package harpy

import (
	"context"

	"github.com/dogmatiq/dodeca/logging"
)

// A Handler is a function that produces a result value (or error) in response
// to a JSON-RPC request.
//
// res is the result value to include in the JSON-RPC response; it is not the
// JSON-RPC response itself. If err is non-nil, a JSON-RPC error response is
// sent instead and res is ignored.
//
// If req is a notification (that is, it does not have a request ID) res is
// always ignored.
type Handler func(ctx context.Context, req Request) (res interface{}, err error)

// HandlerInvoker is an implementation of the Exchanger interface that invokes a
// Handler.
//
// It logs meta-data about each request and response, but does not include the
// complete data structure for request/response information. Such logging can be
// useful for debugging, and is provided by the ExchangeLogger type.
type HandlerInvoker struct {
	// Handle is the function that handles the request.
	Handler Handler

	// Logger is the target for messages about the requests and responses.
	Logger logging.Logger
}

// Call handles a call request and returns the response.
func (i *HandlerInvoker) Call(ctx context.Context, req Request) Response {
	result, err := i.Handler(ctx, req)

	var res Response
	if err == nil {
		res = NewSuccessResponse(req.ID, result)
	} else {
		res = NewErrorResponse(req.ID, err)
	}

	switch res := res.(type) {
	case SuccessResponse:
		logging.Log(
			i.Logger,
			`✓ '%s' CALL`,
			req.Method,
		)
	case ErrorResponse:
		if res.ServerError != nil {
			logging.Log(
				i.Logger,
				`✗ '%s' CALL  %s  [cause: %s]`,
				req.Method,
				res.Error,
				res.ServerError,
			)
		} else {
			logging.Log(
				i.Logger,
				`✗ '%s' CALL  %s`,
				req.Method,
				res.Error,
			)
		}
	}

	return res
}

// Notify handles a notification request.
func (i *HandlerInvoker) Notify(ctx context.Context, req Request) {
	_, err := i.Handler(ctx, req)
	if err != nil {
		logging.Log(
			i.Logger,
			`✗ '%s' NOTIFY  %s`,
			req.Method,
			err,
		)

		return
	}

	logging.Log(
		i.Logger,
		`✓ '%s' NOTIFY`,
		req.Method,
	)
}
