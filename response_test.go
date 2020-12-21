package harpy_test

import (
	"context"
	"encoding/json"
	"errors"

	. "github.com/jmalloc/harpy"
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
