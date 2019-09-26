package arpc

// Error always return 200 status with false ok value
// use this error for validate, precondition failed, etc.
type Error struct {
	Message string `json:"message"`

	err error
}

// Unwrap implements errors.Unwarp
func (err *Error) Unwrap() error {
	return err.err
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

func (err *Error) Error() string {
	return err.Message
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
	errMethodNotAllowed = NewProtocolError("method not allowed")
	errNotFound         = NewProtocolError("not found")
)

type internalError struct{}

func (internalError) Error() string { return "internal error" }
