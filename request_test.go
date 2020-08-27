package voorhees_test

import (
	"encoding/json"
	"io"
	"strings"

	. "github.com/jmalloc/voorhees"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("func ParseRequestSet()", func() {
	It("parses a single request", func() {
		r := strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": 123,
				"method": "<method>",
				"params": [1, 2, 3]
			}`)

		rs, err := ParseRequestSet(r)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rs.IsBatch).To(BeFalse())
		Expect(rs.Requests).To(ConsistOf(
			Request{
				Version:    "2.0",
				ID:         json.RawMessage(`123`),
				Method:     "<method>",
				Parameters: json.RawMessage(`[1, 2, 3]`),
			},
		))
	})

	It("parses a batch request with a single request", func() {
		r := strings.NewReader(`[{
				"jsonrpc": "2.0",
				"id": 123,
				"method": "<method>",
				"params": [1, 2, 3]
			}]`)

		rs, err := ParseRequestSet(r)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rs.IsBatch).To(BeTrue())
		Expect(rs.Requests).To(ConsistOf(
			Request{
				Version:    "2.0",
				ID:         json.RawMessage(`123`),
				Method:     "<method>",
				Parameters: json.RawMessage(`[1, 2, 3]`),
			},
		))
	})

	It("parses a batch request with multiple requests", func() {
		r := strings.NewReader(`[{
				"jsonrpc": "2.0",
				"id": 123,
				"method": "<method-a>",
				"params": [1, 2, 3]
			},{
				"jsonrpc": "2.0",
				"id": 456,
				"method": "<method-b>",
				"params": [4, 5, 6]
			}]`)

		rs, err := ParseRequestSet(r)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rs.IsBatch).To(BeTrue())
		Expect(rs.Requests).To(ConsistOf(
			Request{
				Version:    "2.0",
				ID:         json.RawMessage(`123`),
				Method:     "<method-a>",
				Parameters: json.RawMessage(`[1, 2, 3]`),
			},
			Request{
				Version:    "2.0",
				ID:         json.RawMessage(`456`),
				Method:     "<method-b>",
				Parameters: json.RawMessage(`[4, 5, 6]`),
			},
		))
	})

	It("omits the ID field if it is not present in the request", func() {
		r := strings.NewReader(`{}`)

		rs, err := ParseRequestSet(r)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rs.Requests[0].ID).To(BeNil())
	})

	It("includes the ID field if it set to NULL", func() {
		r := strings.NewReader(`{"id": null}`)

		rs, err := ParseRequestSet(r)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rs.Requests[0].ID).To(Equal(json.RawMessage(`null`)))
	})

	It("returns an error if the request can not be read", func() {
		r := strings.NewReader(``)

		_, err := ParseRequestSet(r)
		Expect(err).To(Equal(io.EOF))
	})

	It("returns an error if a single request is malformed", func() {
		r := strings.NewReader(`""`) // not an array or object

		_, err := ParseRequestSet(r)
		Expect(err).To(MatchError("json: cannot unmarshal string into Go value of type voorhees.Request"))
	})

	It("returns an error if a request within a batch malformed", func() {
		r := strings.NewReader(`[""]`) // not an array or object

		_, err := ParseRequestSet(r)
		Expect(err).To(MatchError("json: cannot unmarshal string into Go value of type voorhees.Request"))
	})
})
