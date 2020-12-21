package harpy_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dogmatiq/dodeca/logging"
	. "github.com/jmalloc/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ Exchanger = (*HandlerInvoker)(nil)

var _ = Describe("type HandlerInvoker", func() {
	var (
		request Request
		logger  *logging.BufferedLogger
		stage   *HandlerInvoker
	)

	BeforeEach(func() {
		request = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`123`),
			Method:     "<method>",
			Parameters: json.RawMessage(`[1, 2, 3]`),
		}

		logger = &logging.BufferedLogger{
			CaptureDebug: true,
		}

		stage = &HandlerInvoker{
			Handler: func(context.Context, Request) (interface{}, error) {
				return nil, nil
			},
			Logger: logger,
		}
	})

	Describe("func Call()", func() {
		It("passes the request to the handler function", func() {
			called := false

			stage.Handler = func(
				_ context.Context,
				req Request,
			) (interface{}, error) {
				called = true
				Expect(req).To(Equal(req))
				return nil, nil
			}

			stage.Call(context.Background(), request)
			Expect(called).To(BeTrue())
		})

		When("the handler succeeds", func() {
			var result interface{}

			BeforeEach(func() {
				result = 456

				stage.Handler = func(
					context.Context,
					Request,
				) (interface{}, error) {
					return result, nil
				}
			})

			It("logs the invocation", func() {
				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `✓ CALL[123] <method>`,
						IsDebug: false,
					},
				))
			})

			It("returns a success response that contains the marshaled result", func() {
				res := stage.Call(context.Background(), request)
				Expect(res).To(Equal(SuccessResponse{
					Version:   `2.0`,
					RequestID: json.RawMessage(`123`),
					Result:    json.RawMessage(`456`),
				}))
			})
		})

		When("the handler returns an error", func() {
			var err error

			BeforeEach(func() {
				err = NewError(789, WithMessage("<error>"))

				stage.Handler = func(
					context.Context,
					Request,
				) (interface{}, error) {
					return nil, err
				}
			})

			It("logs the invocation", func() {
				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `✗ CALL[123] <method>  [789] <error>`,
						IsDebug: false,
					},
				))
			})

			It("returns an error response", func() {
				res := stage.Call(context.Background(), request)
				Expect(res).To(Equal(ErrorResponse{
					Version:   `2.0`,
					RequestID: json.RawMessage(`123`),
					Error: ErrorInfo{
						Code:    789,
						Message: "<error>",
					},
				}))
			})

			When("the error is unrecognised", func() {
				BeforeEach(func() {
					err = errors.New("<error>")
				})

				It("logs the cause of the internal error", func() {
					stage.Call(context.Background(), request)

					Expect(logger.Messages()).To(ContainElement(
						logging.BufferedLogMessage{
							Message: `✗ CALL[123] <method>  [-32603] internal server error  [cause: <error>]`,
							IsDebug: false,
						},
					))
				})
			})
		})
	})

	Describe("func Notify()", func() {
		BeforeEach(func() {
			request.ID = nil
		})

		It("passes the request to the handler", func() {
			called := false

			stage.Handler = func(
				_ context.Context,
				req Request,
			) (interface{}, error) {
				called = true
				Expect(req).To(Equal(req))
				return nil, nil
			}

			stage.Notify(context.Background(), request)
			Expect(called).To(BeTrue())
		})

		It("logs the invocation when the handler succeeds", func() {
			stage.Notify(context.Background(), request)

			Expect(logger.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `✓ NOTIFY <method>`,
					IsDebug: false,
				},
			))
		})

		It("logs the invocation when the handler fails", func() {
			stage.Handler = func(
				_ context.Context,
				req Request,
			) (interface{}, error) {
				return nil, errors.New("<error>")
			}

			stage.Notify(context.Background(), request)

			Expect(logger.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `✗ NOTIFY <method>  <error>`,
					IsDebug: false,
				},
			))
		})
	})
})
