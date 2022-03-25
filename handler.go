package harpy

import (
	"context"
)

// A Handler is a function that produces a result value (or error) in response
// to a JSON-RPC request for a specific method.
//
// res is the result value to include in the JSON-RPC response; it is not the
// JSON-RPC response itself. If err is non-nil, a JSON-RPC error response is
// sent instead and res is ignored.
//
// If req is a notification (that is, it does not have a request ID) res is
// always ignored.
type Handler func(ctx context.Context, req Request) (res any, err error)

// Router is a Exchanger that dispatches to different handlers based on the
// JSON-RPC method name.
type Router map[string]Handler

// Call handles a call request and returns the response.
//
// It invokes the handler associated with the method specified by the request.
// If no such method has been registered it returns a JSON-RPC "method not
// found" error response.
func (r Router) Call(ctx context.Context, req Request) Response {
	h, ok := r[req.Method]
	if !ok {
		return NewErrorResponse(
			req.ID,
			MethodNotFound(),
		)
	}

	result, err := h(ctx, req)
	if err != nil {
		return NewErrorResponse(req.ID, err)
	}

	return NewSuccessResponse(req.ID, result)
}

// Notify handles a notification request.
//
// It invokes the handler associated with the method specified by the request.
// If no such method has been registered it does nothing.
func (r Router) Notify(ctx context.Context, req Request) {
	if h, ok := r[req.Method]; ok {
		h(ctx, req) // nolint:errcheck // notification errors are not reported to the caller
	}
}
