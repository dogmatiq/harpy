package otelharpy

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

// commonAttributes returns the OpenTelemetry attributes that are recorded on
// every span and meter.
func commonAttributes(serviceName string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.RPCSystemKey.String("dogmatiq/harpy"),
	}

	if serviceName != "" {
		attrs = append(
			attrs,
			semconv.RPCServiceKey.String(serviceName),
		)
	}

	return attrs
}
