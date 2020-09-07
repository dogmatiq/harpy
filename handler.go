package voorhees

import (
	"context"
	"errors"

	"github.com/dogmatiq/dodeca/logging"
)

// Handler is a function that produces a result value in response to a
// JSON-RPC request.
//
// res is the result value to include in the JSON-RPC response (it is not the
// JSON-RPC response itself).
//
// If err is non-nil, a JSON-RPC error response is sent instead and res is
// ignored.
//
// If req is a notification (that is, it does not have a request ID) res is
// always ignored.
type Handler func(ctx context.Context, req Request) (res interface{}, err error)

// HandlerInvoker is a PipelineStage that dispatches to a Handler.
type HandlerInvoker struct {
	// Handle is the function that handles the request.
	Handler Handler

	// Logger is the target for messages about the requests and responses.
	Logger logging.Logger
}

// Call handles a call request and returns the response.
func (i *HandlerInvoker) Call(ctx context.Context, req Request) Response {
	result, err := i.Handler(ctx, req)
	if err != nil {
		return i.buildErrorResponse(req, err)
	}

	return i.buildSuccessResponse(req, result)
}

// buildSuccessResponse returns the JSON-RPC response to send after successful
// handling of a call.
func (i *HandlerInvoker) buildSuccessResponse(req Request, result interface{}) Response {
	res, err := NewSuccessResponse(req.ID, result)
	if err != nil {
		return i.buildErrorResponse(req, err)
	}

	logging.Log(
		i.Logger,
		`✓ CALL[%s] %s`,
		req.ID,
		req.Method,
	)

	return res
}

// buildErrorResponse returns the JSON-RPC response to send after a failure to
// handle a call.
func (i *HandlerInvoker) buildErrorResponse(req Request, err error) (res ErrorResponse) {
	if nerr, ok := err.(Error); ok {
		return i.buildNativeErrorResponse(req, nerr)
	} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return i.buildTransparentErrorResponse(req, err)
	} else {
		return i.buildOpaqueErrorResponse(req, err)
	}
}

// buildNativeErrorResponse returns the JSON-RPC response to send when a handler
// returns a native JSON-RPC Error.
func (i *HandlerInvoker) buildNativeErrorResponse(req Request, err Error) ErrorResponse {
	res, rerr := NewErrorResponse(req.ID, err)
	if rerr != nil {
		return i.buildOpaqueErrorResponse(req, rerr)
	}

	logging.Log(
		i.Logger,
		`✗ CALL[%s] %s  %s`,
		req.ID,
		req.Method,
		err,
	)

	return res
}

// buildOpaqueErrorResponse returns the JSON-RPC response to return when the
// handling of a request failed but the direct cause should NOT be reported to
// the caller.
func (i *HandlerInvoker) buildOpaqueErrorResponse(req Request, err error) ErrorResponse {
	res := ErrorResponse{
		Version:   jsonRPCVersion,
		RequestID: req.ID,
		Error: ErrorInfo{
			Code:    InternalErrorCode,
			Message: InternalErrorCode.String(), // Note, we do NOT use the actual error message in the response.
		},
	}

	logging.Log(
		i.Logger,
		`✗ CALL[%s] %s  [%d] %s: %s  (cause not shown to caller)`,
		req.ID,
		req.Method,
		res.Error.Code,
		res.Error.Code,
		err,
	)

	return res
}

// buildTransparentErrorResponse returns the JSON-RPC response to return when
// the handling of a request failed and it is safe to inform the caller of the
// direct cause.
func (i *HandlerInvoker) buildTransparentErrorResponse(req Request, err error) ErrorResponse {
	res := ErrorResponse{
		Version:   jsonRPCVersion,
		RequestID: req.ID,
		Error: ErrorInfo{
			Code:    InternalErrorCode,
			Message: err.Error(), // Note, we use the actual error message in the response.
		},
	}

	logging.Log(
		i.Logger,
		`✗ CALL[%s] %s  [%d] %s: %s`,
		req.ID,
		req.Method,
		res.Error.Code,
		res.Error.Code,
		err,
	)

	return res
}

// Notify handles a notification request.
func (i *HandlerInvoker) Notify(ctx context.Context, req Request) {
	_, err := i.Handler(ctx, req)
	if err != nil {
		logging.Log(
			i.Logger,
			`✗ NOTIFY %s  %s`,
			req.Method,
			err,
		)

		return
	}

	logging.Log(
		i.Logger,
		`✓ NOTIFY %s`,
		req.Method,
	)
}
