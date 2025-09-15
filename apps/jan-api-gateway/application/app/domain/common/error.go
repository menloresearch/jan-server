package common

// Error represents a standardized error with code and message
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewError creates a new Error instance
func NewError(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// IsEmpty checks if the error is empty (no error)
func (e *Error) IsEmpty() bool {
	return e == nil || e.Code == ""
}

// String returns the string representation of the error
func (e *Error) String() string {
	if e == nil {
		return ""
	}
	return e.Code + ": " + e.Message
}

// Error implements the error interface
func (e *Error) Error() string {
	return e.String()
}

// EmptyError represents an empty error (no error occurred)
var EmptyError = &Error{}
