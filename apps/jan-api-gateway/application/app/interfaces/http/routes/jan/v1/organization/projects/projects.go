package projects

import (
	"github.com/gin-gonic/gin"
)

type ProjectsRoute struct {
}

func NewProjectsRoute() *ProjectsRoute {
	return &ProjectsRoute{}
}

func (o *ProjectsRoute) RegisterRouter(router gin.IRouter) {
}
