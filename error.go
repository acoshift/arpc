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
	code string
	msg  string
	err  error
}

// OKError implements OKError
func (err *Error) OKError() {}

// Error implements error
func (err *Error) Error() string {
	s := err.code
	if s != "" {
		s += " "
	}
	return s + err.msg
}

// Unwrap implements errors.Unwarp
func (err *Error) Unwrap() error {
	return err.err
}

// MarshalJSON implements json.Marshaler
func (err *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}{err.code, err.msg})
}

// NewError creates new Error
func NewError(code, message string) error {
	return &Error{code, message, nil}
}

func wrapError(err error) error {
	return &Error{"", err.Error(), err}
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
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewProtocolError(code, message string) error {
	return &ProtocolError{code, message}
}

func (err *ProtocolError) Error() string {
	return err.Message
}

// predefined errors
var (
	ErrNotFound    = NewProtocolError("", "not found")
	ErrUnsupported = NewProtocolError("", "unsupported content type")
)

type internalError struct{}

func (internalError) Error() string { return "internal error" }
