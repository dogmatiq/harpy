package voorhees_test

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/dogmatiq/iago/iotest"
	. "github.com/jmalloc/voorhees"
	. "github.com/jmalloc/voorhees/fixtures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("func Serve()", func() {
	var exchanger *ExchangerStub

	BeforeEach(func() {
		exchanger = &ExchangerStub{}

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
	})

	When("the next request set is a single request", func() {
		request := `{
			"jsonrpc": "2.0",
			"id": 123,
			"params": [1, 2, 3]
		}`

		It("writes the response from the exchanger", func() {
			r := strings.NewReader(request)
			w := &bytes.Buffer{}

			ok, err := Serve(
				context.Background(),
				exchanger,
				r,
				w,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(w.String()).To(MatchJSON(`{
				"jsonrpc": "2.0",
				"id": 123,
				"result": [1, 2, 3]
			}`))
			Expect(ok).To(BeTrue())
		})

		It("returns an error if the request can not be written", func() {
			r := strings.NewReader(request)
			w := iotest.NewFailer(nil, nil)

			_, err := Serve(
				context.Background(),
				exchanger,
				r,
				w,
			)

			Expect(err).To(Equal(iotest.ErrWrite))
		})
	})

	When("the next request set is a batch", func() {
		request := `[
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
		]`

		It("writes the responses from the exchanger", func() {
			r := strings.NewReader(request)
			w := &bytes.Buffer{}

			ok, err := Serve(
				context.Background(),
				exchanger,
				r,
				w,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(w.String()).To(
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
			Expect(ok).To(BeTrue())
		})

		It("returns an error if the request can not be written", func() {
			r := strings.NewReader(request)
			w := iotest.NewFailer(nil, nil)

			_, err := Serve(
				context.Background(),
				exchanger,
				r,
				w,
			)

			Expect(err).To(Equal(iotest.ErrWrite))
		})
	})

	It("writes an error response if the request is malformed", func() {
		r := strings.NewReader(`}`)
		w := &bytes.Buffer{}

		ok, err := Serve(
			context.Background(),
			exchanger,
			r,
			w,
		)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(w.String()).To(MatchJSON(`{
					"jsonrpc": "2.0",
					"id": null,
					"error": {
						"code": -32700,
						"message": "unable to parse request: invalid character '}' looking for beginning of value"
					}
				}`))
		Expect(ok).To(BeFalse())
	})

	It("returns an error if the request can not be read", func() {
		r := strings.NewReader(``)
		w := &bytes.Buffer{}

		_, err := Serve(
			context.Background(),
			exchanger,
			r,
			w,
		)

		Expect(err).To(Equal(io.EOF))
	})
})
