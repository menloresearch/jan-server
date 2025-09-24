package organization

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"

	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses/openai"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type AdminApiKeyAPI struct {
	organizationService *organization.OrganizationService
	authService         *auth.AuthService
	apiKeyService       *apikey.ApiKeyService
	userService         *user.UserService
}

func NewAdminApiKeyAPI(
	organizationService *organization.OrganizationService,
	authService *auth.AuthService,
	apiKeyService *apikey.ApiKeyService,
	userService *user.UserService) *AdminApiKeyAPI {
	return &AdminApiKeyAPI{
		organizationService,
		authService,
		apiKeyService,
		userService,
	}
}

func (adminApiKeyAPI *AdminApiKeyAPI) RegisterRouter(router *gin.RouterGroup) {
	permissionAll := adminApiKeyAPI.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleAll)
	permissionOwnerOnly := adminApiKeyAPI.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleOwnerOnly)
	adminApiKeyRouter := router.Group("/admin_api_keys",
		adminApiKeyAPI.authService.AdminUserAuthMiddleware(),
		adminApiKeyAPI.authService.RegisteredUserMiddleware(),
	)
	adminApiKeyRouter.GET("",
		permissionAll,
		adminApiKeyAPI.GetAdminApiKeys,
	)
	adminApiKeyRouter.POST("",
		permissionOwnerOnly,
		adminApiKeyAPI.CreateAdminApiKey,
	)

	adminKeyPath := fmt.Sprintf("/:%s", auth.ApikeyContextKeyPublicID)
	adminApiKeyIdRoute := adminApiKeyRouter.Group(adminKeyPath, adminApiKeyAPI.authService.GetAdminApiKeyFromQuery())
	adminApiKeyIdRoute.GET("",
		permissionAll,
		adminApiKeyAPI.GetAdminApiKey,
	)
	adminApiKeyIdRoute.DELETE("",
		permissionOwnerOnly,
		adminApiKeyAPI.DeleteAdminApiKey,
	)
}

// GetAdminApiKey godoc
// @Summary Get Admin API Key
// @Description Retrieves a specific admin API key by its ID.
// @Tags Administration API
// @Security BearerAuth
// @Param id path string true "ID of the admin API key"
// @Success 200 {object} OrganizationAdminAPIKeyResponse "Successfully retrieved the admin API key"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - API key with the given ID does not exist or does not belong to the organization"
// @Router /v1/organization/admin_api_keys/{id} [get]
func (api *AdminApiKeyAPI) GetAdminApiKey(reqCtx *gin.Context) {
	entity, ok := auth.GetAdminKeyFromContext(reqCtx)
	if !ok {
		return
	}
	reqCtx.JSON(http.StatusOK, domainToOrganizationAdminAPIKeyResponse(entity))
}

// GetAdminApiKeys godoc
// @Summary List Admin API Keys
// @Description Retrieves a paginated list of all admin API keys for the authenticated organization.
// @Tags Administration API
// @Security BearerAuth
// @Param limit query int false "The maximum number of items to return" default(20)
// @Param after query string false "A cursor for use in pagination. The ID of the last object from the previous page"
// @Success 200 {object} AdminApiKeyListResponse "Successfully retrieved the list of admin API keys"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal Server Error"
// @Router /v1/organization/admin_api_keys [get]
func (api *AdminApiKeyAPI) GetAdminApiKeys(reqCtx *gin.Context) {
	apikeyService := api.apiKeyService
	ctx := reqCtx.Request.Context()
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}

	pagination, err := query.GetCursorPaginationFromQuery(reqCtx, func(lastID string) (*uint, error) {
		apiKey, err := api.apiKeyService.FindOneByFilter(ctx, apikey.ApiKeyFilter{
			PublicID: &lastID,
		})
		if err != nil {
			return nil, err
		}
		return &apiKey.ID, nil
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "5f89e23d-d4a0-45ce-ba43-ae2a9be0ca64",
			ErrorInstance: err,
		})
		return
	}

	// Fetch all API keys for the organization
	filter := apikey.ApiKeyFilter{
		OrganizationID: &orgEntity.ID,
	}
	apiKeys, err := apikeyService.Find(ctx, filter, pagination)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "32d59d1a-2eff-4b6f-a198-30a4fa9ff871",
			Error: "failed to retrieve API keys",
		})
		return
	}
	total, err := apikeyService.Count(ctx, filter)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "6d067ca3-c891-4343-b2e3-eb430278dd28",
		})
		return
	}

	var firstId *string
	var lastId *string
	hasMore := false
	if len(apiKeys) > 0 {
		firstId = &apiKeys[0].PublicID
		lastId = &apiKeys[len(apiKeys)-1].PublicID
		moreRecords, err := apikeyService.Find(ctx, filter, &query.Pagination{
			Order: pagination.Order,
			Limit: ptr.ToInt(1),
			After: &apiKeys[len(apiKeys)-1].ID,
		})
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "814c5eb7-e2e3-4476-9ae4-d8222063654a",
				Error: "failed to retrieve API keys",
			})
			return
		}
		if len(moreRecords) != 0 {
			hasMore = true
		}
	}
	// TODO; owner
	result := functional.Map(apiKeys, func(apikey *apikey.ApiKey) *OrganizationAdminAPIKeyResponse {
		return domainToOrganizationAdminAPIKeyResponse(apikey)
	})

	response := openai.ListResponse[*OrganizationAdminAPIKeyResponse]{
		Object:  "list",
		Data:    result,
		FirstID: firstId,
		LastID:  lastId,
		HasMore: hasMore,
		Total:   total,
	}
	reqCtx.JSON(http.StatusOK, response)
}

// DeleteAdminApiKey godoc
// @Summary Delete Admin API Key
// @Description Deletes an admin API key by its ID.
// @Tags Administration API
// @Security BearerAuth
// @Param id path string true "ID of the admin API key to delete"
// @Success 200 {object} AdminAPIKeyDeletedResponse "Successfully deleted the admin API key"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - API key with the given ID does not exist or does not belong to the organization"
// @Router /v1/organization/admin_api_keys/{id} [delete]
func (api *AdminApiKeyAPI) DeleteAdminApiKey(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	entity, ok := auth.GetAdminKeyFromContext(reqCtx)
	if !ok {
		return
	}

	err := api.apiKeyService.Delete(ctx, entity)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "c9a103b2-985c-44b7-9ccd-38e914a2c82b",
			Error: "invalid or missing API key",
		})
		return
	}
	reqCtx.JSON(http.StatusOK, AdminAPIKeyDeletedResponse{
		ID:      entity.PublicID,
		Object:  "organization.admin_api_key.deleted",
		Deleted: true,
	})
}

// CreateAdminApiKey creates a new admin API key for an organization.
// @Summary Create Admin API Key
// @Description Creates a new admin API key for an organization. Requires a valid admin API key in the Authorization header.
// @Tags Administration API
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateOrganizationAdminAPIKeyRequest true "API key creation request"
// @Success 200 {object} OrganizationAdminAPIKeyResponse "Successfully created admin API key"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Router /v1/organization/admin_api_keys [post]
func (api *AdminApiKeyAPI) CreateAdminApiKey(reqCtx *gin.Context) {
	apikeyService := api.apiKeyService
	ctx := reqCtx.Request.Context()
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		return
	}
	organizationEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}

	var requestPayload CreateOrganizationAdminAPIKeyRequest
	if err := reqCtx.ShouldBindJSON(&requestPayload); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b6cb35be-8a53-478d-95d1-5e1f64f35c09",
			Error: err.Error(),
		})
		return
	}

	key, hash, err := apikeyService.GenerateKeyAndHash(ctx, apikey.ApikeyTypeAdmin)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "e00e6ab3-1b43-490e-90df-aae030697f74",
			Error: err.Error(),
		})
		return
	}
	apikeyEntity, err := apikeyService.CreateApiKey(ctx, &apikey.ApiKey{
		KeyHash:        hash,
		PlaintextHint:  fmt.Sprintf("sk-..%s", key[len(key)-3:]),
		Description:    requestPayload.Name,
		Enabled:        true,
		ApikeyType:     string(apikey.ApikeyTypeAdmin),
		OwnerPublicID:  user.PublicID,
		OrganizationID: &organizationEntity.ID,
		Permissions:    "{}",
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "32d59d1a-2eff-4b6f-a198-30a4fa9ff871",
			Error: err.Error(),
		})
		return
	}
	response := domainToOrganizationAdminAPIKeyResponse(apikeyEntity)
	response.Owner = userToOwnerResponse(user)
	response.Value = key
	reqCtx.JSON(http.StatusOK, response)
}

func domainToOrganizationAdminAPIKeyResponse(entity *apikey.ApiKey) *OrganizationAdminAPIKeyResponse {
	var lastUsedAt *int64
	if entity.LastUsedAt != nil {
		lastUsedAt = ptr.ToInt64(entity.LastUsedAt.Unix())
	}
	return &OrganizationAdminAPIKeyResponse{
		Object:        string(openai.ObjectKeyAdminApiKey),
		ID:            entity.PublicID,
		Name:          entity.Description,
		RedactedValue: entity.PlaintextHint,
		CreatedAt:     entity.CreatedAt.Unix(),
		LastUsedAt:    lastUsedAt,
	}
}

func userToOwnerResponse(user *user.User) Owner {
	return Owner{
		Type:      string(openai.ApikeyTypeUser),
		Object:    string(openai.OwnerObjectOrganizationUser),
		ID:        user.PublicID,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.Unix(),
		Role:      string(openai.OwnerRoleOwner),
	}
}

// CreateOrganizationAdminAPIKeyRequest defines the request payload for creating an admin API key.
type CreateOrganizationAdminAPIKeyRequest struct {
	Name string `json:"name" binding:"required" example:"My Admin API Key" description:"The name of the API key to be created"`
}

// OrganizationAdminAPIKeyResponse defines the response structure for a created admin API key.
type OrganizationAdminAPIKeyResponse struct {
	Object        string `json:"object" example:"api_key" description:"The type of the object, typically 'api_key'"`
	ID            string `json:"id" example:"key_1234567890" description:"Unique identifier for the API key"`
	Name          string `json:"name" example:"My Admin API Key" description:"The name of the API key"`
	RedactedValue string `json:"redacted_value" example:"sk-...abcd" description:"A redacted version of the API key for display purposes"`
	CreatedAt     int64  `json:"created_at" example:"1698765432" description:"Unix timestamp when the API key was created"`
	LastUsedAt    *int64 `json:"last_used_at,omitempty" example:"1698765432" description:"Unix timestamp when the API key was last used, if available"`
	Owner         Owner  `json:"owner" description:"Details of the owner of the API key"`
	Value         string `json:"value,omitempty" example:"sk-abcdef1234567890" description:"The full API key value, included only in the response upon creation"`
}

// Owner defines the structure for the owner of an API key.
type Owner struct {
	Type      string `json:"type" example:"user" description:"The type of the owner, e.g., 'user'"`
	Object    string `json:"object" example:"user" description:"The type of the object, typically 'user'"`
	ID        string `json:"id" example:"user_1234567890" description:"Unique identifier for the owner"`
	Name      string `json:"name" example:"John Doe" description:"The name of the owner"`
	CreatedAt int64  `json:"created_at" example:"1698765432" description:"Unix timestamp when the owner was created"`
	Role      string `json:"role" example:"admin" description:"The role of the owner within the organization"`
}

type AdminApiKeyListResponse struct {
	Object  string                            `json:"object" example:"list" description:"The type of the object, always 'list'"`
	Data    []OrganizationAdminAPIKeyResponse `json:"data" description:"Array of admin API keys"`
	FirstID *string                           `json:"first_id,omitempty"`
	LastID  *string                           `json:"last_id,omitempty"`
	HasMore bool                              `json:"has_more"`
}

type AdminAPIKeyDeletedResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}
