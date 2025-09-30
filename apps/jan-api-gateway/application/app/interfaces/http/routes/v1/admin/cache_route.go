package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

// CacheRoute exposes administrative cache operations.
type CacheRoute struct {
	authService  *auth.AuthService
	cacheService *cache.RedisCacheService
}

// NewCacheRoute constructs a CacheRoute instance.
func NewCacheRoute(authService *auth.AuthService, cacheService *cache.RedisCacheService) *CacheRoute {
	return &CacheRoute{
		authService:  authService,
		cacheService: cacheService,
	}
}

// RegisterRouter wires the administrative cache endpoints.
func (route *CacheRoute) RegisterRouter(router gin.IRouter) {
	adminRouter := router.Group("/admin",
		route.authService.AdminUserAuthMiddleware(),
		route.authService.RegisteredUserMiddleware(),
		route.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleOwnerOnly),
	)

	adminRouter.POST("/cache/invalidate", route.InvalidateCache)
}

// CacheInvalidateResponse represents the result of a cache invalidation request.
type CacheInvalidateResponse struct {
	Object  string `json:"object"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (route *CacheRoute) InvalidateCache(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	if err := route.cacheService.FlushAll(ctx); err != nil {
		logger.GetLogger().Errorf("admin cache: failed to flush cache: %v", err)
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "b0c4f1c8-2a3b-4ad4-8b1d-7a2124d7c7b1",
			Error: "failed to invalidate cache",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, CacheInvalidateResponse{
		Object:  "cache.invalidation",
		Status:  "ok",
		Message: "cache invalidated",
	})
}
