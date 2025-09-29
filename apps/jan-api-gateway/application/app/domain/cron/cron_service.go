package cron

import (
	"context"

	"github.com/mileusna/crontab"
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
	cs.InferenceModelRegistry.CheckInferenceModels(ctx)

	ctab.AddJob("* * * * *", func() {
		cs.InferenceModelRegistry.CheckInferenceModels(ctx)

		// Reload environment variables
		environment_variables.EnvironmentVariables.LoadFromEnv()
	})
}
