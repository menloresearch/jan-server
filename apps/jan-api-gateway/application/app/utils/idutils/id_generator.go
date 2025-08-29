package idutils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// GenerateSecureID generates a cryptographically secure ID with the given prefix and length
// This is used for OpenAI-compatible IDs like "conv_abc123", "msg_def456", etc.
func GenerateSecureID(prefix string, length int) (string, error) {
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
func GenerateConversationID() (string, error) {
	return GenerateSecureID("conv", 16)
}

// GenerateItemID generates an item/message ID with format "msg_..."
func GenerateItemID() (string, error) {
	return GenerateSecureID("msg", 16)
}

// GenerateAPIKeyID generates an API key ID with format "sk_..."
func GenerateAPIKeyID() (string, error) {
	return GenerateSecureID("sk", 24)
}

// GenerateOrganizationID generates an organization ID with format "org_..."
func GenerateOrganizationID() (string, error) {
	return GenerateSecureID("org", 16)
}

// GenerateProjectID generates a project ID with format "proj_..."
func GenerateProjectID() (string, error) {
	return GenerateSecureID("proj", 16)
}

// ValidateIDFormat validates that an ID has the expected format (prefix_alphanumeric)
func ValidateIDFormat(id, expectedPrefix string) bool {
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
func ValidateConversationID(id string) bool {
	return ValidateIDFormat(id, "conv")
}

// ValidateItemID validates an item/message ID format
func ValidateItemID(id string) bool {
	return ValidateIDFormat(id, "msg")
}

// ValidateAPIKeyID validates an API key ID format
func ValidateAPIKeyID(id string) bool {
	return ValidateIDFormat(id, "sk")
}

// ValidateOrganizationID validates an organization ID format
func ValidateOrganizationID(id string) bool {
	return ValidateIDFormat(id, "org")
}

// ValidateProjectID validates a project ID format
func ValidateProjectID(id string) bool {
	return ValidateIDFormat(id, "proj")
}
