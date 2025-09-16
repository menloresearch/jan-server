package responses

type ErrorResponse struct {
	Code          string `json:"code"`
	Error         string `json:"error"`
	ErrorInstance error  `json:"-"`
}

type GeneralResponse[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
}

type ListResponse[T any] struct {
	Status  string  `json:"status"`
	Total   int64   `json:"total"`
	Results []T     `json:"results"`
	FirstID *string `json:"first_id"`
	LastID  *string `json:"last_id"`
	HasMore bool    `json:"has_more"`
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
