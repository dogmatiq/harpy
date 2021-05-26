package harpy

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

// NewError returns a new JSON-RPC error with an application-defined error code.
//
// The error codes from and including -32768 to -32000 are reserved for
// pre-defined errors by the JSON-RPC specification. Use of a code within this
// range causes a panic.
func NewError(code ErrorCode, options ...ErrorOption) Error {
	if code.IsReserved() {
		panic(fmt.Sprintf("the error code %d is reserved by the JSON-RPC specification (%s)", code, code))
	}

	return newError(code, options)
}

// NewErrorWithReservedCode returns a new JSON-RPC error that uses a reserved
// error code.
//
// The error codes from and including -32768 to -32000 are reserved for
// pre-defined errors by the JSON-RPC specification. Use of a code outside this
// range causes a panic.
//
// This function is provided to allow user-defined handlers to produce errors
// with reserved codes if necessary, but forces the developer to be explicit
// about doing so. Under normal circumstances NewError() should be used instead.
func NewErrorWithReservedCode(code ErrorCode, options ...ErrorOption) Error {
	if !code.IsReserved() {
		panic(fmt.Sprintf("the error code %d is not reserved by the JSON-RPC specification", code))
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

// Error returns the error message.
func (e Error) Error() string {
	return describeError(e.code, e.message)
}

// Unwrap returns the cause of e, if known.
func (e Error) Unwrap() error {
	return e.cause
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
func WithMessage(format string, values ...interface{}) ErrorOption {
	return func(e *Error) {
		e.message = fmt.Sprintf(format, values...)
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
// As per the JSON-RPC specification, the error codes from and including -32768
// to -32000 are reserved for pre-defined errors. These known set of predefined
// errors are defined as constants below.
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

// IsPredefined returns true if c is an error code defined by the JSON-RPC
// specification.
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

// String returns a brief description of the error.
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
		return "undefined reserved error"
	}

	return "unknown error"
}

// describeError returns a short string containing the most useful information
// from an error code and message.
func describeError(code ErrorCode, message string) string {
	if message == "" || message == code.String() {
		// The error message does not contain any more information than the
		// description of the error code.
		return fmt.Sprintf("[%d] %s", code, code)
	}

	if code.IsPredefined() {
		// We have some different information in the error message, and the code
		// is predefined so we display both.
		return fmt.Sprintf("[%d] %s: %s", code, code, message)
	}

	// Otherwise, the code is not predefined which makes its description quite
	// meaningless, so we only show the provided error message.
	return fmt.Sprintf("[%d] %s", code, message)
}
