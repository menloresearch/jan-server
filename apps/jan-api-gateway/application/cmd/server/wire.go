//go:build wireinject

package main

import (
	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/domain"
	"menlo.ai/jan-api-gateway/app/interfaces/http"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes"
)

func CreateApplication() (*Application, error) {
	wire.Build(
		domain.ServiceProvider,
		routes.RouteProvider,
		http.NewHttpServer,
		wire.Struct(new(Application), "*"),
	)
	return nil, nil
}
