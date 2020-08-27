package voorhees

// SuccessResponse encapsulates a successful JSON-RPC response.
type SuccessResponse struct {
	// Version is the JSON-RPC version.
	//
	// As per the specification it MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// RequestID is the ID of the request that produced this response.
	RequestID interface{} `json:"id"`

	// Result is the user-defined result value produce in response to the
	// request.
	Result interface{} `json:"result"`
}

// ErrorResponse encapsulates a failed JSON-RPC response.
type ErrorResponse struct {
	// Version is the JSON-RPC version.
	//
	// As per the specification it MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// RequestID is the ID of the request that produced this response.
	RequestID interface{} `json:"id"`

	// Error describes the error produced in response to the request.
	Error ErrorInfo `json:"error"`
}

// ErrorInfo describes a JSON-RPC error. It is included in an ErrorResponse, but
// it is not a Go error.
type ErrorInfo struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
