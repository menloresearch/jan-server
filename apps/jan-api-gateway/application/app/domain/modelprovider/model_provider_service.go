package modelprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/utils/crypto"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type ModelProviderService struct {
	repo             ModelProviderRepository
	encryptionSecret string
}

type CreateOrganizationProviderInput struct {
	OrganizationID uint
	ProjectID      *uint
	Name           string
	Vendor         ProviderVendor
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

func NewModelProviderService(repo ModelProviderRepository) *ModelProviderService {
	secret := environment_variables.EnvironmentVariables.MODEL_PROVIDER_SECRET
	return &ModelProviderService{repo: repo, encryptionSecret: secret}
}

func (s *ModelProviderService) List(ctx context.Context, filter ProviderFilter, pagination *query.Pagination) ([]*ModelProvider, error) {
	return s.repo.Find(ctx, filter, pagination)
}

func (s *ModelProviderService) GetByPublicID(ctx context.Context, publicID string) (*ModelProvider, error) {
	return s.repo.FindByPublicID(ctx, publicID)
}

func (s *ModelProviderService) GetByPublicIDWithKey(ctx context.Context, publicID string) (*ModelProvider, string, error) {
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

func (s *ModelProviderService) RegisterOrganizationProvider(ctx context.Context, input CreateOrganizationProviderInput) (*ModelProvider, error) {
	provider := &ModelProvider{
		OrganizationID: &input.OrganizationID,
		ProjectID:      input.ProjectID,
		Name:           strings.TrimSpace(input.Name),
		Type:           ProviderTypeOrganization,
		Vendor:         input.Vendor,
		BaseURL:        strings.TrimSpace(input.BaseURL),
		Active:         input.Active,
	}
	metadataJSON, err := marshalMetadata(input.Metadata)
	if err != nil {
		return nil, err
	}
	provider.MetadataJSON = metadataJSON

	if err := ValidateCombination(provider.Type, provider.Vendor); err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.APIKey) == "" {
		return nil, ErrMissingAPIKey
	}
	if err := s.applyAPIKey(provider, input.APIKey); err != nil {
		return nil, err
	}

	provider.AssignDefaults()
	if err := provider.EnsureValid(); err != nil {
		return nil, err
	}
	if err := s.ensureVendorUnique(ctx, provider); err != nil {
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

func (s *ModelProviderService) UpdateOrganizationProvider(ctx context.Context, input UpdateOrganizationProviderInput) (*ModelProvider, error) {
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

func (s *ModelProviderService) ensureVendorUnique(ctx context.Context, provider *ModelProvider) error {
	if provider == nil || provider.OrganizationID == nil {
		return nil
	}

	filter := ProviderFilter{
		OrganizationID: provider.OrganizationID,
	}
	vendor := provider.Vendor
	filter.Vendor = &vendor
	providerType := provider.Type
	filter.Type = &providerType

	if provider.ProjectID != nil {
		ids := []uint{*provider.ProjectID}
		filter.ProjectIDs = &ids
	} else {
		ids := []uint{}
		filter.ProjectIDs = &ids
	}

	providers, err := s.repo.Find(ctx, filter, nil)
	if err != nil {
		return err
	}
	if len(providers) > 0 {
		return fmt.Errorf("%w: %s", ErrDuplicateProviderVendor, provider.Vendor)
	}
	return nil
}

func (s *ModelProviderService) applyAPIKey(provider *ModelProvider, apiKey string) error {
	ciphertext, err := crypto.EncryptString(s.encryptionSecret, apiKey)
	if err != nil {
		return err
	}
	provider.UpdateAPIKey(ciphertext, generateHint(apiKey))
	return nil
}

func (s *ModelProviderService) decryptAPIKey(provider *ModelProvider) (string, error) {
	if strings.TrimSpace(provider.EncryptedAPIKey) == "" {
		return "", nil
	}
	return crypto.DecryptString(s.encryptionSecret, provider.EncryptedAPIKey)
}

func (s *ModelProviderService) assignPublicID(provider *ModelProvider) error {
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
