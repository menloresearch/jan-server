package apikey

import (
	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/query"
)

type ApiKeyService struct {
	repo ApiKeyRepository
}

func NewService(repo ApiKeyRepository) *ApiKeyService {
	return &ApiKeyService{repo: repo}
}

func (s *ApiKeyService) CreateApiKey(ctx context.Context, apiKey *ApiKey) (*ApiKey, error) {
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

func (s *ApiKeyService) FindByKey(ctx context.Context, key string) (*ApiKey, error) {
	return s.repo.FindByKey(ctx, key)
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
