package harpy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dogmatiq/harpy"
	. "github.com/dogmatiq/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type stubIDGenerator struct {
	CallNewIDs    func(ctx context.Context) (oteltrace.TraceID, oteltrace.SpanID)
	CallNewSpanID func(ctx context.Context, traceID oteltrace.TraceID) oteltrace.SpanID
}

func (g *stubIDGenerator) NewIDs(ctx context.Context) (oteltrace.TraceID, oteltrace.SpanID) {
	if g.CallNewIDs == nil {
		return [16]byte{}, [8]byte{}
	}

	return g.CallNewIDs(ctx)
}

func (g *stubIDGenerator) NewSpanID(ctx context.Context, traceID oteltrace.TraceID) oteltrace.SpanID {
	if g.CallNewSpanID == nil {
		return [8]byte{}
	}

	return g.CallNewSpanID(ctx, traceID)
}

var _ = Context("type structuredExchangeLogger", func() {
	var (
		ctx                           context.Context
		request                       harpy.Request
		success                       harpy.SuccessResponse
		nativeError                   harpy.ErrorResponse
		nativeErrorNonStandardMessage harpy.ErrorResponse
		nonNativeError                harpy.ErrorResponse
		buffer                        bytes.Buffer
		logger                        ExchangeLogger
		stubIDGen                     *stubIDGenerator
		tracer                        oteltrace.Tracer
	)

	BeforeEach(func() {
		ctx = context.Background()

		exporter, err := stdouttrace.New()
		Expect(err).NotTo(HaveOccurred())

		stubIDGen = &stubIDGenerator{}
		traceID, err := oteltrace.TraceIDFromHex("01020304050607080102040810203040")
		Expect(err).NotTo(HaveOccurred())

		stubIDGen.CallNewIDs = func(ctx context.Context) (oteltrace.TraceID, oteltrace.SpanID) {
			var spanID [8]byte
			return traceID, spanID
		}

		tracer = trace.NewTracerProvider(
			trace.WithIDGenerator(stubIDGen),
			trace.WithBatcher(exporter),
		).Tracer("<tracer>")

		request = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`123`),
			Method:     "method",
			Parameters: json.RawMessage(`[1, 2, 3]`),
		}

		success = harpy.NewSuccessResponse(request.ID, 123).(harpy.SuccessResponse)
		nativeError = harpy.NewErrorResponse(request.ID, MethodNotFound())
		nativeErrorNonStandardMessage = harpy.NewErrorResponse(request.ID, MethodNotFound(WithMessage("<message>")))
		nonNativeError = harpy.NewErrorResponse(request.ID, errors.New("<error>"))

		buffer.Reset()

		logger = NewZapExchangeLogger(
			zap.New(
				zapcore.NewCore(
					zapcore.NewConsoleEncoder(
						zap.NewDevelopmentEncoderConfig(),
					),
					zapcore.AddSync(&buffer),
					zapcore.DebugLevel,
				),
			),
		)
	})

	Describe("func LogError()", func() {
		It("logs details of a native error response", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			logger.LogError(ctx, nativeError)

			substr := fmt.Sprintf(
				`error	{"error_code": -32601, "error": "method not found", "trace_id": "%s"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})

		It("logs details of a native error response with a non-standard message", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			logger.LogError(ctx, nativeErrorNonStandardMessage)

			substr := fmt.Sprintf(
				`error	{"error_code": -32601, "error": "method not found", "trace_id": "%s", "responded_with": "<message>"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})

		It("logs details of a non-native causal error", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			logger.LogError(ctx, nonNativeError)

			substr := fmt.Sprintf(
				`error	{"error_code": -32603, "error": "internal server error", "trace_id": "%s", "caused_by": "<error>"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})
	})

	Describe("func LogNotification()", func() {
		It("logs the request information", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			request.ID = nil
			logger.LogNotification(ctx, request)

			substr := fmt.Sprintf(
				`notify method	{"param_size": 9, "trace_id": "%s"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})

		It("quotes empty method names", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			request.ID = nil
			request.Method = ""
			logger.LogNotification(ctx, request)

			substr := fmt.Sprintf(
				`notify ""	{"param_size": 9, "trace_id": "%s"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})

		It("quotes and escapes methods names that contain whitespace and non-printable characters", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			request.ID = nil
			request.Method = "<the method>\x00"
			logger.LogNotification(ctx, request)

			substr := fmt.Sprintf(
				`notify "<the method>\x00"	{"param_size": 9, "trace_id": "%s"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})
	})

	Describe("func LogCall()", func() {
		It("logs the request and response information", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			logger.LogCall(ctx, request, success)

			substr := fmt.Sprintf(
				`call method	{"param_size": 9, "trace_id": "%s", "result_size": 3}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})

		It("quotes empty method names", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			request.Method = ""
			logger.LogCall(ctx, request, success)

			substr := fmt.Sprintf(
				`call ""	{"param_size": 9, "trace_id": "%s", "result_size": 3}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})

		It("quotes and escapes methods names that contain whitespace and non-printable characters", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			request.Method = "<the method>\x00"
			logger.LogCall(ctx, request, success)

			substr := fmt.Sprintf(
				`call "<the method>\x00"	{"param_size": 9, "trace_id": "%s", "result_size": 3}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})

		It("logs details of a native error response", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			logger.LogCall(ctx, request, nativeError)

			substr := fmt.Sprintf(
				`call method	{"param_size": 9, "trace_id": "%s", "error_code": -32601, "error": "method not found"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})

		It("logs details of a native error response with a non-standard message", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			logger.LogCall(ctx, request, nativeErrorNonStandardMessage)

			substr := fmt.Sprintf(
				`call method	{"param_size": 9, "trace_id": "%s", "error_code": -32601, "error": "method not found", "responded_with": "<message>"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})

		It("logs details of a non-native causal error", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			logger.LogCall(ctx, request, nonNativeError)

			substr := fmt.Sprintf(
				`call method	{"param_size": 9, "trace_id": "%s", "error_code": -32603, "error": "internal server error", "caused_by": "<error>"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})
	})

	Describe("func LogWriterError()", func() {
		It("logs the error", func() {
			ctx, span := tracer.Start(ctx, "<span>")
			defer span.End()

			logger.LogWriterError(ctx, errors.New("<error>"))

			substr := fmt.Sprintf(
				`unable to write JSON-RPC response	{"error": "<error>", "trace_id": "%s"}`,
				"01020304050607080102040810203040",
			)
			Expect(buffer.String()).To(
				ContainSubstring(substr),
			)
		})
	})
})
