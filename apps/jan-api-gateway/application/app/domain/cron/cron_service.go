package cron

import (
	"context"

	"github.com/mileusna/crontab"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	inference_model_registry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type CronService struct {
	JanInferenceClient     *janinference.JanInferenceClient
	InferenceModelRegistry *inference_model_registry.InferenceModelRegistry
}

func NewService(janInferenceClient *janinference.JanInferenceClient, registry *inference_model_registry.InferenceModelRegistry) *CronService {
	return &CronService{
		JanInferenceClient:     janInferenceClient,
		InferenceModelRegistry: registry,
	}
}

func (cs *CronService) Start(ctx context.Context, ctab *crontab.Crontab) {
	// Run initial check
	cs.CheckInferenceModels(ctx)

	// Check every 5 minutes instead of every minute
	ctab.AddJob("*/5 * * * *", func() {
		cs.CheckInferenceModels(ctx)
		environment_variables.EnvironmentVariables.LoadFromEnv()
	})
}

func (cs *CronService) CheckInferenceModels(ctx context.Context) {
	janModelResp, err := cs.JanInferenceClient.GetModels(ctx)
	if err != nil {
		cs.InferenceModelRegistry.RemoveServiceModels(ctx, cs.JanInferenceClient.BaseURL)
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

		cs.InferenceModelRegistry.SetModels(ctx, cs.JanInferenceClient.BaseURL, models)
	}
}
