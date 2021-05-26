package harpy_test

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	. "github.com/jmalloc/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Request", func() {
	Describe("func IsNotification()", func() {
		It("returns false when a request ID is present", func() {
			req := Request{
				Version: "2.0",
				ID:      json.RawMessage(``),
			}

			Expect(req.IsNotification()).To(BeFalse())
		})

		It("returns true when a request ID is not present", func() {
			req := Request{
				Version: "2.0",
			}

			Expect(req.IsNotification()).To(BeTrue())
		})
	})

	Describe("func Validate()", func() {
		DescribeTable(
			"it returns true when the request is valid",
			func(id json.RawMessage) {
				req := Request{
					Version: "2.0",
					ID:      id,
				}

				_, ok := req.Validate()
				Expect(ok).To(BeTrue())
			},
			Entry("string ID", json.RawMessage(`"<id>"`)),
			Entry("integer ID", json.RawMessage(`1`)),
			Entry("decimal ID", json.RawMessage(`1.2`)),
			Entry("null ID", json.RawMessage(`null`)),
			Entry("absent ID", nil),
		)

		It("returns an error if the JSON-RPC version is incorrect", func() {
			req := Request{
				Version: "1.0",
				ID:      json.RawMessage(`1`),
			}

			err, ok := req.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`request version must be "2.0"`),
				),
			))
		})

		It("returns an error if the request ID is an invalid type", func() {
			req := Request{
				Version: "2.0",
				ID:      json.RawMessage(`{}`),
			}

			err, ok := req.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`request ID must be a JSON string, number or null`),
				),
			))
		})

		It("returns an error if the request ID is not valid JSON", func() {
			req := Request{
				Version: "2.0",
				ID:      json.RawMessage(`{`),
			}

			err, ok := req.Validate()
			Expect(ok).To(BeFalse())
			Expect(err.Code()).To(Equal(ParseErrorCode))
			Expect(err.Unwrap()).To(MatchError("unexpected end of JSON input"))
		})
	})

	Describe("func UnmarshalParameters()", func() {
		It("populates the given value with the unmarshaled parameters", func() {
			req := Request{
				Version:    "2.0",
				Parameters: []byte(`{"Value":123}`),
			}

			var params struct {
				Value int
			}
			err := req.UnmarshalParameters(&params)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(params.Value).To(Equal(123))
		})

		It("returns an error if the parameters can not be unmarshaled", func() {
			req := Request{
				Version:    "2.0",
				Parameters: []byte(`]`),
			}

			var params interface{}
			err := req.UnmarshalParameters(&params)

			var jsonErr Error
			ok := errors.As(err, &jsonErr)
			Expect(ok).To(BeTrue())
			Expect(jsonErr.Code()).To(Equal(InvalidParametersCode))
		})

		It("returns an error if the parameters contain unknown fields", func() {
			req := Request{
				Version:    "2.0",
				Parameters: []byte(`{"Value":123}`),
			}

			var params struct{}
			err := req.UnmarshalParameters(&params)

			var jsonErr Error
			ok := errors.As(err, &jsonErr)
			Expect(ok).To(BeTrue())
			Expect(jsonErr.Code()).To(Equal(InvalidParametersCode))
		})

		When("the target type implements the Validatable interface", func() {
			It("returns nil if validation succeeds", func() {
				req := Request{
					Version:    "2.0",
					Parameters: []byte(`{"Value":123}`),
				}

				var params validatableStub
				err := req.UnmarshalParameters(&params)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("returns an error if validation fails", func() {
				req := Request{
					Version:    "2.0",
					Parameters: []byte(`{"Value":123}`),
				}

				params := validatableStub{
					ValidateFunc: func() error {
						return errors.New("<error>")
					},
				}
				err := req.UnmarshalParameters(&params)

				var jsonErr Error
				ok := errors.As(err, &jsonErr)
				Expect(ok).To(BeTrue())
				Expect(jsonErr.Code()).To(Equal(InvalidParametersCode))
				Expect(jsonErr.Unwrap()).To(MatchError("<error>"))
			})
		})
	})
})

// validatableStub is a test implementation of the Validatable interface.
type validatableStub struct {
	ValidateFunc func() error
	Value        int
}

func (p validatableStub) Validate() error {
	if p.ValidateFunc != nil {
		return p.ValidateFunc()
	}

	return nil
}

var _ = Describe("type RequestSet", func() {
	Describe("func ParseRequestSet()", func() {
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

		It("ignores leading whitespace", func() {
			r := strings.NewReader(`    []`)

			rs, err := ParseRequestSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.IsBatch).To(BeTrue())
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

		It("returns an error if the request has invalid syntax", func() {
			r := strings.NewReader(`}`)

			_, err := ParseRequestSet(r)

			var e Error
			Expect(err).To(BeAssignableToTypeOf(e))

			e = err.(Error)
			Expect(e.Code()).To(Equal(ParseErrorCode))
			Expect(e.Unwrap()).To(MatchError("unable to parse request: invalid character '}' looking for beginning of value"))
		})

		It("returns an error if a single request is malformed", func() {
			r := strings.NewReader(`""`) // not an array or object

			_, err := ParseRequestSet(r)

			var e Error
			Expect(err).To(BeAssignableToTypeOf(e))

			e = err.(Error)
			Expect(e.Code()).To(Equal(ParseErrorCode))
			Expect(e.Unwrap()).To(MatchError("unable to parse request: json: cannot unmarshal string into Go value of type harpy.Request"))
		})

		It("returns an error if a request within a batch is malformed", func() {
			r := strings.NewReader(`[""]`) // not an array or object

			_, err := ParseRequestSet(r)

			var e Error
			Expect(err).To(BeAssignableToTypeOf(e))

			e = err.(Error)
			Expect(e.Code()).To(Equal(ParseErrorCode))
			Expect(e.Unwrap()).To(MatchError("unable to parse request: json: cannot unmarshal string into Go value of type harpy.Request"))
		})
	})

	Describe("func Validate()", func() {
		It("returns true if all requests are valid", func() {
			rs := RequestSet{
				Requests: []Request{
					{Version: "2.0"},
					{Version: "2.0"},
				},
				IsBatch: true,
			}

			_, ok := rs.Validate()
			Expect(ok).To(BeTrue())
		})

		It("returns an error if any of the requests is invalid", func() {
			rs := RequestSet{
				Requests: []Request{
					{Version: "2.0"},
					{},
				},
				IsBatch: true,
			}

			err, ok := rs.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`request version must be "2.0"`),
				),
			))
		})

		It("returns an error if a batch contains no requests", func() {
			rs := RequestSet{
				IsBatch: true,
			}

			err, ok := rs.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`batches must contain at least one request`),
				),
			))
		})

		It("returns an error if a non-batch contains no requests", func() {
			rs := RequestSet{
				IsBatch: false,
			}

			err, ok := rs.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`non-batch request sets must contain exactly one request`),
				),
			))
		})

		It("returns an error if a non-batch contains more than one requests", func() {
			rs := RequestSet{
				Requests: []Request{
					{Version: "2.0"},
					{Version: "2.0"},
				},
				IsBatch: false,
			}

			err, ok := rs.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`non-batch request sets must contain exactly one request`),
				),
			))
		})
	})
})
