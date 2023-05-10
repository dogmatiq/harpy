package harpy

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/exp/slog"
)

// ExchangeLogger is an interface for logging JSON-RPC requests, responses and
// errors.
type ExchangeLogger interface {
	// LogError logs about an error that is a result of some problem with the
	// request set as a whole.
	LogError(ctx context.Context, res ErrorResponse)

	// LogWriterError logs about an error that occured when attempting to use a
	// ResponseWriter.
	LogWriterError(ctx context.Context, err error)

	// LogNotification logs about a notification request.
	LogNotification(ctx context.Context, req Request, err error)

	// LogCall logs about a call request/response pair.
	LogCall(ctx context.Context, req Request, res Response)
}

// NewZapExchangeLogger returns an ExchangeLogger that targets the given
// [zap.Logger].
func NewZapExchangeLogger(t *zap.Logger) ExchangeLogger {
	return &structuredExchangeLogger[zap.Field]{
		Target: t,
		Int:    zap.Int,
		String: zap.String,
	}
}

// NewSLogExchangeLogger returns an ExchangeLogger that targets the given
// [slog.Logger].
func NewSLogExchangeLogger(t *slog.Logger) ExchangeLogger {
	return &structuredExchangeLogger[any]{
		Target: t,
		Int: func(n string, v int) any {
			return slog.Int(n, v)
		},
		String: func(n string, v string) any {
			return slog.String(n, v)
		},
	}
}

type structuredExchangeLogger[Attr any] struct {
	Target interface {
		Info(message string, attrs ...Attr)
		Error(message string, attrs ...Attr)
	}
	Int    func(string, int) Attr
	String func(string, string) Attr
}

var _ ExchangeLogger = (*structuredExchangeLogger[any])(nil)

// LogError writes an information about an error response that is a result of
// some problem with the request set as a whole.
func (l structuredExchangeLogger[Attr]) LogError(ctx context.Context, res ErrorResponse) {
	attrs := []Attr{
		l.Int("error_code", int(res.Error.Code)),
		l.String("error", res.Error.Code.String()),
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		attrs = append(attrs, l.String("trace_id", span.SpanContext().TraceID().String()))
	}

	if res.ServerError != nil {
		attrs = append(attrs, l.String("caused_by", res.ServerError.Error()))
	}

	if res.Error.Message != res.Error.Code.String() {
		attrs = append(attrs, l.String("responded_with", res.Error.Message))
	}

	l.Target.Error(
		"error",
		attrs...,
	)
}

// LogWriterError logs about an error that occured when attempting to use a
// ResponseWriter.
func (l structuredExchangeLogger[Attr]) LogWriterError(ctx context.Context, err error) {
	attrs := []Attr{
		l.String("error", err.Error()),
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		attrs = append(attrs, l.String("trace_id", span.SpanContext().TraceID().String()))
	}

	l.Target.Error(
		"unable to write JSON-RPC response",
		attrs...,
	)
}

// LogNotification logs information about a notification request.
func (l structuredExchangeLogger[Attr]) LogNotification(ctx context.Context, req Request, err error) {
	attrs := []Attr{
		l.String("method", req.Method),
		l.Int("param_size", len(req.Parameters)),
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		attrs = append(attrs, l.String("trace_id", span.SpanContext().TraceID().String()))
	}

	switch err := err.(type) {
	case nil:
		l.Target.Info("notify", attrs...)
	case Error:
		attrs = append(
			attrs,
			l.Int("error_code", int(err.Code())),
			l.String("error", err.Message()),
		)

		if cause := err.Unwrap(); cause != nil {
			attrs = append(attrs, l.String("caused_by", cause.Error()))
		}

		l.Target.Error("notify", attrs...)
	default:
		attrs = append(attrs, l.String("error", err.Error()))
		l.Target.Error("notify", attrs...)
	}
}

// LogCall logs information about a call request and its response.
func (l structuredExchangeLogger[Attr]) LogCall(ctx context.Context, req Request, res Response) {
	attrs := []Attr{
		l.String("method", req.Method),
		l.Int("param_size", len(req.Parameters)),
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		attrs = append(attrs, l.String("trace_id", span.SpanContext().TraceID().String()))
	}

	switch res := res.(type) {
	case SuccessResponse:
		attrs = append(attrs, l.Int("result_size", len(res.Result)))
		l.Target.Info(
			"call",
			attrs...,
		)
	case ErrorResponse:
		attrs = append(
			attrs,
			l.Int("error_code", int(res.Error.Code)),
			l.String("error", res.Error.Code.String()),
		)

		if res.ServerError != nil {
			attrs = append(attrs, l.String("caused_by", res.ServerError.Error()))
		}

		if res.Error.Message != res.Error.Code.String() {
			attrs = append(attrs, l.String("responded_with", res.Error.Message))
		}

		l.Target.Error(
			"call",
			attrs...,
		)
	}
}
