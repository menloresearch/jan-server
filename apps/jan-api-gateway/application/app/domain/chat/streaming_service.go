package chat

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/inference"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

// StreamingService handles streaming chat completions
type StreamingService struct {
	inferenceProvider inference.InferenceProvider
}

// NewStreamingService creates a new StreamingService
func NewStreamingService(inferenceProvider inference.InferenceProvider) *StreamingService {
	return &StreamingService{
		inferenceProvider: inferenceProvider,
	}
}

// StreamCompletion handles streaming chat completion
func (s *StreamingService) StreamCompletion(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest) *common.Error {
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

	// Use the two-channel streaming logic
	return s.processStreamingResponse(reqCtx, request)
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

// processStreamingResponse processes the streaming response using two channels
func (s *StreamingService) processStreamingResponse(reqCtx *gin.Context, request openai.ChatCompletionRequest) *common.Error {
	// Create buffered channels for data and errors
	dataChan := make(chan string, ChannelBufferSize)
	errChan := make(chan error, ErrorBufferSize)

	var wg sync.WaitGroup
	wg.Add(1)

	// Start streaming in a goroutine
	go s.streamResponseToChannel(reqCtx, request, dataChan, errChan, &wg)

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
				return common.EmptyError
			}
			_, err := reqCtx.Writer.Write([]byte(line))
			if err != nil {
				reqCtx.AbortWithStatusJSON(
					http.StatusBadRequest,
					responses.ErrorResponse{
						Code: "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4",
					})
				return common.NewError("bc82d69c-685b-4556-9d1f-2a4a80ae8ca4", fmt.Sprintf("Failed to write response: %v", err))
			}
			reqCtx.Writer.Flush()
		case err := <-errChan:
			if err != nil {
				reqCtx.AbortWithStatusJSON(
					http.StatusBadRequest,
					responses.ErrorResponse{
						Code: "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4",
					})
				return common.NewError("bc82d69c-685b-4556-9d1f-2a4a80ae8ca4", fmt.Sprintf("Streaming error: %v", err))
			}
		}
	}
}

// streamResponseToChannel handles the streaming response and sends data/errors to channels
func (s *StreamingService) streamResponseToChannel(reqCtx *gin.Context, request openai.ChatCompletionRequest, dataChan chan<- string, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	// Get streaming reader from inference provider
	reader, err := s.inferenceProvider.CreateCompletionStream(reqCtx.Request.Context(), "", request)
	if err != nil {
		errChan <- err
		return
	}
	defer reader.Close()

	// Process the stream line by line
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// Check if context was cancelled
		if s.checkContextCancellation(reqCtx, errChan) {
			return
		}

		line := scanner.Text()
		if data, found := strings.CutPrefix(line, DataPrefix); found {
			if data == DoneMarker {
				break
			}
			// Send the raw data line directly without any wrapping
			// With Jan Inference follow the Completion format
			dataChan <- line + "\n"
		}
	}

	// Send [DONE] to close the stream
	dataChan <- fmt.Sprintf("%s%s\n", DataPrefix, DoneMarker)
}
