package chat

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	chatdomain "menlo.ai/jan-api-gateway/app/domain/chat"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

type CompletionAPI struct {
	apikeyService    *apikey.ApiKeyService
	chatUseCase      *chatdomain.ChatUseCase
	streamingService *chatdomain.StreamingService
}

func NewCompletionAPI(apikeyService *apikey.ApiKeyService, chatUseCase *chatdomain.ChatUseCase, streamingService *chatdomain.StreamingService) *CompletionAPI {
	return &CompletionAPI{
		apikeyService:    apikeyService,
		chatUseCase:      chatUseCase,
		streamingService: streamingService,
	}
}

func (completionAPI *CompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", completionAPI.PostCompletion)
}

// CreateChatCompletion
// @Summary Create a chat completion
// @Description Generates a model response for the given chat conversation.
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body PostChatCompletionRequest true "Chat completion request payload"
// @Success 200 {object} CompletionResponse "Successful response"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/chat/completions [post]
func (api *CompletionAPI) PostCompletion(reqCtx *gin.Context) {
	var request openai.ChatCompletionRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "cf237451-8932-48d1-9cf6-42c4db2d4805",
			Error: err.Error(),
		})
		return
	}

	// TODO: Implement admin API key check
	key := ""
	// if environment_variables.EnvironmentVariables.ENABLE_ADMIN_API {
	// 	key, ok := requests.GetTokenFromBearer(reqCtx)
	// 	if !ok {
	// 		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
	// 			Code:  "4284adb3-7af4-428b-8064-7073cb9ca2ca",
	// 			Error: "invalid apikey",
	// 		})
	// 		return
	// 	}
	// 	hashed := api.apikeyService.HashKey(reqCtx, key)
	// 	apikeyEntity, err := api.apikeyService.FindByKeyHash(reqCtx, hashed)
	// 	if err != nil {
	// 		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
	// 			Code:  "d14ab75b-586b-4b55-ba65-e520a76d6559",
	// 			Error: "invalid apikey",
	// 		})
	// 		return
	// 	}
	// 	if !apikeyEntity.Enabled {
	// 		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
	// 			Code:  "42bd6104-28a1-45bd-a164-8e32d12b0378",
	// 			Error: "invalid apikey",
	// 		})
	// 		return
	// 	}
	// 	if apikeyEntity.ExpiresAt != nil && apikeyEntity.ExpiresAt.Before(time.Now()) {
	// 		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
	// 			Code:  "f8f2733d-c76f-40e4-95b1-584a5d054225",
	// 			Error: "apikey expired",
	// 		})
	// 		return
	// 	}
	// }

	// Convert to domain request
	domainRequest := &chatdomain.CompletionRequest{
		Model:    request.Model,
		Messages: request.Messages,
		Stream:   request.Stream,
	}

	if request.Temperature != 0 {
		domainRequest.Temperature = &request.Temperature
	}
	if request.MaxTokens != 0 {
		domainRequest.MaxTokens = &request.MaxTokens
	}
	if request.TopP != 0 {
		domainRequest.TopP = &request.TopP
	}
	if request.Metadata != nil {
		// Convert map[string]string to map[string]interface{}
		metadata := make(map[string]interface{})
		for k, v := range request.Metadata {
			metadata[k] = v
		}
		domainRequest.Metadata = metadata
	}

	// Handle streaming vs non-streaming requests
	if request.Stream {
		err := api.streamingService.StreamCompletion(reqCtx, key, request)
		if !err.IsEmpty() {
			// Check if context was cancelled (timeout)
			if reqCtx.Request.Context().Err() == context.DeadlineExceeded {
				reqCtx.AbortWithStatusJSON(
					http.StatusRequestTimeout,
					responses.ErrorResponse{
						Code: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
					})
			} else if reqCtx.Request.Context().Err() == context.Canceled {
				reqCtx.AbortWithStatusJSON(
					http.StatusRequestTimeout,
					responses.ErrorResponse{
						Code: "b2c3d4e5-f6g7-8901-bcde-f23456789012",
					})
			} else {
				reqCtx.AbortWithStatusJSON(
					http.StatusBadRequest,
					responses.ErrorResponse{
						Code:  err.Code,
						Error: err.Message,
					})
			}
			return
		}
		return
	} else {
		response, err := api.chatUseCase.CreateCompletion(reqCtx.Request.Context(), key, domainRequest)
		if !err.IsEmpty() {
			reqCtx.AbortWithStatusJSON(
				http.StatusBadRequest,
				responses.ErrorResponse{
					Code:  err.Code,
					Error: err.Message,
				})
			return
		}
		reqCtx.JSON(http.StatusOK, response)
		return
	}
}

// CompletionResponse represents the response from chat completion
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   Usage              `json:"usage"`
}

type CompletionChoice struct {
	Index        int               `json:"index"`
	Message      CompletionMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type CompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PostChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

type PostChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}
