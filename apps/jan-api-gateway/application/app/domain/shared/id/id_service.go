package id

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// IDService provides centralized ID generation for all domain entities
// This service generates OpenAI-compatible IDs with proper prefixes and secure random suffixes
type IDService struct{}

// NewIDService creates a new instance of IDService
func NewIDService() *IDService {
	return &IDService{}
}

// GenerateSecureID generates a cryptographically secure ID with the given prefix and length
// This is used for OpenAI-compatible IDs like "conv_abc123", "msg_def456", etc.
func (s *IDService) GenerateSecureID(prefix string, length int) (string, error) {
	// Use larger byte array for better entropy (24 bytes = 32 base64 chars)
	bytes := make([]byte, 24)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base64 URL-safe format
	encoded := base64.URLEncoding.EncodeToString(bytes)
	encoded = strings.TrimRight(encoded, "=") // Remove padding

	// Truncate to desired length
	if len(encoded) > length {
		encoded = encoded[:length]
	}

	return fmt.Sprintf("%s_%s", prefix, encoded), nil
}

// GenerateConversationID generates a conversation ID with format "conv_..."
func (s *IDService) GenerateConversationID() (string, error) {
	return s.GenerateSecureID("conv", 16)
}

// GenerateItemID generates an item/message ID with format "msg_..."
func (s *IDService) GenerateItemID() (string, error) {
	return s.GenerateSecureID("msg", 16)
}

// GenerateAPIKeyID generates an API key ID with format "sk_..."
func (s *IDService) GenerateAPIKeyID() (string, error) {
	return s.GenerateSecureID("sk", 24)
}

// GenerateOrganizationID generates an organization ID with format "org_..."
func (s *IDService) GenerateOrganizationID() (string, error) {
	return s.GenerateSecureID("org", 16)
}

// GenerateProjectID generates a project ID with format "proj_..."
func (s *IDService) GenerateProjectID() (string, error) {
	return s.GenerateSecureID("proj", 16)
}

// GenerateUserID generates a user ID with format "user_..."
func (s *IDService) GenerateUserID() (string, error) {
	return s.GenerateSecureID("user", 16)
}

// GenerateAPIKeyPublicID generates an API key public ID with format "key_..."
func (s *IDService) GenerateAPIKeyPublicID() (string, error) {
	return s.GenerateSecureID("key", 16)
}

// ValidateIDFormat validates that an ID has the expected format (prefix_alphanumeric)
func (s *IDService) ValidateIDFormat(id, expectedPrefix string) bool {
	if !strings.HasPrefix(id, expectedPrefix+"_") {
		return false
	}

	// Extract the suffix after the prefix and underscore
	suffix := id[len(expectedPrefix)+1:]

	// Check that suffix is not empty and contains only valid characters
	if len(suffix) == 0 {
		return false
	}

	// Validate characters (base64 URL-safe: A-Z, a-z, 0-9, -, _)
	for _, char := range suffix {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// ValidateConversationID validates a conversation ID format
func (s *IDService) ValidateConversationID(id string) bool {
	return s.ValidateIDFormat(id, "conv")
}

// ValidateItemID validates an item/message ID format
func (s *IDService) ValidateItemID(id string) bool {
	return s.ValidateIDFormat(id, "msg")
}

// ValidateAPIKeyID validates an API key ID format
func (s *IDService) ValidateAPIKeyID(id string) bool {
	return s.ValidateIDFormat(id, "sk")
}

// ValidateOrganizationID validates an organization ID format
func (s *IDService) ValidateOrganizationID(id string) bool {
	return s.ValidateIDFormat(id, "org")
}

// ValidateProjectID validates a project ID format
func (s *IDService) ValidateProjectID(id string) bool {
	return s.ValidateIDFormat(id, "proj")
}

// ValidateUserID validates a user ID format
func (s *IDService) ValidateUserID(id string) bool {
	return s.ValidateIDFormat(id, "user")
}

// ValidateAPIKeyPublicID validates an API key public ID format
func (s *IDService) ValidateAPIKeyPublicID(id string) bool {
	return s.ValidateIDFormat(id, "key")
}
