package healthcheck

import (
	"context"

	"github.com/mileusna/crontab"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	inference_model_registry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
)

type HealthcheckCrontabService struct {
	JanInferenceClient     *janinference.JanInferenceClient
	InferenceModelRegistry *inference_model_registry.InferenceModelRegistry
}

func NewService(janInferenceClient *janinference.JanInferenceClient) *HealthcheckCrontabService {
	return &HealthcheckCrontabService{
		JanInferenceClient:     janInferenceClient,
		InferenceModelRegistry: inference_model_registry.GetInstance(),
	}
}

func (hs *HealthcheckCrontabService) Start(ctx context.Context, ctab *crontab.Crontab) {
	hs.CheckInferenceModels(ctx)
	ctab.AddJob("* * * * *", func() {
		hs.CheckInferenceModels(ctx)
	})
}

func (hs *HealthcheckCrontabService) CheckInferenceModels(ctx context.Context) {
	janModelResp, err := hs.JanInferenceClient.GetModels()
	if err != nil {
		hs.InferenceModelRegistry.RemoveServiceModels(hs.JanInferenceClient.BaseURL)
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
		hs.InferenceModelRegistry.AddModels(hs.JanInferenceClient.BaseURL, models)
	}
}
