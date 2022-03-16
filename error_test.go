package harpy_test

import (
	"errors"

	. "github.com/dogmatiq/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Error", func() {
	Describe("func NewError()", func() {
		It("panics if the error code is reserved", func() {
			Expect(func() {
				NewError(InternalErrorCode)
			}).To(PanicWith("the error code -32603 is reserved by the JSON-RPC specification (internal server error)"))
		})
	})

	Describe("func NewErrorWithReservedCode()", func() {
		It("panics if the error code is not reserved", func() {
			Expect(func() {
				NewErrorWithReservedCode(100)
			}).To(PanicWith("the error code 100 is not reserved by the JSON-RPC specification"))
		})
	})

	Describe("func MethodNotFound()", func() {
		It("returns an error with the correct error code", func() {
			e := MethodNotFound()
			Expect(e.Code()).To(Equal(MethodNotFoundCode))
		})
	})

	Describe("func InvalidParameters()", func() {
		It("returns an error with the correct error code", func() {
			e := InvalidParameters()
			Expect(e.Code()).To(Equal(InvalidParametersCode))
		})
	})

	Describe("func Code()", func() {
		It("returns the error code", func() {
			e := NewError(100)
			Expect(e.Code()).To(BeEquivalentTo(100))
		})
	})

	Describe("func Message()", func() {
		It("returns the user-defined message", func() {
			e := NewError(100, WithMessage("<message>"))
			Expect(e.Message()).To(Equal("<message>"))
		})

		It("returns the error code description when there is no user-defined message", func() {
			e := NewError(100)
			Expect(e.Message()).To(Equal("unknown error"))
		})

		When("a causal error is provided", func() {
			It("returns the error message of the cause", func() {
				cause := errors.New("<cause>")
				e := NewError(100, WithCause(cause))
				Expect(e.Message()).To(Equal("<cause>"))
			})

			It("does not override a user-defined message", func() {
				cause := errors.New("<cause>")
				e := NewError(100, WithMessage("<message>"), WithCause(cause))
				Expect(e.Message()).To(Equal("<message>"))
			})
		})
	})

	Describe("func Data()", func() {
		It("returns the user-defined data", func() {
			e := NewError(100, WithData("<data>"))
			Expect(e.Data()).To(Equal("<data>"))
		})

		It("returns nil if there is no user-defined data", func() {
			e := NewError(100)
			Expect(e.Data()).To(BeNil())
		})
	})

	Describe("func Error()", func() {
		It("includes the error code description when there is no user-defined message", func() {
			e := NewError(100)
			Expect(e.Error()).To(Equal("[100] unknown error"))
		})

		It("includes both the error code description and the user-defined message when the error code is predefined", func() {
			e := NewErrorWithReservedCode(InternalErrorCode, WithMessage("<message>"))
			Expect(e.Error()).To(Equal("[-32603] internal server error: <message>"))
		})

		It("includes only the user-defined message when the error code is not predefined", func() {
			e := NewError(100, WithMessage("<message>"))
			Expect(e.Error()).To(Equal("[100] <message>"))
		})
	})

	Describe("func Unwrap()", func() {
		It("returns the causal error", func() {
			cause := errors.New("<cause>")
			e := NewError(100, WithCause(cause))
			Expect(e.Unwrap()).To(BeIdenticalTo(cause))
		})
	})
})
