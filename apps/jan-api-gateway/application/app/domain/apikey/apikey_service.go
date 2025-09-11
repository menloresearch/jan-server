package apikey

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"

	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type ApikeyContextKey string

const (
	ApikeyContextKeyEntity   ApikeyContextKey = "ApikeyContextKeyEntity"
	ApikeyContextKeyPublicID ApikeyContextKey = "apikey_public_id"
)

type ApiKeyService struct {
	repo                ApiKeyRepository
	organizationService *organization.OrganizationService
}

func NewService(
	repo ApiKeyRepository,
	organizationService *organization.OrganizationService,
) *ApiKeyService {
	return &ApiKeyService{
		repo,
		organizationService,
	}
}

const ApikeyPrefix = "sk"

func (s *ApiKeyService) GenerateKeyAndHash(ctx context.Context, ownerType ApikeyType) (string, string, error) {
	baseKey, err := idgen.GenerateSecureID(ApikeyPrefix, 24)
	if err != nil {
		return "", "", err
	}

	// Business rule: Format as sk_<ownerType>-<random> for identification
	apikey := fmt.Sprintf("%s-%s", ownerType, baseKey)
	hash := s.HashKey(ctx, apikey)
	return apikey, hash, nil
}

func (s *ApiKeyService) generatePublicID() (string, error) {
	return idgen.GenerateSecureID("key", 16)
}

func (s *ApiKeyService) HashKey(ctx context.Context, key string) string {
	h := hmac.New(sha256.New, []byte(environment_variables.EnvironmentVariables.APIKEY_SECRET))
	h.Write([]byte(key))

	return hex.EncodeToString(h.Sum(nil))
}

func (s *ApiKeyService) CreateApiKey(ctx context.Context, apiKey *ApiKey) (*ApiKey, error) {
	publicId, err := s.generatePublicID()
	if err != nil {
		return nil, err
	}
	apiKey.PublicID = publicId
	if err := s.repo.Create(ctx, apiKey); err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (s *ApiKeyService) Delete(ctx context.Context, apiKey *ApiKey) error {
	if err := s.repo.DeleteByID(ctx, apiKey.ID); err != nil {
		return err
	}
	return nil
}

func (s *ApiKeyService) FindById(ctx context.Context, id uint) (*ApiKey, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ApiKeyService) FindByPublicID(ctx context.Context, publicID string) (*ApiKey, error) {
	entities, err := s.repo.FindByFilter(ctx, ApiKeyFilter{
		PublicID: &publicID,
	}, nil)
	if err != nil {
		return nil, err
	}
	if len(entities) != 1 {
		return nil, fmt.Errorf("record not found")
	}
	return entities[0], nil
}

func (s *ApiKeyService) FindByKeyHash(ctx context.Context, key string) (*ApiKey, error) {
	return s.repo.FindByKeyHash(ctx, key)
}

func (s *ApiKeyService) FindByKey(ctx context.Context, key string) (*ApiKey, error) {
	return s.repo.FindByKeyHash(ctx, s.HashKey(ctx, key))
}

func (s *ApiKeyService) Find(ctx context.Context, filter ApiKeyFilter, p *query.Pagination) ([]*ApiKey, error) {
	return s.repo.FindByFilter(ctx, filter, p)
}

func (s *ApiKeyService) Count(ctx context.Context, filter ApiKeyFilter) (int64, error) {
	return s.repo.Count(ctx, filter)
}

func (s *ApiKeyService) Save(ctx context.Context, entity *ApiKey) error {
	return s.repo.Update(ctx, entity)
}

func (s *ApiKeyService) GetAdminApiKeyFromQuery() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		user, ok := user.GetUserFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "72ca928d-bd8b-44f8-af70-1a9e33b58295",
			})
			return
		}

		publicID := reqCtx.Param(string(ApikeyContextKeyPublicID))
		if publicID == "" {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "9c6ed28c-1dab-4fab-945a-f0efa2dec1eb",
				Error: "missing apikey public ID",
			})
			return
		}

		adminKeyEntity, err := s.repo.FindOneByFilter(ctx, ApiKeyFilter{
			PublicID: &publicID,
		})
		if adminKeyEntity == nil || err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code: "f4f47443-0c80-4c7a-bedc-ac30ec49f494",
			})
			return
		}
		memberEntity, err := s.organizationService.FindOneMemberByFilter(ctx, organization.OrganizationMemberFilter{
			UserID:         &user.ID,
			OrganizationID: adminKeyEntity.OrganizationID,
		})
		if memberEntity == nil || err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "56a9fa87-ddd7-40b7-b2d6-94ae41a600f8",
			})
			return
		}
		SetAdminKeyFromContext(reqCtx, adminKeyEntity)
	}
}

func GetAdminKeyFromContext(reqCtx *gin.Context) (*ApiKey, bool) {
	apiKey, ok := reqCtx.Get(string(ApikeyContextKeyEntity))
	if !ok {
		return nil, false
	}
	v, ok := apiKey.(*ApiKey)
	if !ok {
		return nil, false
	}
	return v, true
}

func SetAdminKeyFromContext(reqCtx *gin.Context, apiKey *ApiKey) {
	reqCtx.Set(string(ApikeyContextKeyEntity), apiKey)
}
