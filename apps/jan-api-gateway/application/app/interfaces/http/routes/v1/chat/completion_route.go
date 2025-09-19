package chat

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/inference"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

// Constants for streaming configuration
const (
	RequestTimeout       = 120 * time.Second
	ChannelBufferSize    = 100
	ErrorBufferSize      = 10
	DataPrefix           = "data: "
	DoneMarker           = "[DONE]"
	NewlineChar          = "\n"
	ScannerInitialBuffer = 12 * 1024   // 12KB
	ScannerMaxBuffer     = 1024 * 1024 // 1MB
)

type CompletionAPI struct {
	inferenceProvider inference.InferenceProvider
	authService       *auth.AuthService
}

func NewCompletionAPI(inferenceProvider inference.InferenceProvider, authService *auth.AuthService) *CompletionAPI {
	return &CompletionAPI{
		inferenceProvider: inferenceProvider,
		authService:       authService,
	}
}

func (completionAPI *CompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", completionAPI.PostCompletion)
}

// CreateChatCompletion
// @Summary Create a chat completion
// @Description Generates a model response for the given chat conversation. Supports both streaming and non-streaming modes with conversation management and storage options.
// @Description
// @Description **Streaming Mode (stream=true):**
// @Description - Returns Server-Sent Events (SSE) with real-time streaming
// @Description - First event contains conversation metadata
// @Description - Subsequent events contain completion chunks
// @Description - Final event contains "[DONE]" marker
// @Description
// @Description **Non-Streaming Mode (stream=false or omitted):**
// @Description - Returns single JSON response with complete completion
// @Description - Includes conversation metadata in response
// @Description
// @Description **Storage Options:**
// @Description - `store=true`: Saves user message and assistant response to conversation
// @Description - `store_reasoning=true`: Includes reasoning content in stored messages
// @Description - `conversation`: ID of existing conversation or empty for new conversation
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body openai.ChatCompletionRequest. true "Chat completion request with streaming, storage, and conversation options"
// @Success 200 {object} openai.ChatCompletionResponse. "Successful non-streaming response (when stream=false)"
// @Success 200 {string} string "Successful streaming response (when stream=true) - SSE format with data: {json} events"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload or conversation not found"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found or user not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/chat/completions [post]
func (cApi *CompletionAPI) PostCompletion(reqCtx *gin.Context) {
	var request openai.ChatCompletionRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "0199600b-86d3-7339-8402-8ef1c7840475",
			ErrorInstance: err,
		})
		return
	}

	if len(request.Messages) == 0 {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "0199600f-2cbe-7518-be5c-9989cce59472",
			Error: "messages cannot be empty",
		})
		return
	}

	// Get user ID for saving messages
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok || user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "0199600b-961c-71ba-846b-9ca5b384e382",
			Error: "user not authenticated",
		})
		return
	}

	// TODO: Implement admin API key check

	var err *common.Error
	var response *openai.ChatCompletionResponse

	if request.Stream {
		// Handle streaming completion - streams SSE events and accumulates response
		err = cApi.StreamCompletionResponse(reqCtx, "", request)
	} else {
		// Handle non-streaming completion
		response, err = cApi.CallCompletionAndGetRestResponse(reqCtx.Request.Context(), "", request)
	}

	if err != nil {
		logger.GetLogger().Errorf("completion failed: %v", err)
		reqCtx.AbortWithStatusJSON(
			http.StatusBadRequest,
			responses.ErrorResponse{
				Code:          err.GetCode(),
				ErrorInstance: err.GetError(),
			})
		return
	}

	// Only send JSON response for non-streaming requests (streaming uses SSE)
	if !request.Stream {
		reqCtx.JSON(http.StatusOK, response)
	}
}

// CallCompletionAndGetRestResponse calls the inference model and returns a non-streaming REST response
func (cApi *CompletionAPI) CallCompletionAndGetRestResponse(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, *common.Error) {

	// Call inference provider
	response, err := cApi.inferenceProvider.CreateCompletion(ctx, apiKey, request)
	if err != nil {
		logger.GetLogger().Errorf("inference failed: %v", err)
		return nil, common.NewError(err, "0199600c-3b65-7618-83ca-443a583d91c9")
	}

	return response, nil
}

// StreamCompletionAndAccumulateResponse streams SSE events to client and accumulates a complete response for internal processing
func (cApi *CompletionAPI) StreamCompletionResponse(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest) *common.Error {
	// Add timeout context
	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), RequestTimeout)
	defer cancel()

	// Check for client disconnection
	if reqCtx.Request.Context().Err() != nil {
		return common.NewError(reqCtx.Request.Context().Err(), "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
	}

	// Set up SSE headers
	cApi.setupSSEHeaders(reqCtx)
	// Create buffered channels for data and errors
	dataChan := make(chan string, ChannelBufferSize)
	errChan := make(chan error, ErrorBufferSize)

	var wg sync.WaitGroup
	wg.Add(1)

	// Start streaming in a goroutine
	go cApi.streamResponseToChannel(ctx, apiKey, request, dataChan, errChan, &wg)

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
			if err := cApi.writeSSELine(reqCtx, line); err != nil {
				return common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
			}

			if data, found := strings.CutPrefix(line, DataPrefix); found {
				if data == DoneMarker {
					streamingComplete = true
					break
				}
			}

		case err, ok := <-errChan:
			if !ok {
				// Channel closed, no more errors
				continue
			}
			if err != nil {
				return common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
			}

		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				logger.GetLogger().Errorf("streaming timeout: %v", ctx.Err())
				return common.NewError(ctx.Err(), "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
			}
			return common.NewError(ctx.Err(), "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
		}
	}

	// Wait for streaming goroutine to complete and close channels
	wg.Wait()

	close(dataChan)
	close(errChan)

	return nil
}

// streamResponseToChannel streams the response from inference provider to channels
func (cApi *CompletionAPI) streamResponseToChannel(ctx context.Context, apiKey string, request openai.ChatCompletionRequest, dataChan chan<- string, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	// Ensure channels are closed in case of early return
	defer close(dataChan)
	defer close(errChan)

	// Get streaming reader from inference provider
	reader, err := cApi.inferenceProvider.CreateCompletionStream(ctx, apiKey, request)
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
	//Increase scanner buffer size
	scanner.Buffer(make([]byte, 0, ScannerInitialBuffer), ScannerMaxBuffer)
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
func (cApi *CompletionAPI) setupSSEHeaders(reqCtx *gin.Context) {
	reqCtx.Header("Content-Type", "text/event-stream")
	reqCtx.Header("Cache-Control", "no-cache")
	reqCtx.Header("Connection", "keep-alive")
	reqCtx.Header("Access-Control-Allow-Origin", "*")
	reqCtx.Header("Access-Control-Allow-Headers", "Cache-Control")
}

// writeSSELine writes a line to the SSE stream
func (cApi *CompletionAPI) writeSSELine(reqCtx *gin.Context, line string) error {
	_, err := reqCtx.Writer.Write([]byte(line + NewlineChar))
	if err != nil {
		return err
	}
	reqCtx.Writer.Flush()
	return nil
}
