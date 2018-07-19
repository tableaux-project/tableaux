package order

import (
	"github.com/tableaux-project/tableaux"
	"github.com/tableaux-project/tableaux/config"
)

// LongEnumSorter is an extended EnumSorter, which suffixes ".long" when retrieving translations.
type LongEnumSorter struct {
	*EnumSorter
}

// NewLongEnumSorter creates a new LongEnumSorter instance.
func NewLongEnumSorter(mapper config.EnumMapper, translator config.Translator) Sorter {
	return LongEnumSorter{
		EnumSorter: &EnumSorter{
			direct:     &Direct{},
			mapper:     mapper,
			translator: translator,
		},
	}
}

func (order LongEnumSorter) OrderColumn(path string, column config.TableSchemaColumn, direction tableaux.Order, locale string) (ResolvedOrder, error) {
	return order.orderColumnWithSortFunction(path, column, direction, locale, func(enum config.Enum, locale string, reverse bool) []config.KeyWithTranslation {
		return order.entriesSortedByTranslationWithSuffix(enum, locale, ".long", reverse)
	})
}
