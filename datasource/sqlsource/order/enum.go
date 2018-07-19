package order

import (
	"sort"
	"strings"

	"gopkg.in/birkirb/loggers.v1/log"

	"github.com/tableaux-project/tableaux"
	"github.com/tableaux-project/tableaux/config"
)

// EnumSorter is the base implementation to sort columns by a translated enum.
type EnumSorter struct {
	direct *Direct

	mapper     config.EnumMapper
	translator config.Translator
}

// NewEnumSorter creates a new EnumSorter instance.
func NewEnumSorter(mapper config.EnumMapper, translator config.Translator) Sorter {
	return &EnumSorter{
		direct:     &Direct{},
		mapper:     mapper,
		translator: translator,
	}
}

// EnumSortFunc describes the method that is applicable for sorting Enums.
type EnumSortFunc func(enum config.Enum, locale string, reverse bool) []config.KeyWithTranslation

func (sorter EnumSorter) entriesSortedByKey(enum config.Enum) []config.KeyWithTranslation {
	keys := enum.Entries()

	sort.Slice(keys, func(i, j int) bool {
		return strings.Compare(keys[i].EnumKey, keys[j].EnumKey) < 0
	})

	return keys
}

func (sorter EnumSorter) entriesSortedByTranslationWithSuffix(enum config.Enum, locale string, translationSuffix string, reverse bool) []config.KeyWithTranslation {
	keys := enum.Entries()

	sort.Slice(keys, func(i, j int) bool {
		aTrans, err := sorter.translator.Translate(locale, keys[i].TranslationKey+translationSuffix)

		if err != nil {
			// These are just warnings, not breaking errors
			log.Println(err)
		}

		bTrans, err := sorter.translator.Translate(locale, keys[j].TranslationKey+translationSuffix)

		if err != nil {
			// These are just warnings, not breaking errors
			log.Println(err)
		}

		return reverse != (strings.Compare(aTrans, bTrans) < 0)
	})

	return keys
}

// Checks if two arrays of keys with translations are equal.
// This is used to determinate if an enum order changes if it is translated.
func entriesAreTheSame(a, b []config.KeyWithTranslation) bool {
	if len(a) != len(b) {
		return false
	}

	for i, val := range a {
		if val != b[i] {
			return false
		}
	}

	return true
}

func (sorter EnumSorter) OrderColumn(path string, column config.TableSchemaColumn, direction tableaux.Order, locale string) (ResolvedOrder, error) {
	return sorter.orderColumnWithSortFunction(path, column, direction, locale, func(enum config.Enum, locale string, reverse bool) []config.KeyWithTranslation {
		return sorter.entriesSortedByTranslationWithSuffix(enum, locale, "", reverse)
	})
}

func (sorter EnumSorter) orderColumnWithSortFunction(path string, column config.TableSchemaColumn, direction tableaux.Order, locale string, sortFn EnumSortFunc) (ResolvedOrder, error) {
	enum, err := sorter.mapper.Enum(column.Type)
	if err != nil {
		return ResolvedOrder{}, err
	}

	originalEntries := sorter.entriesSortedByKey(enum)
	sortedEntries := sortFn(enum, locale, false)

	if entriesAreTheSame(originalEntries, sortedEntries) {
		// Nice, order does not change. So we can fall back to simple ordering
		return sorter.direct.OrderColumn(path, column, direction, locale)
	}

	// Check again, if the order might just need to be reversed
	if entriesAreTheSame(originalEntries, sortFn(enum, locale, true)) {
		// Nice, order does not change. So we can fall back to simple ordering
		return sorter.direct.OrderColumn(path, column, direction.Reverse(), locale)
	}

	keys := make([]interface{}, len(sortedEntries))
	for i, entry := range sortedEntries {
		keys[i] = entry.EnumKey
	}

	return ResolvedOrder{Path: path, Dir: direction, SortKeys: keys}, nil
}
