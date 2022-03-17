package harpy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode"
)

// JSONRPCVersion is the version that must appear in the "jsonrpc" field of
// JSON-RPC 2.0 requests and responses.
const JSONRPCVersion = "2.0"

// Request encapsulates a JSON-RPC request.
type Request struct {
	// Version is the JSON-RPC version.
	//
	// As per the JSON-RPC specification it MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// ID uniquely identifies requests that expect a response, that is RPC calls
	// as opposed to notifications.
	//
	// As per the JSON-RPC specification, it MUST be a JSON string, number, or
	// null value. It SHOULD NOT normally not be null. Numbers SHOULD NOT
	// contain fractional parts.
	//
	// If the ID field itself is nil, the request is a notification.
	ID json.RawMessage `json:"id,omitempty"`

	// Method is the name of the RPC method to be invoked.
	//
	// As per the JSON-RPC specification, method names that begin with "rpc."
	// are reserved for system extensions, and MUST NOT be used for anything
	// else. Each system extension is defined in a separate specification. All
	// system extensions are OPTIONAL.
	//
	// Any requests for extension methods that are not handled internally by
	// this package are treated just like any other request, thus allowing
	// extension methods to be implemented by user-defined handlers.
	//
	// This package does not currently handle any extension methods internally.
	//
	// In accordance with the JSON-RPC specification there are no requirements
	// placed on the format of the method name. This allows server
	// implementations that provide methods with an empty name, non-ASCII names,
	// or any other value that can be represented as a JSON string.
	Method string `json:"method"`

	// Parameters holds the parameter values to be used during the invocation of
	// the method.
	//
	// As per the JSON-RPC specification it MUST be a structured value, that is
	// either a JSON array or object.
	//
	// Validation of the parameters is the responsibility of the user-defined
	// handlers.
	Parameters json.RawMessage `json:"params,omitempty"`
}

// IsNotification returns true if r is a notification, as opposed to an RPC call
// that expects a response.
func (r Request) IsNotification() bool {
	return r.ID == nil
}

// ValidateServerSide checks that the request conforms to the JSON-RPC
// specification.
//
// If the request is invalid it returns a JSON-RPC error intended to be sent to
// the caller in an ErrorResponse.
func (r Request) ValidateServerSide() *Error {
	if r.Version != JSONRPCVersion {
		return NewErrorWithReservedCode(
			InvalidRequestCode,
			WithMessage(`request version must be "2.0"`),
		)
	}

	if len(r.ID) != 0 {
		if err := validateRequestID(r.ID); err != nil {
			return err
		}
	}

	// Do our best to validate the type of the parameters without actually
	// unmarshaling them. It is expected that a full unmarshal will be performed
	// in a handler via the UnmarshalParameters() method.
	if len(r.Parameters) == 0 {
		return nil
	}

	if bytes.EqualFold(r.Parameters, []byte(`null`)) {
		return nil
	}

	if len(r.Parameters) < 2 || (r.Parameters[0] != '{' && r.Parameters[0] != '[') {
		return NewErrorWithReservedCode(
			InvalidParametersCode,
			WithMessage(`parameters must be an array, an object, or null`),
		)
	}

	return nil
}

// ValidateClientSide checks that the request conforms to the JSON-RPC
// specification.
//
// It is intended to be called before sending the request to a server; if the
// request is invalid it returns the error that a server would return upon
// receiving the request set.
func (r Request) ValidateClientSide() *Error {
	if err := r.ValidateServerSide(); err != nil {
		err.isServerSide = false
		return err
	}

	return nil
}

// UnmarshalParameters is a convenience method for unmarshaling request
// parameters into a Go value.
//
// It returns the appropriate native JSON-RPC error if r.Parameters can not be
// unmarshaled into v.
//
// If v implements the Validatable interface, it calls v.Validate() after
// unmarshaling successfully. If validation fails it wraps the validation error
// in the appropriate native JSON-RPC error.
func (r Request) UnmarshalParameters(v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(r.Parameters))
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		return InvalidParameters(
			WithCause(err),
		)
	}

	if v, ok := v.(Validatable); ok {
		if err := v.Validate(); err != nil {
			return InvalidParameters(
				WithCause(err),
			)
		}
	}

	return nil
}

// validateRequestID checks that id is a valid request ID according to the
// JSON-RPC specification.
//
// It returns nil if the response is valid.
func validateRequestID(id json.RawMessage) *Error {
	var value interface{}
	if err := json.Unmarshal(id, &value); err != nil {
		return NewErrorWithReservedCode(
			ParseErrorCode,
			WithCause(err),
		)
	}

	switch value.(type) {
	case string:
		return nil
	case float64:
		return nil
	case nil:
		return nil
	default:
		return NewErrorWithReservedCode(
			InvalidRequestCode,
			WithMessage(`request ID must be a JSON string, number or null`),
		)
	}
}

// RequestSet encapsulates one or more JSON-RPC requests that were parsed from a
// single JSON message.
type RequestSet struct {
	// Requests contains the requests parsed from the message.
	Requests []Request

	// IsBatch is true if the requests are part of a batch.
	//
	// This is used to disambiguate between a single request and a batch that
	// contains only one request.
	IsBatch bool
}

// UnmarshalRequestSet unmarshals a JSON-RPC request or request batch from r.
//
// If there is a problem parsing the request or the request is malformed, an
// Error is returned. Any other non-nil error should be considered an IO error.
//
// On success it returns a request set containing well-formed (but not
// necessarily valid) requests.
func UnmarshalRequestSet(r io.Reader) (RequestSet, error) {
	br := bufio.NewReader(r)

	for {
		ch, _, err := br.ReadRune()
		if err != nil {
			return RequestSet{}, err
		}

		if unicode.IsSpace(ch) {
			continue
		}

		if err := br.UnreadRune(); err != nil {
			panic(err) // only occurs if a rune hasn't already been read
		}

		if ch == '[' {
			return unmarshalBatchRequest(br)
		}

		return unmarshalSingleRequest(br)
	}
}

// ValidateServerSide checks that the request set is valid and that the requests
// within conform to the JSON-RPC specification.
//
// If the request set is invalid it returns a JSON-RPC error intended to be sent
// to the caller in an ErrorResponse.
func (rs RequestSet) ValidateServerSide() *Error {
	if rs.IsBatch {
		if len(rs.Requests) == 0 {
			return NewErrorWithReservedCode(
				InvalidRequestCode,
				WithMessage("batches must contain at least one request"),
			)
		}
	} else if len(rs.Requests) != 1 {
		return NewErrorWithReservedCode(
			InvalidRequestCode,
			WithMessage("non-batch request sets must contain exactly one request"),
		)
	}

	for _, req := range rs.Requests {
		if err := req.ValidateServerSide(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateClientSide checks that the request set is valid and that the requests
// within conform to the JSON-RPC specification.
//
// It is intended to be called before sending the request set to a server; if
// the request set is invalid it returns the error that a server would return
// upon receiving the request set.
func (rs RequestSet) ValidateClientSide() *Error {
	if err := rs.ValidateServerSide(); err != nil {
		err.isServerSide = false
		return err
	}

	return nil
}

// unmarshalSingleRequest unmarshals a non-batch JSON-RPC request set.
func unmarshalSingleRequest(r *bufio.Reader) (RequestSet, error) {
	var req Request

	if err := unmarshalJSONForRequest(r, &req); err != nil {
		return RequestSet{}, err
	}

	return RequestSet{
		Requests: []Request{req},
		IsBatch:  false,
	}, nil
}

// unmarshalBatchRequest unmarshals a batched JSON-RPC request set.
func unmarshalBatchRequest(r *bufio.Reader) (RequestSet, error) {
	var batch []Request

	if err := unmarshalJSONForRequest(r, &batch); err != nil {
		return RequestSet{}, err
	}

	return RequestSet{
		Requests: batch,
		IsBatch:  true,
	}, nil
}

// unmarshalJSONForRequest unmarshals JSON content from r into v. If the JSON
// cannot be parsed it returns a JSON-RPC error with the "parse error" code.
func unmarshalJSONForRequest(r io.Reader, v interface{}) error {
	err := unmarshalJSON(r, v)

	if isJSONError(err) {
		return NewErrorWithReservedCode(
			ParseErrorCode,
			WithCause(fmt.Errorf("unable to parse request: %w", err)),
		)
	}

	return err
}

// Validatable is an interface for parameter values that provide their own
// validation.
type Validatable interface {
	// Validate returns a non-nil error if the value is invalid.
	//
	// The returned error, if non-nil, is always wrapped in a JSON-RPC "invalid
	// parameters" error, and therefore should not itself be a JSON-RPC error.
	Validate() error
}
