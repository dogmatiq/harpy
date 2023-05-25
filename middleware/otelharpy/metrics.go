package otelharpy

import (
	"context"
	"sync"
	"time"

	"github.com/dogmatiq/harpy"
	"github.com/dogmatiq/harpy/internal/version"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics is an implementation of harpy.Exchanger that provides OpenTelemetry
// metrics for each JSON-RPC request.
type Metrics struct {
	// Next is the next exchanger in the middleware stack.
	Next harpy.Exchanger

	// MeterProvider is the OpenTelemetry MeterProvider used to create meters.
	MeterProvider metric.MeterProvider

	// ServiceName is an application specific service name to use in the span
	// name and attributes.
	//
	// It may be prefixed with a dot-separated "package name", for example
	// "myapp.test.EchoService".
	//
	// It may be empty, in which case it is omitted from the span.
	ServiceName string

	once          sync.Once
	calls         metric.Int64Counter
	notifications metric.Int64Counter
	errors        metric.Int64Counter
	duration      metric.Int64Histogram
	attributes    []attribute.KeyValue
}

var _ harpy.Exchanger = (*Metrics)(nil)

// Call handles a call request and returns the response.
func (m *Metrics) Call(ctx context.Context, req harpy.Request) harpy.Response {
	m.init()

	attrs := requestAttributes(req)
	attrs = append(attrs, m.attributes...)
	attrOption := metric.WithAttributes(attrs...)

	m.calls.Add(ctx, 1, attrOption)

	start := time.Now()
	res := m.Next.Call(ctx, req)
	elapsed := time.Since(start)

	m.duration.Record(ctx, durationToMillis(elapsed), attrOption)

	if res, ok := res.(harpy.ErrorResponse); ok {
		attrs = append(attrs, errorResponseAttributes(res)...)
		m.errors.Add(ctx, 1, attrOption)
	}

	return res
}

// Notify handles a notification request.
//
// It invokes the handler associated with the method specified by the request.
// If no such method has been registered it does nothing.
func (m *Metrics) Notify(ctx context.Context, req harpy.Request) error {
	m.init()

	attrs := requestAttributes(req)
	attrs = append(attrs, m.attributes...)
	attrOption := metric.WithAttributes(attrs...)

	m.notifications.Add(ctx, 1, attrOption)

	start := time.Now()
	err := m.Next.Notify(ctx, req)
	elapsed := time.Since(start)

	m.duration.Record(ctx, durationToMillis(elapsed), attrOption)

	if err != nil {
		m.errors.Add(ctx, 1, attrOption)
	}

	return err
}

// init initializes the tracer if it has not already been initialized.
func (m *Metrics) init() {
	m.once.Do(func() {
		meter := m.MeterProvider.Meter(
			"github.com/dogmatiq/harpy/middleware/otelharpy",
			metric.WithInstrumentationVersion(version.Version),
		)

		var err error

		m.calls, err = meter.Int64Counter(
			"rpc.server.calls",
			metric.WithDescription("The number of JSON-RPC requests that are 'calls'."),
			metric.WithUnit("1"),
		)
		if err != nil {
			panic(err)
		}

		m.notifications, err = meter.Int64Counter(
			"rpc.server.notifications",
			metric.WithDescription("The number of JSON-RPC requests that are 'calls'."),
			metric.WithUnit("1"),
		)
		if err != nil {
			panic(err)
		}

		m.errors, err = meter.Int64Counter(
			"rpc.server.errors",
			metric.WithDescription("The number of JSON-RPC 'call' requests that result in an error."),
			metric.WithUnit("1"),
		)
		if err != nil {
			panic(err)
		}

		m.duration, err = meter.Int64Histogram(
			"rpc.server.duration",
			metric.WithDescription("The amount of time it takes user-provided handlers to process JSON-RPC requests."),
			metric.WithUnit("ms"),
		)
		if err != nil {
			panic(err)
		}

		m.attributes = commonAttributes(m.ServiceName)
	})
}

// durationToMillis converts a duration to milliseconds.
func durationToMillis(d time.Duration) int64 {
	return int64(d / time.Millisecond)
}
