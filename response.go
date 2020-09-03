package voorhees

import "encoding/json"

// Response is an interface for a JSON-RPC response object.
type Response interface {
	isResponse()
}

// SuccessResponse encapsulates a successful JSON-RPC response.
type SuccessResponse struct {
	// Version is the JSON-RPC version.
	//
	// As per the specification it MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// RequestID is the ID of the request that produced this response.
	RequestID json.RawMessage `json:"id"`

	// Result is the user-defined result value produce in response to the
	// request.
	Result json.RawMessage `json:"result"`
}

func (SuccessResponse) isResponse() {}

// ErrorResponse encapsulates a failed JSON-RPC response.
type ErrorResponse struct {
	// Version is the JSON-RPC version.
	//
	// As per the specification it MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// RequestID is the ID of the request that produced this response.
	RequestID json.RawMessage `json:"id"`

	// Error describes the error produced in response to the request.
	Error ErrorInfo `json:"error"`
}

func (ErrorResponse) isResponse() {}

// ErrorInfo describes a JSON-RPC error. It is included in an ErrorResponse, but
// it is not a Go error.
type ErrorInfo struct {
	Code    ErrorCode       `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}
