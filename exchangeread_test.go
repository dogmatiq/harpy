package harpy_test

import (
	"context"
	"errors"

	. "github.com/dogmatiq/harpy"
	. "github.com/dogmatiq/harpy/internal/fixtures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("func Exchange() (RequestSetReader error conditions)", func() {
	var (
		exchanger *ExchangerStub
		reader    *RequestSetReaderStub
		writer    *ResponseWriterStub
		logs      *observer.ObservedLogs
		logger    ExchangeLogger
		closed    bool
	)

	BeforeEach(func() {
		exchanger = &ExchangerStub{}

		reader = &RequestSetReaderStub{}

		writer = &ResponseWriterStub{
			WriteErrorFunc: func(ErrorResponse) error {
				panic("unexpected call to WriteErrorFunc()")
			},
			WriteUnbatchedFunc: func(Response) error {
				panic("unexpected call to WriteUnbatchedFunc()")
			},
			WriteBatchedFunc: func(Response) error {
				panic("unexpected call to WriteBatchedFunc()")
			},
			CloseFunc: func() error {
				Expect(closed).To(BeFalse(), "response writer was closed multiple times")
				closed = true
				return nil
			},
		}

		var core zapcore.Core
		core, logs = observer.New(zapcore.DebugLevel)
		logger = NewZapExchangeLogger(zap.New(core))

		closed = false
	})

	AfterEach(func() {
		// The response writer must always be closed.
		Expect(closed).To(BeTrue())
	})

	When("reading the next request set", func() {
		When("the context is canceled", func() {
			var ctx context.Context

			BeforeEach(func() {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel()

				reader.ReadFunc = func(ctx context.Context) (RequestSet, error) {
					return RequestSet{}, ctx.Err()
				}
			})

			It("returns the error without writing a response or logging", func() {
				err := Exchange(
					ctx,
					exchanger,
					reader,
					writer,
					logger,
				)

				Expect(err).To(Equal(ctx.Err()))
				Expect(logs.AllUntimed()).To(BeEmpty())
			})
		})

		When("there is an IO error", func() {
			readError := errors.New("<read error>")

			BeforeEach(func() {
				reader.ReadFunc = func(context.Context) (RequestSet, error) {
					return RequestSet{}, readError
				}
			})

			It("writes and logs an error response and returns the IO error", func() {
				writer.WriteErrorFunc = func(
					res ErrorResponse,
				) error {
					Expect(res).To(Equal(
						ErrorResponse{
							Version:   "2.0",
							RequestID: nil,
							Error: ErrorInfo{
								Code:    InternalErrorCode,
								Message: `unable to read JSON-RPC request`,
							},
							ServerError: readError,
						},
					))

					return nil
				}

				err := Exchange(
					context.Background(),
					exchanger,
					reader,
					writer,
					logger,
				)

				Expect(err).To(MatchError("<read error>"))
				Expect(logs.AllUntimed()).To(ContainElement(
					observer.LoggedEntry{
						Entry: zapcore.Entry{
							Level:   zapcore.ErrorLevel,
							Message: `error`,
						},
						Context: []zapcore.Field{
							zap.Int("error_code", int(InternalErrorCode)),
							zap.String("error", "internal server error"),
							zap.String("caused_by", "<read error>"),
							zap.String("responded_with", "unable to read JSON-RPC request"),
						},
					},
				))
			})

			It("logs errors that occur while writing the response", func() {
				writer.WriteErrorFunc = func(
					res ErrorResponse,
				) error {
					return errors.New("<write error>")
				}

				err := Exchange(
					context.Background(),
					exchanger,
					reader,
					writer,
					logger,
				)

				Expect(err).To(MatchError("<read error>")) // note: still returns the original read error
				Expect(logs.AllUntimed()).To(ContainElement(
					observer.LoggedEntry{
						Entry: zapcore.Entry{
							Level:   zapcore.ErrorLevel,
							Message: `unable to write JSON-RPC response`,
						},
						Context: []zapcore.Field{
							zap.String("error", "<write error>"),
						},
					},
				))
			})
		})

		When("the request data is not valid JSON", func() {
			BeforeEach(func() {
				reader.ReadFunc = func(context.Context) (RequestSet, error) {
					// This is the error expected to be returned by
					// UnmarshalRequestSet() when non-JSON data is read.
					return RequestSet{}, NewErrorWithReservedCode(ParseErrorCode)
				}
			})

			It("writes and logs an error response", func() {
				writer.WriteErrorFunc = func(
					res ErrorResponse,
				) error {
					Expect(res).To(Equal(
						ErrorResponse{
							Version:   "2.0",
							RequestID: nil,
							Error: ErrorInfo{
								Code:    ParseErrorCode,
								Message: ParseErrorCode.String(),
							},
						},
					))

					return nil
				}

				err := Exchange(
					context.Background(),
					exchanger,
					reader,
					writer,
					logger,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(logs.AllUntimed()).To(ContainElement(
					observer.LoggedEntry{
						Entry: zapcore.Entry{
							Level:   zapcore.ErrorLevel,
							Message: `error`,
						},
						Context: []zapcore.Field{
							zap.Int("error_code", int(ParseErrorCode)),
							zap.String("error", "parse error"),
						},
					},
				))
			})

			It("logs and returns errors occur while writing the response", func() {
				writer.WriteErrorFunc = func(
					res ErrorResponse,
				) error {
					return errors.New("<write error>")
				}

				err := Exchange(
					context.Background(),
					exchanger,
					reader,
					writer,
					logger,
				)

				Expect(err).To(MatchError("<write error>"))
				Expect(logs.AllUntimed()).To(ContainElement(
					observer.LoggedEntry{
						Entry: zapcore.Entry{
							Level:   zapcore.ErrorLevel,
							Message: `unable to write JSON-RPC response`,
						},
						Context: []zapcore.Field{
							zap.String("error", "<write error>"),
						},
					},
				))
			})
		})

		When("the request set is well-formed JSON but invalid", func() {
			BeforeEach(func() {
				reader.ReadFunc = func(context.Context) (RequestSet, error) {
					return RequestSet{}, nil
				}
			})

			It("writes and logs an error response", func() {
				writer.WriteErrorFunc = func(
					res ErrorResponse,
				) error {
					Expect(res).To(Equal(
						ErrorResponse{
							Version:   "2.0",
							RequestID: nil,
							Error: ErrorInfo{
								Code:    InvalidRequestCode,
								Message: `non-batch request sets must contain exactly one request`,
							},
						},
					))

					return nil
				}

				err := Exchange(
					context.Background(),
					exchanger,
					reader,
					writer,
					logger,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(logs.AllUntimed()).To(ContainElement(
					observer.LoggedEntry{
						Entry: zapcore.Entry{
							Level:   zapcore.ErrorLevel,
							Message: `error`,
						},
						Context: []zapcore.Field{
							zap.Int("error_code", int(InvalidRequestCode)),
							zap.String("error", "invalid request"),
							zap.String("responded_with", "non-batch request sets must contain exactly one request"),
						},
					},
				))
			})
		})

		It("logs and returns errors occur while writing the response", func() {
			writer.WriteErrorFunc = func(
				res ErrorResponse,
			) error {
				return errors.New("<write error>")
			}

			err := Exchange(
				context.Background(),
				exchanger,
				reader,
				writer,
				logger,
			)

			Expect(err).To(MatchError("<write error>"))
			Expect(logs.AllUntimed()).To(ContainElement(
				observer.LoggedEntry{
					Entry: zapcore.Entry{
						Level:   zapcore.ErrorLevel,
						Message: `unable to write JSON-RPC response`,
					},
					Context: []zapcore.Field{
						zap.String("error", "<write error>"),
					},
				},
			))
		})
	})
})
