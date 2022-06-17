package otelharpy

import (
	"github.com/dogmatiq/harpy"
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

// requestAttributes returns the OpenTelemetry attributes that are to be
// recorded for given request on every span and meter.
func requestAttributes(req harpy.Request) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.RPCMethodKey.String(req.Method),
		semconv.RPCJsonrpcVersionKey.String(req.Version),
	}

	return attrs
}

// errorResponseAttributes returns the OpenTelemetry attributes that are to be
// recorded for given error response on every span and meter.
func errorResponseAttributes(res harpy.ErrorResponse) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.RPCJsonrpcErrorCodeKey.Int(int(res.Error.Code)),
		semconv.RPCJsonrpcErrorMessageKey.String(res.Error.Message),
	}
}
