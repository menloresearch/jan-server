package apikeys

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ApiKeyAPI struct {
	apiKeyService *apikey.ApiKeyService
	userService   *user.UserService
}

func NewApiKeyAPI(
	apiKeyService *apikey.ApiKeyService,
	userService *user.UserService) *ApiKeyAPI {
	return &ApiKeyAPI{
		apiKeyService,
		userService,
	}
}

func (api *ApiKeyAPI) RegisterRouter(router *gin.RouterGroup) {
	apiKeyRouter := router.Group("/apikeys")
	apiKeyRouter.GET("/", middleware.AuthMiddleware(), api.ListApiKeys)
	apiKeyRouter.POST("/", middleware.AuthMiddleware(), api.CreateApiKey)
	apiKeyRouter.PUT("/:id", middleware.AuthMiddleware(), api.UpdateApiKey)
	apiKeyRouter.DELETE("/:id", middleware.AuthMiddleware(), api.DeleteApiKey)
}

type ApiKeyResponse struct {
	ID            uint       `json:"id"`
	Key           *string    `json:"key,omitempty"`
	PlaintextHint string     `json:"plaintext_hint"`
	Description   string     `json:"description"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	Enabled       bool       `json:"enabled"`
}

func domainToApiKeyResponse(entity *apikey.ApiKey) ApiKeyResponse {
	return ApiKeyResponse{
		ID:            entity.ID,
		PlaintextHint: entity.PlaintextHint,
		Description:   entity.Description,
		Enabled:       entity.Enabled,
		ExpiresAt:     entity.ExpiresAt,
		CreatedAt:     entity.CreatedAt,
	}
}

type CreateApiKeyRequest struct {
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// @Summary Create an API key
// @Description Create a new API key for the authenticated user.
// @Tags API Keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateApiKeyRequest true "Request body for creating an API key"
// @Success 200 {object} responses.GeneralResponse[ApiKeyResponse] "Successfully created the API key"
// @Failure 400 {object} responses.ErrorResponse "Bad request (e.g., invalid JSON, missing required fields)"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized (e.g., missing or invalid JWT)"
// @Router /jan/v1/apikeys/ [post]
func (api *ApiKeyAPI) CreateApiKey(reqCtx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "9715151d-02ab-4759-bfb7-89d717f05cd3",
			Error: err.Error(),
		})
		return
	}
	user, err := api.userService.FindByEmailAndPlatform(reqCtx, userClaim.Email, string(user.UserPlatformTypeAskJanAI))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "edf9dd05-aad4-4c1e-9795-98bf60ecf57c",
			Error: err.Error(),
		})
		return
	}
	if user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "417cff16-0325-45f7-9826-8ab24d2fef29",
		})
		return
	}

	var req CreateApiKeyRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "e6be168e-498c-41b0-85de-8e3a5bc6dfd3",
			Error: err.Error(),
		})
		return
	}

	key, hash, err := api.apiKeyService.GenerateKeyAndHash(reqCtx, apikey.OwnerTypeEphemeral)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "207373ae-f94a-4b21-bf95-7bbd8d727f84",
			Error: err.Error(),
		})
		return
	}

	// TODO: OwnerTypeEphemeral
	entity, err := api.apiKeyService.CreateApiKey(reqCtx, &apikey.ApiKey{
		KeyHash:        hash,
		PlaintextHint:  fmt.Sprintf("sk-..%s", key[len(key)-3:]),
		Description:    req.Description,
		Enabled:        true,
		OwnerType:      string(apikey.OwnerTypeEphemeral),
		OwnerID:        &user.ID,
		OrganizationID: nil,
		Permissions:    "{}",
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "9f1e1296-c4e8-43c5-94b5-391906fc12a3",
			Error: err.Error(),
		})
		return
	}
	response := domainToApiKeyResponse(entity)
	response.Key = &key
	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[ApiKeyResponse]{
		Status: responses.ResponseCodeOk,
		Result: response,
	})
}

type UpdateApiKeyRequest struct {
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Enabled     bool       `json:"enabled"`
}

// @Summary Update an API key
// @Description Update the description, expiry date, or enabled status of an existing API key.
// @Tags API Keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "API Key ID"
// @Param request body UpdateApiKeyRequest true "Request body for updating an API key"
// @Success 200 {object} responses.GeneralResponse[ApiKeyResponse] "Successfully updated the API key"
// @Failure 400 {object} responses.ErrorResponse "Bad request (e.g., invalid ID format, invalid JSON)"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized (e.g., missing or invalid JWT)"
// @Failure 404 {object} responses.ErrorResponse "API key not found"
// @Router /jan/v1/apikeys/{id} [put]
func (api *ApiKeyAPI) UpdateApiKey(reqCtx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "9715151d-02ab-4759-bfb7-89d717f05cd3",
			Error: err.Error(),
		})
		return
	}
	user, err := api.userService.FindByEmailAndPlatform(reqCtx, userClaim.Email, string(user.UserPlatformTypePlatform))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "edf9dd05-aad4-4c1e-9795-98bf60ecf57c",
			Error: err.Error(),
		})
		return
	}
	if user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "417cff16-0325-45f7-9826-8ab24d2fef29",
		})
		return
	}

	var req UpdateApiKeyRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "18b7d0d0-f385-465b-a661-e25b5a3fb6b7",
			Error: err.Error(),
		})
		return
	}

	apiKeyId, err := requests.GetIntParam(reqCtx, "id")
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b1dc31a0-a690-47de-9ce5-863ccb1e0c6f",
			Error: err.Error(),
		})
		return
	}

	entity, err := api.apiKeyService.FindById(reqCtx, uint(apiKeyId))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "80f6da91-98d9-44ff-99f6-064d5d849976",
			Error: err.Error(),
		})
		return
	}

	// TODO: check permissions
	if apikey.OwnerType(entity.OwnerType) != apikey.OwnerTypeEphemeral || entity.OwnerID != &user.ID {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code: "6f4b4448-4342-4485-8651-50806e91e163",
		})
		return
	}

	entity.Description = req.Description
	entity.ExpiresAt = req.ExpiresAt
	entity.Enabled = req.Enabled

	err = api.apiKeyService.Save(reqCtx, entity)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "80f6da91-98d9-44ff-99f6-064d5d849976",
			Error: err.Error(),
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[ApiKeyResponse]{
		Status: responses.ResponseCodeOk,
		Result: domainToApiKeyResponse(entity),
	})
}

// @Summary Delete an API key
// @Description Deletes a specific API key by its ID.
// @Tags API Keys
// @Security BearerAuth
// @Param id path int true "API Key ID"
// @Success 204 "No Content"
// @Failure 400 {object} responses.ErrorResponse "Bad request (e.g., invalid ID format)"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized (e.g., missing or invalid JWT)"
// @Failure 404 {object} responses.ErrorResponse "API key not found"
// @Router /jan/v1/apikeys/{id} [delete]
func (api *ApiKeyAPI) DeleteApiKey(reqCtx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "9715151d-02ab-4759-bfb7-89d717f05cd3",
			Error: err.Error(),
		})
		return
	}
	user, err := api.userService.FindByEmailAndPlatform(reqCtx, userClaim.Email, string(user.UserPlatformTypePlatform))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "edf9dd05-aad4-4c1e-9795-98bf60ecf57c",
			Error: err.Error(),
		})
		return
	}
	if user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "417cff16-0325-45f7-9826-8ab24d2fef29",
		})
		return
	}

	apiKeyId, err := requests.GetIntParam(reqCtx, "id")
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b1dc31a0-a690-47de-9ce5-863ccb1e0c6f",
			Error: err.Error(),
		})
		return
	}

	entity, err := api.apiKeyService.FindById(reqCtx, uint(apiKeyId))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "80f6da91-98d9-44ff-99f6-064d5d849976",
			Error: err.Error(),
		})
		return
	}

	// TODO: check permissions
	if *entity.OwnerID != user.ID {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code: "3a1541ee-3934-4bc6-a620-712318961555",
		})
		return
	}

	err = api.apiKeyService.Delete(reqCtx, entity)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "fa391d81-699d-43fa-ba02-dd2cb91c1a2a",
			Error: err.Error(),
		})
		return
	}

	reqCtx.Status(http.StatusNoContent)
}

// @Summary List API keys
// @Description Get a list of API keys for the authenticated user with pagination.
// @Tags API Keys
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number for pagination"
// @Param pageSize query int false "Number of items per page"
// @Success 200 {object} responses.ListResponse[[]ApiKeyResponse] "Successfully retrieved the list of API keys"
// @Failure 400 {object} responses.ErrorResponse "Bad request (e.g., invalid query parameters)"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized (e.g., missing or invalid JWT)"
// @Failure 500 {object} responses.ErrorResponse "Internal Server Error"
// @Router /jan/v1/apikeys/ [get]
func (api *ApiKeyAPI) ListApiKeys(reqCtx *gin.Context) {
	userClaim, ok := reqCtx.Get(auth.ContextUserClaim)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "fbc49daf-2f73-4778-9362-5680da391190",
		})
		return
	}
	u, ok := userClaim.(*auth.UserClaim)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "e8a957c3-e107-4244-8625-3f3a1d29ce5c",
		})
		return
	}
	user, err := api.userService.FindByEmailAndPlatform(reqCtx, u.Email, string(user.UserPlatformTypePlatform))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "e8a957c3-e107-4244-8625-3f3a1d29ce5c",
			Error: err.Error(),
		})
		return
	}
	if user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "417cff16-0325-45f7-9826-8ab24d2fef29",
		})
		return
	}

	pagination, err := query.GetPaginationFromQuery(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "a46b14ea-20ef-4965-ad29-3d00c7e68389",
			Error: err.Error(),
		})
		return
	}

	filter := apikey.ApiKeyFilter{
		OwnerType: ptr.ToString(string(apikey.OwnerTypeEphemeral)),
		OwnerID:   ptr.ToUint(user.ID),
	}
	apiKeysCount, err := api.apiKeyService.Count(reqCtx, filter)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "8a1a3660-8945-46c2-916e-8a2645ecf0e3",
			Error: err.Error(),
		})
		return
	}

	entities, err := api.apiKeyService.Find(reqCtx, filter, pagination)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "2620d11a-8018-4a3b-b7f2-a2351ed9f4ce",
			Error: err.Error(),
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.ListResponse[ApiKeyResponse]{
		Status:  responses.ResponseCodeOk,
		Total:   apiKeysCount,
		Results: functional.Map(entities, domainToApiKeyResponse),
	})
}
