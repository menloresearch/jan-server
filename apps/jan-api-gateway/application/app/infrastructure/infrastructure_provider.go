package infrastructure

import (
	"github.com/google/wire"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/infrastructure/inference"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
)

var InfrastructureProvider = wire.NewSet(
	janinference.NewJanInferenceClient,
	inference.NewJanInferenceProvider,
	cache.NewCacheService,
	inferencemodelregistry.NewInferenceModelRegistry,
)
