package voorhees_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/jmalloc/voorhees"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("func Exchange()", func() {
	var (
		requestSet                   RequestSet
		requestA, requestB, requestC Request
		exchanger                    *exchangerStub
		writer                       *responseWriterStub
		closed                       bool
	)

	BeforeEach(func() {
		requestSet = RequestSet{}

		requestA = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`123`),
			Method:     "<method-a>",
			Parameters: json.RawMessage(`[1, 2, 3]`),
		}

		requestB = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`456`),
			Method:     "<method-b>",
			Parameters: json.RawMessage(`[4, 5, 6]`),
		}

		requestC = Request{
			Version:    "2.0",
			ID:         nil, // notification
			Method:     "<method-c>",
			Parameters: json.RawMessage(`[7, 8, 9]`),
		}

		exchanger = &exchangerStub{}

		exchanger.CallFunc = func(
			_ context.Context,
			req Request,
		) Response {
			return SuccessResponse{
				Version:   "2.0",
				RequestID: req.ID,
				Result:    json.RawMessage(`"result"`),
			}
		}

		closed = false
		writer = &responseWriterStub{
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
	})

	AfterEach(func() {
		// The response writer must always be closed.
		Expect(closed).To(BeTrue())
	})

	It("writes an error response if the request set is invalid", func() {
		requestSet = RequestSet{
			IsBatch: true,
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
					Error: ErrorInfo{
						Code:    InvalidRequestCode,
						Message: "batches must contain at least one request",
						Data:    nil,
					},
				},
			))

			return errors.New("<error>")
		}

		err := Exchange(
			context.Background(),
			requestSet,
			exchanger,
			writer,
		)

		Expect(err).To(MatchError("<error>"))
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
				expect := SuccessResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`123`),
					Result:    json.RawMessage(`[10, 20, 30]`),
				}

				exchanger.CallFunc = func(
					_ context.Context,
					req Request,
				) Response {
					Expect(req).To(Equal(requestA))
					return expect
				}

				writer.WriteUnbatchedFunc = func(
					_ context.Context,
					req Request,
					res Response,
				) error {
					Expect(req).To(Equal(requestA))
					Expect(res).To(Equal(expect))

					return errors.New("<error>")
				}

				err := Exchange(
					context.Background(),
					requestSet,
					exchanger,
					writer,
				)

				Expect(err).To(MatchError("<error>"))
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
					requestSet,
					exchanger,
					writer,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(called).To(BeTrue())
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
								Result:    json.RawMessage(`"result"`),
							},
						))

						return errors.New("<error>")
					}

					err := Exchange(
						context.Background(),
						requestSet,
						exchanger,
						writer,
					)

					Expect(err).To(MatchError("<error>"))
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
						requestSet,
						exchanger,
						writer,
					)

					Expect(err).ShouldNot(HaveOccurred())
					Expect(called).To(BeTrue())
				})
			})
		})

		When("the batch contains a multiple requests", func() {
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
					requestSet,
					exchanger,
					writer,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(calls).To(ConsistOf(requestA, requestB))
				Expect(notifications).To(ConsistOf(requestC))
			})

			It("write a batched response for each call (but not notifications)", func() {
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
					requestSet,
					exchanger,
					writer,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(exchanges).To(ConsistOf(
					exchange{
						requestA,
						SuccessResponse{
							Version:   "2.0",
							RequestID: json.RawMessage(`123`),
							Result:    json.RawMessage(`"result"`),
						},
					},
					exchange{
						requestB,
						SuccessResponse{
							Version:   "2.0",
							RequestID: json.RawMessage(`456`),
							Result:    json.RawMessage(`"result"`),
						},
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
						requestSet,
						exchanger,
						writer,
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
						requestSet,
						exchanger,
						writer,
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
						requestSet,
						exchanger,
						writer,
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
						requestSet,
						exchanger,
						writer,
					)
				})
			})
		})
	})
})

// exchangerStub is a test implementation of the Exchanger interface.
type exchangerStub struct {
	CallFunc   func(context.Context, Request) Response
	NotifyFunc func(context.Context, Request)
}

// Call handles a call request and returns the response.
func (s *exchangerStub) Call(ctx context.Context, req Request) Response {
	if s.CallFunc != nil {
		return s.CallFunc(ctx, req)
	}

	return nil
}

// Notify handles a notification request.
func (s *exchangerStub) Notify(ctx context.Context, req Request) {
	if s.NotifyFunc != nil {
		s.NotifyFunc(ctx, req)
	}
}

// responseWriterStub is a test implementation of the ResponseWriter interface.
type responseWriterStub struct {
	WriteErrorFunc     func(context.Context, RequestSet, ErrorResponse) error
	WriteUnbatchedFunc func(context.Context, Request, Response) error
	WriteBatchedFunc   func(context.Context, Request, Response) error
	CloseFunc          func() error
}

func (s *responseWriterStub) WriteError(ctx context.Context, rs RequestSet, res ErrorResponse) error {
	if s.WriteErrorFunc != nil {
		return s.WriteErrorFunc(ctx, rs, res)
	}

	return nil
}

func (s *responseWriterStub) WriteUnbatched(ctx context.Context, req Request, res Response) error {
	if s.WriteUnbatchedFunc != nil {
		return s.WriteUnbatchedFunc(ctx, req, res)
	}

	return nil
}

func (s *responseWriterStub) WriteBatched(ctx context.Context, req Request, res Response) error {
	if s.WriteBatchedFunc != nil {
		return s.WriteBatchedFunc(ctx, req, res)
	}

	return nil
}

func (s *responseWriterStub) Close() error {
	if s.CloseFunc != nil {
		return s.CloseFunc()
	}

	return nil
}
