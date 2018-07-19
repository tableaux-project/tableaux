package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/birkirb/loggers.v1/log"
)

var (
	// ErrUnknownLanguage indicates that a requested language is
	// not known to a Translator.
	ErrUnknownLanguage = errors.New("unknown language")

	// ErrUnknownTranslation indicates that a requested translation
	// key is not known to a LanguageCatalog.
	ErrUnknownTranslation = errors.New("unknown translation key")
)

// LanguageCatalog is a mapping from translation keys to their individual translations.
// E.g. "enum.country.de" => "Germany"
type LanguageCatalog map[string]string

// Translate fetches the translation for a single key, or returns a
// ErrUnknownTranslation, if the key does not exist.
func (languageCatalog LanguageCatalog) Translate(key string) (string, error) {
	if languageCatalog[key] == "" {
		return "??" + key + "??", ErrUnknownTranslation
	}

	return languageCatalog[key], nil
}

// Entries returns all translation keys and their respective translation.
func (languageCatalog LanguageCatalog) Entries() map[string]string {
	entries := make(map[string]string, len(languageCatalog))
	for k, v := range languageCatalog {
		entries[k] = v
	}

	return entries
}

// Translator is a mapper which translates translation keys for different languages.
type Translator struct {
	languages map[string]LanguageCatalog
}

// NewTranslatorFromFolder builds a new translator from a given folder,
// recursively loading all i18n jsons which are found in there. Note,
// that the first level of folders is used for differentiating the language.
// E.g:
//
// /folder
// -- /de
// ---- somefile.json
// -- /en
// ---- anotherfile.json
func NewTranslatorFromFolder(schemaPath string) (Translator, error) {
	folders, err := ioutil.ReadDir(schemaPath)
	if err != nil {
		return Translator{}, err
	}

	maxKeys := -1
	minKeys := -1

	languages := make(map[string]LanguageCatalog)
	for _, f := range folders {
		if f.IsDir() {
			name := f.Name()
			catalog, err := loadTranslationFiles(filepath.Join(schemaPath, name))

			if err != nil {
				return Translator{}, err
			}

			keysCount := len(catalog)

			if maxKeys == -1 || keysCount > maxKeys {
				maxKeys = keysCount
			}

			if minKeys == -1 || keysCount < minKeys {
				minKeys = keysCount
			}

			log.WithFields(
				"name", name,
				"keys", keysCount,
			).Debug("Assembled language")

			languages[name] = catalog
		}
	}

	log.WithField("count", len(languages)).Info("Successfully loaded languages")

	if minKeys != maxKeys {
		log.Warn("Loaded languages with differing key counts - enable debug logging to identify languages")
	}

	return Translator{languages: languages}, nil
}

// Translate is a shortcut method for getting a LanguageCatalog, and immediately
// fetching a translation from it. Might return either an ErrUnknownLanguage or
// ErrUnknownTranslation, if either the language or the key therein does
// not exist.
func (translator Translator) Translate(language, key string) (string, error) {
	languageCatalog, err := translator.Language(language)
	if err != nil {
		return "", err
	}

	return languageCatalog.Translate(key)
}

// Language retrieves a specific language catalog if existing, or returns an
// ErrUnknownLanguage otherwise.
func (translator Translator) Language(language string) (LanguageCatalog, error) {
	if translator.languages[language] == nil {
		return nil, ErrUnknownLanguage
	}

	return translator.languages[language], nil
}

// Languages returns all language catalogs, in no particular order.
func (translator Translator) Languages() []LanguageCatalog {
	languageCatalogs := make([]LanguageCatalog, len(translator.languages))

	i := 0
	for _, v := range translator.languages {
		languageCatalogs[i] = v
		i++
	}

	return languageCatalogs
}

func loadTranslationFiles(path string) (LanguageCatalog, error) {
	catalog := make(LanguageCatalog)

	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == dotJSON {
			keys, err := loadTranslationKeys(path)

			if err != nil {
				return err
			}

			for key, value := range keys {
				catalog[key] = value
			}
		} else if !f.IsDir() {
			log.WithField("file", path).Debug("Ignoring file, as not a json file!")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return catalog, nil
}

func loadTranslationKeys(path string) (map[string]string, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	dat := make(map[string]string)
	if err := json.Unmarshal(file, &dat); err != nil {
		return nil, err
	}

	return dat, nil
}
