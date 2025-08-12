package inferencemodelregistry

import (
	"sync"

	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type InferenceModelRegistry struct {
	endpointToModels map[string][]string
	modelToEndpoints map[string][]string
	modelsDetail     map[string]inferencemodel.Model
	models           []inferencemodel.Model
	mu               sync.RWMutex
}

var (
	once             sync.Once
	registryInstance *InferenceModelRegistry
)

func GetInstance() *InferenceModelRegistry {
	once.Do(func() {
		registryInstance = &InferenceModelRegistry{
			endpointToModels: make(map[string][]string),
			modelToEndpoints: make(map[string][]string),
			modelsDetail:     make(map[string]inferencemodel.Model),
			models:           make([]inferencemodel.Model, 0),
		}
	})
	return registryInstance
}

func (r *InferenceModelRegistry) ListModels() []inferencemodel.Model {
	return r.models
}

func (r *InferenceModelRegistry) AddModels(serviceName string, models []inferencemodel.Model) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endpointToModels[serviceName] = functional.Map(models, func(model inferencemodel.Model) string {
		r.modelsDetail[model.ID] = model
		return model.ID
	})
	r.rebuild()
}

func (r *InferenceModelRegistry) RemoveServiceModels(serviceName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.endpointToModels, serviceName)
	r.rebuild()
}

func (r *InferenceModelRegistry) GetEndpointToModels(serviceName string) ([]string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	models, ok := r.endpointToModels[serviceName]
	return models, ok
}

func (r *InferenceModelRegistry) GetModelToEndpoints() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modelToEndpoints
}

func (r *InferenceModelRegistry) rebuild() {
	newModelToEndpoints := make(map[string][]string)
	newModels := make([]inferencemodel.Model, 0)
	for endpoint, models := range r.endpointToModels {
		for _, model := range models {
			newModelToEndpoints[model] = append(newModelToEndpoints[model], endpoint)
		}
	}
	r.modelToEndpoints = newModelToEndpoints

	for key := range r.modelToEndpoints {
		newModels = append(newModels, r.modelsDetail[key])
	}
	r.models = newModels
}
