package otelharpy

import (
	"context"
	"sync"
	"time"

	"github.com/dogmatiq/harpy"
	"github.com/dogmatiq/harpy/internal/version"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/metric/unit"
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
	meter         metric.Meter
	calls         syncint64.Counter
	notifications syncint64.Counter
	errors        syncint64.Counter
	duration      syncint64.Histogram
	attributes    []attribute.KeyValue
}

// Call handles a call request and returns the response.
func (m *Metrics) Call(ctx context.Context, req harpy.Request) harpy.Response {
	m.init()

	attrs := requestAttributes(req)
	attrs = append(attrs, m.attributes...)

	m.calls.Add(ctx, 1, attrs...)

	start := time.Now()
	res := m.Next.Call(ctx, req)
	elapsed := time.Since(start)

	m.duration.Record(ctx, durationToMillis(elapsed), attrs...)

	if res, ok := res.(harpy.ErrorResponse); ok {
		attrs = append(attrs, errorResponseAttributes(res)...)
		m.errors.Add(ctx, 1, attrs...)
	}

	return res
}

// Notify handles a notification request.
//
// It invokes the handler associated with the method specified by the request.
// If no such method has been registered it does nothing.
func (m *Metrics) Notify(ctx context.Context, req harpy.Request) {
	m.init()

	attrs := requestAttributes(req)
	attrs = append(attrs, m.attributes...)

	m.notifications.Add(ctx, 1, attrs...)

	start := time.Now()
	m.Next.Notify(ctx, req)
	elapsed := time.Since(start)

	m.duration.Record(ctx, durationToMillis(elapsed), attrs...)
}

// init initializes the tracer if it has not already been initialized.
func (m *Metrics) init() {
	m.once.Do(func() {
		m.meter = m.MeterProvider.Meter(
			"github.com/dogmatiq/harpy/middleware/otelharpy",
			metric.WithInstrumentationVersion(version.Version),
		)

		var err error
		m.calls, err = m.meter.SyncInt64().Counter(
			"rpc.server.calls",
			instrument.WithDescription("The number of JSON-RPC requests that are 'calls'."),
			instrument.WithUnit(unit.Dimensionless),
		)
		if err != nil {
			panic(err)
		}

		m.notifications, err = m.meter.SyncInt64().Counter(
			"rpc.server.notifications",
			instrument.WithDescription("The number of JSON-RPC requests that are 'calls'."),
			instrument.WithUnit(unit.Dimensionless),
		)
		if err != nil {
			panic(err)
		}

		m.errors, err = m.meter.SyncInt64().Counter(
			"rpc.server.errors",
			instrument.WithDescription("The number of JSON-RPC 'call' requests that result in an error."),
			instrument.WithUnit(unit.Dimensionless),
		)
		if err != nil {
			panic(err)
		}

		m.duration, err = m.meter.SyncInt64().Histogram(
			"rpc.server.duration",
			instrument.WithDescription("The amount of time it takes user-provided handlers to process JSON-RPC requests."),
			instrument.WithUnit(unit.Milliseconds),
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
