package harpy_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	. "github.com/dogmatiq/harpy"
	"github.com/dogmatiq/iago/iotest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Request", func() {
	Describe("func NewCallRequest()", func() {
		It("returns a call request", func() {
			req, err := NewCallRequest(
				123,
				"<method>",
				[]int{1, 2, 3},
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(req).To(Equal(Request{
				Version:    "2.0",
				ID:         json.RawMessage(`123`),
				Method:     "<method>",
				Parameters: json.RawMessage(`[1,2,3]`),
			}))
		})

		DescribeTable(
			"encodes valid request ID values",
			func(id any, expect json.RawMessage) {
				req, err := NewCallRequest(
					id,
					"<method>",
					[]int{},
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(req.ID).To(Equal(expect))

			},
			Entry("nil", nil, json.RawMessage(`null`)),
			Entry("string", "<id>", json.RawMessage(`"\u003cid\u003e"`)),
			Entry("number", 123, json.RawMessage(`123`)),
		)

		It("returns an error if the ID cannot be marshaled", func() {
			_, err := NewCallRequest(
				make(chan struct{}),
				"<method>",
				[]int{},
			)
			Expect(err).To(MatchError("unable to marshal request ID: json: unsupported type: chan struct {}"))
		})

		It("returns an error if the parameters cannot be marshaled", func() {
			_, err := NewCallRequest(
				json.RawMessage(`123`),
				"<method>",
				make(chan struct{}),
			)
			Expect(err).To(MatchError("unable to marshal request parameters: json: unsupported type: chan struct {}"))
		})
	})

	Describe("func NewNotifyRequest()", func() {
		It("returns a notification request", func() {
			req, err := NewNotifyRequest(
				"<method>",
				[]int{1, 2, 3},
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(req).To(Equal(Request{
				Version:    "2.0",
				Method:     "<method>",
				Parameters: json.RawMessage(`[1,2,3]`),
			}))
		})

		It("returns an error if the parameters cannot be marshaled", func() {
			_, err := NewNotifyRequest(
				"<method>",
				make(chan struct{}),
			)
			Expect(err).To(MatchError("unable to marshal request parameters: json: unsupported type: chan struct {}"))
		})
	})

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

	Describe("func ValidateServerSide()", func() {
		DescribeTable(
			"it returns true when the request is valid (request IDs)",
			func(id json.RawMessage) {
				req := Request{
					Version: "2.0",
					ID:      id,
				}

				err, ok := req.ValidateServerSide()
				if !ok {
					Expect(err).To(Equal(Error{}))
					Expect(ok).To(BeTrue())
				}
			},
			Entry("string ID", json.RawMessage(`"<id>"`)),
			Entry("integer ID", json.RawMessage(`1`)),
			Entry("decimal ID", json.RawMessage(`1.2`)),
			Entry("null ID", json.RawMessage(`null`)),
			Entry("absent ID (nil)", nil),
			Entry("absent ID (empty)", json.RawMessage(``)),
		)

		DescribeTable(
			"it returns true when the request is valid (parameters)",
			func(params json.RawMessage) {
				req := Request{
					Version:    "2.0",
					Parameters: params,
				}

				err, ok := req.ValidateServerSide()
				if !ok {
					Expect(err).To(Equal(Error{}))
					Expect(ok).To(BeTrue())
				}
			},
			Entry("array params", json.RawMessage(`[]`)),
			Entry("object params", json.RawMessage(`{}`)),
			Entry("null params", json.RawMessage(`null`)),
			Entry("absent params (nil)", nil),
			Entry("absent params (empty)", json.RawMessage(``)),
		)

		It("returns an error if the JSON-RPC version is incorrect", func() {
			req := Request{
				Version: "1.0",
				ID:      json.RawMessage(`1`),
			}

			err, ok := req.ValidateServerSide()
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`request version must be "2.0"`),
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if the request ID is an invalid type", func() {
			req := Request{
				Version: "2.0",
				ID:      json.RawMessage(`{}`),
			}

			err, ok := req.ValidateServerSide()
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`request ID must be a JSON string, number or null`),
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if the request ID is not valid JSON", func() {
			req := Request{
				Version: "2.0",
				ID:      json.RawMessage(`{`),
			}

			err, ok := req.ValidateServerSide()
			Expect(err).Should(HaveOccurred())
			Expect(err.Code()).To(Equal(ParseErrorCode))
			Expect(err.Unwrap()).To(MatchError("unexpected end of JSON input"))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if the parameters are an invalid type", func() {
			req := Request{
				Version:    "2.0",
				Parameters: json.RawMessage(`123`),
			}

			err, ok := req.ValidateServerSide()
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidParametersCode,
					WithMessage(`parameters must be an array, an object, or null`),
				),
			))
			Expect(ok).To(BeFalse())
		})

		// See https://github.com/dogmatiq/harpy/issues/13
		It("returns an error if the parameters are an invalid type and the request is call", func() {
			req := Request{
				Version:    "2.0",
				ID:         json.RawMessage(`123`),
				Parameters: json.RawMessage(`456`),
			}

			err, ok := req.ValidateServerSide()
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidParametersCode,
					WithMessage(`parameters must be an array, an object, or null`),
				),
			))
			Expect(ok).To(BeFalse())
		})
	})

	Describe("func ValidateClientSide()", func() {
		DescribeTable(
			"it returns true when the request is valid (request IDs)",
			func(id json.RawMessage) {
				req := Request{
					Version: "2.0",
					ID:      id,
				}

				err, ok := req.ValidateClientSide()
				if !ok {
					Expect(err).To(Equal(Error{}))
					Expect(ok).To(BeTrue())
				}
			},
			Entry("string ID", json.RawMessage(`"<id>"`)),
			Entry("integer ID", json.RawMessage(`1`)),
			Entry("decimal ID", json.RawMessage(`1.2`)),
			Entry("null ID", json.RawMessage(`null`)),
			Entry("absent ID (nil)", nil),
			Entry("absent ID (empty)", json.RawMessage(``)),
		)

		DescribeTable(
			"it returns true when the request is valid (parameters)",
			func(params json.RawMessage) {
				req := Request{
					Version:    "2.0",
					Parameters: params,
				}

				err, ok := req.ValidateClientSide()
				if !ok {
					Expect(err).To(Equal(Error{}))
					Expect(ok).To(BeTrue())
				}
			},
			Entry("array params", json.RawMessage(`[]`)),
			Entry("object params", json.RawMessage(`{}`)),
			Entry("null params", json.RawMessage(`null`)),
			Entry("absent params (nil)", nil),
			Entry("absent params (empty)", json.RawMessage(``)),
		)

		It("returns an error if the JSON-RPC version is incorrect", func() {
			req := Request{
				Version: "1.0",
				ID:      json.RawMessage(`1`),
			}

			err, ok := req.ValidateClientSide()
			Expect(err).To(Equal(
				NewClientSideError(
					InvalidRequestCode,
					`request version must be "2.0"`,
					nil,
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if the request ID is an invalid type", func() {
			req := Request{
				Version: "2.0",
				ID:      json.RawMessage(`{}`),
			}

			err, ok := req.ValidateClientSide()
			Expect(err).To(Equal(
				NewClientSideError(
					InvalidRequestCode,
					`request ID must be a JSON string, number or null`,
					nil,
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if the request ID is not valid JSON", func() {
			req := Request{
				Version: "2.0",
				ID:      json.RawMessage(`{`),
			}

			err, ok := req.ValidateClientSide()
			Expect(err).To(HaveOccurred())
			Expect(err.Code()).To(Equal(ParseErrorCode))
			Expect(err.Unwrap()).To(MatchError("unexpected end of JSON input"))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if the parameters are an invalid type", func() {
			req := Request{
				Version:    "2.0",
				Parameters: json.RawMessage(`123`),
			}

			err, ok := req.ValidateClientSide()
			Expect(err).To(Equal(
				NewClientSideError(
					InvalidParametersCode,
					`parameters must be an array, an object, or null`,
					nil,
				),
			))
			Expect(ok).To(BeFalse())
		})

		// See https://github.com/dogmatiq/harpy/issues/13
		It("returns an error if the parameters are an invalid type and the request is call", func() {
			req := Request{
				Version:    "2.0",
				ID:         json.RawMessage(`123`),
				Parameters: json.RawMessage(`456`),
			}

			err, ok := req.ValidateClientSide()
			Expect(err).To(Equal(
				NewClientSideError(
					InvalidParametersCode,
					`parameters must be an array, an object, or null`,
					nil,
				),
			))
			Expect(ok).To(BeFalse())
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

			var params any
			err := req.UnmarshalParameters(&params)

			var rpcErr Error
			ok := errors.As(err, &rpcErr)
			Expect(ok).To(BeTrue())
			Expect(rpcErr.Code()).To(Equal(InvalidParametersCode))
		})

		It("returns an error if the parameters contain unknown fields", func() {
			req := Request{
				Version:    "2.0",
				Parameters: []byte(`{"Value":123}`),
			}

			var params struct{}
			err := req.UnmarshalParameters(&params)

			var rpcErr Error
			ok := errors.As(err, &rpcErr)
			Expect(ok).To(BeTrue())
			Expect(rpcErr.Code()).To(Equal(InvalidParametersCode))
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

				var rpcErr Error
				ok := errors.As(err, &rpcErr)
				Expect(ok).To(BeTrue())
				Expect(rpcErr.Code()).To(Equal(InvalidParametersCode))
				Expect(rpcErr.Unwrap()).To(MatchError("<error>"))
			})

			It("supports the AllowUnknownFields() option", func() {
				req := Request{
					Version:    "2.0",
					Parameters: []byte(`{"Value":123, "Unknown": 456}`),
				}

				var params struct {
					Value int
				}
				err := req.UnmarshalParameters(&params, AllowUnknownFields(true))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(params.Value).To(Equal(123))
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
	Describe("func UnmarshalRequestSet()", func() {
		It("parses a single request", func() {
			r := strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": 123,
				"method": "<method>",
				"params": [1, 2, 3]
			}`)

			rs, err := UnmarshalRequestSet(r)
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

			rs, err := UnmarshalRequestSet(r)
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

			rs, err := UnmarshalRequestSet(r)
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

			rs, err := UnmarshalRequestSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.IsBatch).To(BeTrue())
		})

		It("omits the ID field if it is not present in the request", func() {
			r := strings.NewReader(`{}`)

			rs, err := UnmarshalRequestSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.Requests[0].ID).To(BeNil())
		})

		It("includes the ID field if it set to NULL", func() {
			r := strings.NewReader(`{"id": null}`)

			rs, err := UnmarshalRequestSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.Requests[0].ID).To(Equal(json.RawMessage(`null`)))
		})

		It("returns an error if the request can not be read", func() {
			r := strings.NewReader(``)

			_, err := UnmarshalRequestSet(r)
			Expect(err).To(Equal(io.EOF))
		})

		It("returns an error if the request has invalid syntax", func() {
			r := strings.NewReader(`}`)

			_, err := UnmarshalRequestSet(r)

			var rpcErr Error
			ok := errors.As(err, &rpcErr)
			Expect(ok).To(BeTrue())
			Expect(rpcErr.Code()).To(Equal(ParseErrorCode))
			Expect(rpcErr.Unwrap()).To(MatchError("unable to parse request: invalid character '}' looking for beginning of value"))
		})

		It("returns an error if a single request is malformed", func() {
			r := strings.NewReader(`""`) // not an array or object

			_, err := UnmarshalRequestSet(r)

			var rpcErr Error
			ok := errors.As(err, &rpcErr)
			Expect(ok).To(BeTrue())
			Expect(rpcErr.Code()).To(Equal(ParseErrorCode))
			Expect(rpcErr.Unwrap()).To(MatchError("unable to parse request: json: cannot unmarshal string into Go value of type harpy.Request"))
		})

		It("returns an error if a request within a batch is malformed", func() {
			r := strings.NewReader(`[""]`) // not an array or object

			_, err := UnmarshalRequestSet(r)

			var rpcErr Error
			ok := errors.As(err, &rpcErr)
			Expect(ok).To(BeTrue())
			Expect(rpcErr.Code()).To(Equal(ParseErrorCode))
			Expect(rpcErr.Unwrap()).To(MatchError("unable to parse request: json: cannot unmarshal string into Go value of type harpy.Request"))
		})
	})

	Describe("func ValidateServerSide()", func() {
		It("returns true if all requests are valid", func() {
			rs := RequestSet{
				Requests: []Request{
					{Version: "2.0"},
					{Version: "2.0"},
				},
				IsBatch: true,
			}

			err, ok := rs.ValidateServerSide()
			if !ok {
				Expect(err).To(Equal(Error{}))
				Expect(ok).To(BeTrue())
			}
		})

		It("returns an error if any of the requests is invalid", func() {
			rs := RequestSet{
				Requests: []Request{
					{Version: "2.0"},
					{},
				},
				IsBatch: true,
			}

			err, ok := rs.ValidateServerSide()
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`request version must be "2.0"`),
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if a batch contains no requests", func() {
			rs := RequestSet{
				IsBatch: true,
			}

			err, ok := rs.ValidateServerSide()
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`batches must contain at least one request`),
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if a non-batch contains no requests", func() {
			rs := RequestSet{
				IsBatch: false,
			}

			err, ok := rs.ValidateServerSide()
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`non-batch request sets must contain exactly one request`),
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if a non-batch contains more than one request", func() {
			rs := RequestSet{
				Requests: []Request{
					{Version: "2.0"},
					{Version: "2.0"},
				},
				IsBatch: false,
			}

			err, ok := rs.ValidateServerSide()
			Expect(err).To(Equal(
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage(`non-batch request sets must contain exactly one request`),
				),
			))
			Expect(ok).To(BeFalse())
		})
	})

	Describe("func ValidateClientSide()", func() {
		It("returns true if all requests are valid", func() {
			rs := RequestSet{
				Requests: []Request{
					{Version: "2.0"},
					{Version: "2.0"},
				},
				IsBatch: true,
			}

			err, ok := rs.ValidateClientSide()
			if !ok {
				Expect(err).To(Equal(Error{}))
				Expect(ok).To(BeTrue())
			}
		})

		It("returns an error if any of the requests is invalid", func() {
			rs := RequestSet{
				Requests: []Request{
					{Version: "2.0"},
					{},
				},
				IsBatch: true,
			}

			err, ok := rs.ValidateClientSide()
			Expect(err).To(Equal(
				NewClientSideError(
					InvalidRequestCode,
					`request version must be "2.0"`,
					nil,
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if a batch contains no requests", func() {
			rs := RequestSet{
				IsBatch: true,
			}

			err, ok := rs.ValidateClientSide()
			Expect(err).To(Equal(
				NewClientSideError(
					InvalidRequestCode,
					`batches must contain at least one request`,
					nil,
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if a non-batch contains no requests", func() {
			rs := RequestSet{
				IsBatch: false,
			}

			err, ok := rs.ValidateClientSide()
			Expect(err).To(Equal(
				NewClientSideError(
					InvalidRequestCode,
					`non-batch request sets must contain exactly one request`,
					nil,
				),
			))
			Expect(ok).To(BeFalse())
		})

		It("returns an error if a non-batch contains more than one request", func() {
			rs := RequestSet{
				Requests: []Request{
					{Version: "2.0"},
					{Version: "2.0"},
				},
				IsBatch: false,
			}

			err, ok := rs.ValidateClientSide()
			Expect(err).To(Equal(
				NewClientSideError(
					InvalidRequestCode,
					`non-batch request sets must contain exactly one request`,
					nil,
				),
			))
			Expect(ok).To(BeFalse())
		})
	})
})

var _ = Describe("type BatchRequestMarshaler", func() {
	var (
		buf        *bytes.Buffer
		marshaler  *BatchRequestMarshaler
		req1, req2 Request
	)

	BeforeEach(func() {
		buf = &bytes.Buffer{}
		marshaler = &BatchRequestMarshaler{
			Target: buf,
		}

		var err error
		req1, err = NewCallRequest(123, "<call>", []int{1, 2, 3})
		Expect(err).ShouldNot(HaveOccurred())

		req2, err = NewNotifyRequest("<notify>", []int{4, 5, 6})
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("func MarshalRequest()", func() {
		It("marshals requests into a batch", func() {
			err := marshaler.MarshalRequest(req1)
			Expect(err).ShouldNot(HaveOccurred())

			err = marshaler.MarshalRequest(req2)
			Expect(err).ShouldNot(HaveOccurred())

			err = marshaler.Close()
			Expect(err).ShouldNot(HaveOccurred())

			rs, err := UnmarshalRequestSet(buf)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(rs.IsBatch).To(BeTrue())
			Expect(rs.Requests).To(ContainElements(req1, req2))
		})

		It("returns an error if the request cannot be marshaled", func() {
			req1.ID = json.RawMessage(`}`)

			err := marshaler.MarshalRequest(req1)
			Expect(err).To(MatchError(`json: error calling MarshalJSON for type json.RawMessage: invalid character '}' looking for beginning of value`))
		})

		It("returns an error if the marshaled request can not be written", func() {
			marshaler.Target = iotest.NewFailer(nil, nil)

			err := marshaler.MarshalRequest(req1)
			Expect(err).To(MatchError(`<induced write error>`))
		})

		It("panics if the marshaler has been closed", func() {
			marshaler.Close()

			Expect(func() {
				marshaler.MarshalRequest(req1)
			}).To(PanicWith("marshaler has been closed"))
		})
	})

	Describe("func Close()", func() {
		It("does not write anything if no requests have been written", func() {
			err := marshaler.Close()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(buf.Bytes()).To(BeEmpty())
		})

		It("returns an error if the closing bracket can not be written", func() {
			err := marshaler.MarshalRequest(req1)

			marshaler.Target = iotest.NewFailer(nil, nil)

			err = marshaler.Close()
			Expect(err).To(MatchError(`<induced write error>`))
		})
	})
})
