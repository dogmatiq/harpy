package voorhees

import (
	"bufio"
	"encoding/json"
	"io"
	"unicode"
)

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
// It returns an error if the request set is malformed, but the requests are not
// validated.
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

func parseSingleRequest(r *bufio.Reader) (RequestSet, error) {
	var req Request

	dec := json.NewDecoder(r)
	if err := dec.Decode(&req); err != nil {
		return RequestSet{}, err
	}

	return RequestSet{
		Requests: []Request{req},
		IsBatch:  false,
	}, nil
}

func parseBatchRequest(r *bufio.Reader) (RequestSet, error) {
	var batch []Request

	dec := json.NewDecoder(r)
	if err := dec.Decode(&batch); err != nil {
		return RequestSet{}, err
	}

	return RequestSet{
		Requests: batch,
		IsBatch:  true,
	}, nil
}
