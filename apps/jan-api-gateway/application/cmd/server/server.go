package main

import (
	"context"

	"github.com/mileusna/crontab"
	"menlo.ai/jan-api-gateway/app/domain/healthcheck"
	"menlo.ai/jan-api-gateway/app/interfaces/http"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/httpclients/serper"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type Application struct {
	HttpServer *http.HttpServer
}

func (application *Application) Start() {
	if err := application.HttpServer.Run(); err != nil {
		panic(err)
	}
}

func init() {
	logger.GetLogger()
	environment_variables.EnvironmentVariables.LoadFromEnv()
	// TODO: refactoring: singleton.
	janinference.Init()
	serper.Init()
}

func main() {
	healthcheckService := healthcheck.NewService(janinference.NewJanInferenceClient(context.Background()))
	cron := crontab.New()
	crontabContext := context.Background()
	healthcheckService.Start(crontabContext, cron)
	application, err := CreateApplication()
	if err != nil {
		panic(err)
	}
	application.Start()
}
