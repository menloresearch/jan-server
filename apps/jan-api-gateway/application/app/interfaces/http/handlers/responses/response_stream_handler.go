package responses

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/response"
	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	responsetypes "menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

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

	// More robust completion check
	fca.Complete = fca.Name != "" && fca.Arguments != "" &&
		(strings.HasSuffix(fca.Arguments, "}") ||
			strings.Count(fca.Arguments, "{") == strings.Count(fca.Arguments, "}"))
}

// StreamHandler handles streaming response requests
type StreamHandler struct {
	*ResponseHandler
}

// NewStreamHandler creates a new StreamHandler instance
func NewStreamHandler(responseHandler *ResponseHandler) *StreamHandler {
	return &StreamHandler{
		ResponseHandler: responseHandler,
	}
}

// Constants for streaming configuration
const (
	RequestTimeout    = 120 * time.Second
	MinWordsPerChunk  = 5
	DataPrefix        = "data: "
	DoneMarker        = "[DONE]"
	SSEEventFormat    = "event: %s\ndata: %s\n\n"
	SSEDataFormat     = "data: %s\n\n"
	ChannelBufferSize = 100
	ErrorBufferSize   = 10
)

// validateRequest validates the incoming request
func (h *StreamHandler) validateRequest(request *requesttypes.CreateResponseRequest) error {
	if request.Model == "" {
		return fmt.Errorf("model is required")
	}
	if request.Input == nil {
		return fmt.Errorf("input is required")
	}
	return nil
}

// checkContextCancellation checks if context was cancelled and sends error to channel
func (h *StreamHandler) checkContextCancellation(ctx context.Context, errChan chan<- error) bool {
	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return true
	default:
		return false
	}
}

// marshalAndSendEvent marshals data and sends it to the data channel with proper error handling
func (h *StreamHandler) marshalAndSendEvent(dataChan chan<- string, eventType string, data any) {
	eventJSON, err := json.Marshal(data)
	if err != nil {
		logger.GetLogger().Errorf("Failed to marshal event: %v", err)
		return
	}
	dataChan <- fmt.Sprintf(SSEEventFormat, eventType, string(eventJSON))
}

// logStreamingMetrics logs streaming completion metrics
func (h *StreamHandler) logStreamingMetrics(responseID string, startTime time.Time, wordCount int) {
	duration := time.Since(startTime)
	logger.GetLogger().Infof("Streaming completed - ID: %s, Duration: %v, Words: %d",
		responseID, duration, wordCount)
}

// createTextDeltaEvent creates a text delta event
func (h *StreamHandler) createTextDeltaEvent(itemID string, sequenceNumber int, delta string) responsetypes.ResponseOutputTextDeltaEvent {
	return responsetypes.ResponseOutputTextDeltaEvent{
		BaseStreamingEvent: responsetypes.BaseStreamingEvent{
			Type:           "response.output_text.delta",
			SequenceNumber: sequenceNumber,
		},
		ItemID:       itemID,
		OutputIndex:  0,
		ContentIndex: 0,
		Delta:        delta,
		Logprobs:     []responsetypes.Logprob{},
	}
}

// createFunctionCallEvent creates a function call delta event
func (h *StreamHandler) createFunctionCallEvent(itemID string, sequenceNumber int, name string, arguments map[string]any) responsetypes.ResponseOutputFunctionCallsDeltaEvent {
	return responsetypes.ResponseOutputFunctionCallsDeltaEvent{
		BaseStreamingEvent: responsetypes.BaseStreamingEvent{
			Type:           "response.output_function_calls.delta",
			SequenceNumber: sequenceNumber,
		},
		ItemID:       itemID,
		OutputIndex:  0,
		ContentIndex: 0,
		Delta: responsetypes.FunctionCallDelta{
			Name:      name,
			Arguments: arguments,
		},
		Logprobs: []responsetypes.Logprob{},
	}
}

// CreateStreamResponse handles the business logic for creating a streaming response
func (h *StreamHandler) CreateStreamResponse(reqCtx *gin.Context, request *requesttypes.CreateResponseRequest, key string, conv *conversation.Conversation, responseEntity *response.Response, chatCompletionRequest *openai.ChatCompletionRequest) {
	// Validate request
	if err := h.validateRequest(request); err != nil {
		reqCtx.JSON(http.StatusBadRequest, responsetypes.ErrorResponse{
			Code: "019929ec-6f89-76c5-8ed4-bd0eb1c6c8db",
		})
		return
	}

	// Add timeout context
	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), RequestTimeout)
	defer cancel()

	// Use ctx for long-running operations
	reqCtx.Request = reqCtx.Request.WithContext(ctx)

	// Set up streaming headers (matching completion API format)
	reqCtx.Header("Content-Type", "text/event-stream")
	reqCtx.Header("Cache-Control", "no-cache")
	reqCtx.Header("Connection", "keep-alive")
	reqCtx.Header("Access-Control-Allow-Origin", "*")
	reqCtx.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Use the public ID from the response entity
	responseID := responseEntity.PublicID

	// Create conversation info
	var conversationInfo *responsetypes.ConversationInfo
	if conv != nil {
		conversationInfo = &responsetypes.ConversationInfo{
			ID: conv.PublicID,
		}
	}

	// Convert input back to the original format for response
	var responseInput any
	switch v := request.Input.(type) {
	case string:
		responseInput = v
	case []any:
		responseInput = v
	default:
		responseInput = request.Input
	}

	// Create initial response object
	response := responsetypes.Response{
		ID:           responseID,
		Object:       "response",
		Created:      time.Now().Unix(),
		Model:        request.Model,
		Status:       responsetypes.ResponseStatusRunning,
		Input:        responseInput,
		Conversation: conversationInfo,
		Stream:       ptr.ToBool(true),
		Temperature:  request.Temperature,
		TopP:         request.TopP,
		MaxTokens:    request.MaxTokens,
		Metadata:     request.Metadata,
	}

	// Emit response.created event
	h.emitStreamEvent(reqCtx, "response.created", responsetypes.ResponseCreatedEvent{
		BaseStreamingEvent: responsetypes.BaseStreamingEvent{
			Type:           "response.created",
			SequenceNumber: 0,
		},
		Response: response,
	})

	// Note: User messages are already added to conversation by the main response handler
	// No need to add them again here to avoid duplication

	// Process with Jan inference client for streaming
	janInferenceClient := janinference.NewJanInferenceClient(reqCtx)
	err := h.processStreamingResponse(reqCtx, janInferenceClient, key, *chatCompletionRequest, responseID, conv)
	if err != nil {
		// Check if context was cancelled (timeout)
		if reqCtx.Request.Context().Err() == context.DeadlineExceeded {
			h.emitStreamEvent(reqCtx, "response.error", responsetypes.ResponseErrorEvent{
				Event:      "response.error",
				Created:    time.Now().Unix(),
				ResponseID: responseID,
				Data: responsetypes.ResponseError{
					Code: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				},
			})
		} else if reqCtx.Request.Context().Err() == context.Canceled {
			h.emitStreamEvent(reqCtx, "response.error", responsetypes.ResponseErrorEvent{
				Event:      "response.error",
				Created:    time.Now().Unix(),
				ResponseID: responseID,
				Data: responsetypes.ResponseError{
					Code: "b2c3d4e5-f6g7-8901-bcde-f23456789012",
				},
			})
		} else {
			h.emitStreamEvent(reqCtx, "response.error", responsetypes.ResponseErrorEvent{
				Event:      "response.error",
				Created:    time.Now().Unix(),
				ResponseID: responseID,
				Data: responsetypes.ResponseError{
					Code: "c3af973c-eada-4e8b-96d9-e92546588cd3",
				},
			})
		}
		return
	}

	// Emit response.completed event
	response.Status = responsetypes.ResponseStatusCompleted
	h.emitStreamEvent(reqCtx, "response.completed", responsetypes.ResponseCompletedEvent{
		BaseStreamingEvent: responsetypes.BaseStreamingEvent{
			Type:           "response.completed",
			SequenceNumber: 9999, // High number to indicate completion
		},
		Response: response,
	})
}

// emitStreamEvent emits a streaming event (matching completion API SSE format)
func (h *StreamHandler) emitStreamEvent(reqCtx *gin.Context, eventType string, data any) {
	// Marshal the data directly without wrapping
	eventJSON, err := json.Marshal(data)
	if err != nil {
		logger.GetLogger().Errorf("Failed to marshal streaming event: %v", err)
		return
	}

	// Use proper SSE format
	reqCtx.Writer.Write([]byte(fmt.Sprintf(SSEEventFormat, eventType, string(eventJSON))))
	reqCtx.Writer.Flush()
}

// processStreamingResponse processes the streaming response from Jan inference using two channels
func (h *StreamHandler) processStreamingResponse(reqCtx *gin.Context, _ *janinference.JanInferenceClient, _ string, request openai.ChatCompletionRequest, responseID string, conv *conversation.Conversation) error {
	// Create buffered channels for data and errors
	dataChan := make(chan string, ChannelBufferSize)
	errChan := make(chan error, ErrorBufferSize)

	var wg sync.WaitGroup
	wg.Add(1)

	// Start streaming in a goroutine
	go h.streamResponseToChannel(reqCtx, request, dataChan, errChan, responseID, conv, &wg)

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
				reqCtx.AbortWithStatusJSON(
					http.StatusBadRequest,
					responsetypes.ErrorResponse{
						Code: "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4",
					})
				return err
			}
			reqCtx.Writer.Flush()
		case err := <-errChan:
			if err != nil {
				reqCtx.AbortWithStatusJSON(
					http.StatusBadRequest,
					responsetypes.ErrorResponse{
						Code: "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4",
					})
				return err
			}
		}
	}
}

// OpenAIStreamData represents the structure of OpenAI streaming data
type OpenAIStreamData struct {
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

// parseOpenAIStreamData parses OpenAI streaming data and extracts content and function call
func (h *StreamHandler) parseOpenAIStreamData(jsonStr string) (string, *openai.FunctionCall) {
	var data OpenAIStreamData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", nil
	}

	// Check if choices array is empty to prevent panic
	if len(data.Choices) == 0 {
		return "", nil
	}

	// Use reasoning_content if content is empty (jan-v1-4b model format)
	content := data.Choices[0].Delta.Content
	if content == "" {
		content = data.Choices[0].Delta.ReasoningContent
	}

	functionCall := data.Choices[0].Delta.FunctionCall

	// Handle tool_calls format
	if functionCall == nil && len(data.Choices[0].Delta.ToolCalls) > 0 {
		toolCall := data.Choices[0].Delta.ToolCalls[0]
		functionCall = &openai.FunctionCall{
			Name:      toolCall.Function.Name,
			Arguments: toolCall.Function.Arguments,
		}
	}

	return content, functionCall
}

// extractContentFromOpenAIStream extracts content from OpenAI streaming format
func (h *StreamHandler) extractContentFromOpenAIStream(chunk string) (string, *openai.FunctionCall) {
	// Format 1: data: {"choices":[{"delta":{"content":"chunk"}}]}
	if len(chunk) >= 6 && chunk[:6] == DataPrefix {
		return h.parseOpenAIStreamData(chunk[6:])
	}

	// Format 2: Direct JSON without "data: " prefix
	if content, functionCall := h.parseOpenAIStreamData(chunk); content != "" || functionCall != nil {
		return content, functionCall
	}

	// Format 3: Simple content string (fallback)
	if len(chunk) > 0 && chunk[0] == '"' && chunk[len(chunk)-1] == '"' {
		var content string
		if err := json.Unmarshal([]byte(chunk), &content); err == nil {
			return content, nil
		}
	}

	return "", nil
}

// streamResponseToChannel handles the streaming response and sends data/errors to channels
func (h *StreamHandler) streamResponseToChannel(reqCtx *gin.Context, request openai.ChatCompletionRequest, dataChan chan<- string, errChan chan<- error, responseID string, conv *conversation.Conversation, wg *sync.WaitGroup) {
	defer wg.Done()

	startTime := time.Now()

	// Generate item ID for the message
	itemID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	sequenceNumber := 1

	// Emit response.in_progress event
	inProgressEvent := responsetypes.ResponseInProgressEvent{
		BaseStreamingEvent: responsetypes.BaseStreamingEvent{
			Type:           "response.in_progress",
			SequenceNumber: sequenceNumber,
		},
		Response: map[string]any{
			"id":     responseID,
			"status": "in_progress",
		},
	}
	eventJSON, _ := json.Marshal(inProgressEvent)
	dataChan <- fmt.Sprintf("event: response.in_progress\ndata: %s\n\n", string(eventJSON))
	sequenceNumber++

	// Emit response.output_item.added event
	outputItemAddedEvent := responsetypes.ResponseOutputItemAddedEvent{
		BaseStreamingEvent: responsetypes.BaseStreamingEvent{
			Type:           "response.output_item.added",
			SequenceNumber: sequenceNumber,
		},
		OutputIndex: 0,
		Item: responsetypes.ResponseOutputItem{
			ID:      itemID,
			Type:    "message",
			Status:  "in_progress",
			Content: []responsetypes.ResponseContentPart{},
			Role:    "assistant",
		},
	}
	eventJSON, _ = json.Marshal(outputItemAddedEvent)
	dataChan <- fmt.Sprintf("event: response.output_item.added\ndata: %s\n\n", string(eventJSON))
	sequenceNumber++

	// Emit response.content_part.added event
	contentPartAddedEvent := responsetypes.ResponseContentPartAddedEvent{
		BaseStreamingEvent: responsetypes.BaseStreamingEvent{
			Type:           "response.content_part.added",
			SequenceNumber: sequenceNumber,
		},
		ItemID:       itemID,
		OutputIndex:  0,
		ContentIndex: 0,
		Part: responsetypes.ResponseContentPart{
			Type:        "output_text",
			Annotations: []responsetypes.Annotation{},
			Logprobs:    []responsetypes.Logprob{},
			Text:        "",
		},
	}
	eventJSON, _ = json.Marshal(contentPartAddedEvent)
	dataChan <- fmt.Sprintf("event: response.content_part.added\ndata: %s\n\n", string(eventJSON))
	sequenceNumber++

	// Create a custom streaming client that processes OpenAI streaming format
	req := janinference.JanInferenceRestyClient.R().SetBody(request)
	resp, err := req.
		SetContext(reqCtx.Request.Context()).
		SetDoNotParseResponse(true).
		Post("/v1/chat/completions")
	if err != nil {
		errChan <- err
		return
	}
	defer resp.RawResponse.Body.Close()

	// Use FunctionCallAccumulator for better function call handling
	accumulator := &FunctionCallAccumulator{}

	// Buffer for accumulating content chunks
	var contentBuffer strings.Builder
	var fullResponse strings.Builder

	// Process the stream line by line
	scanner := bufio.NewScanner(resp.RawResponse.Body)
	for scanner.Scan() {
		// Check if context was cancelled
		if h.checkContextCancellation(reqCtx, errChan) {
			return
		}

		line := scanner.Text()
		if strings.HasPrefix(line, DataPrefix) {
			data := strings.TrimPrefix(line, DataPrefix)
			if data == DoneMarker {
				break
			}

			// Extract content and function call from OpenAI streaming format
			content, functionCall := h.extractContentFromOpenAIStream(data)

			// Handle content
			if content != "" {
				contentBuffer.WriteString(content)
				fullResponse.WriteString(content)

				// Check if we have enough words to send
				bufferedContent := contentBuffer.String()
				words := strings.Fields(bufferedContent)

				if len(words) >= MinWordsPerChunk {
					// Create delta event using helper method
					deltaEvent := h.createTextDeltaEvent(itemID, sequenceNumber, bufferedContent)
					h.marshalAndSendEvent(dataChan, "response.output_text.delta", deltaEvent)
					sequenceNumber++
					// Clear the buffer
					contentBuffer.Reset()
				}
			}

			// Handle function call if present
			if functionCall != nil && !accumulator.Complete {
				accumulator.AddChunk(functionCall)

				// If function call is complete, emit function call event
				if accumulator.Complete {
					// Parse arguments JSON string to map
					var arguments map[string]any
					if err := json.Unmarshal([]byte(accumulator.Arguments), &arguments); err != nil {
						logger.GetLogger().Errorf("Failed to parse function call arguments: %v", err)
						arguments = map[string]any{"raw": accumulator.Arguments}
					}

					functionCallEvent := h.createFunctionCallEvent(itemID, sequenceNumber, accumulator.Name, arguments)
					h.marshalAndSendEvent(dataChan, "response.output_function_calls.delta", functionCallEvent)
					sequenceNumber++
				}
			}
		}
	}

	// Send any remaining buffered content
	if contentBuffer.Len() > 0 {
		deltaEvent := h.createTextDeltaEvent(itemID, sequenceNumber, contentBuffer.String())
		h.marshalAndSendEvent(dataChan, "response.output_text.delta", deltaEvent)
		sequenceNumber++
	}

	// Append assistant's complete response to conversation
	if fullResponse.Len() > 0 && conv != nil {
		assistantMessage := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: fullResponse.String(),
		}
		if err := h.appendMessagesToConversation(reqCtx, conv, []openai.ChatCompletionMessage{assistantMessage}); err != nil {
			// Log error but don't fail the response
			logger.GetLogger().Errorf("Failed to append assistant response to conversation: %v", err)
		}
	}

	// Emit text done event
	if fullResponse.Len() > 0 {
		doneEvent := responsetypes.ResponseOutputTextDoneEvent{
			BaseStreamingEvent: responsetypes.BaseStreamingEvent{
				Type:           "response.output_text.done",
				SequenceNumber: sequenceNumber,
			},
			ItemID:       itemID,
			OutputIndex:  0,
			ContentIndex: 0,
			Text:         fullResponse.String(),
			Logprobs:     []responsetypes.Logprob{},
		}
		eventJSON, _ := json.Marshal(doneEvent)
		dataChan <- fmt.Sprintf("event: response.output_text.done\ndata: %s\n\n", string(eventJSON))
		sequenceNumber++

		// Emit response.content_part.done event
		contentPartDoneEvent := responsetypes.ResponseContentPartDoneEvent{
			BaseStreamingEvent: responsetypes.BaseStreamingEvent{
				Type:           "response.content_part.done",
				SequenceNumber: sequenceNumber,
			},
			ItemID:       itemID,
			OutputIndex:  0,
			ContentIndex: 0,
			Part: responsetypes.ResponseContentPart{
				Type:        "output_text",
				Annotations: []responsetypes.Annotation{},
				Logprobs:    []responsetypes.Logprob{},
				Text:        fullResponse.String(),
			},
		}
		eventJSON, _ = json.Marshal(contentPartDoneEvent)
		dataChan <- fmt.Sprintf("event: response.content_part.done\ndata: %s\n\n", string(eventJSON))
		sequenceNumber++

		// Emit response.output_item.done event
		outputItemDoneEvent := responsetypes.ResponseOutputItemDoneEvent{
			BaseStreamingEvent: responsetypes.BaseStreamingEvent{
				Type:           "response.output_item.done",
				SequenceNumber: sequenceNumber,
			},
			OutputIndex: 0,
			Item: responsetypes.ResponseOutputItem{
				ID:     itemID,
				Type:   "message",
				Status: "completed",
				Content: []responsetypes.ResponseContentPart{
					{
						Type:        "output_text",
						Annotations: []responsetypes.Annotation{},
						Logprobs:    []responsetypes.Logprob{},
						Text:        fullResponse.String(),
					},
				},
				Role: "assistant",
			},
		}
		eventJSON, _ = json.Marshal(outputItemDoneEvent)
		dataChan <- fmt.Sprintf("event: response.output_item.done\ndata: %s\n\n", string(eventJSON))
		sequenceNumber++
	}

	// Send [DONE] to close the stream
	dataChan <- fmt.Sprintf(SSEDataFormat, DoneMarker)

	// Log streaming metrics
	wordCount := len(strings.Fields(fullResponse.String()))
	h.logStreamingMetrics(responseID, startTime, wordCount)
}
