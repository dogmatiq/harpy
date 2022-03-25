package harpy_test

import (
	"context"
	"encoding/json"

	. "github.com/dogmatiq/harpy"
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

	Describe("func NewRouter()", func() {
		It("unmarshals parameters based on the type in the route", func() {
			called := false

			router = NewRouter(
				WithRoute(
					"<method>",
					func(ctx context.Context, params []int) (any, error) {
						called = true
						Expect(params).To(Equal([]int{1, 2, 3}))
						return nil, nil
					},
				),
			)

			router.Call(context.Background(), request)
			Expect(called).To(BeTrue())
		})

		It("allows calls to handlers that don't return a result", func() {
			called := false

			router = NewRouter(
				WithRoute(
					"<method>",
					NoResult(func(ctx context.Context, params []int) error {
						called = true
						Expect(params).To(Equal([]int{1, 2, 3}))
						return nil
					}),
				),
			)

			router.Call(context.Background(), request)
			Expect(called).To(BeTrue())
		})

		It("returns an error response if the parameters can not be unpacked", func() {
			router = NewRouter(
				WithRoute(
					"<method>",
					func(ctx context.Context, params []string) (any, error) {
						panic("unexpected call")
					},
				),
			)

			res := router.Call(context.Background(), request)

			var errorRes ErrorResponse
			Expect(res).To(BeAssignableToTypeOf(errorRes))

			errorRes = res.(ErrorResponse)
			errorRes.ServerError = nil // remove for comparison

			Expect(errorRes).To(Equal(ErrorResponse{
				Version:   `2.0`,
				RequestID: json.RawMessage(`123`),
				Error: ErrorInfo{
					Code:    InvalidParametersCode,
					Message: "json: cannot unmarshal number into Go value of type string",
				},
			}))
		})

		It("panics if two routes refer to the same method", func() {
			Expect(func() {
				NewRouter(
					WithRoute(
						"<method>",
						func(context.Context, []int) (any, error) {
							panic("not implemented")
						},
					),
					WithRoute(
						"<method>",
						func(context.Context, []int) (any, error) {
							panic("not implemented")
						},
					),
				)
			}).To(PanicWith("duplicate route for '<method>' method"))
		})
	})

	Describe("func Call()", func() {
		When("there is a route for the method", func() {
			It("calls the associated handler", func() {
				called := false

				router["<method>"] = func(
					_ context.Context,
					req Request,
				) (any, error) {
					called = true
					Expect(req).To(Equal(req))
					return nil, nil
				}

				router.Call(context.Background(), request)
				Expect(called).To(BeTrue())
			})

			When("the handler succeeds", func() {
				var result any

				BeforeEach(func() {
					result = 456

					router["<method>"] = func(
						context.Context,
						Request,
					) (any, error) {
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
					) (any, error) {
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
				) (any, error) {
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
