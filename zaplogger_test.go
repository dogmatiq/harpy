package harpy_test

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/dogmatiq/harpy"
	. "github.com/dogmatiq/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ = Context("type ZapExchangeLogger", func() {
	var (
		request                       harpy.Request
		success                       harpy.SuccessResponse
		nativeError                   harpy.ErrorResponse
		nativeErrorNonStandardMessage harpy.ErrorResponse
		nonNativeError                harpy.ErrorResponse
		buffer                        bytes.Buffer
		logger                        ZapExchangeLogger
	)

	BeforeEach(func() {
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

		logger = ZapExchangeLogger{
			Target: zap.New(
				zapcore.NewCore(
					zapcore.NewConsoleEncoder(
						zap.NewDevelopmentEncoderConfig(),
					),
					zapcore.AddSync(&buffer),
					zapcore.DebugLevel,
				),
			),
		}
	})

	Describe("func LogError()", func() {
		It("logs details of a native error response", func() {
			logger.LogError(nativeError)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`error	{"error_code": -32601, "error": "method not found"}`,
				),
			)
		})

		It("logs details of a native error response with a non-standard message", func() {
			logger.LogError(nativeErrorNonStandardMessage)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`error	{"error_code": -32601, "error": "method not found", "responded_with": "<message>"}`,
				),
			)
		})

		It("logs details of a non-native causal error", func() {
			logger.LogError(nonNativeError)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`error	{"error_code": -32603, "error": "internal server error", "caused_by": "<error>"}`,
				),
			)
		})
	})

	Describe("func LogNotification()", func() {
		It("logs the request information", func() {
			request.ID = nil
			logger.LogNotification(request)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`notify method	{"param_size": 9}`,
				),
			)
		})

		It("quotes empty method names", func() {
			request.ID = nil
			request.Method = ""
			logger.LogNotification(request)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`notify ""	{"param_size": 9}`,
				),
			)
		})

		It("quotes and escapes methods names that contain whitespace and non-printable characters", func() {
			request.ID = nil
			request.Method = "<the method>\x00"
			logger.LogNotification(request)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`notify "<the method>\x00"	{"param_size": 9}`,
				),
			)
		})
	})

	Describe("func LogCall()", func() {
		It("logs the request and response information", func() {
			logger.LogCall(request, success)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`call method	{"param_size": 9, "result_size": 3}`,
				),
			)
		})

		It("quotes empty method names", func() {
			request.Method = ""
			logger.LogCall(request, success)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`call ""	{"param_size": 9, "result_size": 3}`,
				),
			)
		})

		It("quotes and escapes methods names that contain whitespace and non-printable characters", func() {
			request.Method = "<the method>\x00"
			logger.LogCall(request, success)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`call "<the method>\x00"	{"param_size": 9, "result_size": 3}`,
				),
			)
		})

		It("logs details of a native error response", func() {
			logger.LogCall(request, nativeError)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`call method	{"param_size": 9, "error_code": -32601, "error": "method not found"}`,
				),
			)
		})

		It("logs details of a native error response with a non-standard message", func() {
			logger.LogCall(request, nativeErrorNonStandardMessage)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`call method	{"param_size": 9, "error_code": -32601, "error": "method not found", "responded_with": "<message>"}`,
				),
			)
		})

		It("logs details of a non-native causal error", func() {
			logger.LogCall(request, nonNativeError)
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`call method	{"param_size": 9, "error_code": -32603, "error": "internal server error", "caused_by": "<error>"}`,
				),
			)
		})
	})

	Describe("func LogWriterError()", func() {
		It("logs the error", func() {
			logger.LogWriterError(errors.New("<error>"))
			logger.Target.Sync()

			Expect(buffer.String()).To(
				ContainSubstring(
					`unable to write JSON-RPC response	{"error": "<error>"}`,
				),
			)
		})
	})
})
