package responses

// BaseStreamingEvent represents the base structure for all streaming events
type BaseStreamingEvent struct {
	// The type of event.
	Type string `json:"type"`

	// The sequence number of the event.
	SequenceNumber int `json:"sequence_number"`
}

// ResponseCreatedEvent represents a response.created event
type ResponseCreatedEvent struct {
	BaseStreamingEvent
	Response Response `json:"response"`
}

// ResponseInProgressEvent represents a response.in_progress event
type ResponseInProgressEvent struct {
	BaseStreamingEvent
	Response map[string]any `json:"response"`
}

// ResponseOutputItemAddedEvent represents a response.output_item.added event
type ResponseOutputItemAddedEvent struct {
	BaseStreamingEvent
	OutputIndex int                `json:"output_index"`
	Item        ResponseOutputItem `json:"item"`
}

// ResponseContentPartAddedEvent represents a response.content_part.added event
type ResponseContentPartAddedEvent struct {
	BaseStreamingEvent
	ItemID       string              `json:"item_id"`
	OutputIndex  int                 `json:"output_index"`
	ContentIndex int                 `json:"content_index"`
	Part         ResponseContentPart `json:"part"`
}

// ResponseOutputTextDeltaEvent represents a response.output_text.delta event
type ResponseOutputTextDeltaEvent struct {
	BaseStreamingEvent
	ItemID       string    `json:"item_id"`
	OutputIndex  int       `json:"output_index"`
	ContentIndex int       `json:"content_index"`
	Delta        string    `json:"delta"`
	Logprobs     []Logprob `json:"logprobs"`
	Obfuscation  string    `json:"obfuscation"`
}

// ResponseOutputItem represents an output item
type ResponseOutputItem struct {
	ID      string                `json:"id"`
	Type    string                `json:"type"`
	Status  string                `json:"status"`
	Content []ResponseContentPart `json:"content"`
	Role    string                `json:"role"`
}

// ResponseContentPart represents a content part
type ResponseContentPart struct {
	Type        string       `json:"type"`
	Annotations []Annotation `json:"annotations"`
	Logprobs    []Logprob    `json:"logprobs"`
	Text        string       `json:"text"`
}

// Logprob represents log probability data
type Logprob struct {
	Token       string       `json:"token"`
	Logprob     float64      `json:"logprob"`
	Bytes       []int        `json:"bytes,omitempty"`
	TopLogprobs []TopLogprob `json:"top_logprobs,omitempty"`
}

// TopLogprob represents top log probability data
type TopLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

// TextDelta represents a delta for text output (legacy)
type TextDelta struct {
	// The delta text.
	Delta string `json:"delta"`

	// The annotations for the delta.
	Annotations []Annotation `json:"annotations,omitempty"`
}

// ResponseOutputTextDoneEvent represents a response.output_text.done event
type ResponseOutputTextDoneEvent struct {
	BaseStreamingEvent
	ItemID       string    `json:"item_id"`
	OutputIndex  int       `json:"output_index"`
	ContentIndex int       `json:"content_index"`
	Text         string    `json:"text"`
	Logprobs     []Logprob `json:"logprobs"`
}

// ResponseContentPartDoneEvent represents a response.content_part.done event
type ResponseContentPartDoneEvent struct {
	BaseStreamingEvent
	ItemID       string              `json:"item_id"`
	OutputIndex  int                 `json:"output_index"`
	ContentIndex int                 `json:"content_index"`
	Part         ResponseContentPart `json:"part"`
}

// ResponseOutputItemDoneEvent represents a response.output_item.done event
type ResponseOutputItemDoneEvent struct {
	BaseStreamingEvent
	OutputIndex int                `json:"output_index"`
	Item        ResponseOutputItem `json:"item"`
}

// ResponseCompletedEvent represents a response.completed event
type ResponseCompletedEvent struct {
	BaseStreamingEvent
	Response Response `json:"response"`
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
	BaseStreamingEvent
	ItemID       string            `json:"item_id"`
	OutputIndex  int               `json:"output_index"`
	ContentIndex int               `json:"content_index"`
	Delta        FunctionCallDelta `json:"delta"`
	Logprobs     []Logprob         `json:"logprobs"`
}

// FunctionCallDelta represents a delta for function call
type FunctionCallDelta struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
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

// ResponseReasoningSummaryPartAddedEvent represents a response.reasoning_summary_part.added event
type ResponseReasoningSummaryPartAddedEvent struct {
	BaseStreamingEvent
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	SummaryIndex int    `json:"summary_index"`
	Part         struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"part"`
}

// ResponseReasoningSummaryTextDeltaEvent represents a response.reasoning_summary_text.delta event
type ResponseReasoningSummaryTextDeltaEvent struct {
	BaseStreamingEvent
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	SummaryIndex int    `json:"summary_index"`
	Delta        string `json:"delta"`
	Obfuscation  string `json:"obfuscation"`
}

// ResponseReasoningSummaryTextDoneEvent represents a response.reasoning_summary_text.done event
type ResponseReasoningSummaryTextDoneEvent struct {
	BaseStreamingEvent
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	SummaryIndex int    `json:"summary_index"`
	Text         string `json:"text"`
}

// ResponseReasoningSummaryPartDoneEvent represents a response.reasoning_summary_part.done event
type ResponseReasoningSummaryPartDoneEvent struct {
	BaseStreamingEvent
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	SummaryIndex int    `json:"summary_index"`
	Part         struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"part"`
}
