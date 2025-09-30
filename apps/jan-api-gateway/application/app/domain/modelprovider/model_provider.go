package modelprovider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type ProviderType string

const (
	ProviderTypeJan          ProviderType = "jan"
	ProviderTypeOrganization ProviderType = "organization"
)

type ProviderVendor string

const (
	ProviderVendorJan        ProviderVendor = "jan"
	ProviderVendorOpenRouter ProviderVendor = "openrouter"
	ProviderVendorGemini     ProviderVendor = "gemini"
)

var validVendorsByType = map[ProviderType][]ProviderVendor{
	ProviderTypeJan: {
		ProviderVendorJan,
	},
	ProviderTypeOrganization: {
		ProviderVendorOpenRouter,
		ProviderVendorGemini,
	},
}

type ModelProvider struct {
	ID              uint
	PublicID        string
	OrganizationID  *uint
	ProjectID       *uint
	Name            string
	Type            ProviderType
	Vendor          ProviderVendor
	BaseURL         string
	EncryptedAPIKey string
	APIKeyHint      string
	MetadataJSON    string
	Active          bool
	LastSyncedAt    *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ProviderFilter struct {
	IDs            []uint
	PublicID       *string
	OrganizationID *uint
	ProjectIDs     *[]uint
	Type           *ProviderType
	Vendor         *ProviderVendor
	Active         *bool
	Search         *string
}

func (p ProviderType) String() string {
	return string(p)
}

func (v ProviderVendor) String() string {
	return string(v)
}

func ValidateCombination(providerType ProviderType, vendor ProviderVendor) error {
	vendors, ok := validVendorsByType[providerType]
	if !ok {
		return fmt.Errorf("unsupported provider type: %s", providerType)
	}
	for _, allowed := range vendors {
		if allowed == vendor {
			return nil
		}
	}
	return fmt.Errorf("vendor %s is not supported for provider type %s", vendor, providerType)
}

var ErrMissingAPIKey = errors.New("missing provider api key")
var ErrInvalidName = errors.New("provider name cannot be empty")
var ErrDuplicateProviderVendor = errors.New("provider vendor already exists for this scope")

func (p *ModelProvider) EnsureValid() error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrInvalidName
	}
	if err := ValidateCombination(p.Type, p.Vendor); err != nil {
		return err
	}
	if p.Type == ProviderTypeOrganization && strings.TrimSpace(p.EncryptedAPIKey) == "" {
		return ErrMissingAPIKey
	}
	return nil
}

func (p *ModelProvider) UpdateAPIKey(ciphertext, hint string) {
	p.EncryptedAPIKey = ciphertext
	p.APIKeyHint = hint
}

func (p *ModelProvider) AssignDefaults() {
	if p.MetadataJSON == "" {
		p.MetadataJSON = "{}"
	}
}

type ModelProviderRepository interface {
	Create(ctx context.Context, provider *ModelProvider) error
	Update(ctx context.Context, provider *ModelProvider) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*ModelProvider, error)
	FindByPublicID(ctx context.Context, publicID string) (*ModelProvider, error)
	Find(ctx context.Context, filter ProviderFilter, pagination *query.Pagination) ([]*ModelProvider, error)
}
