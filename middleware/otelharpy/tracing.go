package otelharpy

import (
	"context"
	"strings"
	"sync"

	"github.com/dogmatiq/harpy"
	"github.com/dogmatiq/harpy/internal/version"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracing is an implementation of harpy.Exchanger that provides OpenTelemetry
// tracing for each JSON-RPC request.
//
// It adheres to the OpenTelemetry RPC semantic conventions as specified in
// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/rpc.md.
type Tracing struct {
	// Next is the next exchanger in the middleware stack.
	Next harpy.Exchanger

	// TracerProvider is the OpenTelemetry TracerProvider to use for creating
	// spans.
	TracerProvider trace.TracerProvider

	// ServiceName is an application specific service name to use in the span
	// name and attributes.
	//
	// It may be prefixed with a dot-separated "package name", for example
	// "myapp.test.EchoService".
	//
	// It may be empty, in which case it is omitted from the span.
	ServiceName string

	// CreateNewSpan controls whether a new span is created for each request, or
	// JSON-RPC attributes are added to an existing span.
	//
	// By default it is assumed that the transport layer is responsibe for
	// creating the span, and no new span will be created.
	CreateNewSpan bool

	once           sync.Once
	tracer         trace.Tracer
	spanNamePrefix string
	attributes     []attribute.KeyValue
}

var _ harpy.Exchanger = (*Tracing)(nil)

// Call handles a call request and returns the response.
func (t *Tracing) Call(ctx context.Context, req harpy.Request) harpy.Response {
	var res harpy.Response

	t.withSpan(
		ctx,
		req,
		func(ctx context.Context, span trace.Span) {
			res = t.Next.Call(ctx, req)

			if res, ok := res.(harpy.ErrorResponse); ok {
				span.SetAttributes(errorResponseAttributes(res)...)

				if res.ServerError == nil {
					span.SetStatus(codes.Error, res.Error.Message)
				} else {
					span.SetStatus(codes.Error, res.ServerError.Error())
					span.RecordError(res.ServerError)
				}
			} else {
				span.SetStatus(codes.Ok, "")
			}
		},
	)

	return res
}

// Notify handles a notification request.
//
// It invokes the handler associated with the method specified by the request.
// If no such method has been registered it does nothing.
func (t *Tracing) Notify(ctx context.Context, req harpy.Request) error {
	var err error

	t.withSpan(
		ctx,
		req,
		func(ctx context.Context, span trace.Span) {
			err = t.Next.Notify(ctx, req)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
			} else {
				span.SetStatus(codes.Ok, "")
			}
		},
	)

	return err
}

// withSpan invokes fn with a tracing span.
func (t *Tracing) withSpan(
	ctx context.Context,
	req harpy.Request,
	fn func(context.Context, trace.Span),
) {
	t.init()

	name := t.spanNamePrefix + sanitizeMethodName(req.Method)
	var span trace.Span

	if t.CreateNewSpan {
		ctx, span = t.tracer.Start(
			ctx,
			name,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()
	} else {
		span = trace.SpanFromContext(ctx)
		span.SetName(name)
	}

	span.SetAttributes(t.attributes...)
	span.SetAttributes(requestAttributes(req)...)

	if !req.IsNotification() {
		span.SetAttributes(
			semconv.RPCJsonrpcRequestIDKey.String(sanitizeRequestID(req)),
		)
	}

	fn(ctx, span)
}

// init initializes the tracer if it has not already been initialized.
func (t *Tracing) init() {
	t.once.Do(func() {
		t.tracer = t.TracerProvider.Tracer(
			"github.com/dogmatiq/harpy/middleware/otelharpy",
			trace.WithInstrumentationVersion(version.Version),
		)

		t.attributes = commonAttributes(t.ServiceName)

		if t.ServiceName != "" {
			t.spanNamePrefix = t.ServiceName + "/"
		}
	})
}

// sanitizeRequestID returns a request ID suitable for use as a span attribute.
//
// As per semconv.RPCJsonrpcRequestIDKey it returns an empty string if the
// requestID is null.
func sanitizeRequestID(req harpy.Request) string {
	requestID := string(req.ID)

	if strings.EqualFold(requestID, "null") {
		return ""
	}

	return strings.Trim(requestID, `"`)
}

// sanitizeMethodName returns an RPC method name suitable for use in part of
// span name.
func sanitizeMethodName(n string) string {
	return strings.ReplaceAll(n, "/", "-")
}
