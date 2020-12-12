package voorhees

import (
	"context"
	"encoding/json"
	"errors"
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
//
// If the result can not be marshaled an ErrorResponse is returned instead.
func NewSuccessResponse(requestID json.RawMessage, result interface{}) Response {
	res := SuccessResponse{
		Version:   jsonRPCVersion,
		RequestID: requestID,
	}

	if result != nil {
		var err error
		res.Result, err = json.Marshal(result)
		if err != nil {
			return NewErrorResponse(
				requestID,
				fmt.Errorf("could not marshal success result value: %w", err),
			)
		}
	}

	return res
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

	// ServerError provides more context to internal errors. The value is never
	// sent to the client.
	ServerError error `json:"-"`
}

// NewErrorResponse returns a new ErrorResponse for the given error.
func NewErrorResponse(requestID json.RawMessage, err error) ErrorResponse {
	if err, ok := err.(Error); ok {
		return newNativeErrorResponse(requestID, err)
	}

	if isInternalError(err) {
		return ErrorResponse{
			Version:   jsonRPCVersion,
			RequestID: requestID,
			Error: ErrorInfo{
				Code:    InternalErrorCode,
				Message: InternalErrorCode.String(),
			},
			ServerError: err,
		}
	}

	return ErrorResponse{
		Version:   jsonRPCVersion,
		RequestID: requestID,
		Error: ErrorInfo{
			Code:    InternalErrorCode,
			Message: err.Error(), // Note, we use the actual error message in the response.
		},
	}
}

func newNativeErrorResponse(requestID json.RawMessage, nerr Error) ErrorResponse {
	res := ErrorResponse{
		Version:   jsonRPCVersion,
		RequestID: requestID,
		Error: ErrorInfo{
			Code:    nerr.Code(),
			Message: nerr.Message(),
		},
	}

	if data := nerr.Data(); data != nil {
		var err error
		res.Error.Data, err = json.Marshal(data)
		if err != nil {
			// If an error occurs marshaling the user-defined error data we
			// return an internal server error.
			//
			// We *could* still return the error code and message from nerr, but
			// we can not be sure that the client implementation will behave as
			// intended if presented with that error code but no user-defined
			// data.
			return NewErrorResponse(
				requestID,
				fmt.Errorf("could not marshal user-defined error data in %s: %w", nerr, err),
			)
		}
	}

	return res
}

func (ErrorResponse) isResponse() {}

// ErrorInfo describes a JSON-RPC error. It is included in an ErrorResponse, but
// it is not a Go error.
type ErrorInfo struct {
	Code    ErrorCode       `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e ErrorInfo) String() string {
	return describeError(e.Code, e.Message)
}

// isInternalError returns true if err is considered "internal" to the server,
// and hence should not be shown to the client.
func isInternalError(err error) bool {
	return !errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded)
}
