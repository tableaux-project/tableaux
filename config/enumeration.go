package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/birkirb/loggers.v1/log"
)

var (
	// ErrUnknownEnum indicates that a requested enum is not
	// known to an EnumMapper.
	ErrUnknownEnum = errors.New("unknown enum")

	// ErrUnknownEnumKey indicates that a requested key is
	// not known to an Enum.
	ErrUnknownEnumKey = errors.New("unknown enum key")
)

// Enum is an assignment of enum keys to translation keys.
// E.g. "DE" => "enum.country.de"
type Enum map[string]string

// KeyWithTranslation is a simple tuple of an enumeration key
// to its translation key.
type KeyWithTranslation struct {
	EnumKey, TranslationKey string
}

// TranslationKey retrieves the translation key for a single enum key, or
// returns an ErrUnknownEnumKey, if the key does not exist.
func (enum Enum) TranslationKey(key string) (string, error) {
	if enum[key] == "" {
		return "", ErrUnknownEnumKey
	}

	return enum[key], nil
}

// Entries returns all enum keys and their respective translation keys.
func (enum Enum) Entries() []KeyWithTranslation {
	entries := make([]KeyWithTranslation, len(enum))

	i := 0
	for k, v := range enum {
		entries[i] = KeyWithTranslation{
			EnumKey:        k,
			TranslationKey: v,
		}
		i++
	}

	return entries
}

// EnumMapper is a mapper which maps enum keys to translation keys.
type EnumMapper struct {
	enums map[string]Enum
}

// NewEnumMapperFromFolder builds a new enum mapper from a given folder,
// recursively loading all enum jsons which are found in there.
func NewEnumMapperFromFolder(schemaPath string) (EnumMapper, error) {
	enums := make(map[string]Enum)

	regex := regexp.MustCompile(`[\\/]`)
	err := filepath.Walk(schemaPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == dotJSON {
			relativePath, err := filepath.Rel(schemaPath, path)
			if err != nil {
				return err
			}

			keys, err := loadEnumFile(path)
			if err != nil {
				return err
			}

			name := regex.ReplaceAllString(strings.TrimSuffix(relativePath, filepath.Ext(path)), "")

			enums[name] = keys
		} else if !f.IsDir() {
			log.WithField("file", path).Debug("Ignoring file, as not a json file!")
		}

		return nil
	})

	if err != nil {
		return EnumMapper{}, err
	}

	log.WithField("count", len(enums)).Info("Successfully loaded enums")

	return EnumMapper{enums: enums}, nil
}

// TranslationKeyInEnum is a shortcut method for getting an enum, and immediately
// fetching a translation key from it.
func (enumMapper EnumMapper) TranslationKeyInEnum(enum, key string) (string, error) {
	fetchedEnum, err := enumMapper.Enum(enum)
	if err != nil {
		return "", err
	}

	return fetchedEnum.TranslationKey(key)
}

// Enum retrieves a specific enum from the mapper if existing, or returns an error
// otherwise.
func (enumMapper EnumMapper) Enum(enum string) (Enum, error) {
	if enumMapper.enums[enum] == nil {
		return nil, ErrUnknownEnum
	}

	return enumMapper.enums[enum], nil
}

// Enums returns all enums which the mapper knows, in no particular order.
func (enumMapper EnumMapper) Enums() []Enum {
	enums := make([]Enum, len(enumMapper.enums))

	i := 0
	for _, v := range enumMapper.enums {
		enums[i] = v
		i++
	}

	return enums
}

func loadEnumFile(path string) (Enum, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	dat := Enum{}
	if err := json.Unmarshal(file, &dat); err != nil {
		return nil, err
	}

	return dat, nil
}
