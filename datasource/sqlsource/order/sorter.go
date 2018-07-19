package order

import (
	"github.com/tableaux-project/tableaux"
	"github.com/tableaux-project/tableaux/config"
)

// Sorter is the common sql interface for sorting a single column.
type Sorter interface {
	// OrderColumn takes a path and its respective column meta data, and converts it into an ResolvedOrder
	// by ordering via the supplementary direction and locale.
	// The given path is the resolved path for the provided column, and should be used. However, it is
	// possible to return an entirely different path in the ResolvedOrder.
	OrderColumn(path string, column config.TableSchemaColumn, direction tableaux.Order, locale string) (ResolvedOrder, error)
}

// ResolvedOrder is an abstract data source Order, which has been processed by a Sorter implementation.
type ResolvedOrder struct {
	Path     string
	Dir      tableaux.Order
	SortKeys []interface{}
}
