package harpy

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ZapExchangeLogger is an implementation of ExchangeLogger using zap.Logger.
type ZapExchangeLogger struct {
	// Target is the destination for log messages.
	Target *zap.Logger
}

var _ ExchangeLogger = (*ZapExchangeLogger)(nil)

// LogError writes an information about an error response that is a result of
// some problem with the request set as a whole.
func (l ZapExchangeLogger) LogError(ctx context.Context, res ErrorResponse) {
	fields := []zap.Field{
		zap.Int("error_code", int(res.Error.Code)),
		zap.String("error", res.Error.Code.String()),
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		fields = append(fields, zap.String("trace_id", span.SpanContext().TraceID().String()))
	}

	if res.ServerError != nil {
		fields = append(fields, zap.String("caused_by", res.ServerError.Error()))
	}

	if res.Error.Message != res.Error.Code.String() {
		fields = append(fields, zap.String("responded_with", res.Error.Message))
	}

	l.Target.Error(
		"error",
		fields...,
	)
}

// LogWriterError logs about an error that occured when attempting to use a
// ResponseWriter.
func (l ZapExchangeLogger) LogWriterError(ctx context.Context, err error) {
	fields := []zap.Field{
		zap.String("error", err.Error()),
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		fields = append(fields, zap.String("trace_id", span.SpanContext().TraceID().String()))
	}

	l.Target.Error(
		"unable to write JSON-RPC response",
		fields...,
	)
}

// LogNotification logs information about a notification request.
func (l ZapExchangeLogger) LogNotification(ctx context.Context, req Request) {
	var w strings.Builder

	w.WriteString("notify ")
	writeMethod(&w, req.Method)

	fields := []zap.Field{
		zap.Int("param_size", len(req.Parameters)),
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		fields = append(fields, zap.String("trace_id", span.SpanContext().TraceID().String()))
	}

	l.Target.Info(
		w.String(),
		fields...,
	)
}

// LogCall logs information about a call request and its response.
func (l ZapExchangeLogger) LogCall(ctx context.Context, req Request, res Response) {
	var w strings.Builder

	w.WriteString("call ")
	writeMethod(&w, req.Method)

	fields := []zap.Field{
		zap.Int("param_size", len(req.Parameters)),
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		fields = append(fields, zap.String("trace_id", span.SpanContext().TraceID().String()))
	}

	switch res := res.(type) {
	case SuccessResponse:
		fields = append(fields, zap.Int("result_size", len(res.Result)))
		l.Target.Info(
			w.String(),
			fields...,
		)
	case ErrorResponse:
		fields = append(
			fields,
			zap.Int("error_code", int(res.Error.Code)),
			zap.String("error", res.Error.Code.String()),
		)

		if res.ServerError != nil {
			fields = append(fields, zap.String("caused_by", res.ServerError.Error()))
		}

		if res.Error.Message != res.Error.Code.String() {
			fields = append(fields, zap.String("responded_with", res.Error.Message))
		}

		l.Target.Error(
			w.String(),
			fields...,
		)
	}
}
