package harpy_test

import (
	"context"
	"encoding/json"
	"errors"

	. "github.com/dogmatiq/harpy"
	. "github.com/dogmatiq/harpy/internal/fixtures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("func Exchange() (single request)", func() {
	var (
		exchanger *ExchangerStub
		request   Request
		reader    *RequestSetReaderStub
		writer    *ResponseWriterStub
		logs      *observer.ObservedLogs
		logger    ExchangeLogger
		closed    bool
	)

	BeforeEach(func() {
		exchanger = &ExchangerStub{}

		exchanger.CallFunc = func(
			_ context.Context,
			req Request,
		) Response {
			return SuccessResponse{
				Version:   "2.0",
				RequestID: req.ID,
				Result:    json.RawMessage(`"<result>"`),
			}
		}

		request = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`123`),
			Method:     "<method>",
			Parameters: json.RawMessage(`[]`),
		}

		reader = &RequestSetReaderStub{
			ReadFunc: func(context.Context) (RequestSet, error) {
				return RequestSet{
					Requests: []Request{request},
					IsBatch:  false,
				}, nil
			},
		}

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

	When("the request is a call", func() {
		It("passes the request to the exchanger and writes an unbatched response", func() {
			next := exchanger.CallFunc
			exchanger.CallFunc = func(
				ctx context.Context,
				req Request,
			) Response {
				Expect(req).To(Equal(request))
				return next(ctx, req)
			}

			writer.WriteUnbatchedFunc = func(res Response) error {
				Expect(res).To(Equal(SuccessResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`123`),
					Result:    json.RawMessage(`"<result>"`),
				}))

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
						Level:   zapcore.InfoLevel,
						Message: `call`,
					},
					Context: []zapcore.Field{
						zap.String("method", "<method>"),
						zap.Int("param_size", 2),
						zap.Int("result_size", 10),
					},
				},
			))
		})

		It("logs and returns errors the occur when writing the response", func() {
			writer.WriteUnbatchedFunc = func(Response) error {
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

	When("the request is a notification", func() {
		BeforeEach(func() {
			request.ID = nil
		})

		It("passes the request to the exchanger and does not write any responses", func() {
			called := true
			exchanger.NotifyFunc = func(
				_ context.Context,
				req Request,
			) error {
				Expect(req).To(Equal(request))
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
			Expect(called).To(BeTrue())
			Expect(logs.AllUntimed()).To(ContainElement(
				observer.LoggedEntry{
					Entry: zapcore.Entry{
						Level:   zapcore.InfoLevel,
						Message: `notify`,
					},
					Context: []zapcore.Field{
						zap.String("method", "<method>"),
						zap.Int("param_size", 2),
					},
				},
			))
		})
	})
})
