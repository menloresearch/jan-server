package healthcheck

import (
	"context"

	"github.com/mileusna/crontab"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	inference_model_registry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type HealthcheckCrontabService struct {
	JanInferenceClient     *janinference.JanInferenceClient
	InferenceModelRegistry *inference_model_registry.InferenceModelRegistry
}

func NewService(janInferenceClient *janinference.JanInferenceClient, registry *inference_model_registry.InferenceModelRegistry) *HealthcheckCrontabService {
	return &HealthcheckCrontabService{
		JanInferenceClient:     janInferenceClient,
		InferenceModelRegistry: registry,
	}
}

func (hs *HealthcheckCrontabService) Start(ctx context.Context, ctab *crontab.Crontab) {
	hs.CheckInferenceModels(ctx)
	// Check every 2 minutes instead of every minute
	ctab.AddJob("*/2 * * * *", func() {
		hs.CheckInferenceModels(ctx)
		environment_variables.EnvironmentVariables.LoadFromEnv()
	})
}

func (hs *HealthcheckCrontabService) CheckInferenceModels(ctx context.Context) {
	janModelResp, err := hs.JanInferenceClient.GetModels(ctx)
	if err != nil {
		hs.InferenceModelRegistry.RemoveServiceModels(ctx, hs.JanInferenceClient.BaseURL)
	} else {
		models := make([]inferencemodel.Model, 0)
		for _, model := range janModelResp.Data {
			models = append(models, inferencemodel.Model{
				ID:      model.ID,
				Object:  model.Object,
				Created: model.Created,
				OwnedBy: model.OwnedBy,
			})
		}
		hs.InferenceModelRegistry.AddModels(ctx, hs.JanInferenceClient.BaseURL, models)
	}
}
