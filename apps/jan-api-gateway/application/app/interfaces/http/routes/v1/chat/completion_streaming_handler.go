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
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

// Constants for streaming configuration
const (
	RequestTimeout    = 120 * time.Second
	ChannelBufferSize = 100
	ErrorBufferSize   = 10
	DataPrefix        = "data: "
	DoneMarker        = "[DONE]"
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

// FunctionCallAccumulator handles streaming function call accumulation
type FunctionCallAccumulator struct {
	Name      string
	Arguments string
	Complete  bool
}

// ToolCallAccumulator handles streaming tool call accumulation
type ToolCallAccumulator struct {
	ID       string
	Type     string
	Index    int
	Function struct {
		Name      string
		Arguments string
	}
	Complete bool
}

// StreamCompletionAndAccumulateResponse streams SSE events to client and accumulates a complete response for internal processing
func (s *CompletionStreamHandler) StreamCompletionAndAccumulateResponse(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest, conv *conversation.Conversation, conversationCreated bool, askItemID string, completionItemID string) (*ExtendedCompletionResponse, *common.Error) {
	// Add timeout context
	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), RequestTimeout)
	defer cancel()

	// Set up SSE headers
	s.setupSSEHeaders(reqCtx)

	// Send conversation metadata event first
	if conv != nil {
		if err := s.sendConversationMetadata(reqCtx, conv, conversationCreated, askItemID, completionItemID); err != nil {
			return nil, common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
		}
	}

	// Create buffered channels for data and errors
	dataChan := make(chan string, ChannelBufferSize)
	errChan := make(chan error, ErrorBufferSize)

	var wg sync.WaitGroup
	wg.Add(1)

	// Start streaming in a goroutine
	go s.streamResponseToChannel(ctx, apiKey, request, dataChan, errChan, &wg)

	// Accumulators for different types of content
	var fullContent string
	var fullReasoning string
	var functionCallAccumulator = make(map[int]*FunctionCallAccumulator)
	var toolCallAccumulator = make(map[int]*ToolCallAccumulator)

	// Process data from channels
	streamingComplete := false
	for !streamingComplete {
		select {
		case line, ok := <-dataChan:
			if !ok {
				// Channel closed, streaming complete
				streamingComplete = true
				break
			}

			// Forward the raw line to client
			if err := s.writeSSELine(reqCtx, line); err != nil {
				return nil, common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
			}

			if data, found := strings.CutPrefix(line, DataPrefix); found {
				if data == DoneMarker {
					streamingComplete = true
					break
				}

				// Process stream chunk and accumulate content
				contentChunk, reasoningChunk, functionCallChunk, toolCallChunk := s.processStreamChunkForChannel(data)

				// Accumulate content
				if contentChunk != "" {
					fullContent += contentChunk
				}

				// Accumulate reasoning
				if reasoningChunk != "" {
					fullReasoning += reasoningChunk
				}

				// Handle function call accumulation
				if functionCallChunk != nil {
					s.handleStreamingFunctionCall(functionCallChunk, functionCallAccumulator)
				}

				// Handle tool call accumulation
				if toolCallChunk != nil {
					s.handleStreamingToolCall(toolCallChunk, toolCallAccumulator)
				}
			}

		case err, ok := <-errChan:
			if !ok {
				// Channel closed, no more errors
				continue
			}
			if err != nil {
				return nil, common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
			}

		case <-ctx.Done():
			return nil, common.NewError(ctx.Err(), "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
		}
	}

	// Wait for streaming goroutine to complete and close channels
	wg.Wait()

	close(dataChan)
	close(errChan)

	// Build the complete response
	response := s.buildCompleteResponse(fullContent, fullReasoning, functionCallAccumulator, toolCallAccumulator, completionItemID, request.Model, request)

	// Return as ExtendedCompletionResponse
	return &ExtendedCompletionResponse{
		ChatCompletionResponse: response,
	}, nil
}

// streamResponseToChannel streams the response from inference provider to channels
func (s *CompletionStreamHandler) streamResponseToChannel(ctx context.Context, apiKey string, request openai.ChatCompletionRequest, dataChan chan<- string, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	// Get streaming reader from inference provider
	reader, err := s.inferenceProvider.CreateCompletionStream(ctx, apiKey, request)
	if err != nil {
		errChan <- err
		return
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			// Log the close error but don't send it to errChan to avoid overriding the original error
			// In a production environment, you might want to use a proper logger here
			logger.GetLogger().Errorf("unable to close reader: %v", closeErr)
		}
	}()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
			line := scanner.Text()
			dataChan <- line
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
		return
	}
}

// setupSSEHeaders sets up the required headers for Server-Sent Events
func (s *CompletionStreamHandler) setupSSEHeaders(reqCtx *gin.Context) {
	reqCtx.Header("Content-Type", "text/event-stream")
	reqCtx.Header("Cache-Control", "no-cache")
	reqCtx.Header("Connection", "keep-alive")
	reqCtx.Header("Access-Control-Allow-Origin", "*")
	reqCtx.Header("Access-Control-Allow-Headers", "Cache-Control")
}

// writeSSELine writes a line to the SSE stream
func (s *CompletionStreamHandler) writeSSELine(reqCtx *gin.Context, line string) error {
	_, err := reqCtx.Writer.Write([]byte(line + "\n"))
	if err != nil {
		return err
	}
	reqCtx.Writer.Flush()
	return nil
}

// writeSSEEvent writes a properly formatted SSE event
func (s *CompletionStreamHandler) writeSSEEvent(reqCtx *gin.Context, data string) error {
	_, err := reqCtx.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
	if err != nil {
		return err
	}
	reqCtx.Writer.Flush()
	return nil
}

// sendConversationMetadata sends conversation metadata as SSE event
func (s *CompletionStreamHandler) sendConversationMetadata(reqCtx *gin.Context, conv *conversation.Conversation, conversationCreated bool, askItemID string, completionItemID string) error {
	if conv == nil {
		return nil
	}

	metadata := ResponseMetadata{
		ConversationID:      conv.PublicID,
		ConversationCreated: conversationCreated,
		ConversationTitle:   *conv.Title,
		AskItemId:           askItemID,
		CompletionItemId:    completionItemID,
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// Send proper SSE formatted event with double newline
	return s.writeSSEEvent(reqCtx, string(jsonData))
}

// processStreamChunkForChannel processes a single stream chunk and returns separate chunks
func (s *CompletionStreamHandler) processStreamChunkForChannel(data string) (string, string, *openai.FunctionCall, *openai.ToolCall) {
	// Parse the JSON data to extract content and calls
	var streamData struct {
		Choices []struct {
			Delta struct {
				Content          string               `json:"content"`
				ReasoningContent string               `json:"reasoning_content"`
				FunctionCall     *openai.FunctionCall `json:"function_call,omitempty"`
				ToolCalls        []openai.ToolCall    `json:"tool_calls,omitempty"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(data), &streamData); err != nil {
		// Log JSON parsing errors for debugging
		logger.GetLogger().Errorf("failed to parse stream chunk JSON: %v, data: %s", err, data)
		return "", "", nil, nil
	}

	// Extract content, reasoning content, function calls, and tool calls from all choices
	var contentChunk string
	var reasoningChunk string
	var functionCall *openai.FunctionCall
	var toolCall *openai.ToolCall

	for _, choice := range streamData.Choices {
		// Check for regular content
		if choice.Delta.Content != "" {
			contentChunk += choice.Delta.Content
		}

		// Check for reasoning content
		if choice.Delta.ReasoningContent != "" {
			reasoningChunk += choice.Delta.ReasoningContent
		}

		// Extract function calls (legacy format)
		if choice.Delta.FunctionCall != nil {
			functionCall = choice.Delta.FunctionCall
		}

		// Extract tool calls (new format)
		if len(choice.Delta.ToolCalls) > 0 {
			toolCall = &choice.Delta.ToolCalls[0]
		}
	}

	// Return separate chunks
	return contentChunk, reasoningChunk, functionCall, toolCall
}

// handleStreamingFunctionCall handles function call accumulation
func (s *CompletionStreamHandler) handleStreamingFunctionCall(functionCall *openai.FunctionCall, accumulator map[int]*FunctionCallAccumulator) {
	if functionCall == nil {
		return
	}

	// Use index 0 for function calls (legacy format doesn't have index)
	index := 0
	if accumulator[index] == nil {
		accumulator[index] = &FunctionCallAccumulator{}
	}

	// Add chunk to accumulator
	if functionCall.Name != "" {
		accumulator[index].Name = functionCall.Name
	}
	if functionCall.Arguments != "" {
		accumulator[index].Arguments += functionCall.Arguments
	}

	// Check if complete (has name and arguments ending with })
	if accumulator[index].Name != "" && accumulator[index].Arguments != "" && strings.HasSuffix(accumulator[index].Arguments, "}") {
		accumulator[index].Complete = true
	}
}

// handleStreamingToolCall handles tool call accumulation
func (s *CompletionStreamHandler) handleStreamingToolCall(toolCall *openai.ToolCall, accumulator map[int]*ToolCallAccumulator) {
	if toolCall == nil || toolCall.Index == nil {
		return
	}

	index := *toolCall.Index
	if accumulator[index] == nil {
		accumulator[index] = &ToolCallAccumulator{
			ID:    toolCall.ID,
			Type:  string(toolCall.Type),
			Index: index,
		}
	}

	// Add chunk to accumulator
	if toolCall.Function.Name != "" {
		accumulator[index].Function.Name = toolCall.Function.Name
	}
	if toolCall.Function.Arguments != "" {
		accumulator[index].Function.Arguments += toolCall.Function.Arguments
	}

	// Check if complete (has name and arguments ending with })
	if accumulator[index].Function.Name != "" && accumulator[index].Function.Arguments != "" && strings.HasSuffix(accumulator[index].Function.Arguments, "}") {
		accumulator[index].Complete = true
	}
}

// buildCompleteResponse builds the complete ChatCompletionResponse from accumulated data
func (s *CompletionStreamHandler) buildCompleteResponse(content string, reasoning string, functionCallAccumulator map[int]*FunctionCallAccumulator, toolCallAccumulator map[int]*ToolCallAccumulator, completionItemID string, model string, request openai.ChatCompletionRequest) openai.ChatCompletionResponse {
	// Build a single choice that combines all content, reasoning, and calls
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content,
	}

	// Add reasoning content if present
	if reasoning != "" {
		message.ReasoningContent = reasoning
	}

	var finishReason openai.FinishReason = openai.FinishReasonStop

	// Check for function calls first (legacy format)
	if len(functionCallAccumulator) > 0 {
		for _, acc := range functionCallAccumulator {
			if acc.Complete {
				message.FunctionCall = &openai.FunctionCall{
					Name:      acc.Name,
					Arguments: acc.Arguments,
				}
				finishReason = openai.FinishReasonFunctionCall
				break
			}
		}
	}

	// Check for tool calls (new format) - these take precedence over function calls
	if len(toolCallAccumulator) > 0 {
		var toolCalls []openai.ToolCall
		for _, acc := range toolCallAccumulator {
			if acc.Complete {
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   acc.ID,
					Type: openai.ToolType(acc.Type),
					Function: openai.FunctionCall{
						Name:      acc.Function.Name,
						Arguments: acc.Function.Arguments,
					},
				})
			}
		}

		if len(toolCalls) > 0 {
			message.ToolCalls = toolCalls
			finishReason = openai.FinishReasonToolCalls
		}
	}

	// Create the single choice with all combined content
	choices := []openai.ChatCompletionChoice{
		{
			Index:        0,
			Message:      message,
			FinishReason: finishReason,
		},
	}

	// Calculate token usage
	promptTokens := s.estimateTokens(request.Messages)
	completionTokens := s.estimateTokens([]openai.ChatCompletionMessage{message})
	totalTokens := promptTokens + completionTokens

	return openai.ChatCompletionResponse{
		ID:      completionItemID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: choices,
		Usage: openai.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
	}
}

// TODO it's raw solution, we need to use the official openai tokenizer like tiktoken
// estimateTokens provides a rough estimation of token count for messages
func (s *CompletionStreamHandler) estimateTokens(messages []openai.ChatCompletionMessage) int {
	var allText strings.Builder

	for _, msg := range messages {
		allText.WriteString(msg.Content)
		allText.WriteString(" ")

		if msg.FunctionCall != nil {
			allText.WriteString(msg.FunctionCall.Name)
			allText.WriteString(" ")
			allText.WriteString(msg.FunctionCall.Arguments)
			allText.WriteString(" ")
		}

		for _, toolCall := range msg.ToolCalls {
			allText.WriteString(toolCall.ID)
			allText.WriteString(" ")
			allText.WriteString(toolCall.Function.Name)
			allText.WriteString(" ")
			allText.WriteString(toolCall.Function.Arguments)
			allText.WriteString(" ")
		}
	}

	// Split by spaces and count words, but normalize whitespace
	normalized := strings.Join(strings.Fields(allText.String()), " ") // Collapse multiple spaces
	words := strings.Fields(normalized)
	return len(words)
}
