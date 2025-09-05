package apikey

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

type ApiKeyHandler struct {
	apiKeyService *apikey.ApiKeyService
}

func NewApiKeyHandler(apiKeyService *apikey.ApiKeyService) *ApiKeyHandler {
	return &ApiKeyHandler{
		apiKeyService,
	}
}

type ApiKeyContextKey string

const (
	ApiKeyContextKeyEntity ApiKeyContextKey = "ApiKeyContextKeyEntity"
)

func (handler *ApiKeyHandler) AdminApiKeyMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		adminKeyStr, ok := requests.GetTokenFromBearer(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "277eefea-80b2-4614-8346-409d085ca292",
			})
			return
		}
		adminKey, err := handler.apiKeyService.FindByKeyHash(ctx, adminKeyStr)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "1f45fbe7-4a61-472c-b533-60ab737a9419",
			})
			return
		}
		reqCtx.Set(string(ApiKeyContextKeyEntity), adminKey)
		reqCtx.Next()
	}
}

func (handler *ApiKeyHandler) GetApiKeyFromContext(reqCtx *gin.Context) (*apikey.ApiKey, bool) {
	key, ok := reqCtx.Get(string(ApiKeyContextKeyEntity))
	if !ok {
		return nil, false
	}
	return key.(*apikey.ApiKey), true
}
