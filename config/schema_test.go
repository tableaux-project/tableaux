package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tableaux-project/tableaux/config"
)

var _ = Describe("Schema", func() {
	var (
		mapper config.SchemaMapper
		err    error
	)

	Describe("While working with broken files", func() {
		Context("when trying to load the broken test files", func() {
			var (
				err error
			)

			BeforeEach(func() {
				_, err = config.NewSchemaMapperFromFolder(filepath.Join("testfiles", "schema-broken-files", "simplybroken"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&json.SyntaxError{}))
			})
		})

		Context("when trying to load a non existent path", func() {
			var (
				err error
			)

			BeforeEach(func() {
				_, err = config.NewSchemaMapperFromFolder("i can't exist")
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
			})
		})

		Context("when trying to load a path which contains files with missing references", func() {
			var (
				err error
			)

			BeforeEach(func() {
				_, err = config.NewSchemaMapperFromFolder(filepath.Join("testfiles", "schema-wrong-reference"))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&config.UnresolvableSchemaError{}))
				Expect(err.Error()).To(Equal("cannot resolve table schema subfolder/abstract_entity"))
			})
		})

		Context("when trying to validate a file which contains an unknown enum type", func() {
			var (
				err error
			)

			BeforeEach(func() {
				mapper, mapperErr := config.NewSchemaMapperFromFolder(filepath.Join("testfiles", "schema-unknown-type"))
				Expect(mapperErr).ToNot(HaveOccurred())

				err = mapper.ValidateIntegrity(config.EnumMapper{})
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&config.UnknownColumnTypeError{}))
				Expect(err.Error()).To(Equal("Unknown column type CompanyClassification in column company_companyClassification of schema company"))
			})
		})
	})

	Describe("While working with correct files", func() {
		BeforeEach(func() {
			mapper, err = config.NewSchemaMapperFromFolder(filepath.Join("testfiles", "schema-test-files"))
		})

		Context("when trying to load the test files", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("contain exactly two schemas", func() {
				Expect(len(mapper.Schemas())).To(Equal(2))
				Expect(len(mapper.ResolvedSchemas())).To(Equal(2))
			})

			It("should contain the test schema file", func() {
				validSchema, err := mapper.Schema("companies")
				Expect(err).NotTo(HaveOccurred())
				Expect(validSchema).ToNot(BeNil())
			})

			It("should contain the schema file from the sub folder)", func() {
				validSchema, err := mapper.Schema("subfolder/abstract_entity")
				Expect(err).NotTo(HaveOccurred())
				Expect(validSchema).ToNot(BeNil())
			})

			It("should error, when trying to access an unknown schema", func() {
				tableSchema, err := mapper.Schema("wat")

				Expect(err).To(Equal(config.ErrUnknownSchema))
				Expect(tableSchema).To(Equal(config.TableSchema{}))
			})

			It("should error, when trying to access an unknown resolved schema", func() {
				resolvedTableSchema, err := mapper.ResolvedSchema("wat")

				Expect(err).To(Equal(config.ErrUnknownSchema))
				Expect(resolvedTableSchema).To(Equal(config.ResolvedTableSchema{}))
			})
		})

		Context("when working with an loaded schema", func() {
			var (
				tableSchema config.TableSchema
			)

			JustBeforeEach(func() {
				tableSchema, err = mapper.Schema("companies")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should contain the parsed entity", func() {
				Expect(tableSchema.Entity).To(Equal("company"))
			})

			It("should contain the parsed extensions", func() {
				Expect(len(tableSchema.Extensions)).To(Equal(1))

				Expect(tableSchema.Extensions[0].Key).To(Equal("")) // was null in json
				Expect(tableSchema.Extensions[0].Table).To(Equal("subfolder/abstract_entity"))
				Expect(tableSchema.Extensions[0].Title).To(Equal("columns.abstract.system"))
			})

			It("should contain the parsed exclusions", func() {
				Expect(len(tableSchema.Exclusions)).To(Equal(1))
				Expect(tableSchema.Exclusions[0]).To(Equal(config.TableSchemaExclusion("company_lastModificationDateUtc")))
			})

			It("should contain the parsed columns", func() {
				Expect(len(tableSchema.Columns)).To(Equal(2))

				Expect(tableSchema.Columns[0]).To(Equal(config.TableSchemaColumn{
					Title:        "columns.masterdata.company.companykey",
					Path:         "company_companyKey",
					Type:         "long",
					Filter:       "NumericFilter",
					Order:        "",
					PathResolver: "",
					FrontendHints: map[string]interface{}{
						"showDefault": false,
					},
				}))

				Expect(tableSchema.Columns[1]).To(Equal(config.TableSchemaColumn{
					Title:        "columns.masterdata.company.name",
					Path:         "company_name",
					Type:         "string",
					Filter:       "StringRegExFilter",
					Order:        "",
					PathResolver: "",
					FrontendHints: map[string]interface{}{
						"showDefault": true,
					},
				}))
			})
		})

		Context("when working with an resolved schema", func() {
			var (
				resolvedTableSchema config.ResolvedTableSchema
				originalTableSchema config.TableSchema
			)

			JustBeforeEach(func() {
				var err error

				resolvedTableSchema, err = mapper.ResolvedSchema("companies")
				Expect(err).NotTo(HaveOccurred())

				originalTableSchema, err = mapper.Schema("companies")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should contain the original schema", func() {
				Expect(resolvedTableSchema.OriginalSchema()).To(Equal(originalTableSchema))
			})

			It("should error when trying to access a non existing column", func() {
				column, err := resolvedTableSchema.Column("wat")

				Expect(err).To(Equal(config.ErrUnknownColumn))
				Expect(column).To(Equal(config.TableSchemaColumn{}))
			})

			It("should contain the parsed columns", func() {
				// Important - contains the resolved columns
				Expect(len(resolvedTableSchema.Columns())).To(Equal(4))

				Expect(resolvedTableSchema.Column("company_companyKey")).To(Equal(originalTableSchema.Columns[0]))
				Expect(resolvedTableSchema.Column("company_name")).To(Equal(originalTableSchema.Columns[1]))

				Expect(resolvedTableSchema.Column("company_uuid")).To(Equal(config.TableSchemaColumn{
					Title:        "columns.abstract.uuid",
					Path:         "company_uuid",
					Type:         "string",
					Filter:       "StringRegExFilter",
					Order:        "",
					PathResolver: "",
					FrontendHints: map[string]interface{}{
						"showDefault": false,
					},
				}))

				Expect(resolvedTableSchema.Column("company_createDateUtc")).To(Equal(config.TableSchemaColumn{
					Title:        "columns.abstract.createdate",
					Path:         "company_createDateUtc",
					Type:         "DateTime",
					Filter:       "DateTimeFilter",
					Order:        "",
					PathResolver: "",
					FrontendHints: map[string]interface{}{
						"showDefault": false,
					},
				}))
			})
		})

		Context("when trying to validate a table schema", func() {
			var (
				err error
			)

			BeforeEach(func() {
				err = mapper.ValidateIntegrity(config.EnumMapper{})
			})

			It("should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
