package voorhees_test

import (
	"context"
	"encoding/json"

	"github.com/dogmatiq/dodeca/logging"
	. "github.com/jmalloc/voorhees"
	. "github.com/jmalloc/voorhees/fixtures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ Exchanger = (*ExchangeLogger)(nil)

var _ = Describe("type ExchangeLogger", func() {
	var (
		next    *ExchangerStub
		request Request
		logger  *logging.BufferedLogger
		stage   *ExchangeLogger
	)

	BeforeEach(func() {
		next = &ExchangerStub{}

		request = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`123`),
			Method:     "<method>",
			Parameters: json.RawMessage(`[1, 2, 3]`),
		}

		logger = &logging.BufferedLogger{
			CaptureDebug: true,
		}

		stage = &ExchangeLogger{
			Next:   next,
			Logger: logger,
		}
	})

	Describe("func Call()", func() {
		It("passes the request to the next stage", func() {
			expect := SuccessResponse{
				Result: json.RawMessage(`"expected"`),
			}

			next.CallFunc = func(
				_ context.Context,
				req Request,
			) Response {
				Expect(req).To(Equal(req))
				return expect
			}

			res := stage.Call(context.Background(), request)
			Expect(res).To(Equal(expect))
		})

		It("logs requests that have parameters", func() {
			stage.Call(context.Background(), request)

			Expect(logger.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `▼ CALL[123] <method> WITH PARAMETERS [1, 2, 3]`,
					IsDebug: false,
				},
			))
		})

		It("logs requests that do not have parameters", func() {
			request.Parameters = nil
			stage.Call(context.Background(), request)

			Expect(logger.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `▼ CALL[123] <method> WITHOUT PARAMETERS`,
					IsDebug: false,
				},
			))
		})

		When("the next stage succeeds", func() {
			var response SuccessResponse

			BeforeEach(func() {
				response = SuccessResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`123`),
					Result:    json.RawMessage(`[4, 5, 6]`),
				}

				next.CallFunc = func(
					_ context.Context,
					req Request,
				) Response {
					return response
				}
			})

			It("logs responses that have a result", func() {
				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `▲ CALL[123] <method> SUCCESS WITH RESULT [4, 5, 6]`,
						IsDebug: false,
					},
				))
			})

			It("logs responses that do not have a result", func() {
				response.Result = nil

				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `▲ CALL[123] <method> SUCCESS WITHOUT RESULT`,
						IsDebug: false,
					},
				))
			})
		})

		When("when the next stage fails", func() {
			var response ErrorResponse

			BeforeEach(func() {
				response = ErrorResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`123`),
					Error: ErrorInfo{
						Code:    InternalErrorCode,
						Message: "<error>",
						Data:    json.RawMessage(`[7, 8, 9]`),
					},
				}

				next.CallFunc = func(
					_ context.Context,
					req Request,
				) Response {
					return response
				}
			})

			It("logs responses that have user-defined data", func() {
				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `▲ CALL[123] <method> ERROR [-32603] internal server error: <error> WITH DATA [7, 8, 9]`,
						IsDebug: false,
					},
				))
			})

			It("logs responses that do not have user-defined data", func() {
				response.Error.Data = nil

				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `▲ CALL[123] <method> ERROR [-32603] internal server error: <error> WITHOUT DATA`,
						IsDebug: false,
					},
				))
			})

			It("does not duplicate the error message if it is the same as the error code description", func() {
				response.Error.Message = response.Error.Code.String()

				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `▲ CALL[123] <method> ERROR [-32603] internal server error WITH DATA [7, 8, 9]`,
						IsDebug: false,
					},
				))
			})
		})
	})

	Describe("func Notify()", func() {
		BeforeEach(func() {
			request.ID = nil
		})

		It("passes the request to the next stage", func() {
			called := false
			next.NotifyFunc = func(
				_ context.Context,
				req Request,
			) {
				called = true
				Expect(req).To(Equal(req))
			}

			stage.Notify(context.Background(), request)
			Expect(called).To(BeTrue())
		})

		It("logs requests that have parameters", func() {
			stage.Notify(context.Background(), request)

			Expect(logger.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `▼ NOTIFY <method> WITH PARAMETERS [1, 2, 3]`,
					IsDebug: false,
				},
			))
		})

		It("logs requests that do not have parameters", func() {
			request.Parameters = nil
			stage.Notify(context.Background(), request)

			Expect(logger.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `▼ NOTIFY <method> WITHOUT PARAMETERS`,
					IsDebug: false,
				},
			))
		})
	})
})
