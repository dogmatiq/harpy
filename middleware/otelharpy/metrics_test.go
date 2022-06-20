package otelharpy_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dogmatiq/harpy"
	. "github.com/dogmatiq/harpy/internal/fixtures"
	. "github.com/dogmatiq/harpy/middleware/otelharpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metrictest"
	"go.opentelemetry.io/otel/sdk/metric/number"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

var _ = Describe("type Metrics", func() {
	var (
		request   harpy.Request
		response  harpy.Response
		exchanger *ExchangerStub
		exporter  *metrictest.Exporter
		metrics   *Metrics
	)

	BeforeEach(func() {
		request = harpy.Request{
			Version:    "2.0",
			ID:         json.RawMessage(`123`),
			Method:     "<method/name>",
			Parameters: json.RawMessage(`[1, 2, 3]`),
		}

		response = harpy.SuccessResponse{
			Version:   "2.0",
			RequestID: request.ID,
			Result:    json.RawMessage(`"<result>"`),
		}

		exchanger = &ExchangerStub{
			CallFunc: func(
				_ context.Context,
				req harpy.Request,
			) harpy.Response {
				return response
			},
		}

		var provider metric.MeterProvider
		provider, exporter = metrictest.NewTestMeterProvider()

		metrics = &Metrics{
			Next:          exchanger,
			MeterProvider: provider,
			ServiceName:   "package.subpackage.Service",
		}
	})

	Describe("func Call()", func() {
		It("forwards to the next exchanger", func() {
			exchanger.CallFunc = func(
				_ context.Context,
				req harpy.Request,
			) harpy.Response {
				Expect(req).To(Equal(request))
				return response
			}

			res := metrics.Call(context.Background(), request)
			Expect(res).To(Equal(response))
		})

		It("increments the call count", func() {
			for i := 0; i < 3; i++ {
				metrics.Call(context.Background(), request)
			}

			err := exporter.Collect(context.Background())
			Expect(err).ShouldNot(HaveOccurred())

			rec, err := exporter.GetByName("rpc.server.calls")
			Expect(err).ShouldNot(HaveOccurred())

			Expect(rec.InstrumentationLibrary).To(Equal(metrictest.Library{
				InstrumentationName:    "github.com/dogmatiq/harpy/middleware/otelharpy",
				InstrumentationVersion: "0.0.0-dev",
			}))

			Expect(rec.Attributes).To(ConsistOf(
				semconv.RPCSystemKey.String("dogmatiq/harpy"),
				semconv.RPCServiceKey.String("package.subpackage.Service"),
				semconv.RPCMethodKey.String("<method/name>"),
				semconv.RPCJsonrpcVersionKey.String("2.0"),
			))

			Expect(rec.AggregationKind).To(Equal(aggregation.SumKind))
			Expect(rec.NumberKind).To(Equal(number.Int64Kind))
			Expect(rec.Sum).To(Equal(number.NewInt64Number(3)))
		})

		It("records the duration", func() {
			for i := 0; i < 3; i++ {
				metrics.Call(context.Background(), request)
			}

			err := exporter.Collect(context.Background())
			Expect(err).ShouldNot(HaveOccurred())

			rec, err := exporter.GetByName("rpc.server.duration")
			Expect(err).ShouldNot(HaveOccurred())

			Expect(rec.InstrumentationLibrary).To(Equal(metrictest.Library{
				InstrumentationName:    "github.com/dogmatiq/harpy/middleware/otelharpy",
				InstrumentationVersion: "0.0.0-dev",
			}))

			Expect(rec.Attributes).To(ConsistOf(
				semconv.RPCSystemKey.String("dogmatiq/harpy"),
				semconv.RPCServiceKey.String("package.subpackage.Service"),
				semconv.RPCMethodKey.String("<method/name>"),
				semconv.RPCJsonrpcVersionKey.String("2.0"),
			))

			Expect(rec.AggregationKind).To(Equal(aggregation.HistogramKind))
			Expect(rec.NumberKind).To(Equal(number.Int64Kind))
			Expect(rec.Count).To(BeNumerically("==", 3))
		})

		It("does not increment the notification count", func() {
			metrics.Call(context.Background(), request)

			err := exporter.Collect(context.Background())
			Expect(err).ShouldNot(HaveOccurred())

			_, err = exporter.GetByName("rpc.server.notifications")
			Expect(err).To(MatchError("record not found"))
		})

		When("the call returns a success response", func() {
			It("does not increment the error count", func() {
				metrics.Call(context.Background(), request)

				err := exporter.Collect(context.Background())
				Expect(err).ShouldNot(HaveOccurred())

				_, err = exporter.GetByName("rpc.server.errors")
				Expect(err).To(MatchError("record not found"))
			})
		})

		When("the call returns an error response", func() {
			BeforeEach(func() {
				response = harpy.ErrorResponse{
					Version:   "2.0",
					RequestID: request.ID,
					Error: harpy.ErrorInfo{
						Code:    harpy.InternalErrorCode,
						Message: harpy.InternalErrorCode.String(),
					},
					ServerError: errors.New("<error>"),
				}
			})

			It("increments the error count", func() {
				for i := 0; i < 3; i++ {
					metrics.Call(context.Background(), request)
				}

				err := exporter.Collect(context.Background())
				Expect(err).ShouldNot(HaveOccurred())

				rec, err := exporter.GetByName("rpc.server.errors")
				Expect(err).ShouldNot(HaveOccurred())

				Expect(rec.InstrumentationLibrary).To(Equal(metrictest.Library{
					InstrumentationName:    "github.com/dogmatiq/harpy/middleware/otelharpy",
					InstrumentationVersion: "0.0.0-dev",
				}))

				Expect(rec.Attributes).To(ConsistOf(
					semconv.RPCSystemKey.String("dogmatiq/harpy"),
					semconv.RPCServiceKey.String("package.subpackage.Service"),
					semconv.RPCMethodKey.String("<method/name>"),
					semconv.RPCJsonrpcVersionKey.String("2.0"),
					semconv.RPCJsonrpcErrorCodeKey.Int(int(harpy.InternalErrorCode)),
					semconv.RPCJsonrpcErrorMessageKey.String(harpy.InternalErrorCode.String()),
				))

				Expect(rec.AggregationKind).To(Equal(aggregation.SumKind))
				Expect(rec.NumberKind).To(Equal(number.Int64Kind))
				Expect(rec.Sum).To(Equal(number.NewInt64Number(3)))
			})
		})
	})

	Describe("func Notify()", func() {
		BeforeEach(func() {
			request.ID = nil
		})

		It("forwards to the next exchanger", func() {
			called := false
			exchanger.NotifyFunc = func(
				_ context.Context,
				req harpy.Request,
			) {
				called = true
				Expect(req).To(Equal(request))
			}

			metrics.Notify(context.Background(), request)
			Expect(called).To(BeTrue())
		})

		It("increments the notifications count", func() {
			for i := 0; i < 3; i++ {
				metrics.Notify(context.Background(), request)
			}

			err := exporter.Collect(context.Background())
			Expect(err).ShouldNot(HaveOccurred())

			rec, err := exporter.GetByName("rpc.server.notifications")
			Expect(err).ShouldNot(HaveOccurred())

			Expect(rec.InstrumentationLibrary).To(Equal(metrictest.Library{
				InstrumentationName:    "github.com/dogmatiq/harpy/middleware/otelharpy",
				InstrumentationVersion: "0.0.0-dev",
			}))

			Expect(rec.Attributes).To(ConsistOf(
				semconv.RPCSystemKey.String("dogmatiq/harpy"),
				semconv.RPCServiceKey.String("package.subpackage.Service"),
				semconv.RPCMethodKey.String("<method/name>"),
				semconv.RPCJsonrpcVersionKey.String("2.0"),
			))

			Expect(rec.AggregationKind).To(Equal(aggregation.SumKind))
			Expect(rec.NumberKind).To(Equal(number.Int64Kind))
			Expect(rec.Sum).To(Equal(number.NewInt64Number(3)))
		})

		It("records the duration", func() {
			for i := 0; i < 3; i++ {
				metrics.Notify(context.Background(), request)
			}

			err := exporter.Collect(context.Background())
			Expect(err).ShouldNot(HaveOccurred())

			rec, err := exporter.GetByName("rpc.server.duration")
			Expect(err).ShouldNot(HaveOccurred())

			Expect(rec.InstrumentationLibrary).To(Equal(metrictest.Library{
				InstrumentationName:    "github.com/dogmatiq/harpy/middleware/otelharpy",
				InstrumentationVersion: "0.0.0-dev",
			}))

			Expect(rec.Attributes).To(ConsistOf(
				semconv.RPCSystemKey.String("dogmatiq/harpy"),
				semconv.RPCServiceKey.String("package.subpackage.Service"),
				semconv.RPCMethodKey.String("<method/name>"),
				semconv.RPCJsonrpcVersionKey.String("2.0"),
			))

			Expect(rec.AggregationKind).To(Equal(aggregation.HistogramKind))
			Expect(rec.NumberKind).To(Equal(number.Int64Kind))
			Expect(rec.Count).To(BeNumerically("==", 3))
		})

		It("does not increment the call count", func() {
			metrics.Notify(context.Background(), request)

			err := exporter.Collect(context.Background())
			Expect(err).ShouldNot(HaveOccurred())

			_, err = exporter.GetByName("rpc.server.calls")
			Expect(err).To(MatchError("record not found"))
		})

		It("does not increment the error count", func() {
			metrics.Notify(context.Background(), request)

			err := exporter.Collect(context.Background())
			Expect(err).ShouldNot(HaveOccurred())

			_, err = exporter.GetByName("rpc.server.errors")
			Expect(err).To(MatchError("record not found"))
		})
	})
})
