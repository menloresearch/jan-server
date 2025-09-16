package response

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
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	responsetypes "menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// StreamModelService handles streaming response requests
type StreamModelService struct {
	*ResponseModelService
}

// NewStreamModelService creates a new StreamModelService instance
func NewStreamModelService(responseModelService *ResponseModelService) *StreamModelService {
	return &StreamModelService{
		ResponseModelService: responseModelService,
	}
}

// Constants for streaming configuration
const (
	RequestTimeout    = 120 * time.Second
	MinWordsPerChunk  = 6
	DataPrefix        = "data: "
	DoneMarker        = "[DONE]"
	SSEEventFormat    = "event: %s\ndata: %s\n\n"
	SSEDataFormat     = "data: %s\n\n"
	ChannelBufferSize = 100
	ErrorBufferSize   = 10
)

// validateRequest validates the incoming request
func (h *StreamModelService) validateRequest(request *requesttypes.CreateResponseRequest) (bool, *common.Error) {
	if request.Model == "" {
		return false, common.NewErrorWithMessage("Model is required", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	}
	if request.Input == nil {
		return false, common.NewErrorWithMessage("Input is required", "b2c3d4e5-f6g7-8901-bcde-f23456789012")
	}
	return true, nil
}

// checkContextCancellation checks if context was cancelled and sends error to channel
func (h *StreamModelService) checkContextCancellation(ctx context.Context, errChan chan<- error) bool {
	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return true
	default:
		return false
	}
}

// marshalAndSendEvent marshals data and sends it to the data channel with proper error handling
func (h *StreamModelService) marshalAndSendEvent(dataChan chan<- string, eventType string, data any) {
	eventJSON, err := json.Marshal(data)
	if err != nil {
		logger.GetLogger().Errorf("Failed to marshal event: %v", err)
		return
	}
	dataChan <- fmt.Sprintf(SSEEventFormat, eventType, string(eventJSON))
}

// logStreamingMetrics logs streaming completion metrics
func (h *StreamModelService) logStreamingMetrics(responseID string, startTime time.Time, wordCount int) {
	duration := time.Since(startTime)
	logger.GetLogger().Infof("Streaming completed - ID: %s, Duration: %v, Words: %d",
		responseID, duration, wordCount)
}

// createTextDeltaEvent creates a text delta event
func (h *StreamModelService) createTextDeltaEvent(itemID string, sequenceNumber int, delta string) responsetypes.ResponseOutputTextDeltaEvent {
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
		Obfuscation:  fmt.Sprintf("%x", time.Now().UnixNano())[:10], // Simple obfuscation
	}
}

// CreateStreamResponse handles the business logic for creating a streaming response
func (h *StreamModelService) CreateStreamResponse(reqCtx *gin.Context, request *requesttypes.CreateResponseRequest, key string, conv *conversation.Conversation, responseEntity *Response, chatCompletionRequest *openai.ChatCompletionRequest) {
	// Validate request
	success, err := h.validateRequest(request)
	if !success {
		reqCtx.JSON(http.StatusBadRequest, responsetypes.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.GetMessage(),
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
	streamErr := h.processStreamingResponse(reqCtx, janInferenceClient, key, *chatCompletionRequest, responseID, conv)
	if streamErr != nil {
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
func (h *StreamModelService) emitStreamEvent(reqCtx *gin.Context, eventType string, data any) {
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
func (h *StreamModelService) processStreamingResponse(reqCtx *gin.Context, _ *janinference.JanInferenceClient, _ string, request openai.ChatCompletionRequest, responseID string, conv *conversation.Conversation) error {
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
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
	} `json:"choices"`
}

// parseOpenAIStreamData parses OpenAI streaming data and extracts content
func (h *StreamModelService) parseOpenAIStreamData(jsonStr string) string {
	var data OpenAIStreamData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	// Check if choices array is empty to prevent panic
	if len(data.Choices) == 0 {
		return ""
	}

	// Use reasoning_content if content is empty (jan-v1-4b model format)
	content := data.Choices[0].Delta.Content
	if content == "" {
		content = data.Choices[0].Delta.ReasoningContent
	}

	return content
}

// extractContentFromOpenAIStream extracts content from OpenAI streaming format
func (h *StreamModelService) extractContentFromOpenAIStream(chunk string) string {
	// Format 1: data: {"choices":[{"delta":{"content":"chunk"}}]}
	if len(chunk) >= 6 && chunk[:6] == DataPrefix {
		return h.parseOpenAIStreamData(chunk[6:])
	}

	// Format 2: Direct JSON without "data: " prefix
	if content := h.parseOpenAIStreamData(chunk); content != "" {
		return content
	}

	// Format 3: Simple content string (fallback)
	if len(chunk) > 0 && chunk[0] == '"' && chunk[len(chunk)-1] == '"' {
		var content string
		if err := json.Unmarshal([]byte(chunk), &content); err == nil {
			return content
		}
	}

	return ""
}

// extractReasoningContentFromOpenAIStream extracts reasoning content from OpenAI streaming format
func (h *StreamModelService) extractReasoningContentFromOpenAIStream(chunk string) string {
	// Format 1: data: {"choices":[{"delta":{"reasoning_content":"chunk"}}]}
	if len(chunk) >= 6 && chunk[:6] == DataPrefix {
		return h.parseOpenAIStreamReasoningData(chunk[6:])
	}

	// Format 2: Direct JSON without "data: " prefix
	if reasoningContent := h.parseOpenAIStreamReasoningData(chunk); reasoningContent != "" {
		return reasoningContent
	}

	return ""
}

// parseOpenAIStreamReasoningData parses OpenAI streaming data and extracts reasoning content
func (h *StreamModelService) parseOpenAIStreamReasoningData(jsonStr string) string {
	var data OpenAIStreamData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	// Check if choices array is empty to prevent panic
	if len(data.Choices) == 0 {
		return ""
	}

	// Extract reasoning content
	return data.Choices[0].Delta.ReasoningContent
}

// streamResponseToChannel handles the streaming response and sends data/errors to channels
func (h *StreamModelService) streamResponseToChannel(reqCtx *gin.Context, request openai.ChatCompletionRequest, dataChan chan<- string, errChan chan<- error, responseID string, conv *conversation.Conversation, wg *sync.WaitGroup) {
	defer wg.Done()

	startTime := time.Now()

	// Generate item ID for the message
	itemID, _ := idgen.GenerateSecureID("msg", 42)
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
			Status:  string(conversation.ItemStatusInProgress),
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

	// Buffer for accumulating content chunks
	var contentBuffer strings.Builder
	var fullResponse strings.Builder

	// Buffer for accumulating reasoning content chunks
	var reasoningBuffer strings.Builder
	var fullReasoningResponse strings.Builder
	var reasoningItemID string
	var reasoningSequenceNumber int
	var hasReasoningContent bool
	var reasoningComplete bool

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

			// Extract content from OpenAI streaming format
			content := h.extractContentFromOpenAIStream(data)

			// Handle content - buffer until reasoning is complete
			if content != "" {
				contentBuffer.WriteString(content)
				fullResponse.WriteString(content)

				// Only send content if reasoning is complete or there's no reasoning content
				if reasoningComplete || !hasReasoningContent {
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
			}

			// Handle reasoning content separately
			reasoningContent := h.extractReasoningContentFromOpenAIStream(data)
			if reasoningContent != "" {
				// Initialize reasoning item if not already done
				if !hasReasoningContent {
					reasoningItemID = fmt.Sprintf("rs_%d", time.Now().UnixNano())
					reasoningSequenceNumber = sequenceNumber
					hasReasoningContent = true

					// Emit response.output_item.added event for reasoning
					reasoningItemAddedEvent := responsetypes.ResponseOutputItemAddedEvent{
						BaseStreamingEvent: responsetypes.BaseStreamingEvent{
							Type:           "response.output_item.added",
							SequenceNumber: reasoningSequenceNumber,
						},
						OutputIndex: 0,
						Item: responsetypes.ResponseOutputItem{
							ID:      reasoningItemID,
							Type:    "reasoning",
							Status:  string(conversation.ItemStatusInProgress),
							Content: []responsetypes.ResponseContentPart{},
							Role:    "assistant",
						},
					}
					eventJSON, _ := json.Marshal(reasoningItemAddedEvent)
					dataChan <- fmt.Sprintf("event: response.output_item.added\ndata: %s\n\n", string(eventJSON))
					reasoningSequenceNumber++

					// Emit response.reasoning_summary_part.added event
					reasoningSummaryPartAddedEvent := responsetypes.ResponseReasoningSummaryPartAddedEvent{
						BaseStreamingEvent: responsetypes.BaseStreamingEvent{
							Type:           "response.reasoning_summary_part.added",
							SequenceNumber: reasoningSequenceNumber,
						},
						ItemID:       reasoningItemID,
						OutputIndex:  0,
						SummaryIndex: 0,
						Part: struct {
							Type string `json:"type"`
							Text string `json:"text"`
						}{
							Type: "summary_text",
							Text: "",
						},
					}
					eventJSON, _ = json.Marshal(reasoningSummaryPartAddedEvent)
					dataChan <- fmt.Sprintf("event: response.reasoning_summary_part.added\ndata: %s\n\n", string(eventJSON))
					reasoningSequenceNumber++
				}

				reasoningBuffer.WriteString(reasoningContent)
				fullReasoningResponse.WriteString(reasoningContent)

				// Check if we have enough words to send reasoning content
				bufferedReasoningContent := reasoningBuffer.String()
				reasoningWords := strings.Fields(bufferedReasoningContent)

				if len(reasoningWords) >= MinWordsPerChunk {
					// Emit reasoning summary text delta event
					reasoningSummaryTextDeltaEvent := responsetypes.ResponseReasoningSummaryTextDeltaEvent{
						BaseStreamingEvent: responsetypes.BaseStreamingEvent{
							Type:           "response.reasoning_summary_text.delta",
							SequenceNumber: reasoningSequenceNumber,
						},
						ItemID:       reasoningItemID,
						OutputIndex:  0,
						SummaryIndex: 0,
						Delta:        bufferedReasoningContent,
						Obfuscation:  fmt.Sprintf("%x", time.Now().UnixNano())[:10], // Simple obfuscation
					}
					eventJSON, _ := json.Marshal(reasoningSummaryTextDeltaEvent)
					dataChan <- fmt.Sprintf("event: response.reasoning_summary_text.delta\ndata: %s\n\n", string(eventJSON))
					reasoningSequenceNumber++
					// Clear the reasoning buffer
					reasoningBuffer.Reset()
				}
			}

		}
	}

	// Send any remaining buffered reasoning content
	if hasReasoningContent && reasoningBuffer.Len() > 0 {
		reasoningSummaryTextDeltaEvent := responsetypes.ResponseReasoningSummaryTextDeltaEvent{
			BaseStreamingEvent: responsetypes.BaseStreamingEvent{
				Type:           "response.reasoning_summary_text.delta",
				SequenceNumber: reasoningSequenceNumber,
			},
			ItemID:       reasoningItemID,
			OutputIndex:  0,
			SummaryIndex: 0,
			Delta:        reasoningBuffer.String(),
			Obfuscation:  fmt.Sprintf("%x", time.Now().UnixNano())[:10], // Simple obfuscation
		}
		eventJSON, _ := json.Marshal(reasoningSummaryTextDeltaEvent)
		dataChan <- fmt.Sprintf("event: response.reasoning_summary_text.delta\ndata: %s\n\n", string(eventJSON))
		reasoningSequenceNumber++
	}

	// Handle reasoning completion events
	if hasReasoningContent && fullReasoningResponse.Len() > 0 {
		// Emit reasoning summary text done event
		reasoningSummaryTextDoneEvent := responsetypes.ResponseReasoningSummaryTextDoneEvent{
			BaseStreamingEvent: responsetypes.BaseStreamingEvent{
				Type:           "response.reasoning_summary_text.done",
				SequenceNumber: reasoningSequenceNumber,
			},
			ItemID:       reasoningItemID,
			OutputIndex:  0,
			SummaryIndex: 0,
			Text:         fullReasoningResponse.String(),
		}
		eventJSON, _ := json.Marshal(reasoningSummaryTextDoneEvent)
		dataChan <- fmt.Sprintf("event: response.reasoning_summary_text.done\ndata: %s\n\n", string(eventJSON))
		reasoningSequenceNumber++

		// Emit reasoning summary part done event
		reasoningSummaryPartDoneEvent := responsetypes.ResponseReasoningSummaryPartDoneEvent{
			BaseStreamingEvent: responsetypes.BaseStreamingEvent{
				Type:           "response.reasoning_summary_part.done",
				SequenceNumber: reasoningSequenceNumber,
			},
			ItemID:       reasoningItemID,
			OutputIndex:  0,
			SummaryIndex: 0,
			Part: struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				Type: "summary_text",
				Text: fullReasoningResponse.String(),
			},
		}
		eventJSON, _ = json.Marshal(reasoningSummaryPartDoneEvent)
		dataChan <- fmt.Sprintf("event: response.reasoning_summary_part.done\ndata: %s\n\n", string(eventJSON))
		reasoningSequenceNumber++

		// Mark reasoning as complete
		reasoningComplete = true
	}

	// Send any remaining buffered content (only once, after reasoning is complete or if there's no reasoning content)
	if (reasoningComplete || !hasReasoningContent) && contentBuffer.Len() > 0 {
		deltaEvent := h.createTextDeltaEvent(itemID, sequenceNumber, contentBuffer.String())
		h.marshalAndSendEvent(dataChan, "response.output_text.delta", deltaEvent)
		sequenceNumber++
		contentBuffer.Reset()
	}

	// Append assistant's complete response to conversation
	if fullResponse.Len() > 0 && conv != nil {
		assistantMessage := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: fullResponse.String(),
		}
		// Get response entity to get the internal ID
		responseEntity, err := h.responseService.GetResponseByPublicID(reqCtx, responseID)
		if err == nil && responseEntity != nil {
			success, err := h.responseService.AppendMessagesToConversation(reqCtx, conv, []openai.ChatCompletionMessage{assistantMessage}, &responseEntity.ID)
			if !success {
				// Log error but don't fail the response
				logger.GetLogger().Errorf("Failed to append assistant response to conversation: %s - %s", err.GetCode(), err.Error())
			}
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
				Status: string(conversation.ItemStatusCompleted),
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

	// Update response status to completed and save output
	// Get response entity by public ID to update status
	responseEntity, getErr := h.responseService.GetResponseByPublicID(reqCtx, responseID)
	if getErr == nil && responseEntity != nil {
		// Update status to completed
		success, updateErr := h.responseService.UpdateResponseStatus(reqCtx, responseEntity.ID, ResponseStatusCompleted)
		if !success {
			// Log error but don't fail the request since streaming is already complete
			fmt.Printf("Failed to update response status to completed: %s - %s\n", updateErr.GetCode(), updateErr.Error())
		}

		// Save the output to database
		outputData := map[string]interface{}{
			"type": "text",
			"text": map[string]interface{}{
				"value": fullResponse.String(),
			},
		}
		success, outputErr := h.responseService.UpdateResponseOutput(reqCtx, responseEntity.ID, outputData)
		if !success {
			fmt.Printf("Failed to update response output: %s - %s\n", outputErr.GetCode(), outputErr.Error())
		}
	} else {
		fmt.Printf("Failed to get response entity for status update: %s - %s\n", getErr.GetCode(), getErr.Error())
	}

	// Log streaming metrics
	wordCount := len(strings.Fields(fullResponse.String()))
	h.logStreamingMetrics(responseID, startTime, wordCount)
}
