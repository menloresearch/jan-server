package chat

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type CompletionAPI struct {
	userService   *user.UserService
	apikeyService *apikey.ApiKeyService
}

func NewCompletionAPI(userService *user.UserService, apikeyService *apikey.ApiKeyService) *CompletionAPI {
	return &CompletionAPI{
		userService,
		apikeyService,
	}
}

func (completionAPI *CompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", middleware.OptionalAuthMiddleware(), completionAPI.PostCompletion)
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
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body openai.ChatCompletionRequest true "Chat completion request payload"
// @Success 200 {object} ChatCompletionResponseSwagger "Successful response"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/chat/completions [post]
func (api *CompletionAPI) PostCompletion(reqCtx *gin.Context) {
 	userClaim, _ := auth.GetUserClaimFromRequestContext(reqCtx)
	key := "AnonymousUserKey"
	if userClaim != nil {
		user, err := api.userService.FindByEmailAndPlatform(reqCtx, userClaim.Email, string(user.UserPlatformTypeAskJanAI))
		if err != nil {
			reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "62a772b9-58ec-4332-b669-920c7f4a8821",
				Error: err.Error(),
			})
			return
		}
		apikeyEntities, err := api.apikeyService.Find(reqCtx, apikey.ApiKeyFilter{
			OwnerID:   &user.ID,
			OwnerType: ptr.ToString(string(apikey.OwnerTypeAdmin)),
		}, nil)
		if err != nil {
			reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "7e29d138-8b8e-4895-8edc-c0876ebb1a52",
				Error: err.Error(),
			})
			return
		}
		// TODO: Should we provide a default key to user?
		if len(apikeyEntities) == 0 {
			key, hash, err := api.apikeyService.GenerateKeyAndHash(reqCtx, apikey.OwnerTypeEphemeral)
			if err != nil {
				reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
					Code:  "207373ae-f94a-4b21-bf95-7bbd8d727f84",
					Error: err.Error(),
				})
				return
			}

			// TODO: OwnerTypeEphemeral
			entity, err := api.apikeyService.CreateApiKey(reqCtx, &apikey.ApiKey{
				KeyHash:        hash,
				PlaintextHint:  fmt.Sprintf("sk-..%s", key[len(key)-3:]),
				Description:    "Default Key For User",
				Enabled:        true,
				OwnerType:      string(apikey.OwnerTypeEphemeral),
				OwnerID:        &user.ID,
				OrganizationID: nil,
				Permissions:    "{}",
			})
			if err != nil {
				reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
					Code:  "cfda552d-ec73-4e12-abfb-963b3c3829e9",
					Error: err.Error(),
				})
				return
			}
			apikeyEntities = []*apikey.ApiKey{
				entity,
			}
		}
		key = apikeyEntities[0].KeyHash
	}
	var request openai.ChatCompletionRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "cf237451-8932-48d1-9cf6-42c4db2d4805",
			Error: err.Error(),
		})
		return
	}

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
				err := janInferenceClient.CreateChatCompletionStream(reqCtx, key, request)
				if err != nil {
					reqCtx.AbortWithStatusJSON(
						http.StatusBadRequest,
						responses.ErrorResponse{
							Code:  "c3af973c-eada-4e8b-96d9-e92546588cd3",
							Error: err.Error(),
						})
					return
				}
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
