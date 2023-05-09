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

var _ = Describe("func Exchange()", func() {
	var (
		exchanger *ExchangerStub
		request   Request
		reader    *RequestSetReaderStub
		writer    *ResponseWriterStub
		logs      *observer.ObservedLogs
		logger    ExchangeLogger
	)

	BeforeEach(func() {
		exchanger = &ExchangerStub{}

		request = Request{
			Version:    "2.0",
			Method:     "<method>",
			Parameters: json.RawMessage(`[]`),
		}

		reader = &RequestSetReaderStub{
			ReadFunc: func(context.Context) (RequestSet, error) {
				return RequestSet{
					Requests: []Request{request},
				}, nil
			},
		}

		writer = &ResponseWriterStub{}

		var core zapcore.Core
		core, logs = observer.New(zapcore.DebugLevel)
		logger = NewZapExchangeLogger(zap.New(core))
	})

	When("the writer cannot be closed", func() {
		BeforeEach(func() {
			writer.CloseFunc = func() error {
				return errors.New("<close error>")
			}
		})

		It("logs and returns the error", func() {
			err := Exchange(
				context.Background(),
				exchanger,
				reader,
				writer,
				logger,
			)

			Expect(err).To(MatchError("<close error>"))
			Expect(logs.AllUntimed()).To(ContainElement(
				observer.LoggedEntry{
					Entry: zapcore.Entry{
						Level:   zapcore.ErrorLevel,
						Message: "unable to write JSON-RPC response",
					},
					Context: []zapcore.Field{
						zap.String("error", "<close error>"),
					},
				},
			))
		})

		It("returns the causal error instead, if there is one", func() {
			reader.ReadFunc = func(context.Context) (RequestSet, error) {
				return RequestSet{}, errors.New("<read error>")
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
						Message: "unable to write JSON-RPC response",
					},
					Context: []zapcore.Field{
						zap.String("error", "<close error>"),
					},
				},
			))
		})
	})
})
