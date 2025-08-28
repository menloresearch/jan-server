package apikey

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type ApiKeyService struct {
	repo ApiKeyRepository
}

func NewService(repo ApiKeyRepository) *ApiKeyService {
	return &ApiKeyService{repo: repo}
}

func (s *ApiKeyService) GenerateKeyAndHash(ctx context.Context, ownerType OwnerType) (string, string, error) {
	randomBytes := make([]byte, 117)
	_, err := io.ReadFull(rand.Reader, randomBytes)
	if err != nil {
		return "", "", err
	}
	randomString := base64.URLEncoding.EncodeToString(randomBytes)
	apikey := fmt.Sprintf("sk-%s-%s", ownerType, randomString)
	hash := s.HashKey(ctx, apikey)
	return apikey, hash, nil
}

func (s *ApiKeyService) generatePublicID() (string, error) {
	bytes := make([]byte, 12)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	key := base64.URLEncoding.EncodeToString(bytes)
	key = strings.TrimRight(key, "=")

	if len(key) > 16 {
		key = key[:16]
	} else if len(key) < 16 {
		extra := make([]byte, 16-len(key))
		_, err := rand.Read(extra)
		if err != nil {
			return "", err
		}
		key += base64.URLEncoding.EncodeToString(extra)[:16-len(key)]
	}

	return fmt.Sprintf("key_%s", key), nil
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
