package harpy

import "fmt"

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
// from an error code and a user-defined message.
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
