package arpc

import (
	"encoding/json"
)

// OKError implements this interface to mark errors as 200
type OKError interface {
	OKError()
}

// Error always return 200 status with false ok value
// use this error for validate, precondition failed, etc.
type Error struct {
	msg string
	err error
}

// OKError implements OKError
func (err *Error) OKError() {}

// Error implements error
func (err *Error) Error() string {
	return err.msg
}

// Unwrap implements errors.Unwarp
func (err *Error) Unwrap() error {
	return err.err
}

// MarshalJSON implements json.Marshaler
func (err *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message string `json:"message"`
	}{err.msg})
}

// NewError creates new Error
func NewError(message string) error {
	return &Error{message, nil}
}

func wrapError(err error) error {
	return &Error{err.Error(), err}
}

// WrapError wraps given error with Error
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	switch err.(type) {
	case *Error:
		return err
	case *ProtocolError:
		return err
	default:
		return wrapError(err)
	}
}

// ProtocolError always returns 400 status with false ok value
// only use this error for invalid protocol usages
type ProtocolError struct {
	Message string `json:"message"`
}

func NewProtocolError(message string) error {
	return &ProtocolError{message}
}

func (err *ProtocolError) Error() string {
	return err.Message
}

// predefined errors
var (
	ErrUnsupported = NewProtocolError("unsupported content type")
)

var (
	errNotFound = NewProtocolError("not found")
)

type internalError struct{}

func (internalError) Error() string { return "internal error" }
