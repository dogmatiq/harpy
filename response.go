package harpy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode"
)

// Response is an interface for a JSON-RPC response object.
type Response interface {
	// Validate checks that the response conforms to the JSON-RPC specification.
	//
	// It returns nil if the response is valid.
	Validate() error

	isResponse()
}

// SuccessResponse encapsulates a successful JSON-RPC response.
type SuccessResponse struct {
	// Version is the JSON-RPC version.
	//
	// As per the JSON-RPC specification it MUST be exactly "2.0".
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
		Version:   JSONRPCVersion,
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

// Validate checks that the response conforms to the JSON-RPC specification.
//
// It returns nil if the response is valid.
func (r SuccessResponse) Validate() error {
	if r.Version != JSONRPCVersion {
		return errors.New(`response version must be "2.0"`)
	}

	if err := validateRequestIDInResponse(r.RequestID); err != nil {
		return err
	}

	if len(r.Result) == 0 {
		return errors.New("success response must contain a result")
	}

	return nil
}

func (SuccessResponse) isResponse() {}

// ErrorResponse encapsulates a failed JSON-RPC response.
type ErrorResponse struct {
	// Version is the JSON-RPC version.
	//
	// As per the JSON-RPC specification it MUST be exactly "2.0".
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
			Version:   JSONRPCVersion,
			RequestID: requestID,
			Error: ErrorInfo{
				Code:    InternalErrorCode,
				Message: InternalErrorCode.String(),
			},
			ServerError: err,
		}
	}

	return ErrorResponse{
		Version:   JSONRPCVersion,
		RequestID: requestID,
		Error: ErrorInfo{
			Code:    InternalErrorCode,
			Message: err.Error(), // Note, we use the actual error message in the response.
		},
	}
}

func newNativeErrorResponse(requestID json.RawMessage, nerr Error) ErrorResponse {
	res := ErrorResponse{
		Version:   JSONRPCVersion,
		RequestID: requestID,
		Error: ErrorInfo{
			Code:    nerr.Code(),
			Message: nerr.Message(),
		},
		ServerError: nerr.cause,
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

// Validate checks that the response conforms to the JSON-RPC specification.
//
// It returns nil if the response is valid.
func (r ErrorResponse) Validate() error {
	if r.Version != JSONRPCVersion {
		return errors.New(`response version must be "2.0"`)
	}

	if err := validateRequestIDInResponse(r.RequestID); err != nil {
		return err
	}

	return nil
}

func (ErrorResponse) isResponse() {}

// validateRequestIDInResponse checks that id is a valid request ID for use
// within an RPC response, according to the JSON-RPC specification.
//
// Unlike validateRequestID() it does not allow the id to be absent altogether.
func validateRequestIDInResponse(id json.RawMessage) error {
	if len(id) > 0 {
		var value interface{}
		if err := json.Unmarshal(id, &value); err != nil {
			return err
		}

		switch value.(type) {
		case string:
			return nil
		case float64:
			return nil
		case nil:
			return nil
		}
	}

	return errors.New(`request ID must be a JSON string, number or null`)
}

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

// ResponseSet encapsulates one or more JSON-RPC responses that were parsed from
// a single JSON message.
type ResponseSet struct {
	// Responses contains the responses parsed from the message.
	Responses []Response

	// IsBatch is true if the responses are part of a batch.
	//
	// This is used to disambiguate between a single response and a batch that
	// contains only one response.
	IsBatch bool
}

// UnmarshalResponseSet parses a set of JSON-RPC response set.
func UnmarshalResponseSet(r io.Reader) (ResponseSet, error) {
	br := bufio.NewReader(r)

	for {
		ch, _, err := br.ReadRune()
		if err != nil {
			return ResponseSet{}, err
		}

		if unicode.IsSpace(ch) {
			continue
		}

		if err := br.UnreadRune(); err != nil {
			panic(err) // only occurs if a rune hasn't already been read
		}

		if ch == '[' {
			return unmarshalBatchResponse(br)
		}

		return unmarshalSingleResponse(br)
	}
}

// successOrErrorResponse encapsulates a JSON-RPC response.
type successOrErrorResponse struct {
	// Version is the JSON-RPC version.
	//
	// As per the JSON-RPC specification it MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// RequestID is the ID of the request that produced this response.
	RequestID json.RawMessage `json:"id"`

	// Result is the user-defined result value produce in response to the
	// request.
	Result json.RawMessage `json:"result"`

	// Error describes the error produced in response to the request.
	Error *ErrorInfo `json:"error"`
}

// Validate checks that the response set is valid and that the responses within
// conform to the JSON-RPC specification.
//
// It returns nil if the response set is valid.
func (rs ResponseSet) Validate() error {
	if rs.IsBatch {
		if len(rs.Responses) == 0 {
			return errors.New("batches must contain at least one response")
		}
	} else if len(rs.Responses) != 1 {
		return errors.New("non-batch response sets must contain exactly one response")
	}

	for _, res := range rs.Responses {
		if err := res.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// unmarshalSingleRequest unmarshals a non-batch JSON-RPC request set.
func unmarshalSingleResponse(r *bufio.Reader) (ResponseSet, error) {
	var res successOrErrorResponse

	if err := unmarshalJSONForResponse(r, &res); err != nil {
		return ResponseSet{}, err
	}

	return ResponseSet{
		Responses: []Response{
			normalizeResponse(res),
		},
		IsBatch: false,
	}, nil
}

// unmarshalBatchResponse unmarshals a batched JSON-RPC request set.
func unmarshalBatchResponse(r *bufio.Reader) (ResponseSet, error) {
	var batch []successOrErrorResponse

	if err := unmarshalJSONForResponse(r, &batch); err != nil {
		return ResponseSet{}, err
	}

	set := ResponseSet{
		Responses: make([]Response, len(batch)),
		IsBatch:   true,
	}

	for i, res := range batch {
		set.Responses[i] = normalizeResponse(res)
	}

	return set, nil
}

// unmarshalJSONForResponse unmarshals JSON content from r into v.
func unmarshalJSONForResponse(r io.Reader, v interface{}) error {
	err := unmarshalJSON(r, v)

	if isJSONError(err) {
		return fmt.Errorf("unable to parse response: %w", err)
	}

	return err
}

// normalizeResponse returns a response of a specific type based on the content
// of res.
func normalizeResponse(res successOrErrorResponse) Response {
	if res.Error != nil {
		return ErrorResponse{
			Version:   res.Version,
			RequestID: res.RequestID,
			Error:     *res.Error,
		}
	}

	return SuccessResponse{
		Version:   res.Version,
		RequestID: res.RequestID,
		Result:    res.Result,
	}
}
