package http

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	jan "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1"
	"menlo.ai/jan-api-gateway/app/utils/logger"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "menlo.ai/jan-api-gateway/docs"
)

type HttpServer struct {
	engine   *gin.Engine
	v1Route  *v1.V1Route
	janRoute *jan.JanRoute
}

func (s *HttpServer) bindSwagger() {
	g := s.engine.Group("/")
	g.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	g.GET("/google/testcallback", func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")
		curlCommand := fmt.Sprintf(`curl --request POST \
  --url 'http://localhost:8080/jan/v1/auth/google/callback' \
  --header 'Content-Type: application/json' \
  --cookie 'jan_oauth_state=%s' \
  --data '{"code": "%s", "state": "%s"}'`, state, code, state)
		c.String(http.StatusOK, curlCommand)
	})
}

func NewHttpServer(v1Route *v1.V1Route, janRoute *jan.JanRoute) *HttpServer {
	if os.Getenv("local_dev") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	server := HttpServer{
		gin.New(),
		v1Route,
		janRoute,
	}
	server.engine.Use(middleware.LoggerMiddleware(logger.Logger))
	server.engine.GET("/healthcheck", func(c *gin.Context) {
		c.JSON(200, "ok")
	})
	server.bindSwagger()
	return &server
}

func (httpServer *HttpServer) Run() error {
	port := 8080
	root := httpServer.engine.Group("/")
	httpServer.v1Route.RegisterRouter(root)
	httpServer.janRoute.RegisterRouter(root)
	if err := httpServer.engine.Run(fmt.Sprintf(":%d", port)); err != nil {
		return err
	}
	return nil
}
