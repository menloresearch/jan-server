package common

import "fmt"

// Error represents a standardized error with code and underlying error
type Error struct {
	Err  error  `json:"-"`
	Code string `json:"code"`
}

// NewError creates a new Error instance from an existing error
func NewError(err error, code string) *Error {
	return &Error{
		Err:  err,
		Code: code,
	}
}

// NewErrorWithMessage creates a new Error instance with a custom message
func NewErrorWithMessage(message string, code string) *Error {
	return &Error{
		Err:  fmt.Errorf("%s", message),
		Code: code,
	}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

// String returns the string representation of the error
func (e *Error) String() string {
	return e.Error()
}

// GetMessage returns the error message from the underlying error
func (e *Error) GetMessage() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

// GetCode returns the error code
func (e *Error) GetCode() string {
	return e.Code
}

// GetCode returns the error code
func (e *Error) GetError() error {
	return e.Err
}
