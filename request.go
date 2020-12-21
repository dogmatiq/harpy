package harpy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// jsonRPCVersion is the version that must appear in the "jsonrpc" field of
// JSON-RPC v2 requests and responses.
const jsonRPCVersion = "2.0"

// Request encapsulates a JSON-RPC request.
type Request struct {
	// Version is the JSON-RPC version.
	//
	// As per the specification it MUST be exactly "2.0".
	Version string `json:"jsonrpc"`

	// ID uniquely identifies requests that expect a response, that is RPC calls
	// as opposed to notifications.
	//
	// As per the specification, it MUST be a JSON string, number, or null
	// value. It SHOULD NOT normally not be null. Numbers SHOULD NOT contain
	// fractional parts.
	//
	// If the ID field itself is nil, the request is a notification.
	ID json.RawMessage `json:"id,omitempty"`

	// Method is the name of the RPC method to be invoked.
	//
	// As per the specification, method names that begin with "rpc." are
	// reserved for system extensions, and MUST NOT be used for anything else.
	// Each system extension is defined in a separate specification. All system
	// extensions are OPTIONAL.
	//
	// Any requests for extension methods that are not handled internally by
	// this package are treated just like any other request, allow extension
	// methods to be implemented by user-defined handlers.
	//
	// This package does not currently handle any extension methods internally.
	Method string `json:"method"`

	// Parameters holds the parameter values to be used during the invocation of
	// the method.
	//
	// As per the specification it MUST be a structured value, that is either a
	// JSON array or object.
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

// Validate returns true if the request is valid.
//
// If r is invalid it returns an Error describing the problem.
func (r Request) Validate() (Error, bool) {
	if r.Version != jsonRPCVersion {
		return NewErrorWithReservedCode(
			InvalidRequestCode,
			WithMessage(`request version must be "2.0"`),
		), false
	}

	if r.ID != nil {
		return validateRequestID(r.ID)
	}

	return Error{}, true
}

// validateRequestID returns false if the given request ID is not one of the
// accepted types.
func validateRequestID(id json.RawMessage) (Error, bool) {
	var value interface{}
	if err := json.Unmarshal(id, &value); err != nil {
		return NewErrorWithReservedCode(
			ParseErrorCode,
			WithCause(err),
		), false
	}

	switch value.(type) {
	case string:
		return Error{}, true
	case float64:
		return Error{}, true
	case nil:
		return Error{}, true
	default:
		return NewErrorWithReservedCode(
			InvalidRequestCode,
			WithMessage(`request ID must be a JSON string, number or null`),
		), false
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

// ParseRequestSet reads and parses a JSON-RPC request or request batch from r.
//
// If there is a problem parsing the request or the request is malformed, an
// Error is returned. Any other non-nil error should be considered an I/O error.
//
// On success it returns a request set containing well-formed (but not
// necessarily valid) requests.
func ParseRequestSet(r io.Reader) (RequestSet, error) {
	br := bufio.NewReader(r)

	for {
		ch, _, err := br.ReadRune()
		if err != nil {
			return RequestSet{}, err
		}

		if unicode.IsSpace(ch) {
			continue
		}

		br.UnreadRune()

		if ch == '[' {
			return parseBatchRequest(br)
		}

		return parseSingleRequest(br)
	}
}

// Validate returns true if the request set is valid.
//
// If rs is invalid it returns an Error describing the problem.
func (rs RequestSet) Validate() (Error, bool) {
	if rs.IsBatch {
		if len(rs.Requests) == 0 {
			return NewErrorWithReservedCode(
				InvalidRequestCode,
				WithMessage("batches must contain at least one request"),
			), false
		}
	} else if len(rs.Requests) != 1 {
		return NewErrorWithReservedCode(
			InvalidRequestCode,
			WithMessage("non-batch request sets must contain exactly one request"),
		), false
	}

	for _, req := range rs.Requests {
		if err, ok := req.Validate(); !ok {
			return err, false
		}
	}

	return Error{}, true
}

func parseSingleRequest(r *bufio.Reader) (RequestSet, error) {
	var req Request

	if err := parse(r, &req); err != nil {
		return RequestSet{}, err
	}

	return RequestSet{
		Requests: []Request{req},
		IsBatch:  false,
	}, nil
}

func parseBatchRequest(r *bufio.Reader) (RequestSet, error) {
	var batch []Request

	if err := parse(r, &batch); err != nil {
		return RequestSet{}, err
	}

	return RequestSet{
		Requests: batch,
		IsBatch:  true,
	}, nil
}

func parse(r io.Reader, v interface{}) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	err := dec.Decode(&v)

	if isJSONError(err) {
		return NewErrorWithReservedCode(
			ParseErrorCode,
			WithCause(fmt.Errorf("unable to parse request: %w", err)),
		)
	}

	return err
}

// isJSONError returns true if err indicates a JSON parse failure of some kind.
func isJSONError(err error) bool {
	switch err.(type) {
	case nil:
		return false
	case *json.SyntaxError:
		return true
	case *json.UnmarshalFieldError:
		return true
	case *json.UnmarshalTypeError:
		return true
	default:
		// Unfortunately, some JSON errors do not have distinct types. For
		// example, when parsing using a decoder with DisallowUnknownFields()
		// enabled an unexpected field is reported using the equivalent of:
		//
		//   errors.New(`json: unknown field "<field name>"`)
		return strings.HasPrefix(err.Error(), "json:")
	}
}
