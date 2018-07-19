package config

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Enum mapper internals", func() {
	Context("when trying to load a non existing file", func() {
		var (
			err error
		)

		BeforeEach(func() {
			_, err = loadEnumFile("does-not-exist.json")
		})

		It("should error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
})
