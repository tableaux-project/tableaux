package order

import (
	"github.com/tableaux-project/tableaux"
	"github.com/tableaux-project/tableaux/config"
)

// Direct is the default implementation of a Sorter, which simply sorts by the given
// column path, without any other kind of processing or conversion.
type Direct struct {
}

func (direct Direct) OrderColumn(path string, column config.TableSchemaColumn, sortOrder tableaux.Order, locale string) (ResolvedOrder, error) {
	return ResolvedOrder{Path: path, Dir: sortOrder, SortKeys: nil}, nil
}
