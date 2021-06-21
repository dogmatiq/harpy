package harpy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dogmatiq/dodeca/logging"
	. "github.com/jmalloc/harpy"
	. "github.com/jmalloc/harpy/internal/fixtures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("func Exchange() (batch requests)", func() {
	var (
		exchanger                    *ExchangerStub
		requestA, requestB, requestC Request
		reader                       *RequestSetReaderStub
		writer                       *ResponseWriterStub
		buffer                       *logging.BufferedLogger
		logger                       DefaultExchangeLogger
		closed                       bool
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
				Result:    json.RawMessage(`"result of ` + req.Method + `"`),
			}
		}

		requestA = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`123`),
			Method:     "<method-a>",
			Parameters: json.RawMessage(`[]`),
		}

		requestB = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`456`),
			Method:     "<method-b>",
			Parameters: json.RawMessage(`[]`),
		}

		requestC = Request{
			Version:    "2.0",
			ID:         nil, // notification
			Method:     "<method-c>",
			Parameters: json.RawMessage(`[]`),
		}

		reader = &RequestSetReaderStub{}

		writer = &ResponseWriterStub{
			WriteErrorFunc: func(ErrorResponse) error {
				panic("unexpected call to WriteErrorFunc()")
			},
			WriteUnbatchedFunc: func(Request, Response) error {
				panic("unexpected call to WriteUnbatchedFunc()")
			},
			WriteBatchedFunc: func(Request, Response) error {
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

	When("the batch contains a single request", func() {
		When("the request is a call", func() {
			BeforeEach(func() {
				reader.ReadFunc = func(context.Context) (RequestSet, error) {
					return RequestSet{
						Requests: []Request{requestA},
						IsBatch:  true,
					}, nil
				}
			})

			It("passes the request to the exchanger and writes a batched response", func() {
				writer.WriteBatchedFunc = func(
					req Request,
					res Response,
				) error {
					Expect(req).To(Equal(requestA))
					Expect(res).To(Equal(
						SuccessResponse{
							Version:   "2.0",
							RequestID: req.ID,
							Result:    json.RawMessage(`"result of <method-a>"`),
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
						Message: `call "<method-a>" [params: 2 B, result: 22 B]`,
					},
				))
			})

			It("logs and returns errors that occur when writing the response", func() {
				writer.WriteBatchedFunc = func(
					req Request,
					res Response,
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

		When("the request is a notification", func() {
			BeforeEach(func() {
				reader.ReadFunc = func(context.Context) (RequestSet, error) {
					return RequestSet{
						Requests: []Request{requestC},
						IsBatch:  true,
					}, nil
				}
			})

			It("passes the request to the exchanger and does not write any responses", func() {
				called := true
				exchanger.NotifyFunc = func(
					_ context.Context,
					req Request,
				) {
					Expect(req).To(Equal(requestC))
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
				Expect(buffer.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `notify "<method-c>" [params: 2 B]`,
					},
				))
			})
		})
	})

	When("the batch contains multiple requests", func() {
		BeforeEach(func() {
			reader.ReadFunc = func(context.Context) (RequestSet, error) {
				return RequestSet{
					Requests: []Request{requestA, requestB, requestC},
					IsBatch:  true,
				}, nil
			}
		})

		It("invokes the exchanger for each request", func() {
			var (
				m             sync.Mutex
				calls         []Request
				notifications []Request
			)

			exchanger.CallFunc = func(
				_ context.Context,
				req Request,
			) Response {
				m.Lock()
				defer m.Unlock()

				calls = append(calls, req)
				return SuccessResponse{}
			}

			exchanger.NotifyFunc = func(
				_ context.Context,
				req Request,
			) {
				m.Lock()
				defer m.Unlock()

				notifications = append(notifications, req)
			}

			writer.WriteBatchedFunc = nil // remove default panic behavior

			err := Exchange(
				context.Background(),
				exchanger,
				reader,
				writer,
				logger,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(calls).To(ConsistOf(requestA, requestB))
			Expect(notifications).To(ConsistOf(requestC))
		})

		It("writes a batched response for each call (but not notifications)", func() {
			type exchange struct {
				request  Request
				response Response
			}

			var (
				m         sync.Mutex
				exchanges []exchange
			)

			writer.WriteBatchedFunc = func(
				req Request,
				res Response,
			) error {
				m.Lock()
				defer m.Unlock()

				exchanges = append(exchanges, exchange{req, res})

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
			Expect(exchanges).To(ConsistOf(
				exchange{
					requestA,
					SuccessResponse{
						Version:   "2.0",
						RequestID: json.RawMessage(`123`),
						Result:    json.RawMessage(`"result of <method-a>"`),
					},
				},
				exchange{
					requestB,
					SuccessResponse{
						Version:   "2.0",
						RequestID: json.RawMessage(`456`),
						Result:    json.RawMessage(`"result of <method-b>"`),
					},
				},
			))

			Expect(buffer.Messages()).To(ContainElements(
				logging.BufferedLogMessage{
					Message: `call "<method-a>" [params: 2 B, result: 22 B]`,
				},
				logging.BufferedLogMessage{
					Message: `call "<method-b>" [params: 2 B, result: 22 B]`,
				},
				logging.BufferedLogMessage{
					Message: `notify "<method-c>" [params: 2 B]`,
				},
			))
		})

		When("the response writer returns an error", func() {
			BeforeEach(func() {
				writer.WriteBatchedFunc = func(
					Request,
					Response,
				) error {
					return errors.New("<write error>")
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

				Expect(err).To(MatchError("<write error>"))
				Expect(buffer.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `unable to write JSON-RPC response: <write error>`,
					},
				))
			})

			It("cancels the context given to the exchanger", func() {
				exchanger.CallFunc = func(
					ctx context.Context,
					req Request,
				) Response {
					defer GinkgoRecover()

					if !bytes.Equal(req.ID, requestA.ID) {
						// For any request other than requestA we expect our
						// context to be canceled.
						<-ctx.Done()
						Expect(ctx.Err()).To(Equal(context.Canceled))
					}

					return SuccessResponse{}
				}

				exchanger.NotifyFunc = func(
					ctx context.Context,
					_ Request,
				) {
					defer GinkgoRecover()

					// Just as for calls, we are expect that the context for
					// notifications is canceled.
					<-ctx.Done()
					Expect(ctx.Err()).To(Equal(context.Canceled))
				}

				Exchange(
					context.Background(),
					exchanger,
					reader,
					writer,
					logger,
				)
			})

			It("waits for the pending goroutines to finish", func() {
				var done int32 // atomic

				exchanger.CallFunc = func(
					ctx context.Context,
					req Request,
				) Response {
					time.Sleep(5 * time.Millisecond)
					atomic.AddInt32(&done, 1)
					return SuccessResponse{}
				}

				exchanger.NotifyFunc = func(
					ctx context.Context,
					_ Request,
				) {
					time.Sleep(5 * time.Millisecond)
					atomic.AddInt32(&done, 1)
				}

				Exchange(
					context.Background(),
					exchanger,
					reader,
					writer,
					logger,
				)

				Expect(done).To(BeEquivalentTo(3))
			})

			It("does not write any further responses", func() {
				called := false
				writer.WriteBatchedFunc = func(
					Request,
					Response,
				) error {
					Expect(called).To(BeFalse())
					called = true
					return errors.New("<error>")
				}

				Exchange(
					context.Background(),
					exchanger,
					reader,
					writer,
					logger,
				)
			})
		})
	})
})
