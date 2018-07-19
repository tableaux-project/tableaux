package tableaux

// Order describes an direction to order a column by.
type Order string

const (
	// OrderAsc describes ascending column order.
	OrderAsc Order = "ASC"

	// OrderDesc describes descending column order.
	OrderDesc Order = "DESC"
)

func (order Order) Reverse() Order {
	if order == OrderAsc {
		return OrderDesc
	}

	return OrderAsc
}

// FilterMode is an abstract definition of a mode to filter a column by.
type FilterMode string

const (
	// FilterEquals indicates that the column must match the filter value exactly.
	FilterEquals FilterMode = "EQUALS"

	// FilterGreater indicates that the column must be greater than the filter value.
	FilterGreater FilterMode = "GREATER"

	// FilterGreaterEquals indicates that the column must be greater or equal to the filter value.
	FilterGreaterEquals FilterMode = "GREATER_EQUALS"

	// FilterLesser indicates that the column must be lesser than the filter value.
	FilterLesser FilterMode = "LESSER"

	// FilterLesserEquals indicates that the column must be lesser or equal to the filter value.
	FilterLesserEquals FilterMode = "LESSER_EQUALS"

	// FilterNotEquals indicates that the column must NOT match the exact filter value.
	FilterNotEquals FilterMode = "NOT_EQUALS"
)
