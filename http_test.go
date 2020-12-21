package harpy_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/dogmatiq/iago/iotest"
	. "github.com/jmalloc/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("type HTTPHandler", func() {
	var (
		exchanger *exchangerStub
		handler   *HTTPHandler
		server    *httptest.Server
		request   *strings.Reader
	)

	BeforeEach(func() {
		exchanger = &exchangerStub{}

		exchanger.CallFunc = func(
			_ context.Context,
			req Request,
		) Response {
			return SuccessResponse{
				Version:   "2.0",
				RequestID: req.ID,
				Result:    req.Parameters,
			}
		}

		handler = &HTTPHandler{
			Exchanger: exchanger,
		}

		server = httptest.NewServer(handler)

		request = strings.NewReader(`{
			"jsonrpc": "2.0",
			"id": 123,
			"params": [1, 2, 3]
		}`)
	})

	AfterEach(func() {
		server.Close()
	})

	When("the request is not a batch", func() {
		It("responds with an unbatched response", func() {
			res, err := http.Post(server.URL, "application/json", request)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusOK))

			json, err := ioutil.ReadAll(res.Body)
			res.Body.Close()

			Expect(err).ShouldNot(HaveOccurred())
			Expect(json).To(MatchJSON(`{
				"jsonrpc": "2.0",
				"id": 123,
				"result": [1, 2, 3]
			}`))
		})
	})

	When("the request is a batch", func() {
		BeforeEach(func() {
			request = strings.NewReader(`[
				{
					"jsonrpc": "2.0",
					"id": 123,
					"params": [1, 2, 3]
				},
				{
					"jsonrpc": "2.0",
					"id": 456,
					"params": [4, 5, 6]
				}
			]`)
		})

		It("responds with a batched response", func() {
			res, err := http.Post(server.URL, "application/json", request)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusOK))

			json, err := ioutil.ReadAll(res.Body)
			res.Body.Close()

			Expect(err).ShouldNot(HaveOccurred())
			Expect(json).To(
				Or(
					MatchJSON(`[
						{
							"jsonrpc": "2.0",
							"id": 123,
							"result": [1, 2, 3]
						},
						{
							"jsonrpc": "2.0",
							"id": 456,
							"result": [4, 5, 6]
						}
					]`),
					MatchJSON(`[
						{
							"jsonrpc": "2.0",
							"id": 456,
							"result": [4, 5, 6]
						},
						{
							"jsonrpc": "2.0",
							"id": 123,
							"result": [1, 2, 3]
						}
					]`),
				),
			)
		})
	})

	It("responds with an error if the HTTP method is not POST", func() {
		res, err := http.Get(server.URL)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusMethodNotAllowed))

		json, err := ioutil.ReadAll(res.Body)
		res.Body.Close()

		Expect(err).ShouldNot(HaveOccurred())
		Expect(json).To(MatchJSON(`{
			"jsonrpc": "2.0",
			"id": null,
			"error": {
				"code": -32600,
				"message": "JSON-RPC requests must use the POST method"
			}
		}`))
	})

	It("responds with an error if the content type is not application/json", func() {
		res, err := http.Post(server.URL, "test/plain", request)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusUnsupportedMediaType))

		json, err := ioutil.ReadAll(res.Body)
		res.Body.Close()

		Expect(err).ShouldNot(HaveOccurred())
		Expect(json).To(MatchJSON(`{
			"jsonrpc": "2.0",
			"id": null,
			"error": {
				"code": -32600,
				"message": "JSON-RPC requests must use the application/json content type"
			}
		}`))
	})

	It("responds with an error if the request is malformed", func() {
		request = strings.NewReader(`}`)

		res, err := http.Post(server.URL, "application/json", request)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusBadRequest))

		json, err := ioutil.ReadAll(res.Body)
		res.Body.Close()

		Expect(err).ShouldNot(HaveOccurred())
		Expect(json).To(MatchJSON(`{
			"jsonrpc": "2.0",
			"id": null,
			"error": {
				"code": -32700,
				"message": "unable to parse request: invalid character '}' looking for beginning of value"
			}
		}`))
	})

	It("responds with an error when the request is well-formed but invalid", func() {
		request = strings.NewReader(`{
			"jsonrpc": "2.0",
			"id": {},
			"params": [1, 2, 3]
		}`)

		res, err := http.Post(server.URL, "application/json", request)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusBadRequest))

		json, err := ioutil.ReadAll(res.Body)
		res.Body.Close()

		Expect(err).ShouldNot(HaveOccurred())
		Expect(json).To(MatchJSON(`{
			"jsonrpc": "2.0",
			"id": null,
			"error": {
				"code": -32600,
				"message": "request ID must be a JSON string, number or null"
			}
		}`))
	})

	It("responds with an error when the request can not be read", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", iotest.NewFailer(nil, nil))
		r.Header.Set("Content-Type", "application/json")

		handler.ServeHTTP(w, r)

		Expect(w.Code).To(Equal(http.StatusInternalServerError))

		json, err := ioutil.ReadAll(w.Body)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(json).To(MatchJSON(`{
			"jsonrpc": "2.0",
			"id": null,
			"error": {
				"code": -32603,
				"message": "unable to read request body"
			}
		}`))
	})

	DescribeTable(
		"it maps JSON-RPC error codes to the appropriate HTTP status code",
		func(err error, statusCode int) {
			exchanger.CallFunc = func(
				_ context.Context,
				req Request,
			) Response {
				return NewErrorResponse(req.ID, err)
			}

			res, err := http.Post(server.URL, "application/json", request)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.StatusCode).To(Equal(statusCode))
		},
		Entry("method not found", MethodNotFound(), http.StatusNotImplemented),
		Entry("invalid parameters", InvalidParameters(), http.StatusBadRequest),
		Entry("internal error", NewErrorWithReservedCode(InternalErrorCode), http.StatusInternalServerError),
		Entry("a native JSON-RPC error with an unreserved code", NewError(123), http.StatusOK),
	)
})
