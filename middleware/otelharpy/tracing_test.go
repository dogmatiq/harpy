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
	"github.com/onsi/gomega/gstruct"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

var _ = Describe("type Tracing", func() {
	var (
		request   harpy.Request
		response  harpy.Response
		exchanger *ExchangerStub
		recorder  *tracetest.SpanRecorder
		tracing   *Tracing
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

		recorder = tracetest.NewSpanRecorder()

		tracing = &Tracing{
			Next: exchanger,
			TracerProvider: tracesdk.NewTracerProvider(
				tracesdk.WithSpanProcessor(recorder),
			),
			ServiceName:   "package.subpackage.Service",
			CreateNewSpan: true,
		}
	})

	When("configured to create new spans", func() {
		Describe("func Call()", func() {
			It("forwards to the next exchanger", func() {
				exchanger.CallFunc = func(
					_ context.Context,
					req harpy.Request,
				) harpy.Response {
					Expect(req).To(Equal(request))
					return response
				}

				res := tracing.Call(context.Background(), request)
				Expect(res).To(Equal(response))
			})

			When("the call returns a success response", func() {
				It("records a span", func() {
					tracing.Call(context.Background(), request)

					spans := recorder.Ended()
					Expect(spans).To(HaveLen(1))

					span := spans[0]

					// Note that slashes in the method name are "sanitized" to hyphens
					// as the method name must not contain a slash according to the
					// semantic conventions.
					Expect(span.Name()).To(Equal("package.subpackage.Service/<method-name>"))

					Expect(span.SpanKind()).To(Equal(trace.SpanKindServer))

					// Note that the method name attribute is NOT sanitized, so that we
					// can always see the real method name.
					Expect(span.Attributes()).To(ConsistOf(
						semconv.RPCSystemKey.String("dogmatiq/harpy"),
						semconv.RPCServiceKey.String("package.subpackage.Service"),
						semconv.RPCMethodKey.String("<method/name>"),
						semconv.RPCJsonrpcVersionKey.String("2.0"),
						semconv.RPCJsonrpcRequestIDKey.String("123"),
					))

					Expect(span.Status()).To(Equal(
						tracesdk.Status{
							Code: codes.Ok,
						},
					))

					Expect(span.InstrumentationScope()).To(Equal(
						instrumentation.Scope{
							Name:    "github.com/dogmatiq/harpy/middleware/otelharpy",
							Version: "0.0.0-dev",
						},
					))
				})

				It("uses an empty request ID attribute if the request ID is null", func() {
					request.ID = json.RawMessage(`NULL`)

					tracing.Call(context.Background(), request)

					spans := recorder.Ended()
					Expect(spans).To(HaveLen(1))

					span := spans[0]

					Expect(span.Attributes()).To(ContainElement(
						semconv.RPCJsonrpcRequestIDKey.String(""),
					))
				})

				It("trims quotes from the request ID attribute when the request ID is a strings", func() {
					request.ID = json.RawMessage(`"<id>"`)

					tracing.Call(context.Background(), request)

					spans := recorder.Ended()
					Expect(spans).To(HaveLen(1))

					span := spans[0]

					Expect(span.Attributes()).To(ContainElement(
						semconv.RPCJsonrpcRequestIDKey.String("<id>"),
					))
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

				It("includes error information in the span", func() {
					tracing.Call(context.Background(), request)

					spans := recorder.Ended()
					Expect(spans).To(HaveLen(1))

					span := spans[0]

					// The message key contains the client-facing error message.
					Expect(span.Attributes()).To(ContainElements(
						semconv.RPCJsonrpcErrorCodeKey.Int(int(harpy.InternalErrorCode)),
						semconv.RPCJsonrpcErrorMessageKey.String(harpy.InternalErrorCode.String()),
					))

					// The status contains the causal error.
					Expect(span.Status()).To(Equal(
						tracesdk.Status{
							Code:        codes.Error,
							Description: "<error>",
						},
					))

					Expect(span.Events()).To(ConsistOf(
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Name": Equal("exception"),
							"Attributes": ConsistOf(
								semconv.ExceptionTypeKey.String("*errors.errorString"),
								semconv.ExceptionMessageKey.String("<error>"),
							),
						}),
					))
				})

				It("uses the client-facing error message in the status if there is no ServerError", func() {
					response = harpy.ErrorResponse{
						Version:   "2.0",
						RequestID: request.ID,
						Error: harpy.ErrorInfo{
							Code:    harpy.InternalErrorCode,
							Message: harpy.InternalErrorCode.String(),
						},
					}

					tracing.Call(context.Background(), request)

					spans := recorder.Ended()
					Expect(spans).To(HaveLen(1))

					span := spans[0]

					Expect(span.Status()).To(Equal(
						tracesdk.Status{
							Code:        codes.Error,
							Description: harpy.InternalErrorCode.String(),
						},
					))

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
				) error {
					called = true
					Expect(req).To(Equal(request))
					return nil
				}

				tracing.Notify(context.Background(), request)
				Expect(called).To(BeTrue())
			})

			It("records a span", func() {
				tracing.Notify(context.Background(), request)

				spans := recorder.Ended()
				Expect(spans).To(HaveLen(1))

				span := spans[0]

				// Note that slashes in the method name are "sanitized" to hyphens
				// as the method name must not contain a slash according to the
				// semantic conventions.
				Expect(span.Name()).To(Equal("package.subpackage.Service/<method-name>"))

				Expect(span.SpanKind()).To(Equal(trace.SpanKindServer))

				// Note that the method name attribute is NOT sanitized, so that we
				// can always see the real method name.
				Expect(span.Attributes()).To(ConsistOf(
					semconv.RPCSystemKey.String("dogmatiq/harpy"),
					semconv.RPCServiceKey.String("package.subpackage.Service"),
					semconv.RPCMethodKey.String("<method/name>"),
					semconv.RPCJsonrpcVersionKey.String("2.0"),
				))

				Expect(span.Status()).To(Equal(
					tracesdk.Status{
						Code: codes.Ok,
					},
				))

				Expect(span.InstrumentationScope()).To(Equal(
					instrumentation.Scope{
						Name:    "github.com/dogmatiq/harpy/middleware/otelharpy",
						Version: "0.0.0-dev",
					},
				))
			})

			When("the notification returns an error", func() {
				BeforeEach(func() {
					exchanger.NotifyFunc = func(
						_ context.Context,
						_ harpy.Request,
					) error {
						return errors.New("<error>")
					}
				})

				It("includes error information in the span", func() {
					tracing.Notify(context.Background(), request)

					spans := recorder.Ended()
					Expect(spans).To(HaveLen(1))

					span := spans[0]

					Expect(span.Status()).To(Equal(
						tracesdk.Status{
							Code:        codes.Error,
							Description: "<error>",
						},
					))

					Expect(span.Events()).To(ConsistOf(
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Name": Equal("exception"),
							"Attributes": ConsistOf(
								semconv.ExceptionTypeKey.String("*errors.errorString"),
								semconv.ExceptionMessageKey.String("<error>"),
							),
						}),
					))
				})
			})
		})
	})

	When("configured to modify an existing span", func() {
		var tracer trace.Tracer

		BeforeEach(func() {
			tracer = tracing.TracerProvider.Tracer("test")
			tracing.CreateNewSpan = false
		})

		Describe("func Call()", func() {
			It("modifies the existing span", func() {
				ctx, outerSpan := tracer.Start(context.Background(), "<span>")
				defer outerSpan.End()

				tracing.Call(ctx, request)

				span := outerSpan.(tracesdk.ReadOnlySpan)

				Expect(span.Name()).To(Equal("package.subpackage.Service/<method-name>"))
				Expect(span.Attributes()).To(ConsistOf(
					semconv.RPCSystemKey.String("dogmatiq/harpy"),
					semconv.RPCServiceKey.String("package.subpackage.Service"),
					semconv.RPCMethodKey.String("<method/name>"),
					semconv.RPCJsonrpcVersionKey.String("2.0"),
					semconv.RPCJsonrpcRequestIDKey.String("123"),
				))
				Expect(span.Status()).To(Equal(
					tracesdk.Status{
						Code: codes.Ok,
					},
				))
			})
		})

		Describe("func Notify()", func() {
			It("modifies the existing span", func() {
				ctx, outerSpan := tracer.Start(context.Background(), "<span>")
				defer outerSpan.End()

				tracing.Notify(ctx, request)

				span := outerSpan.(tracesdk.ReadOnlySpan)

				Expect(span.Name()).To(Equal("package.subpackage.Service/<method-name>"))
				Expect(span.Attributes()).To(ConsistOf(
					semconv.RPCSystemKey.String("dogmatiq/harpy"),
					semconv.RPCServiceKey.String("package.subpackage.Service"),
					semconv.RPCMethodKey.String("<method/name>"),
					semconv.RPCJsonrpcVersionKey.String("2.0"),
					semconv.RPCJsonrpcRequestIDKey.String("123"),
				))
				Expect(span.Status()).To(Equal(
					tracesdk.Status{
						Code: codes.Ok,
					},
				))
			})
		})
	})
})
