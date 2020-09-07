package voorhees

import (
	"encoding/json"
	"fmt"
)

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

// NewSuccessResponse returns a new SuccessResponse containing the given result.
func NewSuccessResponse(requestID json.RawMessage, result interface{}) (SuccessResponse, error) {
	var data json.RawMessage
	if result != nil {
		var err error
		data, err = json.Marshal(result)
		if err != nil {
			return SuccessResponse{}, fmt.Errorf("could not marshal success result value: %w", err)
		}
	}

	return SuccessResponse{
		Version:   jsonRPCVersion,
		RequestID: requestID,
		Result:    data,
	}, nil
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

// NewErrorResponse returns a new ErrorResponse for the given Error.
func NewErrorResponse(requestID json.RawMessage, err Error) (ErrorResponse, error) {
	res := ErrorResponse{
		Version:   jsonRPCVersion,
		RequestID: requestID,
		Error: ErrorInfo{
			Code:    err.Code(),
			Message: err.Message(),
		},
	}

	if d := err.Data(); d != nil {
		// The error contains a user-defined data value that needs to be
		// serialized to JSON to be included in the response.
		data, merr := json.Marshal(d)
		if merr != nil {
			return ErrorResponse{}, fmt.Errorf("could not marshal user-defined error data in %s: %w", err, merr)
		}

		res.Error.Data = data
	}

	return res, nil
}

func (ErrorResponse) isResponse() {}

// ErrorInfo describes a JSON-RPC error. It is included in an ErrorResponse, but
// it is not a Go error.
type ErrorInfo struct {
	Code    ErrorCode       `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}
