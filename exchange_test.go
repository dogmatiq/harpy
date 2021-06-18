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
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("func Exchange()", func() {
	var (
		requestSet                   RequestSet
		requestA, requestB, requestC Request
		exchanger                    *ExchangerStub
		reader                       *RequestSetReaderStub
		writer                       *ResponseWriterStub
		buffer                       *logging.BufferedLogger
		logger                       DefaultExchangeLogger
		closed                       bool
	)

	BeforeEach(func() {
		requestSet = RequestSet{}

		requestA = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`123`),
			Method:     "<method-a>",
			Parameters: json.RawMessage(`1`),
		}

		requestB = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`456`),
			Method:     "<method-b>",
			Parameters: json.RawMessage(`22`),
		}

		requestC = Request{
			Version:    "2.0",
			ID:         nil, // notification
			Method:     "<method-c>",
			Parameters: json.RawMessage(`333`),
		}

		exchanger = &ExchangerStub{}

		// Default Call implementation returns its parameter multiplied by 1000.
		exchanger.CallFunc = func(
			_ context.Context,
			req Request,
		) Response {
			var param int
			if err := json.Unmarshal(req.Parameters, &param); err != nil {
				panic(err)
			}

			result, err := json.Marshal(param * 1000)
			if err != nil {
				panic(err)
			}

			return SuccessResponse{
				Version:   "2.0",
				RequestID: req.ID,
				Result:    result,
			}
		}

		reader = &RequestSetReaderStub{
			ReadFunc: func(context.Context) (RequestSet, error) {
				return requestSet, nil
			},
		}

		writer = &ResponseWriterStub{
			WriteErrorFunc: func(context.Context, RequestSet, ErrorResponse) error {
				panic("unexpected call to WriteErrorFunc()")
			},
			WriteUnbatchedFunc: func(context.Context, Request, Response) error {
				panic("unexpected call to WriteUnbatchedFunc()")
			},
			WriteBatchedFunc: func(context.Context, Request, Response) error {
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

	When("the request set is not a batch", func() {
		BeforeEach(func() {
			requestSet = RequestSet{
				Requests: []Request{requestA},
				IsBatch:  false,
			}
		})

		When("the request is a call", func() {
			It("passes the request to the exchanger and writes an unbatched response", func() {
				next := exchanger.CallFunc
				exchanger.CallFunc = func(
					ctx context.Context,
					req Request,
				) Response {
					Expect(req).To(Equal(requestA))
					return next(ctx, req)
				}

				writer.WriteUnbatchedFunc = func(
					_ context.Context,
					req Request,
					res Response,
				) error {
					Expect(req).To(Equal(requestA))
					Expect(res).To(Equal(SuccessResponse{
						Version:   "2.0",
						RequestID: json.RawMessage(`123`),
						Result:    json.RawMessage(`1000`),
					}))

					return errors.New("<error>")
				}

				err := Exchange(
					context.Background(),
					exchanger,
					reader,
					writer,
					logger,
				)

				Expect(err).To(MatchError("<error>"))
				Expect(buffer.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: `call "<method-a>" [params: 1 B, result: 4 B]`,
					},
				))
			})
		})

		When("the request is a notification", func() {
			BeforeEach(func() {
				requestSet.Requests = []Request{requestC}
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
						Message: `notify "<method-c>" [params: 3 B]`,
					},
				))
			})
		})
	})

	When("the request set is a batch", func() {
		BeforeEach(func() {
			requestSet.IsBatch = true
		})

		When("the batch contains a single request", func() {
			BeforeEach(func() {
				requestSet.Requests = []Request{requestA}
			})

			When("the request is a call", func() {
				It("passes the request to the exchanger and writes a batched response", func() {
					writer.WriteBatchedFunc = func(
						ctx context.Context,
						req Request,
						res Response,
					) error {
						Expect(req).To(Equal(requestA))
						Expect(res).To(Equal(
							SuccessResponse{
								Version:   "2.0",
								RequestID: req.ID,
								Result:    json.RawMessage(`1000`),
							},
						))

						return errors.New("<error>")
					}

					err := Exchange(
						context.Background(),
						exchanger,
						reader,
						writer,
						logger,
					)

					Expect(err).To(MatchError("<error>"))
					Expect(buffer.Messages()).To(ContainElement(
						logging.BufferedLogMessage{
							Message: `call "<method-a>" [params: 1 B, result: 4 B]`,
						},
					))
				})
			})

			When("the request is a notification", func() {
				BeforeEach(func() {
					requestSet.Requests = []Request{requestC}
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
							Message: `notify "<method-c>" [params: 3 B]`,
						},
					))
				})
			})
		})

		When("the batch contains multiple requests", func() {
			BeforeEach(func() {
				requestSet.Requests = []Request{requestA, requestB, requestC}
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

				// Remove func that panics.
				writer.WriteBatchedFunc = nil

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
					_ context.Context,
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
							Result:    json.RawMessage(`1000`),
						},
					},
					exchange{
						requestB,
						SuccessResponse{
							Version:   "2.0",
							RequestID: json.RawMessage(`456`),
							Result:    json.RawMessage(`22000`),
						},
					},
				))

				Expect(buffer.Messages()).To(ConsistOf(
					logging.BufferedLogMessage{
						Message: `call "<method-a>" [params: 1 B, result: 4 B]`,
					},
					logging.BufferedLogMessage{
						Message: `call "<method-b>" [params: 2 B, result: 5 B]`,
					},
					logging.BufferedLogMessage{
						Message: `notify "<method-c>" [params: 3 B]`,
					},
				))
			})

			When("the response writer returns an error", func() {
				BeforeEach(func() {
					writer.WriteBatchedFunc = func(
						context.Context,
						Request,
						Response,
					) error {
						return errors.New("<error>")
					}
				})

				It("returns the error", func() {
					err := Exchange(
						context.Background(),
						exchanger,
						reader,
						writer,
						logger,
					)

					Expect(err).To(MatchError("<error>"))
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
						context.Context,
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

	When("there is a problem with the request set", func() {
		DescribeTable(
			"it writes an error response",
			func(
				fn func() (RequestSet, error),
				expectErrInfo ErrorInfo,
				expectLog string,
				expectErr string,
			) {
				reader.ReadFunc = func(context.Context) (RequestSet, error) {
					return fn()
				}

				writer.WriteErrorFunc = func(
					_ context.Context,
					rs RequestSet,
					res ErrorResponse,
				) error {
					Expect(rs).To(Equal(requestSet))
					Expect(res).To(Equal(
						ErrorResponse{
							Version:   "2.0",
							RequestID: nil,
							Error:     expectErrInfo,
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

				Expect(buffer.Messages()).To(ContainElement(
					logging.BufferedLogMessage{
						Message: expectLog,
					},
				))

				if expectErr == "" {
					Expect(err).ShouldNot(HaveOccurred())
				} else {
					Expect(err).To(MatchError(expectErr))
				}
			},
			Entry(
				"IO error when reading the request set",
				func() (RequestSet, error) {
					return RequestSet{}, errors.New("<error>")
				},
				ErrorInfo{
					Code:    InternalErrorCode,
					Message: "unable to read request set: <error>",
				},
				"error: -32603 internal server error, responded with: unable to read request set: <error>",
				"<error>",
			),
			Entry(
				"native JSON-RPC error when reading the request set",
				func() (RequestSet, error) {
					return RequestSet{}, NewErrorWithReservedCode(InvalidRequestCode)
				},
				ErrorInfo{
					Code:    InvalidRequestCode,
					Message: InvalidRequestCode.String(),
				},
				"error: -32600 invalid request",
				"", // Exchange() should not return the error
			),
			Entry(
				"invalid request set",
				func() (RequestSet, error) {
					return RequestSet{}, nil
				},
				ErrorInfo{
					Code:    InvalidRequestCode,
					Message: "non-batch request sets must contain exactly one request",
				},
				"error: -32600 invalid request, responded with: non-batch request sets must contain exactly one request",
				"", // Exchange() should not return the error
			),
		)
	})
})
