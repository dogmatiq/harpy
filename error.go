package voorhees

import "fmt"

// Error is a Go error that describes a JSON-RPC error.
type Error struct {
	code    ErrorCode
	message string
	data    interface{}
	cause   error
}

// newError returns a new Error with the given code.
//
// The options are applied in order.
func newError(code ErrorCode, options []ErrorOption) Error {
	e := Error{
		code: code,
	}

	for _, opt := range options {
		opt(&e)
	}

	return e
}

// NewError returns a new JSON-RPC error.
//
// It panics if code is within the range of codes reserved by the specification.
func NewError(code ErrorCode, options ...ErrorOption) Error {
	if code.IsReserved() {
		panic(fmt.Sprintf("the error code %d is reserved by the JSON-RPC specification (%s)", code, code))
	}

	return newError(code, options)
}

// NewErrorWithReservedCode returns a new JSON-RPC error that uses a code within
// the reserved of codes reserved by the specification.
//
// It panics if the code is not within the range of codes reserved by the
// specification.
//
// This function is provided to allow user-defined handlers to produce errors
// with reserved codes if necessary, but forces the developer to make a concious
// choice to do so.
func NewErrorWithReservedCode(code ErrorCode, options ...ErrorOption) Error {
	if !code.IsReserved() {
		panic(fmt.Sprintf("the error code %d is not reserved by the JSON-RPC specification (%s)", code, code))
	}

	return newError(code, options)
}

// MethodNotFound returns an error that indicates the requested method does not
// exist.
func MethodNotFound(options ...ErrorOption) Error {
	return newError(MethodNotFoundCode, options)
}

// InvalidParameters returns an error that indicates the provided parameters are
// malformed or invalid.
func InvalidParameters(options ...ErrorOption) Error {
	return newError(InvalidParametersCode, options)
}

// Code returns the JSON-RPC error code.
func (e Error) Code() ErrorCode {
	return e.code
}

// Message returns the error message.
func (e Error) Message() string {
	if e.message != "" {
		return e.message
	}

	return e.code.String()
}

// Data returns the user-defined data associated with the error.
func (e Error) Data() interface{} {
	return e.data
}

// Unwrap returns the cause of e, if known.
func (e Error) Unwrap() error {
	return e.cause
}

func (e Error) Error() string {
	if e.message == "" {
		// No user-defined error message was provided, all we really know is the
		// error code and its description.
		return fmt.Sprintf("[%d] %s", e.code, e.code)
	}

	if e.code.IsPredefined() {
		// We have a user-defined error message and the code is predefined (so
		// it has a useful description), so we display both.
		return fmt.Sprintf("[%d] %s: %s", e.code, e.code, e.message)
	}

	// Otherwise, the description of the code is quite meaningless, so we only
	// show the user-defined message.
	return fmt.Sprintf("[%d] %s", e.code, e.message)
}

// ErrorOption is an option that provides further information about an error.
type ErrorOption func(*Error)

// WithCause is an ErrorOption that associates a causal error with a JSON-RPC
// error.
//
// c is wrapped by the resulting JSON-RPC error, such as it can be used with
// errors.Is() and errors.As().
//
// If the JSON-RPC error does not already have a user-defined message, c.Error()
// is used as the user-defined message.
func WithCause(c error) ErrorOption {
	return func(e *Error) {
		e.cause = c

		if e.message == "" {
			// If there is no user-defined error message already provided, use
			// this error as the message.
			e.message = c.Error()
		}
	}
}

// WithMessage is an ErrorOption that provides a user-defined error message for
// a JSON-RPC error.
//
// This message should be used to provide additional information that can help
// diagnose the error.
func WithMessage(m string) ErrorOption {
	return func(e *Error) {
		e.message = m
	}
}

// WithData is an ErrorOption that associates additional data with an error.
//
// The data is provided to the RPC caller via the "data" field of the error
// object in the JSON-RPC response.
func WithData(data interface{}) ErrorOption {
	return func(e *Error) {
		e.data = data
	}
}

// ErrorCode is a JSON-RPC error code.
//
// As per the specification, the error codes from and including -32768 to -32000
// are reserved for pre-defined errors. These known set of predefined errors are
// defined as constants below.
type ErrorCode int

const (
	// ParseErrorCode indicates that the server failed to parse a JSON-RPC
	// request.
	ParseErrorCode ErrorCode = -32700

	// InvalidRequestCode indicates that the server received a well-formed but
	// otherwise invalid JSON-RPC request.
	InvalidRequestCode ErrorCode = -32600

	// MethodNotFoundCode indicates that the server received a request for an
	// RPC method that does not exist.
	MethodNotFoundCode ErrorCode = -32601

	// InvalidParametersCode indicates that the server received a request that
	// contained malformed or invalid parameters.
	InvalidParametersCode ErrorCode = -32602

	// InternalErrorCode indicates that some other error condition was raised
	// within the RPC server.
	InternalErrorCode ErrorCode = -32603
)

// IsReserved returns true if c falls within the range of error codes reserved
// for pre-defined errors.
func (c ErrorCode) IsReserved() bool {
	return c >= -32768 && c <= -32000
}

// IsPredefined returns true if c is an error code defined by the specification.
func (c ErrorCode) IsPredefined() bool {
	switch c {
	case ParseErrorCode,
		InvalidRequestCode,
		MethodNotFoundCode,
		InvalidParametersCode,
		InternalErrorCode:
		return true
	default:
		return false
	}
}

func (c ErrorCode) String() string {
	switch c {
	case ParseErrorCode:
		return "parse error"
	case InvalidRequestCode:
		return "invalid request"
	case MethodNotFoundCode:
		return "method not found"
	case InvalidParametersCode:
		return "invalid parameters"
	case InternalErrorCode:
		return "internal server error"
	}

	if c.IsReserved() {
		return "undefined internal error"
	}

	return "undefined error"
}
