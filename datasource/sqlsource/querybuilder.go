package sqlsource

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/birkirb/loggers.v1/log"

	"github.com/tableaux-project/tableaux"
	"github.com/tableaux-project/tableaux/config"
	"github.com/tableaux-project/tableaux/datasource"
	"github.com/tableaux-project/tableaux/datasource/sqlsource/filter"
	"github.com/tableaux-project/tableaux/datasource/sqlsource/order"
)

type QueryBuilder interface {
	ResolvedToJoinString(resolved Join) string
	CountJoinToJoinString(join CountJoin) string
	IfNull(query string, then interface{}) string
	SelectWithLimitQuery(query string) string

	OrderColumn(path string, direction tableaux.Order) string
	OrderColumnByArray(column string, values []interface{}, direction tableaux.Order) string

	FilterStringFromValues(path string, filter filter.Filter, operator filter.Operator, values []interface{}) (string, error)
	FilterStringFromValue(path string, operator filter.Operator, value string) string
}

// Checks if two string slices are equal.
func stringSlicesEqual(a, b sort.StringSlice) bool {
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

func OrderColumn(queryBuilder QueryBuilder, path string, column config.TableSchemaColumn, sorter order.Sorter, order datasource.Order, locale string) string {
	predefinedSortKeys := order.SortKeys()

	if len(predefinedSortKeys) > 0 {
		// Okay, we have sort keys, so we will commit a case'd order. However, if the sort keys
		// are in order (in either direction), we can omit the cases and do a regular sort instead.

		// This is tricky - we can't do much with interface{}, so lets try to assemble a string array instead
		sanitizedKeys := make(sort.StringSlice, len(predefinedSortKeys))

		bailOut := false
		for i, value := range predefinedSortKeys {
			switch converted := value.(type) {
			case string:
				sanitizedKeys[i] = converted
			case uint64:
				sanitizedKeys[i] = strconv.FormatUint(converted, 10)
			case int64:
				sanitizedKeys[i] = strconv.FormatInt(converted, 10)
			default:
				log.WithFields("type", reflect.TypeOf(value), "value", converted).Debug("Cannot convert type to String")
				bailOut = true
			}

			if bailOut {
				break
			}
		}

		if !bailOut {
			// Okay, we have the keys, now do something with them - first create a copy to compare to
			sortedEntries := make(sort.StringSlice, len(predefinedSortKeys))
			for i, value := range sanitizedKeys {
				sortedEntries[i] = value
			}

			sort.Sort(sortedEntries)
			if stringSlicesEqual(sanitizedKeys, sortedEntries) {
				// Nice, order does not change. So we can fall back to simple ordering
				return queryBuilder.OrderColumn(path, order.Direction())
			}

			// Prepare reversed order
			reversedEntries := sort.Reverse(sortedEntries).(sort.StringSlice)
			sort.Sort(reversedEntries)

			// Check again - the order might just need to be reversed
			if stringSlicesEqual(sanitizedKeys, reversedEntries) {
				// Nice, order does not change. So we can fall back to simple ordering
				return queryBuilder.OrderColumn(path, order.Direction().Reverse())
			}

			// Oh well, order is not linear - so fall back to case'd sort.
			return queryBuilder.OrderColumnByArray(path, predefinedSortKeys, order.Direction())
		}
	}

	orderRequest, err := sorter.OrderColumn(path, column, order.Direction(), locale)
	if err != nil {
		panic(err)
	}

	if orderRequest.SortKeys != nil {
		return queryBuilder.OrderColumnByArray(orderRequest.Path, orderRequest.SortKeys, orderRequest.Dir)
	}

	return queryBuilder.OrderColumn(orderRequest.Path, orderRequest.Dir)
}

func FilterColumn(queryBuilder QueryBuilder, path string, filtery filter.Filter, filterGroups []datasource.FilterGroup) (string, error) {
	var andFilters []string
	for _, filterGroup := range filterGroups {
		// First, we group all filter with the same operator together. This is done, so we can optimize
		// some cases (e.g. multiple EQUALS can be pulled into an IN clause)
		filterModeMap := make(map[filter.Operator][]interface{})
		for _, filterGroupFilter := range filterGroup.Filters() {
			operator, err := filtery.Operator(filterGroupFilter.Value(), filterGroupFilter.FilterMode())
			if err != nil {
				return "", err
			}
			filterModeMap[operator] = append(filterModeMap[operator], filterGroupFilter.Value())
		}

		i := 0
		orFilters := make([]string, len(filterModeMap))
		for filterMode, values := range filterModeMap {
			orFilter, err := queryBuilder.FilterStringFromValues(path, filtery, filterMode, values)
			if err != nil {
				return "", err
			}

			orFilters[i] = orFilter
			i++
		}

		andFilters = append(andFilters, strings.Join(orFilters, " OR "))
	}

	return strings.Join(andFilters, " AND "), nil
}

type CommonQueryBuilder struct {
}

func (commonBuilder CommonQueryBuilder) OrderColumn(path string, direction tableaux.Order) string {
	return path + " " + string(direction)
}

func (commonBuilder CommonQueryBuilder) OrderColumnByArray(path string, values []interface{}, direction tableaux.Order) string {
	cases := make([]string, len(values))

	for index, value := range values {
		switch value.(type) {
		default:
			cases[index] = fmt.Sprintf("WHEN %v THEN %d", value, index)
		case string:
			cases[index] = fmt.Sprintf("WHEN '%s' THEN %d", value, index)
		}
	}

	return fmt.Sprintf("CASE %s %s ELSE -1 END %s", path, strings.Join(cases, " "), string(direction))
}

func (commonBuilder CommonQueryBuilder) ResolvedToJoinString(resolvedJoin Join) string {
	return string(resolvedJoin.JoinType()) + " JOIN " + resolvedJoin.TargetTable() + " AS " + resolvedJoin.JoinAlias() +
		" ON " + resolvedJoin.JoinAlias() + "." + resolvedJoin.TargetColumn() + "=" + resolvedJoin.SourceTable() + "." + resolvedJoin.SourceColumn()
}

func (commonBuilder CommonQueryBuilder) CountJoinToJoinString(resolvedCount CountJoin) string {
	return "LEFT JOIN (" +
		"SELECT " + resolvedCount.CountEntityForeignKey() + ", COUNT(" + resolvedCount.CountEntityPrimaryKey() + ") AS count_result " +
		"FROM " + resolvedCount.CountEntity() + " " +
		"GROUP BY " + resolvedCount.CountEntityForeignKey() +
		") AS " + resolvedCount.Alias() + " ON " + resolvedCount.Alias() + "." + resolvedCount.CountEntityForeignKey() + " = " + resolvedCount.OriginEntity() + "." + resolvedCount.OriginEntityPrimaryKey()
}

// Constructs a single filter expression for a path from multiple values
// multiple values are expected to be OR chained.
func (commonBuilder CommonQueryBuilder) FilterStringFromValues(path string, filtery filter.Filter, operator filter.Operator, values []interface{}) (string, error) {
	parsedValues := parseValues(filtery, values)

	if len(values) == 1 {
		return commonBuilder.FilterStringFromValue(path, operator, parsedValues[0]), nil
	}

	switch operator {
	case filter.OperatorEqual:
		return fmt.Sprintf("%s IN (%s)", path, strings.Join(parsedValues, ",")), nil
	case filter.OperatorNotEqual:
		return fmt.Sprintf("%s NOT IN (%s)", path, strings.Join(parsedValues, ",")), nil
	case filter.OperatorGreater,
		filter.OperatorGreaterEquals,
		filter.OperatorLesser,
		filter.OperatorLesserEquals,
		filter.OperatorLike:
		// There is no IN or NOT IN we can apply to these filter modes, so we classically OR join them
		orChainedValues := make([]string, len(values))

		for i, value := range parsedValues {
			orChainedValues[i] = commonBuilder.FilterStringFromValue(path, operator, value)
		}

		return strings.Join(orChainedValues, " OR "), nil
	default:
		return "", fmt.Errorf("unknown operator %s", operator)
	}
}

func parseValues(filter filter.Filter, values []interface{}) []string {
	parsedValues := make([]string, len(values))

	for i, value := range values {
		parsedValues[i] = filter.ParseValue(value)
	}

	return parsedValues
}

func (commonBuilder CommonQueryBuilder) FilterStringFromValue(path string, operator filter.Operator, value string) string {
	return fmt.Sprintf("%s %s %s", path, operator, value)
}
