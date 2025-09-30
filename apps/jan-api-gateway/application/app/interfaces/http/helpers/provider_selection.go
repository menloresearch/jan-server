package helpers

import (
	"fmt"
	"strings"

	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
	infrainference "menlo.ai/jan-api-gateway/app/infrastructure/inference"
)

func ParseProviderSelection(providerID, providerType, providerVendor string) (infrainference.ProviderSelection, error) {
	selection := infrainference.ProviderSelection{}
	selection.ProviderID = strings.TrimSpace(providerID)

	if providerType != "" {
		normalized := strings.ToLower(providerType)
		pt := modelprovider.ProviderType(normalized)
		switch pt {
		case modelprovider.ProviderTypeJan, modelprovider.ProviderTypeOrganization:
			selection.ProviderType = pt
		default:
			return selection, fmt.Errorf("invalid provider_type: %s", providerType)
		}
	}

	if providerVendor != "" {
		normalized := strings.ToLower(providerVendor)
		vendor := modelprovider.ProviderVendor(normalized)
		switch vendor {
		case modelprovider.ProviderVendorJan, modelprovider.ProviderVendorOpenRouter, modelprovider.ProviderVendorGemini:
			selection.Vendor = vendor
		default:
			return selection, fmt.Errorf("invalid provider_vendor: %s", providerVendor)
		}
	}

	return selection, nil
}
