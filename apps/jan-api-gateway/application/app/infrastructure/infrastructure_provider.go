package infrastructure

import (
	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/infrastructure/inference"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
)

var InfrastructureProvider = wire.NewSet(
	janinference.NewJanInferenceClient,
	inference.NewJanInferenceProvider,
)
