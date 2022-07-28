package httptransport_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/dogmatiq/harpy"
	. "github.com/dogmatiq/harpy/transport/httptransport"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Client", func() {
	var (
		ctx     context.Context
		cancel  context.CancelFunc
		handler http.Handler
		server  *httptest.Server
		client  *Client
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)

		handler = NewHandler(
			harpy.NewRouter(
				harpy.WithRoute(
					"echo",
					func(_ context.Context, params any) (any, error) {
						return params, nil
					},
				),
				harpy.WithRoute(
					"error",
					harpy.NoResult(
						func(_ context.Context, params any) error {
							return harpy.NewError(
								123,
								harpy.WithMessage("<message>"),
								harpy.WithData(params),
							)
						},
					),
				),
			),
		)

		server = httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handler.ServeHTTP(w, r)
			}),
		)

		client = &Client{
			URL: server.URL,
		}
	})

	AfterEach(func() {
		server.Close()
		cancel()
	})

	Describe("func Call()", func() {
		It("returns the JSON-RPC result", func() {
			params := []int{1, 2, 3}
			var result []int
			err := client.Call(ctx, "echo", params, &result)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).To(Equal(params))
		})

		It("returns the JSON-RPC error produced by the server", func() {
			params := []int{1, 2, 3}
			var result any
			err := client.Call(ctx, "error", params, &result)
			Expect(err).Should(HaveOccurred())
			Expect(result).To(BeNil())

			var rpcErr harpy.Error
			ok := errors.As(err, &rpcErr)
			Expect(ok).To(BeTrue())
			Expect(rpcErr.Code()).To(BeNumerically("==", 123))
			Expect(rpcErr.Message()).To(Equal("<message>"))

			var data []int
			ok, err = rpcErr.UnmarshalData(&data)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(data).To(Equal(params))
		})

		It("returns an error if there is a network error", func() {
			server.Close()

			params := []int{1, 2, 3}
			var result []int
			err := client.Call(ctx, "echo", params, &result)
			Expect(err).To(MatchError(
				fmt.Sprintf(
					`unable to call JSON-RPC method (echo): Post "%s": dial tcp %s: connect: connection refused`,
					server.URL,
					strings.TrimPrefix(server.URL, "http://"),
				),
			))
		})

		It("returns an error if the result cannot be unmarshaled", func() {
			params := []int{1, 2, 3}
			var result []string
			err := client.Call(ctx, "echo", params, &result)
			Expect(err).To(MatchError(
				`unable to process JSON-RPC response (echo): unable to unmarshal result: json: cannot unmarshal number into Go value of type string`,
			))
		})

		It("panics if the JSON-RPC request can not be built", func() {
			Expect(func() {
				var result any
				client.Call(
					ctx,
					"<method>",
					make(chan struct{}),
					&result,
				)
			}).To(PanicWith(
				`unable to call JSON-RPC method (<method>): unable to marshal request parameters: json: unsupported type: chan struct {}`,
			))
		})

		It("panics if the JSON-RPC request can not be validated", func() {
			Expect(func() {
				var result any
				client.Call(
					ctx,
					"<method>",
					123,
					&result,
				)
			}).To(PanicWith(
				`unable to call JSON-RPC method (<method>): parameters must be an array, an object, or null`,
			))
		})

		DescribeTable(
			"it panics if the result variable is not a pointer",
			func(result any) {
				Expect(func() {
					client.Call(
						ctx,
						"<method>",
						[]int{1, 2, 3},
						result,
					)
				}).To(PanicWith(
					`unable to call JSON-RPC method (<method>): result must be a non-nil pointer`,
				))
			},
			Entry("nil interface", nil),
			Entry("nil pointer", (*int)(nil)),
			Entry("non-pointer", "<string>"),
		)

		When("the server exhibits unexpected behavior", func() {
			It("returns an error if the server responds with an unexpected content type", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				})

				params := []int{1, 2, 3}
				var result []int
				err := client.Call(ctx, "echo", params, &result)
				Expect(err).To(MatchError("unable to process JSON-RPC response (echo): unexpected content-type in HTTP response (text/plain)"))
			})

			It("returns an error if the JSON-RPC response cannot be parsed", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("{"))
				})

				params := []int{1, 2, 3}
				var result []int
				err := client.Call(ctx, "echo", params, &result)
				Expect(err).To(MatchError("unable to process JSON-RPC response (echo): cannot unmarshal JSON-RPC response: unexpected EOF"))
			})

			It("returns an error if the JSON-RPC response is a batch", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`[{
						"jsonrpc": "2.0",
						"id": 123,
						"result": {}
					}]`))
				})

				params := []int{1, 2, 3}
				var result []int
				err := client.Call(ctx, "echo", params, &result)
				Expect(err).To(MatchError("unable to process JSON-RPC response (echo): unexpected JSON-RPC batch response"))
			})

			It("returns an error if server returns a JSON-RPC success with an unexpected HTTP status", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{
						"jsonrpc": "2.0",
						"id": 1,
						"result": {}
					}`))
				})

				params := []int{1, 2, 3}
				var result []int
				err := client.Call(ctx, "echo", params, &result)
				Expect(err).To(MatchError(
					`unable to process JSON-RPC response (echo): unexpected HTTP 400 (Bad Request) status code with JSON-RPC success response`,
				))
			})

			It("returns an error if server returns a JSON-RPC error response with a non-integer request ID", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{
						"jsonrpc": "2.0",
						"id": "<id>",
						"error": {}
					}`))
				})

				params := []int{1, 2, 3}
				var result []int
				err := client.Call(ctx, "echo", params, &result)
				Expect(err).To(MatchError(
					`unable to process JSON-RPC response (echo): request ID in response is expected to be an integer`,
				))
			})

			It("returns an error if server returns a JSON-RPC success response with a mismatched request ID", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{
						"jsonrpc": "2.0",
						"id": 123,
						"result": {}
					}`))
				})

				params := []int{1, 2, 3}
				var result []int
				err := client.Call(ctx, "echo", params, &result)
				Expect(err).To(MatchError(
					`unable to process JSON-RPC response (echo): request ID in response (123) does not match the actual request ID (1)`,
				))
			})
		})
	})

	Describe("func Notify()", func() {
		It("returns nil on success", func() {
			called := false
			handler = NewHandler(
				harpy.NewRouter(
					harpy.WithRoute(
						"echo",
						func(_ context.Context, params []int) (any, error) {
							Expect(params).To(Equal([]int{1, 2, 3}))
							called = true

							return nil, nil
						},
					),
				),
			)

			params := []int{1, 2, 3}
			err := client.Notify(ctx, "echo", params)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(called).To(BeTrue())
		})

		It("returns the JSON-RPC error produced by the server", func() {
			// We have to force the server to return an error here as the Harpy
			// server only responds with a JSON-RPC error if the request set
			// cannot be parsed, and our client can't produce invalid JSON.
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{
						"jsonrpc": "2.0",
						"id": null,
						"error": {
							"code": 123,
							"message": "<message>",
							"data": [1, 2, 3]
						}
					}`))
			})

			err := client.Notify(ctx, "<method>", []any{})
			Expect(err).Should(HaveOccurred())

			var rpcErr harpy.Error
			ok := errors.As(err, &rpcErr)
			Expect(ok).To(BeTrue())
			Expect(rpcErr.Code()).To(BeNumerically("==", 123))
			Expect(rpcErr.Message()).To(Equal("<message>"))

			var data []int
			ok, err = rpcErr.UnmarshalData(&data)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(data).To(Equal([]int{1, 2, 3}))
		})

		It("returns an error if there is a network error", func() {
			server.Close()

			params := []int{1, 2, 3}
			err := client.Notify(ctx, "echo", params)
			Expect(err).To(MatchError(
				fmt.Sprintf(
					`unable to send JSON-RPC notification (echo): Post "%s": dial tcp %s: connect: connection refused`,
					server.URL,
					strings.TrimPrefix(server.URL, "http://"),
				),
			))
		})

		It("panics if the JSON-RPC request can not be built", func() {
			Expect(func() {
				client.Notify(
					ctx,
					"<method>",
					make(chan struct{}),
				)
			}).To(PanicWith(
				`unable to send JSON-RPC notification (<method>): unable to marshal request parameters: json: unsupported type: chan struct {}`,
			))
		})

		It("panics if the JSON-RPC request can not be validated", func() {
			Expect(func() {
				client.Notify(
					ctx,
					"<method>",
					123,
				)
			}).To(PanicWith(
				`unable to send JSON-RPC notification (<method>): parameters must be an array, an object, or null`,
			))
		})

		When("the server exhibits unexpected behavior", func() {
			It("returns an error if the server responds with an unexpected content type", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("OK"))
				})

				params := []int{1, 2, 3}
				err := client.Notify(ctx, "echo", params)
				Expect(err).To(MatchError("unable to process JSON-RPC response (echo): unexpected content-type in HTTP response (text/plain)"))
			})

			It("returns an error if the JSON-RPC response cannot be parsed", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("{"))
				})

				params := []int{1, 2, 3}
				err := client.Notify(ctx, "echo", params)
				Expect(err).To(MatchError("unable to process JSON-RPC response (echo): cannot unmarshal JSON-RPC response: unexpected EOF"))
			})

			It("returns an error if the JSON-RPC response is a batch", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`[{
							"jsonrpc": "2.0",
							"id": 123,
							"result": {}
						}]`))
				})

				params := []int{1, 2, 3}
				err := client.Notify(ctx, "echo", params)
				Expect(err).To(MatchError("unable to process JSON-RPC response (echo): unexpected JSON-RPC batch response"))
			})

			It("returns an error if server returns a JSON-RPC success response", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{
							"jsonrpc": "2.0",
							"id": null,
							"result": {}
						}`))
				})

				params := []int{1, 2, 3}
				err := client.Notify(ctx, "echo", params)
				Expect(err).To(MatchError(
					`unable to process JSON-RPC response (echo): did not expect a successful JSON-RPC response to a notification, HTTP status code is 400 (Bad Request)`,
				))
			})

			It("returns an error if server returns a JSON-RPC error response with a non-null request ID", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{
							"jsonrpc": "2.0",
							"id": "<id>",
							"error": {}
						}`))
				})

				params := []int{1, 2, 3}
				err := client.Notify(ctx, "echo", params)
				Expect(err).To(MatchError(
					`unable to process JSON-RPC response (echo): request ID in response is expected to be null`,
				))
			})

			It("returns an error if server responds with a successful HTTP status code and content", func() {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{
							"jsonrpc": "2.0",
							"id": null,
							"result": {}
						}`))
				})

				params := []int{1, 2, 3}
				err := client.Notify(ctx, "echo", params)
				Expect(err).To(MatchError(
					`unable to process JSON-RPC response (echo): unexpected HTTP 200 (OK) status code in response to JSON-RPC notification`,
				))
			})
		})
	})
})
