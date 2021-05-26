package harpy

import (
	"context"
	"fmt"

	"github.com/dogmatiq/dodeca/logging"
)

// ExchangeLogger is an implementation of the Exchanger interface that logs
// complete request and response information that is passed to/from another
// Exchanger.
type ExchangeLogger struct {
	// Next is an Exchanger to which requests are forwarded.
	Next Exchanger

	// Logger is the target for log messages about the requests and responses.
	Logger logging.Logger
}

// Call handles a call request and returns the response.
func (l *ExchangeLogger) Call(ctx context.Context, req Request) Response {
	if req.Parameters == nil {
		logging.Log(
			l.Logger,
			`▼ CALL[%s] %s WITHOUT PARAMETERS`,
			req.ID,
			req.Method,
		)
	} else {
		logging.Log(
			l.Logger,
			`▼ CALL[%s] %s WITH PARAMETERS %s`,
			req.ID,
			req.Method,
			req.Parameters,
		)
	}

	res := l.Next.Call(ctx, req)

	switch res := res.(type) {
	case SuccessResponse:
		l.logSuccessResponse(req, res)
	case ErrorResponse:
		l.logErrorResponse(req, res)
	}

	return res
}

// Notify handles a notification request.
func (l *ExchangeLogger) Notify(ctx context.Context, req Request) {
	if req.Parameters == nil {
		logging.Log(
			l.Logger,
			`▼ NOTIFY %s WITHOUT PARAMETERS`,
			req.Method,
		)
	} else {
		logging.Log(
			l.Logger,
			`▼ NOTIFY %s WITH PARAMETERS %s`,
			req.Method,
			req.Parameters,
		)
	}

	l.Next.Notify(ctx, req)
}

// logSuccessResponse logs the details of a success response.
func (l *ExchangeLogger) logSuccessResponse(req Request, res SuccessResponse) {
	if res.Result == nil {
		logging.Log(
			l.Logger,
			`▲ CALL[%s] %s SUCCESS WITHOUT RESULT`,
			req.ID,
			req.Method,
		)
	} else {
		logging.Log(
			l.Logger,
			`▲ CALL[%s] %s SUCCESS WITH RESULT %s`,
			req.ID,
			req.Method,
			res.Result,
		)
	}
}

// logErrorResponse logs the details of an error response.
func (l *ExchangeLogger) logErrorResponse(req Request, res ErrorResponse) {
	var desc string
	if res.Error.Message == res.Error.Code.String() {
		// The error message is identical to the error code description, so only
		// display it once.
		desc = res.Error.Message
	} else {
		// The error message is more specific than the description of the error
		// code, so display them both.
		desc = fmt.Sprintf("%s: %s", res.Error.Code, res.Error.Message)
	}

	if res.Error.Data == nil {
		logging.Log(
			l.Logger,
			`▲ CALL[%s] %s ERROR [%d] %s WITHOUT DATA`,
			req.ID,
			req.Method,
			res.Error.Code,
			desc,
		)
	} else {
		logging.Log(
			l.Logger,
			`▲ CALL[%s] %s ERROR [%d] %s WITH DATA %s`,
			req.ID,
			req.Method,
			res.Error.Code,
			desc,
			res.Error.Data,
		)
	}
}
