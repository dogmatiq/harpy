package harpy_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	. "github.com/dogmatiq/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("func NewSuccessResponse()", func() {
	It("returns a SuccessResponse that contains the marshaled result", func() {
		res := NewSuccessResponse(
			json.RawMessage(`123`),
			456,
		)

		Expect(res).To(Equal(SuccessResponse{
			Version:   `2.0`,
			RequestID: json.RawMessage(`123`),
			Result:    json.RawMessage(`456`),
		}))
	})

	It("returns a SuccessResponse with an empty result when the result is nil", func() {
		res := NewSuccessResponse(
			json.RawMessage(`123`),
			nil,
		)

		Expect(res).To(Equal(SuccessResponse{
			Version:   `2.0`,
			RequestID: json.RawMessage(`123`),
		}))
	})

	It("returns an ErrorResponse if the result can not be marshaled", func() {
		res := NewSuccessResponse(
			json.RawMessage(`123`),
			10i+1, // JSON can not represent complex numbers
		)

		Expect(res).To(MatchAllFields(
			Fields{
				"Version":   Equal(`2.0`),
				"RequestID": Equal(json.RawMessage(`123`)),
				"Error": Equal(ErrorInfo{
					Code:    InternalErrorCode,
					Message: "internal server error",
				}),
				"ServerError": MatchError("could not marshal success result value: json: unsupported type: complex128"),
			},
		))
	})
})

var _ = Describe("func NewErrorResponse()", func() {
	When("the error is a native JSON-RPC error", func() {
		It("returns an ErrorResponse", func() {
			res := NewErrorResponse(
				json.RawMessage(`123`),
				NewError(789, WithMessage("<error>")),
			)

			Expect(res).To(Equal(ErrorResponse{
				Version:   `2.0`,
				RequestID: json.RawMessage(`123`),
				Error: ErrorInfo{
					Code:    789,
					Message: "<error>",
				},
			}))
		})

		It("returns an ErrorResponse that contains marshaled user-defined data", func() {
			res := NewErrorResponse(
				json.RawMessage(`123`),
				NewError(
					789,
					WithMessage("<error>"),
					WithData([]int{100, 200, 300}),
				),
			)

			Expect(res).To(Equal(ErrorResponse{
				Version:   `2.0`,
				RequestID: json.RawMessage(`123`),
				Error: ErrorInfo{
					Code:    789,
					Message: "<error>",
					Data:    json.RawMessage(`[100,200,300]`),
				},
			}))
		})

		It("returns an ErrorResponse indicating an internal error when user-defined data can not be marshaled", func() {
			res := NewErrorResponse(
				json.RawMessage(`123`),
				NewError(
					789,
					WithMessage("<error>"),
					WithData(10i+1), // JSON can not represent complex numbers
				),
			)

			Expect(res).To(MatchAllFields(
				Fields{
					"Version":   Equal(`2.0`),
					"RequestID": Equal(json.RawMessage(`123`)),
					"Error": Equal(ErrorInfo{
						Code:    InternalErrorCode,
						Message: "internal server error",
					}),
					"ServerError": MatchError("could not marshal user-defined error data in [789] <error>: json: unsupported type: complex128"),
				},
			))
		})
	})

	When("the error is not a native JSON-RPC error, and it is not an internal error", func() {
		DescribeTable(
			"it returns an ErrorResponse that includes the error message",
			func(err error) {
				res := NewErrorResponse(
					json.RawMessage(`123`),
					err,
				)

				Expect(res).To(Equal(ErrorResponse{
					Version:   `2.0`,
					RequestID: json.RawMessage(`123`),
					Error: ErrorInfo{
						Code:    InternalErrorCode,
						Message: err.Error(),
					},
				}))
			},
			Entry("context deadline exceeded", context.DeadlineExceeded),
			Entry("context canceled", context.Canceled),
		)
	})

	When("the error is any other error that is not a native JSON-RPC error", func() {
		It("returns an ErrorResponse that does NOT include the error message", func() {
			err := errors.New("<error>")

			res := NewErrorResponse(
				json.RawMessage(`123`),
				err,
			)

			Expect(res).To(Equal(ErrorResponse{
				Version:   `2.0`,
				RequestID: json.RawMessage(`123`),
				Error: ErrorInfo{
					Code:    InternalErrorCode,
					Message: "internal server error",
				},
				ServerError: err,
			}))
		})
	})
})

var _ = Describe("type ErrorInfo", func() {
	Describe("func String()", func() {
		It("includes the error code description when there is no message", func() {
			i := ErrorInfo{
				Code:    100,
				Message: "",
			}

			Expect(i.String()).To(Equal("[100] unknown error"))
		})

		It("does not include the message if it is the same as the error code description", func() {
			i := ErrorInfo{
				Code:    100,
				Message: "unknown error",
			}

			Expect(i.String()).To(Equal("[100] unknown error"))
		})

		It("includes both the error code description and the message when the error code is predefined", func() {
			i := ErrorInfo{
				Code:    InternalErrorCode,
				Message: "<message>",
			}

			Expect(i.String()).To(Equal("[-32603] internal server error: <message>"))
		})

		It("includes only the message when the error code is not predefined", func() {
			i := ErrorInfo{
				Code:    100,
				Message: "<message>",
			}

			Expect(i.String()).To(Equal("[100] <message>"))
		})
	})
})

var _ = Describe("type ResponseSet", func() {
	Describe("func UnmarshalResponseSet()", func() {
		It("parses a single success response", func() {
			r := strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": 123,
				"result": [1, 2, 3]
			}`)

			rs, err := UnmarshalResponseSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.IsBatch).To(BeFalse())
			Expect(rs.Responses).To(ConsistOf(
				SuccessResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`123`),
					Result:    json.RawMessage(`[1, 2, 3]`),
				},
			))
		})

		It("parses a single error response", func() {
			r := strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": 123,
				"error": {
					"code": 456,
					"message": "<error message>",
					"data": [1, 2, 3]
				}
			}`)

			rs, err := UnmarshalResponseSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.IsBatch).To(BeFalse())
			Expect(rs.Responses).To(ConsistOf(
				ErrorResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`123`),
					Error: ErrorInfo{
						Code:    456,
						Message: "<error message>",
						Data:    json.RawMessage(`[1, 2, 3]`),
					},
				},
			))
		})

		It("parses a batch response with a single response", func() {
			r := strings.NewReader(`[{
				"jsonrpc": "2.0",
				"id": 123,
				"result": [1, 2, 3]
			}]`)

			rs, err := UnmarshalResponseSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.IsBatch).To(BeTrue())
			Expect(rs.Responses).To(ConsistOf(
				SuccessResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`123`),
					Result:    json.RawMessage(`[1, 2, 3]`),
				},
			))
		})

		It("parses a batch response with multiple responses", func() {
			r := strings.NewReader(`[{
				"jsonrpc": "2.0",
				"id": 123,
				"result": [1, 2, 3]
			},{
				"jsonrpc": "2.0",
				"id": 456,
				"error": {
					"code": 789,
					"message": "<error message>",
					"data": [4, 5, 6]
				}
			}]`)

			rs, err := UnmarshalResponseSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.IsBatch).To(BeTrue())
			Expect(rs.Responses).To(ConsistOf(
				SuccessResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`123`),
					Result:    json.RawMessage(`[1, 2, 3]`),
				},
				ErrorResponse{
					Version:   "2.0",
					RequestID: json.RawMessage(`456`),
					Error: ErrorInfo{
						Code:    789,
						Message: "<error message>",
						Data:    json.RawMessage(`[4, 5, 6]`),
					},
				},
			))
		})

		It("ignores leading whitespace", func() {
			r := strings.NewReader(`    []`)

			rs, err := UnmarshalResponseSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.IsBatch).To(BeTrue())
		})

		It("includes the ID field if it set to NULL", func() {
			r := strings.NewReader(`{"id": null}`)

			rs, err := UnmarshalResponseSet(r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rs.Responses).To(ConsistOf(
				SuccessResponse{
					RequestID: json.RawMessage(`null`),
				},
			))
		})

		It("returns an error if the response can not be read", func() {
			r := strings.NewReader(``)

			_, err := UnmarshalResponseSet(r)
			Expect(err).To(Equal(io.EOF))
		})

		It("returns an error if the response has invalid syntax", func() {
			r := strings.NewReader(`}`)

			_, err := UnmarshalResponseSet(r)
			Expect(err).To(MatchError("unable to parse response: invalid character '}' looking for beginning of value"))
		})

		It("returns an error if a single response is malformed", func() {
			r := strings.NewReader(`""`) // not an array or object

			_, err := UnmarshalResponseSet(r)
			Expect(err).To(MatchError("unable to parse response: json: cannot unmarshal string into Go value of type harpy.successOrErrorResponse"))
		})

		It("returns an error if a response within a batch is malformed", func() {
			r := strings.NewReader(`[""]`) // not an array or object

			_, err := UnmarshalResponseSet(r)
			Expect(err).To(MatchError("unable to parse response: json: cannot unmarshal string into Go value of type harpy.successOrErrorResponse"))
		})
	})
})
