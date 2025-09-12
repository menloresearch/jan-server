package idgen

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// GenerateSecureID generates a cryptographically secure ID with the given prefix and length
// This is a pure utility function that only handles the crypto and formatting logic
func GenerateSecureID(prefix string, length int) (string, error) {
	// The byte length required is about 3/4 of the desired string length.
	// We add 2 to be safe and avoid rounding issues or insufficient bytes.
	byteLength := (length * 3 / 4) + 2
	bytes := make([]byte, byteLength)
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

// ValidateIDFormat validates that an ID has the expected format (prefix_alphanumeric)
// This is a pure utility function that only handles format validation
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
