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

	// Check if complete
	if fca.Name != "" && fca.Arguments != "" && strings.HasSuffix(fca.Arguments, "}") {
		fca.Complete = true
	}
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

// Constants for timeout management
const (
	RequestTimeout = 120 * time.Second
)

// CreateStreamResponse handles the business logic for creating a streaming response
func (h *StreamHandler) CreateStreamResponse(reqCtx *gin.Context, request *requesttypes.CreateResponseRequest, key string, conv *conversation.Conversation) {
	// Add timeout context
	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), RequestTimeout)
	defer cancel()

	// Use ctx for long-running operations
	reqCtx.Request = reqCtx.Request.WithContext(ctx)
	// Convert response request to chat completion request
	chatCompletionRequest := h.convertToChatCompletionRequest(request)
	if chatCompletionRequest == nil {
		reqCtx.JSON(http.StatusBadRequest, responsetypes.ErrorResponse{
			Code:  "019929ec-6f89-76c5-8ed4-bd0eb1c6c8db",
			Error: "unsupported input type for chat completion",
		})
		return
	}

	// Set up streaming headers (matching completion API format)
	reqCtx.Header("Content-Type", "text/event-stream")
	reqCtx.Header("Cache-Control", "no-cache")
	reqCtx.Header("Connection", "keep-alive")
	reqCtx.Header("Access-Control-Allow-Origin", "*")
	reqCtx.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Generate response ID
	responseID := fmt.Sprintf("response_%d", time.Now().UnixNano())

	// Create conversation info
	var conversationInfo *responsetypes.ConversationInfo
	if conv != nil {
		conversationInfo = &responsetypes.ConversationInfo{
			ID: conv.PublicID,
		}
	}

	// Convert input back to the original format for response
	var responseInput interface{}
	switch v := request.Input.(type) {
	case string:
		responseInput = v
	case []interface{}:
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

	// Add user message to conversation if conversation exists
	if conv != nil {
		// Extract user message from the last message in the chat completion request
		if len(chatCompletionRequest.Messages) > 0 {
			lastMessage := chatCompletionRequest.Messages[len(chatCompletionRequest.Messages)-1]
			if lastMessage.Role == openai.ChatMessageRoleUser {
				userMessage := openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: lastMessage.Content,
				}
				if err := h.appendMessagesToConversation(reqCtx, conv, []openai.ChatCompletionMessage{userMessage}); err != nil {
					// Log error but don't fail the response
					logger.GetLogger().Errorf("Failed to append user message to conversation: %v", err)
				}
			}
		}
	}

	// Process with Jan inference client for streaming
	janInferenceClient := janinference.NewJanInferenceClient(reqCtx)
	err := h.processStreamingResponse(reqCtx, janInferenceClient, key, *chatCompletionRequest, responseID, conv)
	if err != nil {
		// Check if context was cancelled (timeout)
		if ctx.Err() == context.DeadlineExceeded {
			h.emitStreamEvent(reqCtx, "response.error", responsetypes.ResponseErrorEvent{
				Event:      "response.error",
				Created:    time.Now().Unix(),
				ResponseID: responseID,
				Data: responsetypes.ResponseError{
					Code:    "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
					Message: "Request timeout exceeded",
				},
			})
		} else if ctx.Err() == context.Canceled {
			h.emitStreamEvent(reqCtx, "response.error", responsetypes.ResponseErrorEvent{
				Event:      "response.error",
				Created:    time.Now().Unix(),
				ResponseID: responseID,
				Data: responsetypes.ResponseError{
					Code:    "b2c3d4e5-f6g7-8901-bcde-f23456789012",
					Message: "Request was cancelled",
				},
			})
		} else {
			h.emitStreamEvent(reqCtx, "response.error", responsetypes.ResponseErrorEvent{
				Event:      "response.error",
				Created:    time.Now().Unix(),
				ResponseID: responseID,
				Data: responsetypes.ResponseError{
					Code:    "c3af973c-eada-4e8b-96d9-e92546588cd3",
					Message: err.Error(),
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
func (h *StreamHandler) emitStreamEvent(reqCtx *gin.Context, eventType string, data interface{}) {
	// Marshal the data directly without wrapping
	eventJSON, err := json.Marshal(data)
	if err != nil {
		logger.GetLogger().Errorf("Failed to marshal streaming event: %v", err)
		return
	}

	// Use proper SSE format
	reqCtx.Writer.Write([]byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(eventJSON))))
	reqCtx.Writer.Flush()
}

// processStreamingResponse processes the streaming response from Jan inference using two channels
func (h *StreamHandler) processStreamingResponse(reqCtx *gin.Context, _ *janinference.JanInferenceClient, _ string, request openai.ChatCompletionRequest, responseID string, conv *conversation.Conversation) error {
	// Create channels for data and errors
	dataChan := make(chan string)
	errChan := make(chan error)

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
						Code:  "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4",
						Error: err.Error(),
					})
				return err
			}
			reqCtx.Writer.Flush()
		case err := <-errChan:
			if err != nil {
				reqCtx.AbortWithStatusJSON(
					http.StatusBadRequest,
					responsetypes.ErrorResponse{
						Code:  "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4",
						Error: err.Error(),
					})
				return err
			}
		}
	}
}

// extractContentFromOpenAIStream extracts content from OpenAI streaming format
func (h *StreamHandler) extractContentFromOpenAIStream(chunk string) (string, *openai.FunctionCall) {
	// Handle different streaming formats

	// Format 1: data: {"choices":[{"delta":{"content":"chunk"}}]}
	if len(chunk) >= 6 && chunk[:6] == "data: " {
		jsonStr := chunk[6:]

		// Parse the JSON with both function_call and tool_calls support
		var data struct {
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

		if err := json.Unmarshal([]byte(jsonStr), &data); err == nil && len(data.Choices) > 0 {
			// Use reasoning_content if content is empty (jan-v1-4b model format)
			content := data.Choices[0].Delta.Content
			if content == "" {
				content = data.Choices[0].Delta.ReasoningContent
			}

			functionCall := data.Choices[0].Delta.FunctionCall

			// Handle tool_calls format
			if functionCall == nil && len(data.Choices[0].Delta.ToolCalls) > 0 {
				toolCall := data.Choices[0].Delta.ToolCalls[0]

				// Create function call even if name or arguments are empty (they will be accumulated)
				functionCall = &openai.FunctionCall{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				}
			}

			return content, functionCall
		}
	}

	// Format 2: Direct JSON without "data: " prefix
	// Try to parse as direct JSON
	var data struct {
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

	if err := json.Unmarshal([]byte(chunk), &data); err == nil && len(data.Choices) > 0 {
		// Use reasoning_content if content is empty (jan-v1-4b model format)
		content := data.Choices[0].Delta.Content
		if content == "" {
			content = data.Choices[0].Delta.ReasoningContent
		}

		functionCall := data.Choices[0].Delta.FunctionCall

		// Handle tool_calls format
		if functionCall == nil && len(data.Choices[0].Delta.ToolCalls) > 0 {
			toolCall := data.Choices[0].Delta.ToolCalls[0]

			// Create function call even if name or arguments are empty (they will be accumulated)
			functionCall = &openai.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			}
		}

		return content, functionCall
	}

	// Format 3: Simple content string (fallback)
	// If it's just a string, return it as content
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

	// Generate item ID for the message
	itemID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	sequenceNumber := 1

	// Emit response.in_progress event
	inProgressEvent := responsetypes.ResponseInProgressEvent{
		BaseStreamingEvent: responsetypes.BaseStreamingEvent{
			Type:           "response.in_progress",
			SequenceNumber: sequenceNumber,
		},
		Response: map[string]interface{}{
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
	const minWordsPerChunk = 5

	// Process the stream line by line
	scanner := bufio.NewScanner(resp.RawResponse.Body)
	for scanner.Scan() {
		// Check if context was cancelled
		select {
		case <-reqCtx.Request.Context().Done():
			errChan <- reqCtx.Request.Context().Err()
			return
		default:
		}

		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
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

				if len(words) >= minWordsPerChunk {
					// Create delta event in OpenAI format
					deltaEvent := responsetypes.ResponseOutputTextDeltaEvent{
						BaseStreamingEvent: responsetypes.BaseStreamingEvent{
							Type:           "response.output_text.delta",
							SequenceNumber: sequenceNumber,
						},
						ItemID:       itemID,
						OutputIndex:  0,
						ContentIndex: 0,
						Delta:        bufferedContent,
						Logprobs:     []responsetypes.Logprob{},
					}
					eventJSON, _ := json.Marshal(deltaEvent)
					dataChan <- fmt.Sprintf("event: response.output_text.delta\ndata: %s\n\n", string(eventJSON))
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

					functionCallEvent := responsetypes.ResponseOutputFunctionCallsDeltaEvent{
						BaseStreamingEvent: responsetypes.BaseStreamingEvent{
							Type:           "response.output_function_calls.delta",
							SequenceNumber: sequenceNumber,
						},
						ItemID:       itemID,
						OutputIndex:  0,
						ContentIndex: 0,
						Delta: responsetypes.FunctionCallDelta{
							Name:      accumulator.Name,
							Arguments: arguments,
						},
						Logprobs: []responsetypes.Logprob{},
					}
					eventJSON, _ := json.Marshal(functionCallEvent)
					dataChan <- fmt.Sprintf("event: response.output_function_calls.delta\ndata: %s\n\n", string(eventJSON))
					sequenceNumber++
				}
			}
		}
	}

	// Send any remaining buffered content
	if contentBuffer.Len() > 0 {
		deltaEvent := responsetypes.ResponseOutputTextDeltaEvent{
			BaseStreamingEvent: responsetypes.BaseStreamingEvent{
				Type:           "response.output_text.delta",
				SequenceNumber: sequenceNumber,
			},
			ItemID:       itemID,
			OutputIndex:  0,
			ContentIndex: 0,
			Delta:        contentBuffer.String(),
			Logprobs:     []responsetypes.Logprob{},
		}
		eventJSON, _ := json.Marshal(deltaEvent)
		dataChan <- fmt.Sprintf("event: response.output_text.delta\ndata: %s\n\n", string(eventJSON))
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
	dataChan <- "data: [DONE]\n\n"
}
