package voorhees

import "encoding/json"

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
	ID interface{} `json:"id,omitempty"`

	// IsNotification is true if the request represents a notification, as
	// opposed to an RPC call.
	//
	// This is used to disambiguate between a request that was made without an
	// ID value vs one made with a NULL ID (which is allowed, though
	// discouraged, by the specification).
	IsNotification bool

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
