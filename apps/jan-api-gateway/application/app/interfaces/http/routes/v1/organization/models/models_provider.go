package models

import (
	"github.com/google/wire"
)

// ModelsProvider provides all models-related APIs
var ModelsProvider = wire.NewSet(
	NewModelsAPI,
	NewKubernetesAPI,
)
