package harpy_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dogmatiq/dodeca/logging"
	. "github.com/jmalloc/harpy"
	. "github.com/jmalloc/harpy/internal/fixtures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("func Exchange()", func() {
	var (
		exchanger *ExchangerStub
		request   Request
		reader    *RequestSetReaderStub
		writer    *ResponseWriterStub
		buffer    *logging.BufferedLogger
		logger    DefaultExchangeLogger
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

		buffer = &logging.BufferedLogger{}

		logger = DefaultExchangeLogger{
			Target: buffer,
		}
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
			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `unable to write JSON-RPC response: <close error>`,
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
			Expect(buffer.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: `unable to write JSON-RPC response: <close error>`,
				},
			))
		})
	})
})
