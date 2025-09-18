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
	var functionCallReasoningContent string
	var hasFunctionCall bool

	// Use FunctionCallAccumulator for better function call handling
	accumulator := &FunctionCallAccumulator{}

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
			processedData, contentChunk, reasoningChunk, chunkFunctionCall := s.processStreamChunkForChannel(data)
			dataChan <- processedData

			// Collect content for conversation saving
			if contentChunk != "" {
				fullResponse += contentChunk
			}

			// Collect reasoning content for conversation saving
			if reasoningChunk != "" {
				// Always accumulate reasoning content for function call
				functionCallReasoningContent += reasoningChunk
			}

			// Handle function call if present
			s.handleStreamingFunctionCall(reqCtx.Request.Context(), chunkFunctionCall, accumulator, conv, user, functionCallReasoningContent, &hasFunctionCall, &functionCallReasoningContent)
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
		return
	}

	// Save the complete assistant message to conversation if we have content
	// Note: reasoning content is saved to function call items, not assistant message items
	if conv != nil && fullResponse != "" && !hasFunctionCall {
		s.saveAssistantMessageToConversation(reqCtx.Request.Context(), conv, user, assistantItemID, fullResponse, "")
	}
}

// handleStreamingFunctionCall handles function call processing in streaming responses
func (s *CompletionStreamHandler) handleStreamingFunctionCall(ctx context.Context, chunkFunctionCall *openai.FunctionCall, accumulator *FunctionCallAccumulator, conv *conversation.Conversation, user *user.User, functionCallReasoningContent string, hasFunctionCall *bool, functionCallReasoningContentPtr *string) {
	if chunkFunctionCall != nil && !accumulator.Complete {
		accumulator.AddChunk(chunkFunctionCall)

		// If function call is complete, save it to conversation
		if accumulator.Complete {
			completeFunctionCall := &openai.FunctionCall{
				Name:      accumulator.Name,
				Arguments: accumulator.Arguments,
			}

			// Save function call to conversation immediately with reasoning content
			s.saveFunctionCallToConversation(ctx, conv, user, completeFunctionCall, functionCallReasoningContent)

			// Mark that we have a function call - reasoning content should not go to assistant message
			*hasFunctionCall = true

			// Reset function call reasoning content after saving
			*functionCallReasoningContentPtr = ""
		}
	}
}

// FunctionCallAccumulator handles streaming function call accumulation
type FunctionCallAccumulator struct {
	Name      string
	Arguments string
	Complete  bool
}

func (fca *FunctionCallAccumulator) AddChunk(functionCall *openai.FunctionCall) {
	if functionCall.Name != "" {
		fca.Name = functionCall.Name
	}
	if functionCall.Arguments != "" {
		fca.Arguments += functionCall.Arguments
	}

	// Check if complete
	if fca.Name != "" && fca.Arguments != "" && strings.HasSuffix(fca.Arguments, "}") {
		fca.Complete = true
	}
}

// extractFunctionCallFromStreamChunk extracts function calls from stream chunk delta
func (s *CompletionStreamHandler) extractFunctionCallFromStreamChunk(delta struct {
	Content          string               `json:"content"`
	ReasoningContent string               `json:"reasoning_content"`
	FunctionCall     *openai.FunctionCall `json:"function_call,omitempty"`
	ToolCalls        []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Index    int    `json:"index"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls,omitempty"`
}) *openai.FunctionCall {
	var functionCall *openai.FunctionCall

	// Check for legacy function calls
	if delta.FunctionCall != nil {
		functionCall = delta.FunctionCall
	}

	// Handle tool_calls format
	if functionCall == nil && len(delta.ToolCalls) > 0 {
		toolCall := delta.ToolCalls[0]
		functionCall = &openai.FunctionCall{
			Name:      toolCall.Function.Name,
			Arguments: toolCall.Function.Arguments,
		}
	}

	return functionCall
}

// processStreamChunkForChannel processes a single stream chunk and returns formatted data, content, reasoning content, and function call
func (s *CompletionStreamHandler) processStreamChunkForChannel(data string) (string, string, string, *openai.FunctionCall) {
	// Parse the JSON data to extract content and function calls
	var streamData struct {
		Choices []struct {
			Delta struct {
				Content          string               `json:"content"`
				ReasoningContent string               `json:"reasoning_content"`
				FunctionCall     *openai.FunctionCall `json:"function_call,omitempty"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Index    int    `json:"index"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls,omitempty"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(data), &streamData); err != nil {
		// If JSON parsing fails, still send raw data but with empty content
		return fmt.Sprintf("data: %s\n\n", data), "", "", nil
	}

	// Extract content, reasoning content, and function calls from all choices
	var contentChunk string
	var reasoningChunk string
	var functionCall *openai.FunctionCall

	for _, choice := range streamData.Choices {
		// Check for regular content
		if choice.Delta.Content != "" {
			contentChunk += choice.Delta.Content
		}

		// Check for reasoning content
		if choice.Delta.ReasoningContent != "" {
			reasoningChunk += choice.Delta.ReasoningContent
		}

		// Extract function calls and tool calls
		functionCall = s.extractFunctionCallFromStreamChunk(choice.Delta)
	}

	// Return formatted data, extracted content, reasoning content, and function call
	return fmt.Sprintf("data: %s\n\n", data), contentChunk, reasoningChunk, functionCall
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

// convertOpenAIMessageToConversationContent converts OpenAI message content to conversation content
func (s *CompletionStreamHandler) convertOpenAIMessageToConversationContent(msg openai.ChatCompletionMessage) []conversation.Content {
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

	return content
}

// isFunctionToolResult checks if a message is a function tool result
func (s *CompletionStreamHandler) isFunctionToolResult(msg openai.ChatCompletionMessage) bool {
	// Check if the message has tool role (MCP function results)
	if msg.Role == openai.ChatMessageRoleTool {
		return true
	}

	// Check if the message has tool_calls or function_call_result indicators
	if len(msg.ToolCalls) > 0 {
		return true
	}

	// Check content for function result patterns
	if msg.Content != "" {
		content := strings.ToLower(msg.Content)
		// Look for common function result patterns
		if strings.Contains(content, "function_result") ||
			strings.Contains(content, "tool_result") ||
			strings.Contains(content, "call_id") {
			return true
		}
	}

	return false
}

// parseFunctionToolResult parses function tool result from message content
func (s *CompletionStreamHandler) parseFunctionToolResult(msg openai.ChatCompletionMessage) *conversation.Content {
	if !s.isFunctionToolResult(msg) {
		return nil
	}

	// Handle tool role messages (MCP function results)
	if msg.Role == openai.ChatMessageRoleTool {
		// Try to parse the JSON content to extract function name and result
		var toolResult map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Content), &toolResult); err == nil {
			// Extract function name from the result structure
			var functionName string
			if searchParams, ok := toolResult["searchParameters"].(map[string]interface{}); ok {
				if query, exists := searchParams["q"]; exists {
					functionName = fmt.Sprintf("google_search (query: %v)", query)
				} else {
					functionName = "google_search"
				}
			} else {
				functionName = "unknown_function"
			}

			resultContent := fmt.Sprintf("MCP Function Result - %s\nResult: %s", functionName, msg.Content)

			return &conversation.Content{
				Type: "text",
				Text: &conversation.Text{
					Value: resultContent,
				},
			}
		}

		// Fallback for non-JSON tool results
		resultContent := fmt.Sprintf("MCP Function Result: %s", msg.Content)

		return &conversation.Content{
			Type: "text",
			Text: &conversation.Text{
				Value: resultContent,
			},
		}
	}

	// If message has tool_calls, create function call result content
	if len(msg.ToolCalls) > 0 {
		toolCall := msg.ToolCalls[0]
		resultContent := fmt.Sprintf("Function Result - Call ID: %s\nType: %s\nResult: %s",
			toolCall.ID, toolCall.Type, msg.Content)
		reasoningContent := fmt.Sprintf("Tool call %s (%s) executed. Call ID: %s. Result: %s",
			toolCall.ID, toolCall.Type, toolCall.ID, msg.Content)

		return &conversation.Content{
			Type: "text",
			Text: &conversation.Text{
				Value: resultContent,
			},
			ReasoningContent: &reasoningContent,
		}
	}

	// For text-based function results, format them appropriately
	resultContent := fmt.Sprintf("Function Result: %s", msg.Content)
	reasoningContent := fmt.Sprintf("Generic function execution completed. Result: %s", msg.Content)

	return &conversation.Content{
		Type: "text",
		Text: &conversation.Text{
			Value: resultContent,
		},
		ReasoningContent: &reasoningContent,
	}
}

// convertOpenAIRoleToConversationRole converts OpenAI role to conversation role
func (s *CompletionStreamHandler) convertOpenAIRoleToConversationRole(role string) conversation.ItemRole {
	switch role {
	case openai.ChatMessageRoleSystem:
		return conversation.ItemRoleSystem
	case openai.ChatMessageRoleUser:
		return conversation.ItemRoleUser
	case openai.ChatMessageRoleAssistant:
		return conversation.ItemRoleAssistant
	case openai.ChatMessageRoleTool:
		return conversation.ItemRoleTool // Tool results use dedicated tool role
	default:
		return conversation.ItemRoleUser
	}
}

// saveInputMessagesToConversation saves input messages to the conversation
func (s *CompletionStreamHandler) saveInputMessagesToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage, userItemID string) {
	if conv == nil {
		return
	}

	// Convert OpenAI messages to conversation items
	for i, msg := range messages {
		role := s.convertOpenAIRoleToConversationRole(msg.Role)
		content := s.convertOpenAIMessageToConversationContent(msg)

		// Check if this is a function tool result
		var itemType conversation.ItemType = conversation.ItemTypeMessage
		if s.isFunctionToolResult(msg) {
			itemType = conversation.ItemTypeFunctionCall
			// Parse function tool result content
			if functionResultContent := s.parseFunctionToolResult(msg); functionResultContent != nil {
				content = []conversation.Content{*functionResultContent}
			}
		}

		// Add item to conversation - use userItemID for the last user message
		if i == len(messages)-1 && msg.Role == openai.ChatMessageRoleUser {
			s.conversationService.AddItemWithID(ctx, conv, userID, itemType, &role, content, userItemID)
		} else {
			s.conversationService.AddItem(ctx, conv, userID, itemType, &role, content)
		}
	}
}

// saveAssistantMessageToConversation saves the complete assistant message to the conversation
func (s *CompletionStreamHandler) saveAssistantMessageToConversation(ctx context.Context, conv *conversation.Conversation, user *user.User, itemID string, content string, reasoningContent string) {
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

	// Add reasoning content if present
	if reasoningContent != "" {
		conversationContent[0].ReasoningContent = &reasoningContent
	}

	// Add the assistant message to conversation with the provided itemID
	assistantRole := conversation.ItemRoleAssistant
	s.conversationService.AddItemWithID(ctx, conv, user.ID, conversation.ItemTypeMessage, &assistantRole, conversationContent, itemID)
}

// saveFunctionCallToConversation saves a function call to the conversation
func (s *CompletionStreamHandler) saveFunctionCallToConversation(ctx context.Context, conv *conversation.Conversation, user *user.User, functionCall *openai.FunctionCall, reasoningContent string) {
	if conv == nil || functionCall == nil {
		return
	}

	// Create function call content structure
	functionCallContent := []conversation.Content{
		{
			Type: "text",
			Text: &conversation.Text{
				Value: fmt.Sprintf("Function: %s\nArguments: %s", functionCall.Name, functionCall.Arguments),
			},
		},
	}

	// Add reasoning content if present
	if reasoningContent != "" {
		functionCallContent[0].ReasoningContent = &reasoningContent
	}

	// Add the function call to conversation as a separate item
	assistantRole := conversation.ItemRoleAssistant
	s.conversationService.AddItem(ctx, conv, user.ID, conversation.ItemTypeFunction, &assistantRole, functionCallContent)
}
