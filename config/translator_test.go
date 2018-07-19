package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"path/filepath"

	"github.com/tableaux-project/tableaux/config"
)

var _ = Describe("Translator", func() {
	var (
		mapper config.Translator
		err    error
	)

	BeforeEach(func() {
		mapper, err = config.NewTranslatorFromFolder(filepath.Join("testfiles", "i18n-test-files"))
	})

	Context("when trying to load the test files", func() {
		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("contain exactly two languages", func() {
			Expect(len(mapper.Languages())).To(Equal(2))
		})

		It("should contain the DE language catalog", func() {
			languageCatalog, err := mapper.Language("de")
			Expect(err).NotTo(HaveOccurred())
			Expect(languageCatalog).ToNot(BeNil())
		})

		It("should error, when trying to access a non existing language", func() {
			_, err := mapper.Language("wat")

			Expect(err).To(Equal(config.ErrUnknownLanguage))
		})

		It("should be able to access a single, specific key from a language catalog directly", func() {
			translation, err := mapper.Translate("de", "enum.addresstype.street.short")

			Expect(err).NotTo(HaveOccurred())
			Expect(translation).To(Equal("Strassenanschrift"))
		})

		It("should error, when trying to directly access a specific key from a non existing language", func() {
			translationKey, err := mapper.Translate("wat", "doesntMatter")

			Expect(err).To(Equal(config.ErrUnknownLanguage))
			Expect(translationKey).To(Equal(""))
		})
	})

	Context("when trying to load the broken test files", func() {
		var (
			err error
		)

		BeforeEach(func() {
			_, err = config.NewTranslatorFromFolder(filepath.Join("testfiles", "i18n-broken-files"))
		})

		It("should error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&json.SyntaxError{}))
		})
	})

	Context("when working with a language catalog", func() {
		var (
			languageCatalog config.LanguageCatalog
		)

		JustBeforeEach(func() {
			languageCatalog, err = mapper.Language("de")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be able to access all keys at once", func() {
			Expect(len(languageCatalog.Entries())).To(Equal(38))
		})

		It("should be able to access a single, specific key", func() {
			translationKey, err := languageCatalog.Translate("enum.addresstype.street.short")

			Expect(err).NotTo(HaveOccurred())
			Expect(translationKey).To(Equal("Strassenanschrift"))
		})

		It("should error when trying to access an unknown key", func() {
			translationKey, err := languageCatalog.Translate("wat")

			Expect(err).To(Equal(config.ErrUnknownTranslation))
			Expect(translationKey).To(Equal("??wat??"))
		})
	})
})
