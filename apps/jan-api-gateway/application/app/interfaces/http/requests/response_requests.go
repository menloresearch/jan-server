package requests

// CreateResponseRequest represents the request body for creating a response
// Reference: https://platform.openai.com/docs/api-reference/responses/create
type CreateResponseRequest struct {
	// The ID of the model to use for this response.
	Model string `json:"model" binding:"required"`

	// The input to the model. Can be a string or array of strings.
	Input interface{} `json:"input" binding:"required"`

	// The system prompt to use for this response.
	SystemPrompt *string `json:"system_prompt,omitempty"`

	// The maximum number of tokens to generate.
	MaxTokens *int `json:"max_tokens,omitempty"`

	// The temperature to use for this response.
	Temperature *float64 `json:"temperature,omitempty"`

	// The top_p to use for this response.
	TopP *float64 `json:"top_p,omitempty"`

	// The top_k to use for this response.
	TopK *int `json:"top_k,omitempty"`

	// The repetition penalty to use for this response.
	RepetitionPenalty *float64 `json:"repetition_penalty,omitempty"`

	// The seed to use for this response.
	Seed *int `json:"seed,omitempty"`

	// The stop sequences to use for this response.
	Stop []string `json:"stop,omitempty"`

	// The presence penalty to use for this response.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// The frequency penalty to use for this response.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// The logit bias to use for this response.
	LogitBias map[string]float64 `json:"logit_bias,omitempty"`

	// The response format to use for this response.
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// The tools to use for this response.
	Tools []Tool `json:"tools,omitempty"`

	// The tool choice to use for this response.
	ToolChoice *ToolChoice `json:"tool_choice,omitempty"`

	// The metadata to use for this response.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Whether to stream the response.
	Stream *bool `json:"stream,omitempty"`

	// Whether to run the response in the background.
	Background *bool `json:"background,omitempty"`

	// The timeout in seconds for this response.
	Timeout *int `json:"timeout,omitempty"`

	// The user to use for this response.
	User *string `json:"user,omitempty"`
}

// CreateResponseInput represents the input to the model
type CreateResponseInput struct {
	// The type of input.
	Type InputType `json:"type" binding:"required"`

	// The text input (required for text type).
	Text *string `json:"text,omitempty"`

	// The image input (required for image type).
	Image *ImageInput `json:"image,omitempty"`

	// The file input (required for file type).
	File *FileInput `json:"file,omitempty"`

	// The web search input (required for web_search type).
	WebSearch *WebSearchInput `json:"web_search,omitempty"`

	// The file search input (required for file_search type).
	FileSearch *FileSearchInput `json:"file_search,omitempty"`

	// The streaming input (required for streaming type).
	Streaming *StreamingInput `json:"streaming,omitempty"`

	// The function calls input (required for function_calls type).
	FunctionCalls *FunctionCallsInput `json:"function_calls,omitempty"`

	// The reasoning input (required for reasoning type).
	Reasoning *ReasoningInput `json:"reasoning,omitempty"`
}

// InputType represents the type of input
type InputType string

const (
	InputTypeText          InputType = "text"
	InputTypeImage         InputType = "image"
	InputTypeFile          InputType = "file"
	InputTypeWebSearch     InputType = "web_search"
	InputTypeFileSearch    InputType = "file_search"
	InputTypeStreaming     InputType = "streaming"
	InputTypeFunctionCalls InputType = "function_calls"
	InputTypeReasoning     InputType = "reasoning"
)

// ImageInput represents an image input
type ImageInput struct {
	// The URL of the image.
	URL *string `json:"url,omitempty"`

	// The base64 encoded image data.
	Data *string `json:"data,omitempty"`

	// The detail level for the image.
	Detail *string `json:"detail,omitempty"`
}

// FileInput represents a file input
type FileInput struct {
	// The ID of the file.
	FileID string `json:"file_id" binding:"required"`
}

// WebSearchInput represents a web search input
type WebSearchInput struct {
	// The query to search for.
	Query string `json:"query" binding:"required"`

	// The number of results to return.
	MaxResults *int `json:"max_results,omitempty"`

	// The search engine to use.
	SearchEngine *string `json:"search_engine,omitempty"`
}

// FileSearchInput represents a file search input
type FileSearchInput struct {
	// The query to search for.
	Query string `json:"query" binding:"required"`

	// The IDs of the files to search in.
	FileIDs []string `json:"file_ids" binding:"required"`

	// The number of results to return.
	MaxResults *int `json:"max_results,omitempty"`
}

// StreamingInput represents a streaming input
type StreamingInput struct {
	// The URL to stream from.
	URL string `json:"url" binding:"required"`

	// The headers to send with the request.
	Headers map[string]string `json:"headers,omitempty"`

	// The method to use for the request.
	Method *string `json:"method,omitempty"`

	// The body to send with the request.
	Body *string `json:"body,omitempty"`
}

// FunctionCallsInput represents function calls input
type FunctionCallsInput struct {
	// The function calls to make.
	Calls []FunctionCall `json:"calls" binding:"required"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	// The name of the function to call.
	Name string `json:"name" binding:"required"`

	// The arguments to pass to the function.
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ReasoningInput represents a reasoning input
type ReasoningInput struct {
	// The reasoning task to perform.
	Task string `json:"task" binding:"required"`

	// The context for the reasoning task.
	Context *string `json:"context,omitempty"`
}

// ResponseFormat represents the format of the response
type ResponseFormat struct {
	// The type of response format.
	Type string `json:"type" binding:"required"`
}

// Tool represents a tool that can be used by the model
type Tool struct {
	// The type of tool.
	Type string `json:"type" binding:"required"`

	// The function definition for function tools.
	Function *FunctionDefinition `json:"function,omitempty"`
}

// FunctionDefinition represents a function definition
type FunctionDefinition struct {
	// The name of the function.
	Name string `json:"name" binding:"required"`

	// The description of the function.
	Description *string `json:"description,omitempty"`

	// The parameters of the function.
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ToolChoice represents the tool choice for the model
type ToolChoice struct {
	// The type of tool choice.
	Type string `json:"type" binding:"required"`

	// The function to use for function tool choice.
	Function *FunctionChoice `json:"function,omitempty"`
}

// FunctionChoice represents a function choice
type FunctionChoice struct {
	// The name of the function.
	Name string `json:"name" binding:"required"`
}
