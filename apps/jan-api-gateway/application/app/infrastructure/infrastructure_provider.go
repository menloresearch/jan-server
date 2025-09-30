package infrastructure

import (
	"github.com/google/wire"
	inferencedomain "menlo.ai/jan-api-gateway/app/domain/inference"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/infrastructure/inference"
	geminiclient "menlo.ai/jan-api-gateway/app/utils/httpclients/gemini"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	openrouterclient "menlo.ai/jan-api-gateway/app/utils/httpclients/openrouter"
)

var InfrastructureProvider = wire.NewSet(
	janinference.NewJanInferenceClient,
	wire.Bind(new(inferencedomain.InferenceProvider), new(*inference.MultiProviderInference)),
	openrouterclient.NewClient,
	geminiclient.NewClient,
	inference.NewJanProvider,
	inference.NewMultiProviderInference,
	cache.NewRedisCacheService,
	inferencemodelregistry.NewInferenceModelRegistry,
)
