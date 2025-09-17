//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/domain"
	"menlo.ai/jan-api-gateway/app/infrastructure"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository"
	"menlo.ai/jan-api-gateway/app/interfaces/http"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes"
)

func CreateApplication() (*Application, error) {
	wire.Build(
		database.NewDB,
		repository.RepositoryProvider,
		infrastructure.InfrastructureProvider,
		domain.ServiceProvider,
		routes.RouteProvider,
		http.NewHttpServer,
		wire.Struct(new(Application), "*"),
		provideContext,
	)
	return nil, nil
}

func provideContext() context.Context {
	return context.Background()
}
