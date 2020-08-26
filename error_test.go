package voorhees_test

import (
	. "github.com/onsi/gingko"
	. "github.com/onsi/gomega"
)

Describe("type Error", func() {
	Describe("func Code()", func() {
		It("returns the error code", func() {
		})
	})

	Describe("func Message()", func() {
		It("returns the user-defined message", func() {
		})

		It("returns the error code description when there is no user-defined message", func() {
		})
	})

	Describe("func Data()", func() {
		It("returns the user-defined data", func() {
		})

		It("returns nil if there is no user-defined data", func() {
		})
	})

	Describe("func Error()", func() {
		It("includes the error code description when there is no user-defined message", func() {
		})

		It("includes both the error code description and the user-defined message when the error code is predefined", func() {
		})

		It("includes only the user-defined message when the error code is not predefined", func() {
		})
	})
})
