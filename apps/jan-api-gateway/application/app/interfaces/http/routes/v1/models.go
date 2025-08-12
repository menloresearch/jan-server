package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type ModelAPI struct {
}

func NewModelAPI() *ModelAPI {
	return &ModelAPI{}
}

func (modelAPI *ModelAPI) RegisterRouter(router *gin.RouterGroup) {
	router.GET("models", modelAPI.GetModels)
}

func (modelAPI *ModelAPI) GetModels(reqCtx *gin.Context) {
	registry := inferencemodelregistry.GetInstance()
	registry.ListModels()
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
