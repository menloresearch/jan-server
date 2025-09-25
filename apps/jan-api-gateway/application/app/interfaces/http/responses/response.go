package responses

import (
	"net/http"
	"time"

	"menlo.ai/jan-api-gateway/config"
)

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

type PageCursor struct {
	FirstID *string
	LastID  *string
	HasMore bool
	Total   int64
}

func BuildCursorPage[T any](
	items []*T,
	getID func(*T) *string,
	hasMoreFunc func() ([]*T, error),
	CountFunc func() (int64, error),
) (*PageCursor, error) {
	cursorPage := &PageCursor{}
	if len(items) > 0 {
		cursorPage.FirstID = getID(items[0])
		cursorPage.LastID = getID(items[len(items)-1])
		moreRecords, err := hasMoreFunc()
		if len(moreRecords) > 0 {
			cursorPage.HasMore = true
		}
		if err != nil {
			return nil, err
		}
	}
	count, err := CountFunc()
	if err != nil {
		return cursorPage, err
	}
	cursorPage.Total = count
	return cursorPage, nil
}

func NewCookieWithSecurity(name string, value string, expires time.Time) *http.Cookie {
	if config.IsDev() {
		return &http.Cookie{
			Name:     name,
			Value:    value,
			Expires:  expires,
			HttpOnly: false,
			Secure:   false,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
		}
	}
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}
}
