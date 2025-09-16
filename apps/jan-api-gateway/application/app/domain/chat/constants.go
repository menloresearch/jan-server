package chat

import "time"

// Constants for streaming configuration
const (
	RequestTimeout    = 120 * time.Second
	DataPrefix        = "data: "
	DoneMarker        = "[DONE]"
	ChannelBufferSize = 100
	ErrorBufferSize   = 10
)
