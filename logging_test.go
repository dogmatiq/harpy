package harpy_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/harpy"
	. "github.com/dogmatiq/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Context("type DefaultExchangeLogger", func() {
	var (
		ctx                           context.Context
		request                       harpy.Request
		success                       harpy.SuccessResponse
		nativeError                   harpy.ErrorResponse
		nativeErrorNonStandardMessage harpy.ErrorResponse
		nonNativeError                harpy.ErrorResponse
		buffer                        *logging.BufferedLogger
		logger                        DefaultExchangeLogger
	)

	BeforeEach(func() {
		ctx = context.Background()

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

		buffer = &logging.BufferedLogger{
			CaptureDebug: true,
		}

		logger = DefaultExchangeLogger{
			Target: buffer,
		}
	})

	Describe("func LogError()", func() {
		It("logs details of a native error response", func() {
			logger.LogError(ctx, nativeError)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `error: -32601 method not found`,
					IsDebug: false,
				},
			))
		})

		It("logs details of a native error response with a non-standard message", func() {
			logger.LogError(ctx, nativeErrorNonStandardMessage)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `error: -32601 method not found, responded with: <message>`,
					IsDebug: false,
				},
			))
		})

		It("logs details of a non-native causal error", func() {
			logger.LogError(ctx, nonNativeError)
			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `error: -32603 internal server error, caused by: <error>`,
					IsDebug: false,
				},
			))
		})
	})

	Describe("func LogNotification()", func() {
		It("logs the request information", func() {
			request.ID = nil
			logger.LogNotification(ctx, request)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `notify method [params: 9 B]`,
					IsDebug: false,
				},
			))
		})

		It("quotes empty method names", func() {
			request.ID = nil
			request.Method = ""
			logger.LogNotification(ctx, request)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `notify "" [params: 9 B]`,
					IsDebug: false,
				},
			))
		})

		It("quotes and escapes methods names that contain whitespace and non-printable characters", func() {
			request.ID = nil
			request.Method = "<the method>\x00"
			logger.LogNotification(ctx, request)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `notify "<the method>\x00" [params: 9 B]`,
					IsDebug: false,
				},
			))
		})
	})

	Describe("func LogCall()", func() {
		It("logs the request and response information", func() {
			logger.LogCall(ctx, request, success)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `call method [params: 9 B, result: 3 B]`,
					IsDebug: false,
				},
			))
		})

		It("quotes empty method names", func() {
			request.Method = ""
			logger.LogCall(ctx, request, success)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `call "" [params: 9 B, result: 3 B]`,
					IsDebug: false,
				},
			))
		})

		It("quotes and escapes methods names that contain whitespace and non-printable characters", func() {
			request.Method = "<the method>\x00"
			logger.LogCall(ctx, request, success)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `call "<the method>\x00" [params: 9 B, result: 3 B]`,
					IsDebug: false,
				},
			))
		})

		It("logs details of a native error response", func() {
			logger.LogCall(ctx, request, nativeError)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `call method [params: 9 B, error: -32601 method not found]`,
					IsDebug: false,
				},
			))
		})

		It("logs details of a native error response with a non-standard message", func() {
			logger.LogCall(ctx, request, nativeErrorNonStandardMessage)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `call method [params: 9 B, error: -32601 method not found, responded with: <message>]`,
					IsDebug: false,
				},
			))
		})

		It("logs details of a non-native causal error", func() {
			logger.LogCall(ctx, request, nonNativeError)

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `call method [params: 9 B, error: -32603 internal server error, caused by: <error>]`,
					IsDebug: false,
				},
			))
		})
	})

	Describe("func LogWriterError()", func() {
		It("logs the error", func() {
			logger.LogWriterError(ctx, errors.New("<error>"))

			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `unable to write JSON-RPC response: <error>`,
					IsDebug: false,
				},
			))
		})
	})
})
