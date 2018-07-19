// Package datasource contains the implementation agnostic tableaux data retrieval core.
// Sub packages contain common code for certain implementation details, such as SQL.
package datasource

import (
	"github.com/tableaux-project/tableaux"
	"github.com/tableaux-project/tableaux/config"
)

// Connector defines the central contract between tableaux and an implementing data source.
type Connector interface {
	// ValidateRequest validates if the implementation is able to serve the request. Any error
	// indicates that execution of FetchData will probably fail, and is not expected to work.
	// This methods primary use case is to validate user-made requests for errors.
	ValidateRequest(columns []config.TableSchemaColumn, schema config.ResolvedTableSchema, filters []FilterGroup,
		orders []Order, globalSearch string, limit, offset uint64, locale string) error

	// FetchData is the entry point for retrieving data from a data source.
	FetchData(columns []config.TableSchemaColumn, schema config.ResolvedTableSchema, filters []FilterGroup,
		orders []Order, globalSearch string, limit, offset uint64, locale string) (result *Result,
		totalCount uint64, filteredCount uint64, error error)
}

// Result is the abstract data retrieval result of a data source implementation.
// A Result is mapping the fetched paths to their type-safe implementations
type Result []map[string]interface{}

// FilterGroup designates a path to be filtered by one or multiple actual Filters.
// A FilterGroup acts as an OR-chain. That is, all Filters contained in a single FilterGroup
// must be "OR'd" to each other. On the other hand, if multiple FilterGroups for one path
// exist, the individual results of each FilterGroup must be "AND'd".
type FilterGroup struct {
	path    string
	filters []Filter
}

// NewFilterGroup constructs a new FilterGroup.
func NewFilterGroup(path string, filters []Filter) FilterGroup {
	return FilterGroup{
		path:    path,
		filters: filters,
	}
}

// NewSimpleFilterGroup is a shortcut method of constructing a new FilterGroup with a one
// or multiple Filter with the same FilterMode inside. This is essentially a shortcut for
// generating an OR group over a single FilterMode.
func NewSimpleFilterGroup(path string, filterMode tableaux.FilterMode, values []interface{}) FilterGroup {
	newFilters := make([]Filter, len(values))
	for i, value := range values {
		newFilters[i] = NewFilter(
			filterMode,
			value,
		)
	}

	return NewFilterGroup(path, newFilters)
}

// Path is the path which is to be filtered on
func (f FilterGroup) Path() string {
	return f.path
}

// Filters returns all filters which are to be OR-chained in a single FilterGroup.
func (f *FilterGroup) Filters() []Filter {
	return f.filters
}

// Filter describes a single FilterMode with an applicable value to be filtered by.
type Filter struct {
	filterMode tableaux.FilterMode
	value      interface{}
}

// NewFilter constructs a new Filter.
func NewFilter(filterMode tableaux.FilterMode, value interface{}) Filter {
	return Filter{
		filterMode: filterMode,
		value:      value,
	}
}

// FilterMode is the mode in which to filter the column by
func (f Filter) FilterMode() tableaux.FilterMode {
	return f.filterMode
}

// Value returns the actual value to filter by. Its up to the filter implementation
// to make sense of the interface value, and error out if unexpected values are supplied.
func (f Filter) Value() interface{} {
	return f.value
}

// Order designates a path to be ordered in a certain direction.
// Additional sort keys might be supplied, to indicate a fixed order.
type Order struct {
	path      string
	direction tableaux.Order
	sortKeys  []interface{}
}

// Path is the path which is to be ordered
func (o Order) Path() string {
	return o.path
}

func (o Order) Direction() tableaux.Order {
	return o.direction
}

func (o Order) SortKeys() []interface{} {
	return o.sortKeys
}

func NewOrder(path string, direction tableaux.Order, sortKeys []interface{}) Order {
	return Order{
		path:      path,
		direction: direction,
		sortKeys:  sortKeys,
	}
}
