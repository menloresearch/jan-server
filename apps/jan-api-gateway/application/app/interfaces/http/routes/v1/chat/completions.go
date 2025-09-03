package chat

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
)

type CompletionAPI struct {
	apikeyService *apikey.ApiKeyService
}

func NewCompletionAPI(apikeyService *apikey.ApiKeyService) *CompletionAPI {
	return &CompletionAPI{
		apikeyService,
	}
}

func (completionAPI *CompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", completionAPI.PostCompletion)
}

// ChatCompletionResponseSwagger is a doc-only version without http.Header
type ChatCompletionResponseSwagger struct {
	ID      string                        `json:"id"`
	Object  string                        `json:"object"`
	Created int64                         `json:"created"`
	Model   string                        `json:"model"`
	Choices []openai.ChatCompletionChoice `json:"choices"`
	Usage   openai.Usage                  `json:"usage"`
}

// CreateChatCompletion
// @Summary Create a chat completion
// @Description Generates a model response for the given chat conversation.
// @Tags Platform, Platform-Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body openai.ChatCompletionRequest true "Chat completion request payload"
// @Success 200 {object} ChatCompletionResponseSwagger "Successful response"
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

	modelRegistry := inferencemodelregistry.GetInstance()
	mToE := modelRegistry.GetModelToEndpoints()
	endpoints, ok := mToE[request.Model]
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "59253517-df33-44bf-9333-c927402e4e2e",
			Error: fmt.Sprintf("Model: %s does not exist", request.Model),
		})
		return
	}

	janInferenceClient := janinference.NewJanInferenceClient(reqCtx)
	for _, endpoint := range endpoints {
		if endpoint == janInferenceClient.BaseURL {
			if request.Stream {
				dataChan := make(chan string)

				reqCtx.Writer.Header().Set("Content-Type", "text/event-stream")
				reqCtx.Writer.Header().Set("Cache-Control", "no-cache")
				reqCtx.Writer.Header().Set("Connection", "keep-alive")
				reqCtx.Writer.Header().Set("Transfer-Encoding", "chunked")

				func() error {
					req := janinference.JanInferenceRestyClient.R().SetBody(request)
				resp, err := req.
					SetDoNotParseResponse(true).
					Post("/v1/chat/completions")

				defer resp.RawResponse.Body.Close()
				scanner := bufio.NewScanner(resp.RawResponse.Body)
				for scanner.Scan() {
					line := scanner.Text()
					reqCtx.Writer.Write([]byte(line + "\n"))
					reqCtx.Writer.Flush()
				}
				}
				req := janinference.JanInferenceRestyClient.R().SetBody(request)
				resp, err := req.
					SetDoNotParseResponse(true).
					Post("/v1/chat/completions")

				defer resp.RawResponse.Body.Close()
				scanner := bufio.NewScanner(resp.RawResponse.Body)
				for scanner.Scan() {
					line := scanner.Text()
					reqCtx.Writer.Write([]byte(line + "\n"))
					reqCtx.Writer.Flush()
				}
				reqCtx.Writer.Flush()
				// err := janInferenceClient.CreateChatCompletionStream(reqCtx, key, request)
				// if err != nil {
				// 	reqCtx.AbortWithStatusJSON(
				// 		http.StatusBadRequest,
				// 		responses.ErrorResponse{
				// 			Code:  "c3af973c-eada-4e8b-96d9-e92546588cd3",
				// 			Error: err.Error(),
				// 		})
				// 	return
				// }
				return
			} else {
				response, err := janInferenceClient.CreateChatCompletion(reqCtx.Request.Context(), key, request)
				if err != nil {
					reqCtx.AbortWithStatusJSON(
						http.StatusBadRequest,
						responses.ErrorResponse{
							Code:  "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4",
							Error: err.Error(),
						})
					return
				}
				reqCtx.JSON(http.StatusOK, response)
				return
			}
		}
	}

	reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
		Code:  "6c6e4ea0-53d2-4c6c-8617-3a645af59f43",
		Error: "Client does not exist",
	})
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

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type PostChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}
