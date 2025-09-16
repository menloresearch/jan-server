package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/inference"
	"menlo.ai/jan-api-gateway/app/domain/user"
)

// StreamingService handles streaming chat completions
type StreamingService struct {
	inferenceProvider   inference.InferenceProvider
	conversationService *conversation.ConversationService
}

// NewStreamingService creates a new StreamingService
func NewStreamingService(inferenceProvider inference.InferenceProvider, conversationService *conversation.ConversationService) *StreamingService {
	return &StreamingService{
		inferenceProvider:   inferenceProvider,
		conversationService: conversationService,
	}
}

// StreamCompletion handles streaming chat completion
func (s *StreamingService) StreamCompletion(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest, conv *conversation.Conversation, user *user.User) *common.Error {
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

	// Variables to collect full response
	var fullResponse string

	// Use simplified streaming logic that sends raw data and collects content
	streamErr := s.streamCompletionWithCallbacks(reqCtx, request, func(rawData string, contentChunk string) {
		// Send the raw OpenAI streaming data to client
		reqCtx.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", rawData)))
		reqCtx.Writer.Flush()

		// Collect content for conversation saving
		if contentChunk != "" {
			fullResponse += contentChunk
			fmt.Printf("DEBUG: Collected content chunk: '%s', total length: %d\n", contentChunk, len(fullResponse))
		}
	})

	if streamErr != nil {
		fmt.Printf("DEBUG: Streaming error: %s\n", streamErr.Message)
		return streamErr
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

	return common.EmptyError
}

// streamCompletionWithCallbacks handles streaming with callback functions for raw data and content
func (s *StreamingService) streamCompletionWithCallbacks(reqCtx *gin.Context, request openai.ChatCompletionRequest, onData func(string, string)) *common.Error {
	// Get streaming reader from inference provider
	reader, err := s.inferenceProvider.CreateCompletionStream(reqCtx.Request.Context(), "", request)
	if err != nil {
		return common.NewError("a1b2c3d4-e5f6-7890-abcd-ef1234567890", fmt.Sprintf("Failed to create completion stream: %v", err))
	}
	defer reader.Close()

	// Process the stream line by line
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// Check if context was cancelled
		if s.checkContextCancellation(reqCtx.Request.Context(), make(chan error, 1)) {
			return common.NewError("b2c3d4e5-f6g7-8901-bcde-f23456789012", "Context cancelled")
		}

		line := scanner.Text()
		if data, found := strings.CutPrefix(line, DataPrefix); found {
			if data == DoneMarker {
				break
			}

			// Send raw data and extract content
			s.processStreamChunk(data, onData)
		}
	}

	if err := scanner.Err(); err != nil {
		return common.NewError("c3d4e5f6-g7h8-9012-cdef-345678901234", fmt.Sprintf("Scanner error: %v", err))
	}

	return common.EmptyError
}

// processStreamChunk processes a single stream chunk and calls appropriate callbacks
func (s *StreamingService) processStreamChunk(data string, onData func(string, string)) {
	// Parse the JSON data to extract content
	var streamData struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(data), &streamData); err != nil {
		fmt.Printf("DEBUG: JSON parsing failed for data: %s, error: %v\n", data, err)
		// If JSON parsing fails, still send raw data but with empty content
		if onData != nil {
			onData(data, "")
		}
		return
	}

	// Debug: Print the actual JSON structure for first few chunks
	if len(data) < 300 { // Only print for smaller chunks to avoid spam
		fmt.Printf("DEBUG: Raw JSON data: %s\n", data)
	}

	// Extract content from all choices
	var contentChunk string
	for i, choice := range streamData.Choices {
		if choice.Delta.Content != "" {
			contentChunk += choice.Delta.Content
			fmt.Printf("DEBUG: Found content in choice %d: '%s'\n", i, choice.Delta.Content)
		}
	}

	// Debug: Show if we have choices but no content
	if len(streamData.Choices) > 0 && contentChunk == "" {
		fmt.Printf("DEBUG: Has %d choices but no content. First choice delta: %+v\n", len(streamData.Choices), streamData.Choices[0].Delta)
	}

	fmt.Printf("DEBUG: Processed chunk - raw data length: %d, content chunk: '%s'\n", len(data), contentChunk)

	// Send raw data and extracted content
	if onData != nil {
		onData(data, contentChunk)
	}
}

// checkContextCancellation checks if context was cancelled and sends error to channel
func (s *StreamingService) checkContextCancellation(ctx context.Context, errChan chan<- error) bool {
	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return true
	default:
		return false
	}
}

// generateAssistantItemID generates a unique ID for the assistant message item
func (s *StreamingService) generateAssistantItemID() string {
	// For now, use a simple UUID-like string
	// TODO: Use conversation service's ID generation when method is made public
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// saveAssistantMessageToConversation saves the complete assistant message to the conversation
func (s *StreamingService) saveAssistantMessageToConversation(ctx context.Context, conv *conversation.Conversation, user *user.User, itemID string, content string) {
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
	if !err.IsEmpty() {
		// Log error but don't fail the streaming
		fmt.Printf("Warning: Failed to save assistant message to conversation: %s\n", err.Message)
	}
}
