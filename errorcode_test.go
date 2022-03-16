package harpy_test

import (
	. "github.com/dogmatiq/harpy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("type ErrorCode", func() {
	Describe("func String()", func() {
		DescribeTable(
			"it returns a description of the error code",
			func(c ErrorCode, d string) {
				Expect(c.String()).To(Equal(d))
			},
			Entry("parse error", ParseErrorCode, "parse error"),
			Entry("invalid request", InvalidRequestCode, "invalid request"),
			Entry("method not found", MethodNotFoundCode, "method not found"),
			Entry("invalid parameters", InvalidParametersCode, "invalid parameters"),
			Entry("internal server error", InternalErrorCode, "internal server error"),
			Entry("undefined reserved code", ErrorCode(-32000), "undefined reserved error"),
			Entry("user-defined error", ErrorCode(100), "unknown error"),
		)
	})
})
