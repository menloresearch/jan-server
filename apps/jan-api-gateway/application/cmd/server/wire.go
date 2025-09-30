//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"gorm.io/gorm"
	"menlo.ai/jan-api-gateway/app/domain"
	cron "menlo.ai/jan-api-gateway/app/domain/cron"
	"menlo.ai/jan-api-gateway/app/infrastructure"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository"
	infrainference "menlo.ai/jan-api-gateway/app/infrastructure/inference"
	"menlo.ai/jan-api-gateway/app/interfaces/http"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes"
)

func CreateApplication() (*Application, error) {
	wire.Build(
		database.NewDB,
		repository.RepositoryProvider,
		wire.Bind(new(cron.JanModelRefresher), new(*infrainference.JanProvider)),
		infrastructure.InfrastructureProvider,
		domain.ServiceProvider,
		routes.RouteProvider,
		http.NewHttpServer,
		wire.Struct(new(Application), "*"),
		provideContext,
	)
	return nil, nil
}

func ProvideDatabase() *gorm.DB {
	return database.DB
}

func CreateDataInitializer() (*DataInitializer, error) {
	wire.Build(
		ProvideDatabase,
		repository.RepositoryProvider,
		infrastructure.InfrastructureProvider,
		domain.ServiceProvider,
		wire.Struct(new(DataInitializer), "*"),
	)
	return nil, nil
}

func provideContext() context.Context {
	return context.Background()
}
