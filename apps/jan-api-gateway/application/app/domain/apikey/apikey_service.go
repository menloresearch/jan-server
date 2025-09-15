package apikey

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"

	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/config/environment_variables"
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
	prefix := fmt.Sprintf("%s-%s", ApikeyPrefix, ownerType)
	baseKey, err := idgen.GenerateSecureID(prefix, 24)
	if err != nil {
		return "", "", err
	}

	hash := s.HashKey(ctx, baseKey)
	return baseKey, hash, nil
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

func (s *ApiKeyService) FindOneByFilter(ctx context.Context, filter ApiKeyFilter) (*ApiKey, error) {
	return s.repo.FindOneByFilter(ctx, filter)
}
