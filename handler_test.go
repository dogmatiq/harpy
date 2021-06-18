package harpy_test

import (
	"context"
	"encoding/json"

	. "github.com/jmalloc/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Router", func() {
	var (
		request Request
		router  Router
	)

	BeforeEach(func() {
		request = Request{
			Version:    "2.0",
			ID:         json.RawMessage(`123`),
			Method:     "<method>",
			Parameters: json.RawMessage(`[1, 2, 3]`),
		}

		router = Router{}
	})

	Describe("func Call()", func() {
		When("there is a route for the method", func() {
			It("calls the associated handler", func() {
				called := false

				router["<method>"] = func(
					_ context.Context,
					req Request,
				) (interface{}, error) {
					called = true
					Expect(req).To(Equal(req))
					return nil, nil
				}

				router.Call(context.Background(), request)
				Expect(called).To(BeTrue())
			})

			When("the handler succeeds", func() {
				var result interface{}

				BeforeEach(func() {
					result = 456

					router["<method>"] = func(
						context.Context,
						Request,
					) (interface{}, error) {
						return result, nil
					}
				})

				It("returns a success response that contains the marshaled result", func() {
					res := router.Call(context.Background(), request)
					Expect(res).To(Equal(SuccessResponse{
						Version:   `2.0`,
						RequestID: json.RawMessage(`123`),
						Result:    json.RawMessage(`456`),
					}))
				})
			})

			When("the handler returns an error", func() {
				var err error

				BeforeEach(func() {
					err = NewError(789, WithMessage("<error>"))

					router["<method>"] = func(
						context.Context,
						Request,
					) (interface{}, error) {
						return nil, err
					}
				})

				It("returns an error response", func() {
					res := router.Call(context.Background(), request)
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
		})

		When("there is no route for the method", func() {
			It("returns an error response", func() {
				res := router.Call(context.Background(), request)
				Expect(res).To(Equal(ErrorResponse{
					Version:   `2.0`,
					RequestID: json.RawMessage(`123`),
					Error: ErrorInfo{
						Code:    MethodNotFoundCode,
						Message: "method not found",
					},
				}))
			})
		})
	})

	Describe("func Notify()", func() {
		BeforeEach(func() {
			request.ID = nil
		})

		When("when there is a route for the method", func() {
			It("calls the associated handler", func() {
				called := false

				router["<method>"] = func(
					_ context.Context,
					req Request,
				) (interface{}, error) {
					called = true
					Expect(req).To(Equal(req))
					return nil, nil
				}

				router.Notify(context.Background(), request)
				Expect(called).To(BeTrue())
			})
		})

		When("there is no route for the method", func() {
			It("ignores the request", func() {
				Expect(func() {
					router.Notify(context.Background(), request)
				}).NotTo(Panic())
			})
		})
	})
})
