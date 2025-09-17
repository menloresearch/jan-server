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
func (s *CompletionStreamHandler) StreamCompletion(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest, conv *conversation.Conversation, user *user.User, userItemID string, assistantItemID string) *common.Error {
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
	go s.streamResponseToChannel(reqCtx, request, dataChan, errChan, conv, user, userItemID, assistantItemID, &wg)

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
func (s *CompletionStreamHandler) streamResponseToChannel(reqCtx *gin.Context, request openai.ChatCompletionRequest, dataChan chan<- string, errChan chan<- error, conv *conversation.Conversation, user *user.User, userItemID string, assistantItemID string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Save input messages to conversation first
	if conv != nil {
		// Save messages to conversation and get the assistant message item
		var latestMessage []openai.ChatCompletionMessage
		if len(request.Messages) > 0 {
			latestMessage = []openai.ChatCompletionMessage{request.Messages[len(request.Messages)-1]}
		}
		s.saveInputMessagesToConversation(reqCtx.Request.Context(), conv, user.ID, latestMessage, userItemID)
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
			}
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
		return
	}

	// Save the complete assistant message to conversation if we have content
	if conv != nil && fullResponse != "" {
		s.saveAssistantMessageToConversation(reqCtx.Request.Context(), conv, user, assistantItemID, fullResponse)
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
		// If JSON parsing fails, still send raw data but with empty content
		return fmt.Sprintf("data: %s\n\n", data), ""
	}

	// Extract content from all choices
	var contentChunk string
	for _, choice := range streamData.Choices {
		// Check for regular content
		if choice.Delta.Content != "" {
			contentChunk += choice.Delta.Content
		}

		// Check for reasoning content (internal reasoning, don't save to conversation)
		// Note: reasoning_content is not saved to conversation

		// Check for tool calls
		// Note: tool_calls are logged for debugging but not processed here
	}

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

// saveInputMessagesToConversation saves input messages to the conversation
func (s *CompletionStreamHandler) saveInputMessagesToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage, userItemID string) {
	if conv == nil {
		return
	}

	// Convert OpenAI messages to conversation items
	for i, msg := range messages {
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

		// Add item to conversation - use userItemID for the last user message
		if i == len(messages)-1 && msg.Role == openai.ChatMessageRoleUser {
			s.conversationService.AddItemWithID(ctx, conv, userID, conversation.ItemTypeMessage, &role, content, userItemID)
		} else {
			s.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeMessage, &role, content)
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

	// Add the assistant message to conversation with the provided itemID
	assistantRole := conversation.ItemRoleAssistant
	s.conversationService.AddItemWithID(ctx, conv, user.ID, conversation.ItemTypeMessage, &assistantRole, conversationContent, itemID)
}
