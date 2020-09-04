package voorhees_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dogmatiq/dodeca/logging"
	. "github.com/jmalloc/voorhees"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ PipelineStage = (*HandlerInvoker)(nil)

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
			It("logs the invocation", func() {
				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `✓ CALL[123] <method>`,
						IsDebug: false,
					},
				))
			})

			When("the result is nil", func() {
				It("returns a success response", func() {
					res := stage.Call(context.Background(), request)
					Expect(res).To(Equal(SuccessResponse{
						Version:   `2.0`,
						RequestID: json.RawMessage(`123`),
					}))
				})
			})

			When("the response is non-nil", func() {
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

				It("returns a success response that contains the marshaled result", func() {
					res := stage.Call(context.Background(), request)
					Expect(res).To(Equal(SuccessResponse{
						Version:   `2.0`,
						RequestID: json.RawMessage(`123`),
						Result:    json.RawMessage(`456`),
					}))
				})

				When("the result can not be marshaled", func() {
					BeforeEach(func() {
						result = 10i + 1
					})

					It("returns an error response", func() {
						res := stage.Call(context.Background(), request)
						Expect(res).To(Equal(ErrorResponse{
							Version:   `2.0`,
							RequestID: json.RawMessage(`123`),
							Error: ErrorInfo{
								Code:    InternalErrorCode,
								Message: "internal server error",
							},
						}))
					})

					It("logs the invocation", func() {
						stage.Call(context.Background(), request)

						Expect(logger.Messages()).To(ContainElement(
							logging.BufferedLogMessage{
								Message: `✗ CALL[123] <method>  [-32603] internal server error: handler succeeded but the result could not be marshaled: json: unsupported type: complex128  (cause not shown to caller)`,
								IsDebug: false,
							},
						))
					})
				})
			})
		})

		When("the handler returns a native JSON-RPC error", func() {
			var err Error

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

			When("the error does not contain any user-defined data", func() {
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
			})

			When("the error contains user-defined data", func() {
				BeforeEach(func() {
					err = NewError(
						789,
						WithMessage("<error>"),
						WithData([]int{100, 200, 300}),
					)
				})

				It("returns an error response containing the user-defined data", func() {
					res := stage.Call(context.Background(), request)
					Expect(res).To(Equal(ErrorResponse{
						Version:   `2.0`,
						RequestID: json.RawMessage(`123`),
						Error: ErrorInfo{
							Code:    789,
							Message: "<error>",
							Data:    json.RawMessage(`[100,200,300]`),
						},
					}))
				})

				When("the data can not be marshaled", func() {
					BeforeEach(func() {
						err = NewError(
							789,
							WithMessage("<error>"),
							WithData(10i+1),
						)
					})

					It("returns an error response", func() {
						res := stage.Call(context.Background(), request)
						Expect(res).To(Equal(ErrorResponse{
							Version:   `2.0`,
							RequestID: json.RawMessage(`123`),
							Error: ErrorInfo{
								Code:    InternalErrorCode,
								Message: "internal server error",
							},
						}))
					})

					It("logs the invocation", func() {
						stage.Call(context.Background(), request)

						Expect(logger.Messages()).To(ContainElement(
							logging.BufferedLogMessage{
								Message: `✗ CALL[123] <method>  [-32603] internal server error: handler failed ([789] <error>), but the user-defined error data could not be marshaled: json: unsupported type: complex128  (cause not shown to caller)`,
								IsDebug: false,
							},
						))
					})
				})
			})
		})

		When("the handler returns a context.DeadlineExceeded error", func() {
			BeforeEach(func() {
				stage.Handler = func(
					context.Context,
					Request,
				) (interface{}, error) {
					return nil, context.DeadlineExceeded
				}
			})

			It("logs the invocation", func() {
				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `✗ CALL[123] <method>  [-32603] internal server error: context deadline exceeded`,
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
						Code:    InternalErrorCode,
						Message: "context deadline exceeded",
					},
				}))
			})
		})

		When("the handler returns a context.Canceled error", func() {
			BeforeEach(func() {
				stage.Handler = func(
					context.Context,
					Request,
				) (interface{}, error) {
					return nil, context.Canceled
				}
			})

			It("logs the invocation", func() {
				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `✗ CALL[123] <method>  [-32603] internal server error: context canceled`,
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
						Code:    InternalErrorCode,
						Message: "context canceled",
					},
				}))
			})
		})

		When("the handler returns an unrecognised error", func() {
			BeforeEach(func() {
				stage.Handler = func(
					context.Context,
					Request,
				) (interface{}, error) {
					return nil, errors.New("<error>")
				}
			})

			It("logs the invocation", func() {
				stage.Call(context.Background(), request)

				Expect(logger.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `✗ CALL[123] <method>  [-32603] internal server error: <error>  (cause not shown to caller)`,
						IsDebug: false,
					},
				))
			})

			It("returns an error response that does NOT include the causal error message", func() {
				res := stage.Call(context.Background(), request)
				Expect(res).To(Equal(ErrorResponse{
					Version:   `2.0`,
					RequestID: json.RawMessage(`123`),
					Error: ErrorInfo{
						Code:    InternalErrorCode,
						Message: "internal server error",
					},
				}))
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
