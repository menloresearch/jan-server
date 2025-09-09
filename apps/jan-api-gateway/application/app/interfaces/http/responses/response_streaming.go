package responses

// StreamingEvent represents a streaming event
// Reference: https://platform.openai.com/docs/api-reference/responses/streaming
type StreamingEvent struct {
	// The type of event.
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The data for the event.
	Data interface{} `json:"data"`
}

// ResponseCreatedEvent represents a response.created event
type ResponseCreatedEvent struct {
	// The type of event, always "response.created".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The response data.
	Data Response `json:"data"`
}

// ResponseOutputTextDeltaEvent represents a response.output_text.delta event
type ResponseOutputTextDeltaEvent struct {
	// The type of event, always "response.output_text.delta".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The delta data.
	Data TextDelta `json:"data"`
}

// TextDelta represents a delta for text output
type TextDelta struct {
	// The delta text.
	Delta string `json:"delta"`

	// The annotations for the delta.
	Annotations []Annotation `json:"annotations,omitempty"`
}

// ResponseOutputTextDoneEvent represents a response.output_text.done event
type ResponseOutputTextDoneEvent struct {
	// The type of event, always "response.output_text.done".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The completion data.
	Data TextCompletion `json:"data"`
}

// TextCompletion represents the completion of text output
type TextCompletion struct {
	// The final text.
	Value string `json:"value"`

	// The annotations for the text.
	Annotations []Annotation `json:"annotations,omitempty"`
}

// ResponseOutputImageDeltaEvent represents a response.output_image.delta event
type ResponseOutputImageDeltaEvent struct {
	// The type of event, always "response.output_image.delta".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The delta data.
	Data ImageDelta `json:"data"`
}

// ImageDelta represents a delta for image output
type ImageDelta struct {
	// The delta image data.
	Delta ImageOutput `json:"delta"`
}

// ResponseOutputImageDoneEvent represents a response.output_image.done event
type ResponseOutputImageDoneEvent struct {
	// The type of event, always "response.output_image.done".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The completion data.
	Data ImageCompletion `json:"data"`
}

// ImageCompletion represents the completion of image output
type ImageCompletion struct {
	// The final image data.
	Value ImageOutput `json:"value"`
}

// ResponseOutputFileDeltaEvent represents a response.output_file.delta event
type ResponseOutputFileDeltaEvent struct {
	// The type of event, always "response.output_file.delta".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The delta data.
	Data FileDelta `json:"data"`
}

// FileDelta represents a delta for file output
type FileDelta struct {
	// The delta file data.
	Delta FileOutput `json:"delta"`
}

// ResponseOutputFileDoneEvent represents a response.output_file.done event
type ResponseOutputFileDoneEvent struct {
	// The type of event, always "response.output_file.done".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The completion data.
	Data FileCompletion `json:"data"`
}

// FileCompletion represents the completion of file output
type FileCompletion struct {
	// The final file data.
	Value FileOutput `json:"value"`
}

// ResponseOutputWebSearchDeltaEvent represents a response.output_web_search.delta event
type ResponseOutputWebSearchDeltaEvent struct {
	// The type of event, always "response.output_web_search.delta".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The delta data.
	Data WebSearchDelta `json:"data"`
}

// WebSearchDelta represents a delta for web search output
type WebSearchDelta struct {
	// The delta web search data.
	Delta WebSearchOutput `json:"delta"`
}

// ResponseOutputWebSearchDoneEvent represents a response.output_web_search.done event
type ResponseOutputWebSearchDoneEvent struct {
	// The type of event, always "response.output_web_search.done".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The completion data.
	Data WebSearchCompletion `json:"data"`
}

// WebSearchCompletion represents the completion of web search output
type WebSearchCompletion struct {
	// The final web search data.
	Value WebSearchOutput `json:"value"`
}

// ResponseOutputFileSearchDeltaEvent represents a response.output_file_search.delta event
type ResponseOutputFileSearchDeltaEvent struct {
	// The type of event, always "response.output_file_search.delta".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The delta data.
	Data FileSearchDelta `json:"data"`
}

// FileSearchDelta represents a delta for file search output
type FileSearchDelta struct {
	// The delta file search data.
	Delta FileSearchOutput `json:"delta"`
}

// ResponseOutputFileSearchDoneEvent represents a response.output_file_search.done event
type ResponseOutputFileSearchDoneEvent struct {
	// The type of event, always "response.output_file_search.done".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The completion data.
	Data FileSearchCompletion `json:"data"`
}

// FileSearchCompletion represents the completion of file search output
type FileSearchCompletion struct {
	// The final file search data.
	Value FileSearchOutput `json:"value"`
}

// ResponseOutputStreamingDeltaEvent represents a response.output_streaming.delta event
type ResponseOutputStreamingDeltaEvent struct {
	// The type of event, always "response.output_streaming.delta".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The delta data.
	Data StreamingDelta `json:"data"`
}

// StreamingDelta represents a delta for streaming output
type StreamingDelta struct {
	// The delta streaming data.
	Delta StreamingOutput `json:"delta"`
}

// ResponseOutputStreamingDoneEvent represents a response.output_streaming.done event
type ResponseOutputStreamingDoneEvent struct {
	// The type of event, always "response.output_streaming.done".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The completion data.
	Data StreamingCompletion `json:"data"`
}

// StreamingCompletion represents the completion of streaming output
type StreamingCompletion struct {
	// The final streaming data.
	Value StreamingOutput `json:"value"`
}

// ResponseOutputFunctionCallsDeltaEvent represents a response.output_function_calls.delta event
type ResponseOutputFunctionCallsDeltaEvent struct {
	// The type of event, always "response.output_function_calls.delta".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The delta data.
	Data FunctionCallsDelta `json:"data"`
}

// FunctionCallsDelta represents a delta for function calls output
type FunctionCallsDelta struct {
	// The delta function calls data.
	Delta FunctionCallsOutput `json:"delta"`
}

// ResponseOutputFunctionCallsDoneEvent represents a response.output_function_calls.done event
type ResponseOutputFunctionCallsDoneEvent struct {
	// The type of event, always "response.output_function_calls.done".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The completion data.
	Data FunctionCallsCompletion `json:"data"`
}

// FunctionCallsCompletion represents the completion of function calls output
type FunctionCallsCompletion struct {
	// The final function calls data.
	Value FunctionCallsOutput `json:"value"`
}

// ResponseOutputReasoningDeltaEvent represents a response.output_reasoning.delta event
type ResponseOutputReasoningDeltaEvent struct {
	// The type of event, always "response.output_reasoning.delta".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The delta data.
	Data ReasoningDelta `json:"data"`
}

// ReasoningDelta represents a delta for reasoning output
type ReasoningDelta struct {
	// The delta reasoning data.
	Delta ReasoningOutput `json:"delta"`
}

// ResponseOutputReasoningDoneEvent represents a response.output_reasoning.done event
type ResponseOutputReasoningDoneEvent struct {
	// The type of event, always "response.output_reasoning.done".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The completion data.
	Data ReasoningCompletion `json:"data"`
}

// ReasoningCompletion represents the completion of reasoning output
type ReasoningCompletion struct {
	// The final reasoning data.
	Value ReasoningOutput `json:"value"`
}

// ResponseDoneEvent represents a response.done event
type ResponseDoneEvent struct {
	// The type of event, always "response.done".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The completion data.
	Data ResponseCompletion `json:"data"`
}

// ResponseCompletion represents the completion of a response
type ResponseCompletion struct {
	// The final response data.
	Value Response `json:"value"`
}

// ResponseErrorEvent represents a response.error event
type ResponseErrorEvent struct {
	// The type of event, always "response.error".
	Event string `json:"event"`

	// The Unix timestamp (in seconds) when the event was created.
	Created int64 `json:"created"`

	// The ID of the response this event belongs to.
	ResponseID string `json:"response_id"`

	// The error data.
	Data ResponseError `json:"data"`
}
