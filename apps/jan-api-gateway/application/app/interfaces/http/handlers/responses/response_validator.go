package responses

import (
	"fmt"
	"strings"

	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// ValidateCreateResponseRequest validates a CreateResponseRequest
func ValidateCreateResponseRequest(req *requesttypes.CreateResponseRequest) *ValidationErrors {
	var errors []ValidationError

	// Validate model
	if req.Model == "" {
		errors = append(errors, ValidationError{
			Field:   "model",
			Message: "model is required",
		})
	}

	// Validate input
	if err := validateInput(req.Input); err != nil {
		errors = append(errors, *err...)
	}

	// Validate temperature
	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 2 {
			errors = append(errors, ValidationError{
				Field:   "temperature",
				Message: "temperature must be between 0 and 2",
			})
		}
	}

	// Validate top_p
	if req.TopP != nil {
		if *req.TopP < 0 || *req.TopP > 1 {
			errors = append(errors, ValidationError{
				Field:   "top_p",
				Message: "top_p must be between 0 and 1",
			})
		}
	}

	// Validate top_k
	if req.TopK != nil {
		if *req.TopK < 1 {
			errors = append(errors, ValidationError{
				Field:   "top_k",
				Message: "top_k must be greater than 0",
			})
		}
	}

	// Validate repetition_penalty
	if req.RepetitionPenalty != nil {
		if *req.RepetitionPenalty < 0 || *req.RepetitionPenalty > 2 {
			errors = append(errors, ValidationError{
				Field:   "repetition_penalty",
				Message: "repetition_penalty must be between 0 and 2",
			})
		}
	}

	// Validate presence_penalty
	if req.PresencePenalty != nil {
		if *req.PresencePenalty < -2 || *req.PresencePenalty > 2 {
			errors = append(errors, ValidationError{
				Field:   "presence_penalty",
				Message: "presence_penalty must be between -2 and 2",
			})
		}
	}

	// Validate frequency_penalty
	if req.FrequencyPenalty != nil {
		if *req.FrequencyPenalty < -2 || *req.FrequencyPenalty > 2 {
			errors = append(errors, ValidationError{
				Field:   "frequency_penalty",
				Message: "frequency_penalty must be between -2 and 2",
			})
		}
	}

	// Validate max_tokens
	if req.MaxTokens != nil {
		if *req.MaxTokens < 1 {
			errors = append(errors, ValidationError{
				Field:   "max_tokens",
				Message: "max_tokens must be greater than 0",
			})
		}
	}

	// Validate timeout
	if req.Timeout != nil {
		if *req.Timeout < 1 {
			errors = append(errors, ValidationError{
				Field:   "timeout",
				Message: "timeout must be greater than 0",
			})
		}
	}

	// Validate response_format
	if req.ResponseFormat != nil {
		if err := validateResponseFormat(req.ResponseFormat); err != nil {
			errors = append(errors, *err...)
		}
	}

	// Validate tools
	if req.Tools != nil {
		if err := validateTools(req.Tools); err != nil {
			errors = append(errors, *err...)
		}
	}

	// Validate tool_choice
	if req.ToolChoice != nil {
		if err := validateToolChoice(req.ToolChoice); err != nil {
			errors = append(errors, *err...)
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// validateInput validates the input field (can be string or array of strings)
func validateInput(input interface{}) *[]ValidationError {
	var errors []ValidationError

	if input == nil {
		errors = append(errors, ValidationError{
			Field:   "input",
			Message: "input is required",
		})
		return &errors
	}

	switch v := input.(type) {
	case string:
		if v == "" {
			errors = append(errors, ValidationError{
				Field:   "input",
				Message: "input string cannot be empty",
			})
		}
	case []interface{}:
		if len(v) == 0 {
			errors = append(errors, ValidationError{
				Field:   "input",
				Message: "input array cannot be empty",
			})
		}
		for i, item := range v {
			switch itemVal := item.(type) {
			case string:
				if itemVal == "" {
					errors = append(errors, ValidationError{
						Field:   fmt.Sprintf("input[%d]", i),
						Message: "input array string items cannot be empty",
					})
				}
			case map[string]interface{}:
				// Validate message object format
				if err := validateMessageObject(itemVal, i); err != nil {
					errors = append(errors, *err...)
				}
			default:
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("input[%d]", i),
					Message: "input array items must be strings or message objects with 'role' and 'content'",
				})
			}
		}
	default:
		errors = append(errors, ValidationError{
			Field:   "input",
			Message: "input must be a string or array of strings/message objects",
		})
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateMessageObject validates a message object in the input array
func validateMessageObject(msg map[string]interface{}, index int) *[]ValidationError {
	var errors []ValidationError

	// Check for required role field
	role, hasRole := msg["role"]
	if !hasRole {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("input[%d].role", index),
			Message: "role is required for message objects",
		})
	} else if roleStr, ok := role.(string); !ok || roleStr == "" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("input[%d].role", index),
			Message: "role must be a non-empty string",
		})
	} else if roleStr != "system" && roleStr != "user" && roleStr != "assistant" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("input[%d].role", index),
			Message: "role must be one of: system, user, assistant",
		})
	}

	// Check for required content field
	content, hasContent := msg["content"]
	if !hasContent {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("input[%d].content", index),
			Message: "content is required for message objects",
		})
	} else if contentStr, ok := content.(string); !ok || contentStr == "" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("input[%d].content", index),
			Message: "content must be a non-empty string",
		})
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateCreateResponseInput validates a CreateResponseInput (legacy function for backward compatibility)
func validateCreateResponseInput(input *requesttypes.CreateResponseInput) *[]ValidationError {
	var errors []ValidationError

	// Validate type
	if input.Type == "" {
		errors = append(errors, ValidationError{
			Field:   "input.type",
			Message: "input.type is required",
		})
		return &errors
	}

	// Validate type-specific fields
	switch input.Type {
	case requesttypes.InputTypeText:
		if input.Text == nil || *input.Text == "" {
			errors = append(errors, ValidationError{
				Field:   "input.text",
				Message: "input.text is required for text type",
			})
		}
	case requesttypes.InputTypeImage:
		if input.Image == nil {
			errors = append(errors, ValidationError{
				Field:   "input.image",
				Message: "input.image is required for image type",
			})
		} else {
			if err := validateImageInput(input.Image); err != nil {
				errors = append(errors, *err...)
			}
		}
	case requesttypes.InputTypeFile:
		if input.File == nil {
			errors = append(errors, ValidationError{
				Field:   "input.file",
				Message: "input.file is required for file type",
			})
		} else {
			if err := validateFileInput(input.File); err != nil {
				errors = append(errors, *err...)
			}
		}
	case requesttypes.InputTypeWebSearch:
		if input.WebSearch == nil {
			errors = append(errors, ValidationError{
				Field:   "input.web_search",
				Message: "input.web_search is required for web_search type",
			})
		} else {
			if err := validateWebSearchInput(input.WebSearch); err != nil {
				errors = append(errors, *err...)
			}
		}
	case requesttypes.InputTypeFileSearch:
		if input.FileSearch == nil {
			errors = append(errors, ValidationError{
				Field:   "input.file_search",
				Message: "input.file_search is required for file_search type",
			})
		} else {
			if err := validateFileSearchInput(input.FileSearch); err != nil {
				errors = append(errors, *err...)
			}
		}
	case requesttypes.InputTypeStreaming:
		if input.Streaming == nil {
			errors = append(errors, ValidationError{
				Field:   "input.streaming",
				Message: "input.streaming is required for streaming type",
			})
		} else {
			if err := validateStreamingInput(input.Streaming); err != nil {
				errors = append(errors, *err...)
			}
		}
	case requesttypes.InputTypeFunctionCalls:
		if input.FunctionCalls == nil {
			errors = append(errors, ValidationError{
				Field:   "input.function_calls",
				Message: "input.function_calls is required for function_calls type",
			})
		} else {
			if err := validateFunctionCallsInput(input.FunctionCalls); err != nil {
				errors = append(errors, *err...)
			}
		}
	case requesttypes.InputTypeReasoning:
		if input.Reasoning == nil {
			errors = append(errors, ValidationError{
				Field:   "input.reasoning",
				Message: "input.reasoning is required for reasoning type",
			})
		} else {
			if err := validateReasoningInput(input.Reasoning); err != nil {
				errors = append(errors, *err...)
			}
		}
	default:
		errors = append(errors, ValidationError{
			Field:   "input.type",
			Message: fmt.Sprintf("invalid input type: %s", input.Type),
		})
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateImageInput validates an ImageInput
func validateImageInput(image *requesttypes.ImageInput) *[]ValidationError {
	var errors []ValidationError

	// Either URL or data must be provided
	if image.URL == nil && image.Data == nil {
		errors = append(errors, ValidationError{
			Field:   "input.image",
			Message: "either url or data must be provided for image input",
		})
	}

	// Both URL and data cannot be provided
	if image.URL != nil && image.Data != nil {
		errors = append(errors, ValidationError{
			Field:   "input.image",
			Message: "either url or data must be provided, not both",
		})
	}

	// Validate URL if provided
	if image.URL != nil && *image.URL != "" {
		if !strings.HasPrefix(*image.URL, "http://") && !strings.HasPrefix(*image.URL, "https://") {
			errors = append(errors, ValidationError{
				Field:   "input.image.url",
				Message: "url must be a valid HTTP or HTTPS URL",
			})
		}
	}

	// Validate data if provided
	if image.Data != nil && *image.Data != "" {
		if !strings.HasPrefix(*image.Data, "data:image/") {
			errors = append(errors, ValidationError{
				Field:   "input.image.data",
				Message: "data must be a valid base64-encoded image with data URL format",
			})
		}
	}

	// Validate detail if provided
	if image.Detail != nil {
		if *image.Detail != "low" && *image.Detail != "high" && *image.Detail != "auto" {
			errors = append(errors, ValidationError{
				Field:   "input.image.detail",
				Message: "detail must be one of: low, high, auto",
			})
		}
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateFileInput validates a FileInput
func validateFileInput(file *requesttypes.FileInput) *[]ValidationError {
	var errors []ValidationError

	if file.FileID == "" {
		errors = append(errors, ValidationError{
			Field:   "input.file.file_id",
			Message: "file_id is required",
		})
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateWebSearchInput validates a WebSearchInput
func validateWebSearchInput(webSearch *requesttypes.WebSearchInput) *[]ValidationError {
	var errors []ValidationError

	if webSearch.Query == "" {
		errors = append(errors, ValidationError{
			Field:   "input.web_search.query",
			Message: "query is required",
		})
	}

	if webSearch.MaxResults != nil {
		if *webSearch.MaxResults < 1 || *webSearch.MaxResults > 20 {
			errors = append(errors, ValidationError{
				Field:   "input.web_search.max_results",
				Message: "max_results must be between 1 and 20",
			})
		}
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateFileSearchInput validates a FileSearchInput
func validateFileSearchInput(fileSearch *requesttypes.FileSearchInput) *[]ValidationError {
	var errors []ValidationError

	if fileSearch.Query == "" {
		errors = append(errors, ValidationError{
			Field:   "input.file_search.query",
			Message: "query is required",
		})
	}

	if len(fileSearch.FileIDs) == 0 {
		errors = append(errors, ValidationError{
			Field:   "input.file_search.file_ids",
			Message: "file_ids is required and cannot be empty",
		})
	}

	if fileSearch.MaxResults != nil {
		if *fileSearch.MaxResults < 1 || *fileSearch.MaxResults > 20 {
			errors = append(errors, ValidationError{
				Field:   "input.file_search.max_results",
				Message: "max_results must be between 1 and 20",
			})
		}
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateStreamingInput validates a StreamingInput
func validateStreamingInput(streaming *requesttypes.StreamingInput) *[]ValidationError {
	var errors []ValidationError

	if streaming.URL == "" {
		errors = append(errors, ValidationError{
			Field:   "input.streaming.url",
			Message: "url is required",
		})
	} else if !strings.HasPrefix(streaming.URL, "http://") && !strings.HasPrefix(streaming.URL, "https://") {
		errors = append(errors, ValidationError{
			Field:   "input.streaming.url",
			Message: "url must be a valid HTTP or HTTPS URL",
		})
	}

	if streaming.Method != nil {
		method := strings.ToUpper(*streaming.Method)
		if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
			errors = append(errors, ValidationError{
				Field:   "input.streaming.method",
				Message: "method must be one of: GET, POST, PUT, DELETE, PATCH",
			})
		}
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateFunctionCallsInput validates a FunctionCallsInput
func validateFunctionCallsInput(functionCalls *requesttypes.FunctionCallsInput) *[]ValidationError {
	var errors []ValidationError

	if len(functionCalls.Calls) == 0 {
		errors = append(errors, ValidationError{
			Field:   "input.function_calls.calls",
			Message: "calls is required and cannot be empty",
		})
	}

	for i, call := range functionCalls.Calls {
		if call.Name == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("input.function_calls.calls[%d].name", i),
				Message: "name is required",
			})
		}
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateReasoningInput validates a ReasoningInput
func validateReasoningInput(reasoning *requesttypes.ReasoningInput) *[]ValidationError {
	var errors []ValidationError

	if reasoning.Task == "" {
		errors = append(errors, ValidationError{
			Field:   "input.reasoning.task",
			Message: "task is required",
		})
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateResponseFormat validates a ResponseFormat
func validateResponseFormat(format *requesttypes.ResponseFormat) *[]ValidationError {
	var errors []ValidationError

	if format.Type == "" {
		errors = append(errors, ValidationError{
			Field:   "response_format.type",
			Message: "type is required",
		})
	} else if format.Type != "text" && format.Type != "json_object" {
		errors = append(errors, ValidationError{
			Field:   "response_format.type",
			Message: "type must be one of: text, json_object",
		})
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateTools validates a slice of Tools
func validateTools(tools []requesttypes.Tool) *[]ValidationError {
	var errors []ValidationError

	for i, tool := range tools {
		if tool.Type == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("tools[%d].type", i),
				Message: "type is required",
			})
		} else if tool.Type != "function" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("tools[%d].type", i),
				Message: "type must be 'function'",
			})
		}

		if tool.Type == "function" && tool.Function == nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("tools[%d].function", i),
				Message: "function is required for function type tools",
			})
		}

		if tool.Function != nil {
			if tool.Function.Name == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("tools[%d].function.name", i),
					Message: "function name is required",
				})
			}
		}
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// validateToolChoice validates a ToolChoice
func validateToolChoice(choice *requesttypes.ToolChoice) *[]ValidationError {
	var errors []ValidationError

	if choice.Type == "" {
		errors = append(errors, ValidationError{
			Field:   "tool_choice.type",
			Message: "type is required",
		})
	} else if choice.Type != "none" && choice.Type != "auto" && choice.Type != "function" {
		errors = append(errors, ValidationError{
			Field:   "tool_choice.type",
			Message: "type must be one of: none, auto, function",
		})
	}

	if choice.Type == "function" && choice.Function == nil {
		errors = append(errors, ValidationError{
			Field:   "tool_choice.function",
			Message: "function is required for function type tool choice",
		})
	}

	if choice.Function != nil {
		if choice.Function.Name == "" {
			errors = append(errors, ValidationError{
				Field:   "tool_choice.function.name",
				Message: "function name is required",
			})
		}
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// ValidateResponseID validates a response ID
func ValidateResponseID(responseID string) *ValidationError {
	if responseID == "" {
		return &ValidationError{
			Field:   "response_id",
			Message: "response_id is required",
		}
	}

	return nil
}
