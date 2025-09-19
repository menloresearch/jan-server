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

// PostCompletion
// @Summary Create a chat completion
// @Description Generates a model response for the given chat conversation. This is a standard chat completion API that supports both streaming and non-streaming modes without conversation persistence.
// @Description
// @Description **Streaming Mode (stream=true):**
// @Description - Returns Server-Sent Events (SSE) with real-time streaming
// @Description - Streams completion chunks directly from the inference model
// @Description - Final event contains "[DONE]" marker
// @Description
// @Description **Non-Streaming Mode (stream=false or omitted):**
// @Description - Returns single JSON response with complete completion
// @Description - Standard OpenAI ChatCompletionResponse format
// @Description
// @Description **Features:**
// @Description - Supports all OpenAI ChatCompletionRequest parameters
// @Description - User authentication required
// @Description - Direct inference model integration
// @Description - No conversation persistence (stateless)
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body openai.ChatCompletionRequest true "Chat completion request with streaming options"
// @Success 200 {object} openai.ChatCompletionResponse "Successful non-streaming response (when stream=false)"
// @Success 200 {string} string "Successful streaming response (when stream=true) - SSE format with data: {json} events"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload, empty messages, or inference failure"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
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

	// Get authenticated user (required for API access)
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok || user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "0199600b-961c-71ba-846b-9ca5b384e382",
			Error: "user not authenticated",
		})
		return
	}

	// TODO: Implement admin API key check for enhanced security

	var err *common.Error
	var response *openai.ChatCompletionResponse

	if request.Stream {
		// Handle streaming completion - streams SSE events directly to client
		err = cApi.StreamCompletionResponse(reqCtx, "", request)
	} else {
		// Handle non-streaming completion - returns complete response
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

	// Send JSON response for non-streaming requests (streaming responses use SSE)
	if !request.Stream {
		reqCtx.JSON(http.StatusOK, response)
	}
}

// CallCompletionAndGetRestResponse calls the inference model and returns a complete non-streaming response
func (cApi *CompletionAPI) CallCompletionAndGetRestResponse(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, *common.Error) {
	// Call inference provider to get complete response
	response, err := cApi.inferenceProvider.CreateCompletion(ctx, apiKey, request)
	if err != nil {
		logger.GetLogger().Errorf("inference failed: %v", err)
		return nil, common.NewError(err, "0199600c-3b65-7618-83ca-443a583d91c9")
	}

	return response, nil
}

// StreamCompletionResponse streams SSE events directly to the client
func (cApi *CompletionAPI) StreamCompletionResponse(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest) *common.Error {
	// Create timeout context for streaming request
	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), RequestTimeout)
	defer cancel()

	// Check for client disconnection
	if reqCtx.Request.Context().Err() != nil {
		return common.NewError(reqCtx.Request.Context().Err(), "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
	}

	// Set up SSE headers for streaming response
	cApi.setupSSEHeaders(reqCtx)
	// Create buffered channels for streaming data and errors
	dataChan := make(chan string, ChannelBufferSize)
	errChan := make(chan error, ErrorBufferSize)

	var wg sync.WaitGroup
	wg.Add(1)

	// Start streaming from inference model in a goroutine
	go cApi.streamResponseToChannel(ctx, apiKey, request, dataChan, errChan, &wg)

	// Process streaming data from channels
	streamingComplete := false
	for !streamingComplete {
		select {
		case line, ok := <-dataChan:
			if !ok {
				// Channel closed, streaming complete
				streamingComplete = true
				break
			}

			// Forward streaming line directly to client
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

	// Wait for streaming goroutine to complete
	wg.Wait()

	// Clean up channels
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
			logger.GetLogger().Errorf("unable to close reader: %v", closeErr)
		}
	}()

	scanner := bufio.NewScanner(reader)
	// Increase scanner buffer size for better performance with large responses
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

// setupSSEHeaders sets up the required headers for Server-Sent Events streaming
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
