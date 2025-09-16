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
