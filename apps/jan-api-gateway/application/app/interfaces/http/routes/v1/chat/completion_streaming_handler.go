package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/inference"
	"menlo.ai/jan-api-gateway/app/domain/user"
)

// Constants for streaming configuration
const (
	RequestTimeout    = 120 * time.Second
	DataPrefix        = "data: "
	DoneMarker        = "[DONE]"
	ChannelBufferSize = 100
	ErrorBufferSize   = 10
)

// CompletionStreamHandler handles streaming chat completions
type CompletionStreamHandler struct {
	inferenceProvider   inference.InferenceProvider
	conversationService *conversation.ConversationService
}

// NewCompletionStreamHandler creates a new CompletionStreamHandler
func NewCompletionStreamHandler(inferenceProvider inference.InferenceProvider, conversationService *conversation.ConversationService) *CompletionStreamHandler {
	return &CompletionStreamHandler{
		inferenceProvider:   inferenceProvider,
		conversationService: conversationService,
	}
}

// StreamCompletion handles streaming chat completion using buffered channels
func (s *CompletionStreamHandler) StreamCompletion(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest, conv *conversation.Conversation, user *user.User) *common.Error {
	// Add timeout context
	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), RequestTimeout)
	defer cancel()

	// Use ctx for long-running operations
	reqCtx.Request = reqCtx.Request.WithContext(ctx)

	// Set up streaming headers
	reqCtx.Header("Content-Type", "text/event-stream")
	reqCtx.Header("Cache-Control", "no-cache")
	reqCtx.Header("Connection", "keep-alive")
	reqCtx.Header("Access-Control-Allow-Origin", "*")
	reqCtx.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Create buffered channels for data and errors
	dataChan := make(chan string, ChannelBufferSize)
	errChan := make(chan error, ErrorBufferSize)

	var wg sync.WaitGroup
	wg.Add(1)

	// Start streaming in a goroutine
	go s.streamResponseToChannel(reqCtx, request, dataChan, errChan, conv, user, &wg)

	// Wait for streaming to complete and close channels
	go func() {
		wg.Wait()
		close(dataChan)
		close(errChan)
	}()

	// Process data and errors from channels
	for {
		select {
		case line, ok := <-dataChan:
			if !ok {
				return nil
			}
			_, err := reqCtx.Writer.Write([]byte(line))
			if err != nil {
				return common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
			}
			reqCtx.Writer.Flush()
		case err := <-errChan:
			if err != nil {
				return common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
			}
		}
	}
}

// streamResponseToChannel handles streaming and sends data to channels
func (s *CompletionStreamHandler) streamResponseToChannel(reqCtx *gin.Context, request openai.ChatCompletionRequest, dataChan chan<- string, errChan chan<- error, conv *conversation.Conversation, user *user.User, wg *sync.WaitGroup) {
	defer wg.Done()

	// Save input messages to conversation first
	if conv != nil {
		s.saveInputMessagesToConversation(reqCtx.Request.Context(), conv, user.ID, request.Messages)
	}

	// Get streaming reader from inference provider
	reader, err := s.inferenceProvider.CreateCompletionStream(reqCtx.Request.Context(), "", request)
	if err != nil {
		errChan <- err
		return
	}
	defer reader.Close()

	// Variables to collect full response for conversation saving
	var fullResponse string

	// Process the stream line by line
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// Check if context was cancelled
		if s.checkContextCancellation(reqCtx.Request.Context(), errChan) {
			return
		}

		line := scanner.Text()
		if data, found := strings.CutPrefix(line, DataPrefix); found {
			if data == DoneMarker {
				break
			}

			// Process stream chunk and send to data channel
			processedData, contentChunk := s.processStreamChunkForChannel(data)
			dataChan <- processedData

			// Collect content for conversation saving
			if contentChunk != "" {
				fullResponse += contentChunk
				fmt.Printf("DEBUG: Collected content chunk: '%s', total length: %d\n", contentChunk, len(fullResponse))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
		return
	}

	fmt.Printf("DEBUG: Streaming completed. Full response length: %d\n", len(fullResponse))

	// Save the complete assistant message to conversation if we have content
	if conv != nil && fullResponse != "" {
		fmt.Printf("DEBUG: Saving assistant message to conversation. Content: '%s'\n", fullResponse)
		assistantItemID := s.generateAssistantItemID()
		s.saveAssistantMessageToConversation(reqCtx.Request.Context(), conv, user, assistantItemID, fullResponse)
	} else {
		fmt.Printf("DEBUG: Not saving message - conv: %v, fullResponse length: %d\n", conv != nil, len(fullResponse))
	}
}

// processStreamChunkForChannel processes a single stream chunk and returns formatted data and content
func (s *CompletionStreamHandler) processStreamChunkForChannel(data string) (string, string) {
	// Parse the JSON data to extract content
	var streamData struct {
		Choices []struct {
			Delta struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				ToolCalls        []struct {
					Function struct {
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(data), &streamData); err != nil {
		fmt.Printf("DEBUG: JSON parsing failed for data: %s, error: %v\n", data, err)
		// If JSON parsing fails, still send raw data but with empty content
		return fmt.Sprintf("data: %s\n\n", data), ""
	}

	// Debug: Print the actual JSON structure for first few chunks
	if len(data) < 300 { // Only print for smaller chunks to avoid spam
		fmt.Printf("DEBUG: Raw JSON data: %s\n", data)
	}

	// Extract content from all choices
	var contentChunk string
	for i, choice := range streamData.Choices {
		// Check for regular content
		if choice.Delta.Content != "" {
			contentChunk += choice.Delta.Content
			fmt.Printf("DEBUG: Found content in choice %d: '%s'\n", i, choice.Delta.Content)
		}

		// Check for reasoning content (internal reasoning, don't save to conversation)
		if choice.Delta.ReasoningContent != "" {
			fmt.Printf("DEBUG: Found reasoning content in choice %d: '%s' (not saving to conversation)\n", i, choice.Delta.ReasoningContent)
		}

		// Check for tool calls
		if len(choice.Delta.ToolCalls) > 0 {
			fmt.Printf("DEBUG: Found %d tool calls in choice %d\n", len(choice.Delta.ToolCalls), i)
			for j, toolCall := range choice.Delta.ToolCalls {
				fmt.Printf("DEBUG: Tool call %d arguments: '%s'\n", j, toolCall.Function.Arguments)
			}
		}
	}

	// Debug: Show if we have choices but no content
	if len(streamData.Choices) > 0 && contentChunk == "" {
		fmt.Printf("DEBUG: Has %d choices but no content. First choice delta: %+v\n", len(streamData.Choices), streamData.Choices[0].Delta)
	}

	fmt.Printf("DEBUG: Processed chunk - raw data length: %d, content chunk: '%s'\n", len(data), contentChunk)

	// Return formatted data and extracted content
	return fmt.Sprintf("data: %s\n\n", data), contentChunk
}

// checkContextCancellation checks if context was cancelled and sends error to channel
func (s *CompletionStreamHandler) checkContextCancellation(ctx context.Context, errChan chan<- error) bool {
	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return true
	default:
		return false
	}
}

// generateAssistantItemID generates a unique ID for the assistant message item
func (s *CompletionStreamHandler) generateAssistantItemID() string {
	// For now, use a simple UUID-like string
	// TODO: Use conversation service's ID generation when method is made public
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// saveInputMessagesToConversation saves input messages to the conversation
func (s *CompletionStreamHandler) saveInputMessagesToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage) {
	if conv == nil {
		return
	}

	// Convert OpenAI messages to conversation items
	for _, msg := range messages {
		// Convert role
		var role conversation.ItemRole
		switch msg.Role {
		case openai.ChatMessageRoleSystem:
			role = conversation.ItemRoleSystem
		case openai.ChatMessageRoleUser:
			role = conversation.ItemRoleUser
		case openai.ChatMessageRoleAssistant:
			role = conversation.ItemRoleAssistant
		default:
			role = conversation.ItemRoleUser
		}

		// Convert content
		content := make([]conversation.Content, 0, len(msg.MultiContent))
		for _, contentPart := range msg.MultiContent {
			if contentPart.Type == openai.ChatMessagePartTypeText {
				content = append(content, conversation.Content{
					Type: "text",
					Text: &conversation.Text{
						Value: contentPart.Text,
					},
				})
			}
		}

		// If no multi-content, use simple text content
		if len(content) == 0 && msg.Content != "" {
			content = append(content, conversation.Content{
				Type: "text",
				Text: &conversation.Text{
					Value: msg.Content,
				},
			})
		}

		// Add item to conversation
		_, err := s.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeMessage, &role, content)
		if err != nil {
			// Log error but don't fail the streaming
			fmt.Printf("Warning: Failed to save input message to conversation: %s\n", err.GetMessage())
		}
	}
}

// saveAssistantMessageToConversation saves the complete assistant message to the conversation
func (s *CompletionStreamHandler) saveAssistantMessageToConversation(ctx context.Context, conv *conversation.Conversation, user *user.User, itemID string, content string) {
	if conv == nil || content == "" {
		return
	}

	// Create content structure
	conversationContent := []conversation.Content{
		{
			Type: "text",
			Text: &conversation.Text{
				Value: content,
			},
		},
	}

	// Add the assistant message to conversation
	assistantRole := conversation.ItemRoleAssistant
	_, err := s.conversationService.AddItem(ctx, conv, user.ID, conversation.ItemTypeMessage, &assistantRole, conversationContent)
	if err != nil {
		// Log error but don't fail the streaming
		fmt.Printf("Warning: Failed to save assistant message to conversation: %s\n", err.GetMessage())
	}
}
