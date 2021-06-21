package harpy_test

import (
	"context"
	"errors"

	"github.com/dogmatiq/dodeca/logging"
	. "github.com/jmalloc/harpy"
	. "github.com/jmalloc/harpy/internal/fixtures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("func Exchange() (RequestSetReader error conditions)", func() {
	var (
		exchanger *ExchangerStub
		reader    *RequestSetReaderStub
		writer    *ResponseWriterStub
		buffer    *logging.BufferedLogger
		logger    DefaultExchangeLogger
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

		buffer = &logging.BufferedLogger{}

		logger = DefaultExchangeLogger{
			Target: buffer,
		}

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
				Expect(buffer.Messages()).To(BeEmpty())
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
				Expect(buffer.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `error: -32603 internal server error, caused by: <read error>, responded with: unable to read JSON-RPC request`,
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
				Expect(buffer.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `unable to write JSON-RPC response: <write error>`,
					},
				))
			})
		})

		When("the request data is not valid JSON", func() {
			BeforeEach(func() {
				reader.ReadFunc = func(context.Context) (RequestSet, error) {
					// This is the error expected to be returned by
					// ParseRequestSet() when non-JSON data is read.
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
				Expect(buffer.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `error: -32700 parse error`,
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
				Expect(buffer.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `unable to write JSON-RPC response: <write error>`,
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
				Expect(buffer.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `error: -32600 invalid request, responded with: non-batch request sets must contain exactly one request`,
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
			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `unable to write JSON-RPC response: <write error>`,
				},
			))
		})
	})
})
