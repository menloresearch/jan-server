package conversation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"menlo.ai/jan-api-gateway/app/domain/shared/id"
)

// ValidationConfig holds conversation validation rules
type ValidationConfig struct {
	MaxTitleLength          int
	MaxMetadataKeys         int
	MaxMetadataKeyLength    int
	MaxMetadataValueLength  int
	MaxContentBlocks        int
	MaxTextContentLength    int
	MaxItemsPerConversation int
	MaxItemsPerBatch        int
}

// DefaultValidationConfig returns production-ready validation rules
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxTitleLength:          200,   // Reduced from 255
		MaxMetadataKeys:         20,    // Reduced from 50
		MaxMetadataKeyLength:    50,    // Reduced from 100
		MaxMetadataValueLength:  500,   // Reduced from 1000
		MaxContentBlocks:        10,    // Reduced from 20
		MaxTextContentLength:    50000, // Reduced from 100000
		MaxItemsPerConversation: 1000,  // New limit
		MaxItemsPerBatch:        50,    // Reduced from 100
	}
}

type ConversationValidator struct {
	config *ValidationConfig
	// Compiled regex patterns for performance
	publicIDPattern    *regexp.Regexp
	metadataKeyPattern *regexp.Regexp
	idService          *id.IDService
}

// NewConversationValidator creates a new validator with security patterns
func NewConversationValidator(config *ValidationConfig) *ConversationValidator {
	return &ConversationValidator{
		config: config,
		// Validate public ID format (prevent injection)
		publicIDPattern: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
		// Validate metadata keys (alphanumeric + underscore only)
		metadataKeyPattern: regexp.MustCompile(`^[a-zA-Z0-9_]+$`),
		idService:          id.NewIDService(),
	}
}

// ValidateConversationInput performs comprehensive validation
func (v *ConversationValidator) ValidateConversationInput(title *string, metadata map[string]string) error {
	// Validate title
	if title != nil {
		if err := v.validateTitle(*title); err != nil {
			return fmt.Errorf("invalid title: %w", err)
		}
	}

	// Validate metadata
	if metadata != nil {
		if err := v.validateMetadata(metadata); err != nil {
			return fmt.Errorf("invalid metadata: %w", err)
		}
	}

	return nil
}

// ValidateItemContent performs comprehensive content validation
func (v *ConversationValidator) ValidateItemContent(content []Content) error {
	if len(content) == 0 {
		return fmt.Errorf("item must have at least one content block")
	}

	if len(content) > v.config.MaxContentBlocks {
		return fmt.Errorf("item cannot have more than %d content blocks", v.config.MaxContentBlocks)
	}

	for i, c := range content {
		if err := v.validateContentBlock(c, i); err != nil {
			return fmt.Errorf("content block %d: %w", i, err)
		}
	}

	return nil
}

// ValidatePublicID ensures public ID format is secure
func (v *ConversationValidator) ValidatePublicID(publicID string) error {
	if publicID == "" {
		return fmt.Errorf("public ID cannot be empty")
	}

	if len(publicID) < 5 || len(publicID) > 50 {
		return fmt.Errorf("public ID must be between 5 and 50 characters")
	}

	// Use centralized ID validation logic
	if strings.HasPrefix(publicID, "conv_") {
		if !v.idService.ValidateConversationID(publicID) {
			return fmt.Errorf("invalid conversation ID format")
		}
	} else if strings.HasPrefix(publicID, "msg_") {
		if !v.idService.ValidateItemID(publicID) {
			return fmt.Errorf("invalid item ID format")
		}
	} else {
		// Fallback to regex pattern for unknown prefixes
		if !v.publicIDPattern.MatchString(publicID) {
			return fmt.Errorf("public ID contains invalid characters")
		}
	}

	return nil
}

// ValidateBatchSize ensures batch operations are within limits
func (v *ConversationValidator) ValidateBatchSize(itemCount int) error {
	if itemCount == 0 {
		return fmt.Errorf("batch cannot be empty")
	}

	if itemCount > v.config.MaxItemsPerBatch {
		return fmt.Errorf("cannot process more than %d items in a single batch", v.config.MaxItemsPerBatch)
	}

	return nil
}

// Private validation methods

func (v *ConversationValidator) validateTitle(title string) error {
	// Check length
	if utf8.RuneCountInString(title) > v.config.MaxTitleLength {
		return fmt.Errorf("title cannot exceed %d characters", v.config.MaxTitleLength)
	}

	// Check for suspicious content
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("title cannot be empty or only whitespace")
	}

	return nil
}

func (v *ConversationValidator) validateMetadata(metadata map[string]string) error {
	if len(metadata) > v.config.MaxMetadataKeys {
		return fmt.Errorf("metadata cannot have more than %d keys", v.config.MaxMetadataKeys)
	}

	for key, value := range metadata {
		if err := v.validateMetadataKey(key); err != nil {
			return fmt.Errorf("invalid metadata key '%s': %w", key, err)
		}

		if err := v.validateMetadataValue(key, value); err != nil {
			return fmt.Errorf("invalid metadata value for key '%s': %w", key, err)
		}
	}

	return nil
}

func (v *ConversationValidator) validateMetadataKey(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("metadata key cannot be empty")
	}

	if len(key) > v.config.MaxMetadataKeyLength {
		return fmt.Errorf("metadata key cannot exceed %d characters", v.config.MaxMetadataKeyLength)
	}

	if !v.metadataKeyPattern.MatchString(key) {
		return fmt.Errorf("metadata key contains invalid characters (only alphanumeric and underscore allowed)")
	}

	return nil
}

func (v *ConversationValidator) validateMetadataValue(key, value string) error {
	if utf8.RuneCountInString(value) > v.config.MaxMetadataValueLength {
		return fmt.Errorf("metadata value cannot exceed %d characters", v.config.MaxMetadataValueLength)
	}

	return nil
}

func (v *ConversationValidator) validateContentBlock(content Content, index int) error {
	if content.Type == "" {
		return fmt.Errorf("content type cannot be empty")
	}

	// Validate based on content type
	switch content.Type {
	case "text":
		if content.Text != nil {
			return v.validateTextContent(content.Text.Value)
		}
	case "input_text":
		if content.InputText != nil {
			return v.validateTextContent(*content.InputText)
		}
	case "output_text":
		if content.OutputText != nil {
			return v.validateTextContent(content.OutputText.Text)
		}
	case "image":
		if content.Image != nil {
			return v.validateImageContent(content.Image)
		}
	case "file":
		if content.File != nil {
			return v.validateFileContent(content.File)
		}
	default:
		return fmt.Errorf("unsupported content type: %s", content.Type)
	}

	return nil
}

func (v *ConversationValidator) validateTextContent(text string) error {
	if utf8.RuneCountInString(text) > v.config.MaxTextContentLength {
		return fmt.Errorf("text content cannot exceed %d characters", v.config.MaxTextContentLength)
	}

	return nil
}

func (v *ConversationValidator) validateImageContent(image *ImageContent) error {
	if image.URL == "" && image.FileID == "" {
		return fmt.Errorf("image content must have either URL or file ID")
	}

	if image.URL != "" {
		// Basic URL validation
		if !strings.HasPrefix(image.URL, "http://") && !strings.HasPrefix(image.URL, "https://") && !strings.HasPrefix(image.URL, "data:") {
			return fmt.Errorf("invalid image URL format")
		}
	}

	if image.Detail != "" {
		validDetails := []string{"low", "high", "auto"}
		isValid := false
		for _, valid := range validDetails {
			if image.Detail == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid image detail level: %s", image.Detail)
		}
	}

	return nil
}

func (v *ConversationValidator) validateFileContent(file *FileContent) error {
	if file.FileID == "" {
		return fmt.Errorf("file content must have a file ID")
	}

	if file.Size < 0 {
		return fmt.Errorf("file size cannot be negative")
	}

	// Validate file size (100MB limit)
	const maxFileSize = 100 * 1024 * 1024
	if file.Size > maxFileSize {
		return fmt.Errorf("file size cannot exceed 100MB")
	}

	return nil
}
