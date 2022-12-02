package harpy

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
	LogNotification(ctx context.Context, req Request)

	// LogCall logs about a call request/response pair.
	LogCall(ctx context.Context, req Request, res Response)
}

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

// writeMethod formats a JSON-RPC method name for display and writes it to w.
func writeMethod(w *strings.Builder, m string) {
	if m == "" || !isAlphaNumeric(m) {
		fmt.Fprintf(w, "%#v", m)
	} else {
		w.WriteString(m)
	}
}

// isAlphaNumeric returns true if s consists of only letters and digits.
func isAlphaNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}

	return true
}

// writeDataSize writes a human-readable representation of the given size (in
// bytes) to w.
func writeDataSize(w *strings.Builder, n int) {
	if n < 1000 {
		fmt.Fprintf(w, "%d B", n)
		return
	}

	f := float64(n)
	const units = "KMGT"

	for _, u := range units {
		f /= 1000
		if f < 1000 {
			fmt.Fprintf(w, "%0.1f %cB", f, u)
			return
		}
	}

	fmt.Fprintf(w, "%0.1f PB", f/1000)
}

// writeRequestDetails writes the details of a request to w.
func writeRequestDetails(w *strings.Builder, req Request) {
	w.WriteString("params: ")
	writeDataSize(w, len(req.Parameters))
}

// writeSuccessResponseDetails writes the details of a success response to w.
func writeSuccessResponseDetails(w *strings.Builder, res SuccessResponse) {
	w.WriteString("result: ")
	writeDataSize(w, len(res.Result))
}

// writeErrorResponseDetails writes the details of an error response to w.
func writeErrorResponseDetails(w *strings.Builder, res ErrorResponse) {
	fmt.Fprintf(w, "error: %d %s", res.Error.Code, res.Error.Code.String())

	if res.ServerError != nil {
		w.WriteString(", caused by: ")
		w.WriteString(res.ServerError.Error())
	}

	if res.Error.Message != res.Error.Code.String() {
		w.WriteString(", responded with: ")
		w.WriteString(res.Error.Message)
	}
}
