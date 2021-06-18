package harpy

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("func writeDataSize()", func() {
	DescribeTable(
		"it writes a human-readable data size", func(n int, expect string) {
			var w strings.Builder
			writeDataSize(&w, n)
			Expect(w.String()).To(Equal(expect))
		},
		Entry("zero", 0, "0 B"),
		Entry("bytes", 995, "995 B"),
		Entry("kilobytes", 999_500, "999.5 KB"),
		Entry("megabytes", 999_500_000, "999.5 MB"),
		Entry("gigabytes", 999_500_000_000, "999.5 GB"),
		Entry("terabytes", 999_500_000_000_000, "999.5 TB"),
		Entry("petabytes", 999_500_000_000_000_000, "999.5 PB"),
	)
})
