package chat

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
)

type CompletionAPI struct {
}

func NewCompletionAPI() *CompletionAPI {
	return &CompletionAPI{}
}

func (completionAPI *CompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", completionAPI.PostCompletion)
}

func (CompletionAPI *CompletionAPI) PostCompletion(reqCtx *gin.Context) {
	var request openai.ChatCompletionRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "cf237451-8932-48d1-9cf6-42c4db2d4805",
			Error: err.Error(),
		})
		return
	}

	modelRegistry := inferencemodelregistry.GetInstance()
	mToE := modelRegistry.GetModelToEndpoints()
	endpoints, ok := mToE[request.Model]
	if !ok {
		reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "59253517-df33-44bf-9333-c927402e4e2e",
			Error: fmt.Sprintf("Model: %s does not exist", request.Model),
		})
		return
	}

	janInferenceClient := janinference.NewJanInferenceClient(reqCtx)
	for _, endpoint := range endpoints {
		if endpoint == janInferenceClient.BaseURL {
			response, err := janInferenceClient.CreateChatCompletion(reqCtx.Request.Context(), "test-api-key", request)
			if err != nil {
				reqCtx.JSON(
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

	reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
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
