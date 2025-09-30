package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/modelproviderrepo"
	"menlo.ai/jan-api-gateway/app/utils/crypto"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

type repository interface {
	Create(ctx context.Context, provider *modelprovider.ModelProvider) error
	Update(ctx context.Context, provider *modelprovider.ModelProvider) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*modelprovider.ModelProvider, error)
	FindByPublicID(ctx context.Context, publicID string) (*modelprovider.ModelProvider, error)
	Find(ctx context.Context, filter modelprovider.ProviderFilter, pagination *query.Pagination) ([]*modelprovider.ModelProvider, error)
	Count(ctx context.Context, filter modelprovider.ProviderFilter) (int64, error)
}

var _ repository = (*modelproviderrepo.ModelProviderGormRepository)(nil)

type ModelProviderService struct {
	repo             repository
	encryptionSecret string
}

type CreateOrganizationProviderInput struct {
	OrganizationID uint
	ProjectID      *uint
	Name           string
	Vendor         modelprovider.ProviderVendor
	BaseURL        string
	APIKey         string
	Metadata       map[string]any
	Active         bool
}

type UpdateOrganizationProviderInput struct {
	PublicID string
	Name     *string
	BaseURL  *string
	APIKey   *string
	Active   *bool
	Metadata map[string]any
}

func NewService(repo repository, encryptionSecret string) (*ModelProviderService, error) {
	if encryptionSecret == "" {
		return nil, crypto.ErrSecretEmpty
	}
	return &ModelProviderService{repo: repo, encryptionSecret: encryptionSecret}, nil
}

func (s *ModelProviderService) List(ctx context.Context, filter modelprovider.ProviderFilter, pagination *query.Pagination) ([]*modelprovider.ModelProvider, error) {
	return s.repo.Find(ctx, filter, pagination)
}

func (s *ModelProviderService) Count(ctx context.Context, filter modelprovider.ProviderFilter) (int64, error) {
	return s.repo.Count(ctx, filter)
}

func (s *ModelProviderService) GetByPublicID(ctx context.Context, publicID string) (*modelprovider.ModelProvider, error) {
	return s.repo.FindByPublicID(ctx, publicID)
}

func (s *ModelProviderService) GetByPublicIDWithKey(ctx context.Context, publicID string) (*modelprovider.ModelProvider, string, error) {
	provider, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, "", err
	}
	key, err := s.decryptAPIKey(provider)
	if err != nil {
		return nil, "", err
	}
	return provider, key, nil
}

func (s *ModelProviderService) RegisterOrganizationProvider(ctx context.Context, input CreateOrganizationProviderInput) (*modelprovider.ModelProvider, error) {
	provider := &modelprovider.ModelProvider{
		OrganizationID: &input.OrganizationID,
		ProjectID:      input.ProjectID,
		Name:           strings.TrimSpace(input.Name),
		Type:           modelprovider.ProviderTypeOrganization,
		Vendor:         input.Vendor,
		BaseURL:        strings.TrimSpace(input.BaseURL),
		Active:         input.Active,
	}
	metadataJSON, err := marshalMetadata(input.Metadata)
	if err != nil {
		return nil, err
	}
	provider.MetadataJSON = metadataJSON

	if err := modelprovider.ValidateCombination(provider.Type, provider.Vendor); err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.APIKey) == "" {
		return nil, modelprovider.ErrMissingAPIKey
	}
	if err := s.applyAPIKey(provider, input.APIKey); err != nil {
		return nil, err
	}

	provider.AssignDefaults()
	if err := provider.EnsureValid(); err != nil {
		return nil, err
	}
	if err := s.assignPublicID(provider); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *ModelProviderService) UpdateOrganizationProvider(ctx context.Context, input UpdateOrganizationProviderInput) (*modelprovider.ModelProvider, error) {
	provider, err := s.repo.FindByPublicID(ctx, input.PublicID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		provider.Name = strings.TrimSpace(*input.Name)
	}
	if input.BaseURL != nil {
		provider.BaseURL = strings.TrimSpace(*input.BaseURL)
	}
	if input.Active != nil {
		provider.Active = *input.Active
	}
	if input.Metadata != nil {
		metadataJSON, err := marshalMetadata(input.Metadata)
		if err != nil {
			return nil, err
		}
		provider.MetadataJSON = metadataJSON
	}
	if input.APIKey != nil {
		if err := s.applyAPIKey(provider, *input.APIKey); err != nil {
			return nil, err
		}
	}
	if err := provider.EnsureValid(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *ModelProviderService) DeleteByPublicID(ctx context.Context, publicID string) error {
	provider, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return err
	}
	return s.repo.DeleteByID(ctx, provider.ID)
}

func (s *ModelProviderService) applyAPIKey(provider *modelprovider.ModelProvider, apiKey string) error {
	ciphertext, err := crypto.EncryptString(s.encryptionSecret, apiKey)
	if err != nil {
		return err
	}
	provider.UpdateAPIKey(ciphertext, generateHint(apiKey))
	return nil
}

func (s *ModelProviderService) decryptAPIKey(provider *modelprovider.ModelProvider) (string, error) {
	if strings.TrimSpace(provider.EncryptedAPIKey) == "" {
		return "", nil
	}
	return crypto.DecryptString(s.encryptionSecret, provider.EncryptedAPIKey)
}

func (s *ModelProviderService) assignPublicID(provider *modelprovider.ModelProvider) error {
	publicID, err := idgen.GenerateSecureID("prov", 16)
	if err != nil {
		return err
	}
	provider.PublicID = publicID
	return nil
}

func marshalMetadata(metadata map[string]any) (string, error) {
	if metadata == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func generateHint(apiKey string) string {
	clean := strings.TrimSpace(apiKey)
	if len(clean) <= 4 {
		return clean
	}
	return fmt.Sprintf("***%s", clean[len(clean)-4:])
}

func (s *ModelProviderService) TouchSync(ctx context.Context, publicID string) error {
	provider, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return err
	}
	now := time.Now()
	provider.LastSyncedAt = &now
	return s.repo.Update(ctx, provider)
}
