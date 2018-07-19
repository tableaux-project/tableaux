package tableaux

import (
	"os"
	"path/filepath"

	"github.com/tableaux-project/tableaux/config"
)

// NewEnumMapperFromAssets creates a new EnumMapper from the asset folder, relative to the executing binary.
func NewEnumMapperFromAssets() (config.EnumMapper, error) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return config.NewEnumMapperFromFolder(filepath.Dir(ex) + "/assets/enum/")
}

// NewTranslatorFromAssets creates a new Translator from the asset folder, relative to the executing binary.
func NewTranslatorFromAssets() (config.Translator, error) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return config.NewTranslatorFromFolder(filepath.Dir(ex) + "/assets/i18n/")
}

// NewSchemaMapperFromAssets creates a new SchemaMapper from the asset folder, relative to the executing binary.
func NewSchemaMapperFromAssets() (config.SchemaMapper, error) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return config.NewSchemaMapperFromFolder(filepath.Dir(ex) + "/assets/schema/")
}
