package main

import (
	"menlo.ai/jan-api-gateway/app/interfaces/http"
	"menlo.ai/jan-api-gateway/app/utils/httpclients"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
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
	environment_variables.EnvironmentVariables.LoadFromEnv()
	httpclients.Init()
	janinference.Init()
}

func main() {
	application, err := CreateApplication()
	if err != nil {
		panic(err)
	}
	application.Start()
}
