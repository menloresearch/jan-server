package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/chat"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/conversations"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type V1Route struct {
	auth          *auth.AuthRoute
	chat          *chat.ChatRoute
	conversations *conversations.ConversationAPI
	organizations *organization.OrganizationRoute
}

func NewV1Route(
	auth *auth.AuthRoute,
	chat *chat.ChatRoute,
	conversations *conversations.ConversationAPI,
	organizations *organization.OrganizationRoute) *V1Route {
	return &V1Route{
		auth,
		chat,
		conversations,
		organizations,
	}
}

func (v1Route *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")
	v1Route.auth.RegisterRouter(v1Router)
	v1Route.chat.RegisterRouter(v1Router)
	v1Route.conversations.RegisterRouter(v1Router)
	v1Route.organizations.RegisterRouter(v1Router)
	v1Router.GET("/models", v1Route.GetModels)
}

// ListModels
// @Summary List available models
// @Description Retrieves a list of available models that can be used for chat completions or other tasks.
// @Tags Jan, Jan-Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} ModelsResponse "Successful response"
// @Router /jan/v1/models [get]
func (v1Route *V1Route) GetModels(reqCtx *gin.Context) {
	registry := inferencemodelregistry.GetInstance()
	reqCtx.JSON(http.StatusOK, ModelsResponse{
		Object: "list",
		Data: functional.Map(registry.ListModels(), func(model inferencemodel.Model) Model {
			return Model{
				ID:      model.ID,
				Object:  model.Object,
				Created: model.Created,
				OwnedBy: model.OwnedBy,
			}
		}),
	})
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}
