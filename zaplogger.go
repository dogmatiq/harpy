package harpy

import (
	"strings"

	"go.uber.org/zap"
)

// ZapExchangeLogger is an implementation of ExchangeLogger using zap.Logger.
type ZapExchangeLogger struct {
	// Target is the destination for log messages.
	Target *zap.Logger
}

// LogError writes an information about an error response that is a result of
// some problem with the request set as a whole.
func (l ZapExchangeLogger) LogError(res ErrorResponse) {
	fieldCount := 2
	fields := [4]zap.Field{
		zap.Int("error_code", int(res.Error.Code)),
		zap.String("error", res.Error.Code.String()),
	}

	if res.ServerError != nil {
		fields[fieldCount] = zap.String("caused_by", res.ServerError.Error())
		fieldCount++
	}

	if res.Error.Message != res.Error.Code.String() {
		fields[fieldCount] = zap.String("responded_with", res.Error.Message)
		fieldCount++
	}

	l.Target.Error(
		"error in response",
		fields[:fieldCount]...,
	)
}

// LogWriterError logs about an error that occured when attempting to use a
// ResponseWriter.
func (l ZapExchangeLogger) LogWriterError(err error) {
	l.Target.Error(
		"unable to write JSON-RPC response",
		zap.String("error", err.Error()),
	)
}

// LogNotification logs information about a notification request.
func (l ZapExchangeLogger) LogNotification(req Request) {
	var w strings.Builder

	w.WriteString("notify ")
	writeMethod(&w, req.Method)

	l.Target.Info(
		w.String(),
		zap.Int("param_size", len(req.Parameters)),
	)
}

// LogCall logs information about a call request and its response.
func (l ZapExchangeLogger) LogCall(req Request, res Response) {
	var w strings.Builder

	w.WriteString("call ")
	writeMethod(&w, req.Method)

	switch res := res.(type) {
	case SuccessResponse:
		l.Target.Info(
			w.String(),
			zap.Int("param_size", len(req.Parameters)),
			zap.Int("result_size", len(res.Result)),
		)
	case ErrorResponse:
		fieldCount := 3
		fields := [5]zap.Field{
			zap.Int("param_size", len(req.Parameters)),
			zap.Int("error_code", int(res.Error.Code)),
			zap.String("error", res.Error.Code.String()),
		}

		if res.ServerError != nil {
			fields[fieldCount] = zap.String("caused_by", res.ServerError.Error())
			fieldCount++
		}

		if res.Error.Message != res.Error.Code.String() {
			fields[fieldCount] = zap.String("responded_with", res.Error.Message)
			fieldCount++
		}

		l.Target.Error(
			w.String(),
			fields[:fieldCount]...,
		)
	}
}
