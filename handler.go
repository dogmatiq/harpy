package voorhees

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dogmatiq/dodeca/logging"
)

// A Handler is a user-defined function that produces a result value in response
// to a JSON-RPC request.
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

// Invoker invokes a handler with a JSON-RPC request in order to obtain the
// JSON-RPC response.
type Invoker struct {
	// Handler is the function that handles the request.
	Handler Handler

	// Logger is the target for messages about the requests and responses.
	Logger logging.Logger
}

// Invoke dispatches a request to the handler and returns a JSON-RPC response
// containing the handler's result.
//
// If req is a call, ok is true and res is the JSON-RPC response to send. If req
// is a notification, ok is false and res is unused.
func (i *Invoker) Invoke(ctx context.Context, req Request) (res Response, ok bool) {
	if req.IsNotification() {
		i.notify(ctx, req)
		return nil, false
	}

	return i.call(ctx, req), true
}

// notify invokes the handler for a notification request.
func (i *Invoker) notify(ctx context.Context, req Request) {
	if req.Parameters == nil {
		logging.Debug(
			i.Logger,
			`▼ NOTIFY %s WITHOUT PARAMETERS`,
			req.Method,
		)
	} else {
		logging.Debug(
			i.Logger,
			`▼ NOTIFY %s WITH PARAMETERS %s`,
			req.Method,
			req.Parameters,
		)
	}

	_, err := i.Handler(ctx, req)

	if err == nil {
		logging.Log(
			i.Logger,
			`✓ NOTIFY %s`,
			req.Method,
		)
	} else {
		logging.Log(
			i.Logger,
			`✗ NOTIFY %s  %s`,
			req.Method,
			err,
		)
	}
}

// notify invokes the handler for a call request.
func (i *Invoker) call(ctx context.Context, req Request) Response {
	if req.Parameters == nil {
		logging.Debug(
			i.Logger,
			`▼ CALL[%s] %s WITHOUT PARAMETERS`,
			req.ID,
			req.Method,
		)
	} else {
		logging.Debug(
			i.Logger,
			`▼ CALL[%s] %s WITH PARAMETERS %s`,
			req.ID,
			req.Method,
			req.Parameters,
		)
	}

	result, err := i.Handler(ctx, req)

	if err != nil {
		return i.buildErrorResponse(req, err)
	}

	return i.buildSuccessResponse(req, result)
}

// buildSuccessResponse returns the JSON-RPC response to send after successful
// handling of a call.
func (i *Invoker) buildSuccessResponse(req Request, result interface{}) Response {
	var resultJSON json.RawMessage
	if result != nil {
		var err error
		resultJSON, err = json.Marshal(result)
		if err != nil {
			return i.buildErrorResponse(
				req,
				fmt.Errorf("handler succeeded but the result could not be marshaled: %w", err),
			)
		}
	}

	logging.Log(
		i.Logger,
		`✓ CALL[%s] %s`,
		req.ID,
		req.Method,
	)

	if result == nil {
		logging.Debug(
			i.Logger,
			`▲ CALL[%s] %s SUCCESS WITHOUT RESULT`,
			req.ID,
			req.Method,
		)
	} else {
		logging.Debug(
			i.Logger,
			`▲ CALL[%s] %s SUCCESS WITH RESULT %s`,
			req.ID,
			req.Method,
			resultJSON,
		)
	}

	return SuccessResponse{
		Version:   jsonRPCVersion,
		RequestID: req.ID,
		Result:    resultJSON,
	}
}

// buildErrorResponse returns the JSON-RPC response to send after a failure to
// handle a call.
func (i *Invoker) buildErrorResponse(req Request, err error) (res ErrorResponse) {
	if nerr, ok := err.(Error); ok {
		res = i.buildNativeErrorResponse(req, nerr)
	} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		res = i.buildTransparentErrorResponse(req, err)
	} else {
		res = i.buildOpaqueErrorResponse(req, err)
	}

	var desc string
	if res.Error.Message == res.Error.Code.String() {
		desc = res.Error.Message
	} else {
		desc = fmt.Sprintf("%s: %s", res.Error.Code, res.Error.Message)
	}

	if res.Error.Data == nil {
		logging.Debug(
			i.Logger,
			`▲ CALL[%s] %s ERROR [%d] %s WITHOUT DATA`,
			req.ID,
			req.Method,
			res.Error.Code,
			desc,
		)
	} else {
		logging.Debug(
			i.Logger,
			`▲ CALL[%s] %s ERROR [%d] %s WITH DATA %s`,
			req.ID,
			req.Method,
			res.Error.Code,
			desc,
			res.Error.Data,
		)
	}

	return res
}

// buildNativeErrorResponse returns the JSON-RPC response to send when a handler
// returns a native JSON-RPC Error.
func (i *Invoker) buildNativeErrorResponse(req Request, err Error) ErrorResponse {
	res := ErrorResponse{
		Version:   jsonRPCVersion,
		RequestID: req.ID,
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
			return i.buildOpaqueErrorResponse(
				req,
				fmt.Errorf("handler failed (%s), but the user-defined error data could not be marshaled: %w", err, merr),
			)
		}

		res.Error.Data = data
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
func (i *Invoker) buildOpaqueErrorResponse(req Request, err error) ErrorResponse {
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
func (i *Invoker) buildTransparentErrorResponse(req Request, err error) ErrorResponse {
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
