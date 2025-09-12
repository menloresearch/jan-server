package organization

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"

	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses/openai"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type AdminApiKeyAPI struct {
	organizationService *organization.OrganizationService
	apiKeyService       *apikey.ApiKeyService
	userService         *user.UserService
}

func NewAdminApiKeyAPI(organizationService *organization.OrganizationService, apiKeyService *apikey.ApiKeyService, userService *user.UserService) *AdminApiKeyAPI {
	return &AdminApiKeyAPI{
		organizationService,
		apiKeyService,
		userService,
	}
}

func (adminApiKeyAPI *AdminApiKeyAPI) RegisterRouter(router *gin.RouterGroup) {
	adminApiKeyRouter := router.Group("/admin_api_keys")
	adminApiKeyRouter.GET("", adminApiKeyAPI.GetAdminApiKeys)
	adminApiKeyRouter.GET("/:id", adminApiKeyAPI.GetAdminApiKey)
	adminApiKeyRouter.POST("", adminApiKeyAPI.CreateAdminApiKey)
	adminApiKeyRouter.DELETE("/:id", adminApiKeyAPI.DeleteAdminApiKey)
}

// GetAdminApiKey godoc
// @Summary Get Admin API Key
// @Description Retrieves a specific admin API key by its ID.
// @Tags Organizations
// @Security BearerAuth
// @Param id path string true "ID of the admin API key"
// @Success 200 {object} OrganizationAdminAPIKeyResponse "Successfully retrieved the admin API key"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - API key with the given ID does not exist or does not belong to the organization"
// @Router /v1/organization/admin_api_keys/{id} [get]
func (api *AdminApiKeyAPI) GetAdminApiKey(reqCtx *gin.Context) {
	apikeyService := api.apiKeyService
	ctx := reqCtx.Request.Context()
	adminKeyEntity, err := api.validateAdminKey(reqCtx)
	if err != nil {
		return
	}
	publicID := reqCtx.Param("id")
	if publicID == "" {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "2ab393c5-708d-42bc-a785-dcbdcf429ad1",
			Error: "invalid or missing API key",
		})
		return
	}
	entity, err := apikeyService.FindByPublicID(ctx, publicID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "26b68f41-0eb0-4fca-8365-613742ef9204",
			Error: "invalid or missing API key",
		})
		return
	}
	if entity.OrganizationID != adminKeyEntity.OrganizationID {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "4b656858-4212-451a-9ab6-23bc09dcc357",
			Error: "invalid or missing API key",
		})
		return
	}
	reqCtx.JSON(http.StatusOK, domainToOrganizationAdminAPIKeyResponse(entity))
}

// GetAdminApiKeys godoc
// @Summary List Admin API Keys
// @Description Retrieves a paginated list of all admin API keys for the authenticated organization.
// @Tags Organizations
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
	adminKeyEntity, err := api.validateAdminKey(reqCtx)
	if err != nil {
		return
	}

	pagination, err := query.GetPaginationFromQuery(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "5a5931c8-2d34-453a-9cc1-f31a69c97bc8",
			Error: "invalid or missing query parameter",
		})
		return
	}
	afterStr := reqCtx.Query("after")
	if afterStr != "" {
		entity, err := apikeyService.Find(ctx, apikey.ApiKeyFilter{
			PublicID: &afterStr,
		}, &query.Pagination{
			Limit: ptr.ToInt(1),
		})
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "f6bce0e5-8534-41f5-9f44-701894ddfd47",
				Error: err.Error(),
			})
			return
		}
		if len(entity) == 0 {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "f6bce0e5-8534-41f5-9f44-701894ddfd47",
				Error: "invalid or missing API key",
			})
			return
		}
		pagination.After = &entity[0].ID
	}

	// Fetch all API keys for the organization
	filter := apikey.ApiKeyFilter{
		OrganizationID: adminKeyEntity.OrganizationID,
	}
	apiKeys, err := apikeyService.Find(ctx, filter, pagination)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "32d59d1a-2eff-4b6f-a198-30a4fa9ff871",
			Error: "failed to retrieve API keys",
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

	// TODO: Join/Select With Owner
	result := functional.Map(apiKeys, func(apikey *apikey.ApiKey) OrganizationAdminAPIKeyResponse {
		return domainToOrganizationAdminAPIKeyResponse(apikey)
	})

	response := AdminApiKeyListResponse{
		Object:  "list",
		Data:    result,
		FirstID: firstId,
		LastID:  lastId,
		HasMore: hasMore,
	}
	reqCtx.JSON(http.StatusOK, response)
}

// DeleteAdminApiKey godoc
// @Summary Delete Admin API Key
// @Description Deletes an admin API key by its ID.
// @Tags Organizations
// @Security BearerAuth
// @Param id path string true "ID of the admin API key to delete"
// @Success 200 {object} AdminAPIKeyDeletedResponse "Successfully deleted the admin API key"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - API key with the given ID does not exist or does not belong to the organization"
// @Router /v1/organization/admin_api_keys/{id} [delete]
func (api *AdminApiKeyAPI) DeleteAdminApiKey(reqCtx *gin.Context) {
	apikeyService := api.apiKeyService
	ctx := reqCtx.Request.Context()

	adminKeyEntity, err := api.validateAdminKey(reqCtx)
	if err != nil {
		return
	}

	publicID := reqCtx.Param("id")
	if publicID == "" {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "2ab393c5-708d-42bc-a785-dcbdcf429ad1",
			Error: "invalid or missing API key",
		})
		return
	}
	entity, err := apikeyService.FindByPublicID(ctx, publicID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "26b68f41-0eb0-4fca-8365-613742ef9204",
			Error: "invalid or missing API key",
		})
		return
	}
	if entity.ApikeyType != string(apikey.ApikeyTypeAdmin) {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "f5a31a69-10b0-416c-8fa2-8183dca24eb9",
			Error: "invalid or missing API key",
		})
		return
	}

	if entity.OrganizationID != adminKeyEntity.OrganizationID {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "13e172b1-7eec-4b1a-af81-3107ca5a6b0e",
			Error: "invalid or missing API key",
		})
		return
	}

	err = apikeyService.Delete(ctx, entity)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
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
// @Tags Organizations
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
	userService := api.userService
	ctx := reqCtx.Request.Context()

	var requestPayload CreateOrganizationAdminAPIKeyRequest
	if err := reqCtx.ShouldBindJSON(&requestPayload); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b6cb35be-8a53-478d-95d1-5e1f64f35c09",
			Error: err.Error(),
		})
		return
	}

	adminKeyEntity, err := api.validateAdminKey(reqCtx)
	if err != nil {
		return
	}

	userEntity, err := userService.FindByPublicID(ctx, adminKeyEntity.OwnerPublicID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "f773c7a0-618a-42e1-ab14-3792e6311fe7",
			Error: "invalid or missing API key",
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
		OwnerPublicID:  adminKeyEntity.OwnerPublicID,
		OrganizationID: adminKeyEntity.OrganizationID,
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
	response.Owner = domainToOwnerResponse(userEntity)
	response.Value = key
	reqCtx.JSON(http.StatusOK, response)
}

func (api *AdminApiKeyAPI) validateAdminKey(reqCtx *gin.Context) (*apikey.ApiKey, error) {
	apikeyService := api.apiKeyService
	ctx := reqCtx.Request.Context()
	// Extract and validate the admin API key from the Authorization header
	adminKey, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "7ce7c4e8-4fd3-4f60-ac7e-aee87e2e8d4d",
			Error: "invalid or missing API key",
		})
		return nil, fmt.Errorf("invalid token")
	}

	// Verify the provided admin API key
	adminKeyEntity, err := apikeyService.FindByKey(ctx, adminKey)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "21882dd7-4945-4d7b-9582-07cec0f450ce",
			Error: "invalid or missing API key",
		})
		return nil, err
	}

	if adminKeyEntity.ApikeyType != string(apikey.ApikeyTypeAdmin) {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "27828731-bcb8-450b-81c0-3f9e2ff5ef12",
			Error: "invalid or missing API key",
		})
		return nil, fmt.Errorf("invalid or missing API key")
	}
	return adminKeyEntity, nil
}

func domainToOrganizationAdminAPIKeyResponse(entity *apikey.ApiKey) OrganizationAdminAPIKeyResponse {
	var lastUsedAt *int64
	if entity.LastUsedAt != nil {
		lastUsedAt = ptr.ToInt64(entity.LastUsedAt.Unix())
	}
	return OrganizationAdminAPIKeyResponse{
		Object:        string(openai.ObjectKeyAdminApiKey),
		ID:            entity.PublicID,
		Name:          entity.Description,
		RedactedValue: entity.PlaintextHint,
		CreatedAt:     entity.CreatedAt.Unix(),
		LastUsedAt:    lastUsedAt,
	}
}

func domainToOwnerResponse(user *user.User) Owner {
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
