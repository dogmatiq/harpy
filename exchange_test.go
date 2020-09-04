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
	. "github.com/jmalloc/voorhees/fixtures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("func Exchange()", func() {
	var (
		pipeline                     *PipelineStageStub
		requestSet                   RequestSet
		requestA, requestB, requestC Request
		respond                      Responder
	)

	BeforeEach(func() {
		pipeline = &PipelineStageStub{}

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

		pipeline.CallFunc = func(
			_ context.Context,
			req Request,
		) Response {
			return SuccessResponse{
				Version:   "2.0",
				RequestID: req.ID,
				Result:    json.RawMessage(`"result"`),
			}
		}

		respond = func(Request, Response, bool) error {
			Fail("unexpected call to respond()")
			return nil
		}
	})

	When("the request set is a single request", func() {
		BeforeEach(func() {
			requestSet = RequestSet{
				Requests: []Request{requestA},
				IsBatch:  false,
			}
		})

		It("panics if there are no requests", func() {
			requestSet.Requests = nil

			Expect(func() {
				Exchange(context.Background(), requestSet, pipeline, respond)
			}).To(PanicWith("non-batch request sets must contain exactly one request"))
		})

		It("panics if there is more than one request", func() {
			requestSet.Requests = []Request{Request{}, Request{}}

			Expect(func() {
				Exchange(context.Background(), requestSet, pipeline, respond)
			}).To(PanicWith("non-batch request sets must contain exactly one request"))
		})

		When("the request is a call", func() {
			It("passes the request to the pipeline and returns a single response", func() {
				expect := SuccessResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`123`),
					Result:    json.RawMessage(`[10, 20, 30]`),
				}

				pipeline.CallFunc = func(
					_ context.Context,
					req Request,
				) Response {
					Expect(req).To(Equal(requestA))
					return expect
				}

				res, single, err := Exchange(
					context.Background(),
					requestSet,
					pipeline,
					respond,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(single).To(BeTrue())
				Expect(res).To(Equal(expect))
			})
		})

		When("the request is a notification", func() {
			BeforeEach(func() {
				requestSet.Requests = []Request{requestC}
			})

			It("passes the request to the pipeline and does not produce any responses", func() {
				called := true
				pipeline.NotifyFunc = func(
					_ context.Context,
					req Request,
				) {
					Expect(req).To(Equal(requestC))
				}

				_, single, err := Exchange(
					context.Background(),
					requestSet,
					pipeline,
					respond,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(single).To(BeFalse())
				Expect(called).To(BeTrue())
			})
		})
	})

	When("the request set is a batch", func() {
		BeforeEach(func() {
			requestSet = RequestSet{
				Requests: []Request{requestA, requestB, requestC},
				IsBatch:  true,
			}
		})

		When("the batch is empty", func() {
			It("returns a single error response", func() {
				requestSet.Requests = nil

				res, single, err := Exchange(
					context.Background(),
					requestSet,
					pipeline,
					respond,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(single).To(BeTrue())
				Expect(res).To(Equal(
					ErrorResponse{
						Version:   "2.0",
						RequestID: nil,
						Error: ErrorInfo{
							Code:    InvalidRequestCode,
							Message: "request batches must contain at least one request",
							Data:    nil,
						},
					},
				))
			})
		})

		When("the batch contains a single request", func() {
			BeforeEach(func() {
				requestSet.Requests = []Request{requestA}
			})

			When("the request is a call", func() {
				It("passes the request to the pipeline and invokes respond() with the response", func() {
					called := false
					respond = func(
						req Request,
						res Response,
						isLast bool,
					) error {
						called = true

						Expect(req).To(Equal(requestA))
						Expect(res).To(Equal(
							SuccessResponse{
								Version:   "2.0",
								RequestID: req.ID,
								Result:    json.RawMessage(`"result"`),
							},
						))
						Expect(isLast).To(BeTrue())

						return nil
					}

					_, single, err := Exchange(
						context.Background(),
						requestSet,
						pipeline,
						respond,
					)

					Expect(err).ShouldNot(HaveOccurred())
					Expect(single).To(BeFalse())
					Expect(called).To(BeTrue())
				})

				When("respond() returns an error", func() {
					BeforeEach(func() {
						respond = func(
							Request,
							Response,
							bool,
						) error {
							return errors.New("<error>")
						}
					})

					It("returns the error", func() {
						_, _, err := Exchange(
							context.Background(),
							requestSet,
							pipeline,
							respond,
						)

						Expect(err).To(MatchError("<error>"))
					})
				})
			})

			When("the request is a notification", func() {
				BeforeEach(func() {
					requestSet.Requests = []Request{requestC}
				})

				It("passes the request to the pipeline and does not produce any responses", func() {
					called := true
					pipeline.NotifyFunc = func(
						_ context.Context,
						req Request,
					) {
						Expect(req).To(Equal(requestC))
					}

					_, single, err := Exchange(
						context.Background(),
						requestSet,
						pipeline,
						respond,
					)

					Expect(err).ShouldNot(HaveOccurred())
					Expect(single).To(BeFalse())
					Expect(called).To(BeTrue())
				})
			})
		})

		When("the batch contains a multiple requests", func() {
			BeforeEach(func() {
				respond = func(
					Request,
					Response,
					bool,
				) error {
					return nil
				}
			})

			It("invokes the pipeline for each request", func() {
				var (
					m             sync.Mutex
					calls         []Request
					notifications []Request
				)

				pipeline.CallFunc = func(
					_ context.Context,
					req Request,
				) Response {
					m.Lock()
					defer m.Unlock()

					calls = append(calls, req)

					return SuccessResponse{}
				}

				pipeline.NotifyFunc = func(
					_ context.Context,
					req Request,
				) {
					m.Lock()
					defer m.Unlock()

					notifications = append(notifications, req)
				}

				_, single, err := Exchange(
					context.Background(),
					requestSet,
					pipeline,
					respond,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(single).To(BeFalse())

				Expect(calls).To(ConsistOf(requestA, requestB))
				Expect(notifications).To(ConsistOf(requestC))
			})

			It("calls respond() for each call (but not notifications)", func() {
				type exchange struct {
					request  Request
					response Response
				}

				var (
					m         sync.Mutex
					exchanges []exchange
				)

				respond = func(
					req Request,
					res Response,
					_ bool,
				) error {
					m.Lock()
					defer m.Unlock()

					exchanges = append(exchanges, exchange{req, res})
					return nil
				}

				_, single, err := Exchange(
					context.Background(),
					requestSet,
					pipeline,
					respond,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(single).To(BeFalse())
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

			It("sets the isLast parameter of respond()", func() {
				count := 0

				respond = func(
					_ Request,
					_ Response,
					isLast bool,
				) error {
					count++
					Expect(isLast).To(Equal(count == 2)) // there are 2 calls and 1 notification
					return nil
				}

				Exchange(
					context.Background(),
					requestSet,
					pipeline,
					respond,
				)
			})

			When("respond() returns an error", func() {
				BeforeEach(func() {
					respond = func(
						Request,
						Response,
						bool,
					) error {
						return errors.New("<error>")
					}
				})

				It("returns the error", func() {
					_, _, err := Exchange(
						context.Background(),
						requestSet,
						pipeline,
						respond,
					)

					Expect(err).To(MatchError("<error>"))
				})

				It("cancels the context given to the pipeline", func() {
					pipeline.CallFunc = func(
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

					pipeline.NotifyFunc = func(
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
						pipeline,
						respond,
					)
				})

				It("waits for the pending goroutines to finish", func() {
					var done int32 // atomic

					pipeline.CallFunc = func(
						ctx context.Context,
						req Request,
					) Response {
						time.Sleep(5 * time.Millisecond)
						atomic.AddInt32(&done, 1)
						return SuccessResponse{}
					}

					pipeline.NotifyFunc = func(
						ctx context.Context,
						_ Request,
					) {
						time.Sleep(5 * time.Millisecond)
						atomic.AddInt32(&done, 1)
					}

					Exchange(
						context.Background(),
						requestSet,
						pipeline,
						respond,
					)

					Expect(done).To(BeEquivalentTo(3))
				})

				It("does not call respond() again", func() {
					called := false
					respond = func(
						Request,
						Response,
						bool,
					) error {
						Expect(called).To(BeFalse())
						called = true
						return errors.New("<error>")
					}

					Exchange(
						context.Background(),
						requestSet,
						pipeline,
						respond,
					)
				})
			})
		})
	})
})
