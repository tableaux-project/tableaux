package order

import (
	"github.com/tableaux-project/tableaux"
	"github.com/tableaux-project/tableaux/config"
)

// ShortEnumSorter is an extended EnumSorter, which suffixes ".short" when retrieving translations.
type ShortEnumSorter struct {
	*EnumSorter
}

// NewShortEnumSorter creates a new ShortEnumSorter instance.
func NewShortEnumSorter(mapper config.EnumMapper, translator config.Translator) Sorter {
	return ShortEnumSorter{
		EnumSorter: &EnumSorter{
			direct:     &Direct{},
			mapper:     mapper,
			translator: translator,
		},
	}
}

func (order ShortEnumSorter) OrderColumn(path string, column config.TableSchemaColumn, direction tableaux.Order, locale string) (ResolvedOrder, error) {
	return order.orderColumnWithSortFunction(path, column, direction, locale, func(enum config.Enum, locale string, reverse bool) []config.KeyWithTranslation {
		return order.entriesSortedByTranslationWithSuffix(enum, locale, ".short", reverse)
	})
}
