package sqlsource

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/birkirb/loggers.v1/log"

	"github.com/tableaux-project/tableaux"
	"github.com/tableaux-project/tableaux/config"
	"github.com/tableaux-project/tableaux/datasource"
	"github.com/tableaux-project/tableaux/datasource/sqlsource/filter"
	"github.com/tableaux-project/tableaux/datasource/sqlsource/order"
	"github.com/tableaux-project/tableaux/datasource/sqlsource/path"
	"github.com/tableaux-project/tableaux/datasource/sqlsource/util"
)

// Connector is the entry point for sql related
type Connector struct {
	dbConnector  DatabaseConnector
	enumMapper   config.EnumMapper
	schemaMapper config.SchemaMapper
	translator   config.Translator
	resolvers    map[string]datasource.PathResolver
	sorters      map[string]order.Sorter
	filters      map[string]filter.Filter
}

func NewConnector(databaseConnector DatabaseConnector, enumMapper config.EnumMapper, translator config.Translator, schemaMapper config.SchemaMapper) (datasource.Connector, error) {
	if err := schemaMapper.ValidateIntegrity(enumMapper); err != nil {
		return nil, err
	}

	return &Connector{
		databaseConnector,
		enumMapper,
		schemaMapper,
		translator,
		map[string]datasource.PathResolver{
			"":                 path.SimpleResolver{},
			"SizePathResolver": path.SizeResolver{},
		},
		map[string]order.Sorter{
			"":               order.Direct{},
			"EnumOrder":      order.NewEnumSorter(enumMapper, translator),
			"ShortEnumOrder": order.NewShortEnumSorter(enumMapper, translator),
			"LongEnumOrder":  order.NewLongEnumSorter(enumMapper, translator),
		},
		map[string]filter.Filter{
			"BooleanFilter":     filter.Boolean{Common: &filter.Common{}},
			"StringFilter":      filter.PlainString{Common: &filter.Common{}},
			"StringRegExFilter": filter.RegexString{Common: &filter.Common{}},
			"EnumFilter":        filter.PlainString{Common: &filter.Common{}}, // TODO
			"NumericFilter":     filter.Numeric{Common: &filter.Common{}},     // TODO
			"DateFilter":        filter.PlainString{Common: &filter.Common{}}, // TODO
			"DateTimeFilter":    filter.PlainString{Common: &filter.Common{}}, // TODO
		},
	}, nil
}

func (th Connector) ValidateRequest(columns []config.TableSchemaColumn, schema config.ResolvedTableSchema,
	filters []datasource.FilterGroup, orders []datasource.Order, globalSearch string, limit, offset uint64,
	locale string) error {
	if len(columns) == 0 {
		return errors.New("no columns selected")
	}

	if _, err := th.translator.Language(locale); err != nil {
		return fmt.Errorf("unknown locale %s", locale)
	}

	for _, column := range columns {
		columnPath := column.Path

		_, err := schema.Column(columnPath)
		if err != nil {
			return fmt.Errorf("unknown column %s", columnPath)
		}

		pathResolver := th.resolvers[column.PathResolver]
		if pathResolver == nil {
			return fmt.Errorf("unknown path resolver %s on column %s", column.PathResolver, columnPath)
		}

		columnFilter := th.filters[column.Filter]
		if columnFilter == nil {
			return fmt.Errorf("unknown filter %s on column %s", column.Filter, columnPath)
		}

		columnOrder := th.sorters[column.Order]
		if columnOrder == nil {
			return fmt.Errorf("unknown order %s on column %s", column.Order, columnPath)
		}
	}

	for _, filterGroup := range filters {
		columnPath := filterGroup.Path()

		if _, err := schema.Column(columnPath); err == config.ErrUnknownColumn {
			return fmt.Errorf("unknown filter column %s", columnPath)
		}
	}

	for _, column := range orders {
		columnPath := column.Path()

		if _, err := schema.Column(columnPath); err == config.ErrUnknownColumn {
			return fmt.Errorf("unknown order column %s", columnPath)
		}
	}

	return nil
}

func (th Connector) FetchData(columns []config.TableSchemaColumn, schema config.ResolvedTableSchema,
	filters []datasource.FilterGroup, orders []datasource.Order, globalSearch string,
	limit, offset uint64, locale string) (*datasource.Result, uint64, uint64, error) {
	start := time.Now()

	entity := schema.OriginalSchema().Entity

	// Kick-off the result counting - we need that at the end, so it can run in parallel
	totalCountChannel := make(chan uint64, 1)
	go th.countQuery(schema, totalCountChannel, nil)
	//defer close(totalCountChannel)

	// Only count filtered results if we actually have filters
	var filterCountChannel chan uint64
	if len(filters) > 0 {
		filterCountChannel = make(chan uint64, 1)
		go th.countQuery(schema, filterCountChannel, filters)
		//defer close(filterCountChannel)
	}

	// --------

	useDeferredLoading := adviseDeferredLoading(columns, orders, schema)
	if useDeferredLoading {
		// For deferred loading, we only care about selecting the primary key
		primaryKeyPath := entity + "_" + util.IdentifierToDescriptor(th.dbConnector.KeyResolver().ResolvePrimaryKey(entity)[0])

		// --------

		// Fetch the primary keys
		rows, err := th.fetchData([]config.TableSchemaColumn{
			{Path: primaryKeyPath},
		}, filters, orders, schema, limit, offset, locale)
		if err != nil {
			return nil, 0, 0, err
		}

		defer util.LoggingRowsCloser(rows, "deferredLoading-PK-fetch")

		var primaryKey string
		var primaryKeys []interface{}

		for rows.Next() {
			err := rows.Scan(&primaryKey)
			if err != nil {
				return nil, 0, 0, err
			}

			primaryKeys = append(primaryKeys, primaryKey)
		}

		// No keys? Then short-circuit to the empty response
		if len(primaryKeys) == 0 {
			totalCount := waitAndCloseChannel(totalCountChannel)
			return &datasource.Result{}, 0, totalCount, nil
		}

		// Apply the primary keys as the new order of the actual data fetch
		orders = []datasource.Order{
			datasource.NewOrder(
				primaryKeyPath,
				tableaux.OrderAsc,
				primaryKeys,
			),
		}

		// Replace existing filters with primary key filter
		filters = []datasource.FilterGroup{
			datasource.NewSimpleFilterGroup(
				primaryKeyPath,
				tableaux.FilterEquals,
				primaryKeys,
			),
		}

		// Ensure that the data fetch does neither offset nor limit
		limit = 0
		offset = 0
	}

	rows, err := th.fetchData(columns, filters, orders, schema, limit, offset, locale)
	if err != nil {
		return nil, 0, 0, err
	}

	defer util.LoggingRowsCloser(rows, "datafetch")

	cols, err := rows.Columns()
	if err != nil {
		log.Fatal("Failed to get selectColumns", err)
	}

	// Result is your slice string.
	result := make([][]byte, len(cols))
	dest := make([]interface{}, len(cols))

	// A temporary interface{} slice
	for i := range result {
		dest[i] = &result[i] // Put pointers to each string in the interface slice
	}

	types, _ := rows.ColumnTypes()

	dataResult := datasource.Result{}
	for rows.Next() {
		err := rows.Scan(dest...)
		if err != nil {
			log.Fatal(err)
		}

		row := make(map[string]interface{}, len(result))
		for i := 0; i < len(result); i++ {
			name := strings.Replace(types[i].Name(), ".", "_", -1)

			value, err := th.dbConnector.MakeItemTypeSafe(result[i], types[i])
			if err != nil {
				return nil, 0, 0, err
			}

			row[name] = value
		}

		dataResult = append(dataResult, row)
	}

	totalCount := waitAndCloseChannel(totalCountChannel)

	filteredCount := totalCount
	if filterCountChannel != nil {
		filteredCount = waitAndCloseChannel(filterCountChannel)
	}

	log.WithFields(
		"time", time.Since(start),
		"totalCount", totalCount,
		"filteredCount", filteredCount,
		"count", len(dataResult),
	).Info("Data fetched")

	return &dataResult, totalCount, filteredCount, nil
}

func waitAndCloseChannel(channel chan uint64) uint64 {
	count := <-channel
	close(channel)
	return count
}

// Calculates all paths that are participating in the request, be it trough selection, filtering or ordering.
func mergedParticipatingPaths(columns []config.TableSchemaColumn, orders []datasource.Order,
	filters []datasource.FilterGroup) map[string]interface{} {
	pathMap := make(map[string]interface{})

	for _, column := range columns {
		pathMap[column.Path] = struct{}{}
	}

	for _, columnOrder := range orders {
		pathMap[columnOrder.Path()] = struct{}{}
	}

	for _, columnFilter := range filters {
		pathMap[columnFilter.Path()] = struct{}{}
	}

	return pathMap
}

// Calculates all the paths that require joins, for a given request. This method looks at the selected columns,
// ordering and filtering to determinate what needs to be joined.
func calculatePathsForJoins(columns []config.TableSchemaColumn, orders []datasource.Order,
	filters []datasource.FilterGroup) []string {
	participatingPaths := mergedParticipatingPaths(columns, orders, filters)
	if len(participatingPaths) == 0 {
		return []string{}
	}

	joinPaths := make(map[string]bool)

	for columnPath := range participatingPaths {
		pathParts := strings.Split(columnPath, "_")

		if len(pathParts) > 2 {
			for i := 2; i < len(pathParts); i++ {
				// Path required to join
				joinPaths[strings.Join(pathParts[0:i], "_")] = true
			}
		}
	}

	// Sort the joins, and then convert them to database specific joins
	joinStrings := make([]string, len(joinPaths))

	i := 0
	for k := range joinPaths {
		joinStrings[i] = k
		i++
	}
	sort.Strings(joinStrings)

	return joinStrings
}

func calculatePathsForCountJoins(columns []config.TableSchemaColumn, orders []datasource.Order,
	filters []datasource.FilterGroup, schema config.ResolvedTableSchema) []string {
	participatingPaths := mergedParticipatingPaths(columns, orders, filters)
	if len(participatingPaths) == 0 {
		return []string{}
	}

	var countPaths []string

	// Finished resolving the primary joins. Now we resolve the count joins.
	for columnPath := range participatingPaths {
		columnSchema, err := schema.Column(columnPath)
		if err == nil && columnSchema.PathResolver == "SizePathResolver" {
			countPaths = append(countPaths, columnPath)
		}
	}

	return countPaths
}

// Returns true, if it is advisable to use deferred loading
func adviseDeferredLoading(columns []config.TableSchemaColumn, orders []datasource.Order, schema config.ResolvedTableSchema) bool {
	// TODO: Well, it will be when query hints are implemented
	/*for _, column := range columns {
		// Do we have a size path selected? Then deferred loading is about twice as fast!
		if column.PathResolver == "SizePathResolver" {
			return true
		}
	}*/

	for _, columnOrder := range orders {
		if len(strings.Split(columnOrder.Path(), "_")) > 2 {
			return true
		}

		resolvedColumn, err := schema.Column(columnOrder.Path())
		if err != nil {
			log.WithField("column", columnOrder.Path()).Error("Unable to resolve column to advise on deferred loading")
			return false
		}

		// Ordering on a size path is unusable-slow without deferred loading!
		if resolvedColumn.PathResolver == "SizePathResolver" {
			return true
		}
	}

	return false
}

func (th Connector) fetchData(columns []config.TableSchemaColumn, filters []datasource.FilterGroup, orders []datasource.Order,
	schema config.ResolvedTableSchema, limit, offset uint64, locale string) (*sql.Rows, error) {
	var err error

	queryBuilder := th.dbConnector.QueryBuilder()
	keyResolver := th.dbConnector.KeyResolver()

	db := th.dbConnector.DatabaseObject()
	entity := schema.OriginalSchema().Entity

	// ---------------------------

	joinString, err := th.resolveJoinString(columns, orders, schema, filters)
	if err != nil {
		return nil, err
	}

	// ---------------------------

	selectColumns := make([]string, len(columns))
	for i, column := range columns {
		resolver := th.resolvers[column.PathResolver]
		selectColumns[i] = resolver.ResolvePathName(column) + " AS " + column.Path
	}

	// ---------------------------

	pkPath := entity + "_" + keyResolver.ResolvePrimaryKey(entity)[0]
	containsPKOrder := false
	for _, value := range orders {
		if value.Path() == pkPath {
			containsPKOrder = true
			break
		}
	}

	// Guarantee primary key sort, so we get stable results
	if !containsPKOrder {
		log.Info("Request does not contain order on primary key - adding order to ensure consistent results")
		orders = append(orders, datasource.NewOrder(pkPath, tableaux.OrderAsc, nil))
	}

	sortColumns := make([]string, len(orders))
	builder := th.dbConnector.QueryBuilder()
	for i, value := range orders {
		resolver := th.resolvers[""]

		column, colErr := schema.Column(value.Path())
		if colErr == nil {
			resolver = th.resolvers[column.PathResolver]
		} else {
			column = config.TableSchemaColumn{
				Path: value.Path(),
			}
			log.WithFields(
				"path", value.Path(),
				"schema", schema.OriginalSchema().Entity,
			).Warn("Ordering on column which is unknown to schema - using default path resolver")
		}

		resolvedPath := resolver.ResolvePathName(column)

		sortColumns[i] = OrderColumn(builder, resolvedPath, column, th.sorters[column.Order], value, locale)
	}

	// ---------------------------

	queryString := strings.Join(selectColumns, ",") + " FROM " + entity
	if joinString != "" {
		queryString += " " + joinString
	}

	filterString, err := th.filterString(filters, schema)
	if err != nil {
		return nil, err
	}

	if filterString != "" {
		queryString += " WHERE " + filterString
	}

	queryString += " ORDER BY " + strings.Join(sortColumns, ",")

	if limit > 0 {
		queryString = queryBuilder.SelectWithLimitQuery(queryString)
	} else {
		queryString = "SELECT " + queryString
	}

	statement, err := db.Prepare(queryString)
	if err != nil {
		log.WithField("query", queryString).Error("Failed to prepare query")
		return nil, err
	}

	log.WithField("query", queryString).Debug("Executing query")

	start := time.Now()
	var (
		rows    *sql.Rows
		rowsErr error
	)
	if limit > 0 {
		rows, rowsErr = statement.Query(limit)
	} else {
		rows, rowsErr = statement.Query()
	}

	log.WithFields(
		"time", time.Since(start),
		"columns", len(columns),
	).Debug("Query successfully executed for data source")

	return rows, rowsErr
}

func (th Connector) filterString(filters []datasource.FilterGroup, schema config.ResolvedTableSchema) (string, error) {
	queryBuilder := th.dbConnector.QueryBuilder()

	uniqueFilterPaths := make(map[string][]datasource.FilterGroup)
	for _, filterGroup := range filters {
		uniqueFilterPaths[filterGroup.Path()] = append(uniqueFilterPaths[filterGroup.Path()], filterGroup)
	}

	andFilterStrings := make([]string, len(uniqueFilterPaths))
	i := 0
	for rawPath, filterGroups := range uniqueFilterPaths {
		schemaColumn, err := schema.Column(rawPath)
		if err != nil {
			return "", err
		}

		columnFilter := th.filters[schemaColumn.Filter]
		resolver := th.resolvers[schemaColumn.PathResolver]
		resolvedPath := resolver.ResolvePathName(schemaColumn)

		columnFilterString, err := FilterColumn(queryBuilder, resolvedPath, columnFilter, filterGroups)
		if err != nil {
			return "", err
		}

		andFilterStrings[i] = columnFilterString
		i++
	}

	return strings.Join(andFilterStrings, " AND "), nil
}

func (th Connector) resolveJoinString(columns []config.TableSchemaColumn, orders []datasource.Order, schema config.ResolvedTableSchema, filters []datasource.FilterGroup) (string, error) {
	queryBuilder := th.dbConnector.QueryBuilder()
	joinResolver := th.dbConnector.JoinResolver()
	keyResolver := th.dbConnector.KeyResolver()

	// Sort the joins, and then convert them to database specific joins
	joinStrings := calculatePathsForJoins(columns, orders, filters)
	for index, columnPath := range joinStrings {
		resolvedPath, err := joinResolver.ResolvePath(columnPath)
		if err != nil {
			return "", err
		}

		joinStrings[index] = queryBuilder.ResolvedToJoinString(resolvedPath)
	}

	// ---------------------------

	// Finished resolving the primary joins. Now we resolve the count joins.
	for _, columnPath := range calculatePathsForCountJoins(columns, orders, filters, schema) {
		countJoin, err := joinResolver.ResolveCountJoin(columnPath, th.schemaMapper, keyResolver)
		if err != nil {
			log.WithField("path", columnPath).Error("Cannot resolve count join")
			return "", err
		}

		joinStrings = append(joinStrings, queryBuilder.CountJoinToJoinString(countJoin))
	}

	return strings.Join(joinStrings, " "), nil
}

func (th Connector) countQuery(schema config.ResolvedTableSchema, countChannel chan uint64, filters []datasource.FilterGroup) {
	var count uint64

	pk := th.dbConnector.KeyResolver().ResolvePrimaryKey(schema.OriginalSchema().Entity)[0]
	joinString, err := th.resolveJoinString([]config.TableSchemaColumn{}, []datasource.Order{}, schema, filters)
	if err != nil {
		log.Fatal(err)
	}

	queryString := "SELECT count(" + schema.OriginalSchema().Entity + "." + pk + ") FROM " + schema.OriginalSchema().Entity
	if joinString != "" {
		queryString += " " + joinString
	}

	filterString, err := th.filterString(filters, schema)
	if err != nil {
		panic(err)
	}

	if filterString != "" {
		queryString += " WHERE " + filterString
	}

	log.WithField("query", queryString).Debug("Executing query")

	err = th.dbConnector.DatabaseObject().QueryRow(queryString).Scan(&count)
	if err != nil {
		log.Error(err)
	}

	countChannel <- count
}
