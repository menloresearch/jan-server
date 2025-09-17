package response

import (
	"fmt"
	"strings"

	"menlo.ai/jan-api-gateway/app/domain/common"
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
func ValidateCreateResponseRequest(req *requesttypes.CreateResponseRequest) (bool, *common.Error) {
	// Validate model
	if req.Model == "" {
		return false, common.NewErrorWithMessage("model is required", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	}

	// Validate input
	if err := validateInput(req.Input); err != nil {
		return false, common.NewErrorWithMessage("input validation error", "b2c3d4e5-f6g7-8901-bcde-f23456789012")
	}

	// Validate temperature
	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 2 {
			return false, common.NewErrorWithMessage("temperature must be between 0 and 2", "c3d4e5f6-g7h8-9012-cdef-345678901234")
		}
	}

	// Validate top_p
	if req.TopP != nil {
		if *req.TopP < 0 || *req.TopP > 1 {
			return false, common.NewErrorWithMessage("top_p must be between 0 and 1", "d4e5f6g7-h8i9-0123-defg-456789012345")
		}
	}

	// Validate top_k
	if req.TopK != nil {
		if *req.TopK < 1 {
			return false, common.NewErrorWithMessage("top_k must be greater than 0", "e5f6g7h8-i9j0-1234-efgh-567890123456")
		}
	}

	// Validate repetition_penalty
	if req.RepetitionPenalty != nil {
		if *req.RepetitionPenalty < 0 || *req.RepetitionPenalty > 2 {
			return false, common.NewErrorWithMessage("repetition_penalty must be between 0 and 2", "f6g7h8i9-j0k1-2345-fghi-678901234567")
		}
	}

	// Validate presence_penalty
	if req.PresencePenalty != nil {
		if *req.PresencePenalty < -2 || *req.PresencePenalty > 2 {
			return false, common.NewErrorWithMessage("presence_penalty must be between -2 and 2", "g7h8i9j0-k1l2-3456-ghij-789012345678")
		}
	}

	// Validate frequency_penalty
	if req.FrequencyPenalty != nil {
		if *req.FrequencyPenalty < -2 || *req.FrequencyPenalty > 2 {
			return false, common.NewErrorWithMessage("frequency_penalty must be between -2 and 2", "h8i9j0k1-l2m3-4567-hijk-890123456789")
		}
	}

	// Validate max_tokens
	if req.MaxTokens != nil {
		if *req.MaxTokens < 1 {
			return false, common.NewErrorWithMessage("max_tokens must be greater than 0", "i9j0k1l2-m3n4-5678-ijkl-901234567890")
		}
	}

	// Validate timeout
	if req.Timeout != nil {
		if *req.Timeout < 1 {
			return false, common.NewErrorWithMessage("timeout must be greater than 0", "j0k1l2m3-n4o5-6789-jklm-012345678901")
		}
	}

	// Validate response_format
	if req.ResponseFormat != nil {
		if err := validateResponseFormat(req.ResponseFormat); err != nil {
			return false, common.NewErrorWithMessage("response_format validation error", "k1l2m3n4-o5p6-7890-klmn-123456789012")
		}
	}

	// Validate tools
	if req.Tools != nil {
		if err := validateTools(req.Tools); err != nil {
			return false, common.NewErrorWithMessage("tools validation error", "l2m3n4o5-p6q7-8901-lmno-234567890123")
		}
	}

	// Validate tool_choice
	if req.ToolChoice != nil {
		if err := validateToolChoice(req.ToolChoice); err != nil {
			return false, common.NewErrorWithMessage("tool_choice validation error", "m3n4o5p6-q7r8-9012-mnop-345678901234")
		}
	}

	return true, nil
}

// validateInput validates the input field (can be string, array of strings, or structured CreateResponseInput)
func validateInput(input any) *[]ValidationError {
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
	case []any:
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
			case map[string]any:
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
	case map[string]any:
		// Check if this is a structured CreateResponseInput object
		if structuredInput := convertToCreateResponseInput(v); structuredInput != nil {
			// Delegate to structured input validation
			if err := validateCreateResponseInput(structuredInput); err != nil {
				errors = append(errors, *err...)
			}
		} else {
			// Treat as a single message object
			if err := validateMessageObject(v, 0); err != nil {
				errors = append(errors, *err...)
			}
		}
	default:
		errors = append(errors, ValidationError{
			Field:   "input",
			Message: "input must be a string, array of strings/message objects, or structured input object",
		})
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

// convertToCreateResponseInput attempts to convert a map to CreateResponseInput
// Returns nil if the map doesn't represent a structured input
func convertToCreateResponseInput(inputMap map[string]any) *requesttypes.CreateResponseInput {
	// Check if this looks like a structured input by looking for a 'type' field
	typeField, hasType := inputMap["type"]
	if !hasType {
		return nil
	}

	typeStr, ok := typeField.(string)
	if !ok {
		return nil
	}

	// Check if it's a valid input type
	switch requesttypes.InputType(typeStr) {
	case requesttypes.InputTypeText,
		requesttypes.InputTypeImage,
		requesttypes.InputTypeFile,
		requesttypes.InputTypeWebSearch,
		requesttypes.InputTypeFileSearch,
		requesttypes.InputTypeStreaming,
		requesttypes.InputTypeFunctionCalls,
		requesttypes.InputTypeReasoning:
		// This looks like a structured input, create the object
		structuredInput := &requesttypes.CreateResponseInput{
			Type: requesttypes.InputType(typeStr),
		}

		// Extract type-specific fields based on the input type
		switch requesttypes.InputType(typeStr) {
		case requesttypes.InputTypeText:
			if text, ok := inputMap["text"].(string); ok {
				structuredInput.Text = &text
			}
		case requesttypes.InputTypeImage:
			if imageData, ok := inputMap["image"].(map[string]any); ok {
				imageInput := &requesttypes.ImageInput{}
				if url, ok := imageData["url"].(string); ok {
					imageInput.URL = &url
				}
				if data, ok := imageData["data"].(string); ok {
					imageInput.Data = &data
				}
				if detail, ok := imageData["detail"].(string); ok {
					imageInput.Detail = &detail
				}
				structuredInput.Image = imageInput
			}
		case requesttypes.InputTypeFile:
			if fileData, ok := inputMap["file"].(map[string]any); ok {
				if fileID, ok := fileData["file_id"].(string); ok {
					structuredInput.File = &requesttypes.FileInput{
						FileID: fileID,
					}
				}
			}
		case requesttypes.InputTypeWebSearch:
			if webSearchData, ok := inputMap["web_search"].(map[string]any); ok {
				if query, ok := webSearchData["query"].(string); ok {
					webSearchInput := &requesttypes.WebSearchInput{
						Query: query,
					}
					if maxResults, ok := webSearchData["max_results"].(float64); ok {
						maxResultsInt := int(maxResults)
						webSearchInput.MaxResults = &maxResultsInt
					}
					if searchEngine, ok := webSearchData["search_engine"].(string); ok {
						webSearchInput.SearchEngine = &searchEngine
					}
					structuredInput.WebSearch = webSearchInput
				}
			}
		case requesttypes.InputTypeFileSearch:
			if fileSearchData, ok := inputMap["file_search"].(map[string]any); ok {
				if query, ok := fileSearchData["query"].(string); ok {
					fileSearchInput := &requesttypes.FileSearchInput{
						Query: query,
					}
					if fileIDs, ok := fileSearchData["file_ids"].([]any); ok {
						fileSearchInput.FileIDs = make([]string, len(fileIDs))
						for i, id := range fileIDs {
							if idStr, ok := id.(string); ok {
								fileSearchInput.FileIDs[i] = idStr
							}
						}
					}
					if maxResults, ok := fileSearchData["max_results"].(float64); ok {
						maxResultsInt := int(maxResults)
						fileSearchInput.MaxResults = &maxResultsInt
					}
					structuredInput.FileSearch = fileSearchInput
				}
			}
		case requesttypes.InputTypeStreaming:
			if streamingData, ok := inputMap["streaming"].(map[string]any); ok {
				if url, ok := streamingData["url"].(string); ok {
					streamingInput := &requesttypes.StreamingInput{
						URL: url,
					}
					if method, ok := streamingData["method"].(string); ok {
						streamingInput.Method = &method
					}
					if body, ok := streamingData["body"].(string); ok {
						streamingInput.Body = &body
					}
					if headers, ok := streamingData["headers"].(map[string]any); ok {
						streamingInput.Headers = make(map[string]string)
						for k, v := range headers {
							if vStr, ok := v.(string); ok {
								streamingInput.Headers[k] = vStr
							}
						}
					}
					structuredInput.Streaming = streamingInput
				}
			}
		case requesttypes.InputTypeFunctionCalls:
			if functionCallsData, ok := inputMap["function_calls"].(map[string]any); ok {
				if calls, ok := functionCallsData["calls"].([]any); ok {
					functionCallsInput := &requesttypes.FunctionCallsInput{
						Calls: make([]requesttypes.FunctionCall, len(calls)),
					}
					for i, call := range calls {
						if callData, ok := call.(map[string]any); ok {
							if name, ok := callData["name"].(string); ok {
								functionCallsInput.Calls[i] = requesttypes.FunctionCall{
									Name: name,
								}
								if args, ok := callData["arguments"].(map[string]any); ok {
									functionCallsInput.Calls[i].Arguments = args
								}
							}
						}
					}
					structuredInput.FunctionCalls = functionCallsInput
				}
			}
		case requesttypes.InputTypeReasoning:
			if reasoningData, ok := inputMap["reasoning"].(map[string]any); ok {
				if task, ok := reasoningData["task"].(string); ok {
					reasoningInput := &requesttypes.ReasoningInput{
						Task: task,
					}
					if context, ok := reasoningData["context"].(string); ok {
						reasoningInput.Context = &context
					}
					structuredInput.Reasoning = reasoningInput
				}
			}
		}

		return structuredInput
	default:
		return nil
	}
}

// validateMessageObject validates a message object in the input array
func validateMessageObject(msg map[string]any, index int) *[]ValidationError {
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
