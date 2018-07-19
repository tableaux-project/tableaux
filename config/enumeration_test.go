package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"path/filepath"

	"github.com/tableaux-project/tableaux/config"
)

var _ = Describe("Enum mapper", func() {
	var (
		mapper config.EnumMapper
		err    error
	)

	BeforeEach(func() {
		mapper, err = config.NewEnumMapperFromFolder(filepath.Join("testfiles", "enum-test-files"))
	})

	Context("when trying to load the test files", func() {
		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("contain exactly two enums", func() {
			Expect(len(mapper.Enums())).To(Equal(2))
		})

		It("should contain the test enum file", func() {
			validEnum, err := mapper.Enum("country")
			Expect(err).NotTo(HaveOccurred())
			Expect(validEnum).ToNot(BeNil())
		})

		It("should contain the enum file from the sub folder (preserving casing)", func() {
			validEnum, err := mapper.Enum("addressType")
			Expect(err).NotTo(HaveOccurred())
			Expect(validEnum).ToNot(BeNil())
		})

		It("should be able to access a single, specific key from a enum directly", func() {
			translationKey, err := mapper.TranslationKeyInEnum("country", "DE")

			Expect(err).NotTo(HaveOccurred())
			Expect(translationKey).To(Equal("enum.country.de"))
		})

		It("should error, when trying to directly access a specific key from a non existing enum", func() {
			translationKey, err := mapper.TranslationKeyInEnum("wat", "doesntMatter")

			Expect(err).To(Equal(config.ErrUnknownEnum))
			Expect(translationKey).To(Equal(""))
		})
	})

	Context("when trying to load the broken test files", func() {
		var (
			err error
		)

		BeforeEach(func() {
			_, err = config.NewEnumMapperFromFolder(filepath.Join("testfiles", "enum-broken-files"))
		})

		It("should error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&json.SyntaxError{}))
		})
	})

	Context("when working with an enum", func() {
		var (
			enum config.Enum
		)

		JustBeforeEach(func() {
			enum, err = mapper.Enum("country")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be able to access all keys at once", func() {
			Expect(len(enum.Entries())).To(Equal(2))
		})

		It("should be able to access a single, specific key", func() {
			translationKey, err := enum.TranslationKey("DE")

			Expect(err).NotTo(HaveOccurred())
			Expect(translationKey).To(Equal("enum.country.de"))
		})

		It("should error when trying to access an unknown key", func() {
			translationKey, err := enum.TranslationKey("wat")

			Expect(err).To(Equal(config.ErrUnknownEnumKey))
			Expect(translationKey).To(Equal(""))
		})
	})
})
