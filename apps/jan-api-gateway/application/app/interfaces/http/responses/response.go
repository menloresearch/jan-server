package responses

import (
	"encoding/json"
)

type ErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

type GeneralResponse[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
}

type ListResponse[T any] struct {
	Status  string `json:"status"`
	Total   int64  `json:"total"`
	Results []T    `json:"results"`
}

// OpenAIGeneralResponse includes common fields and inline embedding
// All fields of T will be promoted to the top level of the JSON response
// All fields except T are nullable (omitempty)
type OpenAIGeneralResponse[T any] struct {
	JanStatus *string     `json:"jan_status,omitempty"`
	Object    *ObjectType `json:"object"`
	T         T           `json:",inline"` // Embedded type T with inline tag
}

// Goland does not support inline embedding, so we need to marshal the embedded struct T to get its fields
// MarshalJSON implements custom JSON marshaling to flatten the embedded struct
func (r OpenAIGeneralResponse[T]) MarshalJSON() ([]byte, error) {
	// Create a map to hold the flattened result
	result := make(map[string]interface{})

	// Add the wrapper fields
	if r.JanStatus != nil {
		result["jan_status"] = *r.JanStatus
	}
	if r.Object != nil {
		result["object"] = *r.Object
	}

	// Marshal the embedded struct T to get its fields
	tBytes, err := json.Marshal(r.T)
	if err != nil {
		return nil, err
	}

	// Unmarshal T into a map to get its fields
	var tMap map[string]interface{}
	if err := json.Unmarshal(tBytes, &tMap); err != nil {
		return nil, err
	}

	// Add all fields from T to the result map
	for key, value := range tMap {
		result[key] = value
	}

	// Marshal the final result
	return json.Marshal(result)
}

// OpenAIListResponse includes common fields and inline embedding
// All fields of T will be promoted to the top level of the JSON response
// All fields except T are nullable (omitempty)
type OpenAIListResponse[T any] struct {
	JanStatus *string     `json:"jan_status,omitempty"`
	Object    *ObjectType `json:"object"`
	FirstID   *string     `json:"first_id,omitempty"`
	LastID    *string     `json:"last_id,omitempty"`
	HasMore   *bool       `json:"has_more,omitempty"`
	T         []T         `json:"data,inline"` // Inline T - all fields of T will be at the top level
}

// ObjectType represents the type of object in responses
type ObjectType string

const (
	ObjectTypeResponse ObjectType = "response"
	ObjectTypeList     ObjectType = "list"
)

const ResponseCodeOk = "000000"
