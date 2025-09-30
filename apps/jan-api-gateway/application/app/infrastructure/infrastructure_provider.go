package infrastructure

import (
	"github.com/google/wire"
	inferenceDomain "menlo.ai/jan-api-gateway/app/domain/inference"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/infrastructure/inference"
	geminiclient "menlo.ai/jan-api-gateway/app/utils/httpclients/gemini"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	openrouterclient "menlo.ai/jan-api-gateway/app/utils/httpclients/openrouter"
)

var InfrastructureProvider = wire.NewSet(
	janinference.NewJanInferenceClient,
	openrouterclient.NewClient,
	geminiclient.NewClient,
	inference.NewJanProvider,
	inference.NewMultiProviderInference,
	wire.Bind(new(inferenceDomain.InferenceProvider), new(*inference.MultiProviderInference)),
	cache.NewRedisCacheService,
	inferencemodelregistry.NewInferenceModelRegistry,
)
