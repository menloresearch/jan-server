package http

import (
	"fmt"

	"github.com/gin-gonic/gin"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1"
)

type HttpServer struct {
	engine  *gin.Engine
	v1Route *v1.V1Route
}

func NewHttpServer(v1Route *v1.V1Route) *HttpServer {
	gin.SetMode(gin.ReleaseMode)
	server := HttpServer{
		engine:  gin.New(),
		v1Route: v1Route,
	}
	server.engine.GET("/health-check", func(c *gin.Context) {
		c.JSON(200, "ok")
	})
	return &server
}

func (httpServer *HttpServer) Run() error {
	port := 8080
	httpServer.v1Route.RegisterRouter(httpServer.engine.Group("/"))
	if err := httpServer.engine.Run(fmt.Sprintf(":%d", port)); err != nil {
		return err
	}
	return nil
}
