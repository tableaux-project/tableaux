package sqlsource

import (
	"database/sql"
	"errors"
	"strings"

	"gopkg.in/birkirb/loggers.v1/log"

	"github.com/tableaux-project/tableaux/config"
	"github.com/tableaux-project/tableaux/datasource/sqlsource/util"
)

// TableColumn is a simple doublet of a table and a column
// therein.
type TableColumn struct {
	Table, Column string
}

// ColumnInformation encapsulates all the relevant information
// about an individual column for join behavior.
type ColumnInformation struct {
	Nullable bool
}

// JoinResolver is a helping resolver, which resolves joins for
// individual paths.
type JoinResolver interface {
	ResolvePath(joinPath string) (Join, error)
	ResolveCountJoin(path string, schemaMapper config.SchemaMapper, keyResolver KeyResolver) (CountJoin, error)
}

// CommonJoinResolver encapsulates common JoinResolver behavior,
// so that implementations only need to feed the objects caches
// for proper usage.
type CommonJoinResolver struct {
	// Cache to improve join performance, by analyzing column characteristics
	columnCache map[TableColumn]ColumnInformation

	// Cache to map a table with its foreign key to a different table and its primary key
	foreignKeyMap map[TableColumn]TableColumn

	// Cache for remembering already visited join paths
	joinPathCache map[string]Join
}

// NewCommonJoinResolver creates a new CommonJoinResolver instance.
func NewCommonJoinResolver(
	columnCache map[TableColumn]ColumnInformation,
	foreignKeyMap map[TableColumn]TableColumn,
) *CommonJoinResolver {
	return &CommonJoinResolver{
		columnCache:   columnCache,
		foreignKeyMap: foreignKeyMap,
		joinPathCache: make(map[string]Join),
	}
}

// Searches the table and field that match the foreign key column in a given table.
func (joinResolver CommonJoinResolver) findRelationTarget(joinSource TableColumn) (TableColumn, TableColumn, error) {
	// First, see if we can get an exact match, by applying some tricks (because its faster than iterating all possible values)
	shortcutKey := TableColumn{Table: joinSource.Table, Column: joinSource.Column + "_uuid"}
	if idMatch, exists := joinResolver.foreignKeyMap[shortcutKey]; exists {
		return idMatch, shortcutKey, nil
	}

	log.WithFields(
		"column", joinSource.Column,
		"table", joinSource.Table,
	).Warn("Unable to resolve column in table via lookup - using iteration approach")

	for k, v := range joinResolver.foreignKeyMap {
		if k.Table == joinSource.Table && strings.HasPrefix(k.Column, joinSource.Column) {
			return v, TableColumn{Table: joinSource.Table, Column: k.Column}, nil
		}
	}

	return TableColumn{}, TableColumn{}, errors.New("cannot find relation target")
}

// ResolvePath resolves an given path to a Join, which must be applied during query
// building for the query to succeed.
func (joinResolver *CommonJoinResolver) ResolvePath(joinPath string) (Join, error) {
	joinAlias := util.DescriptorToIdentifier(joinPath)

	// Has the path already been resolved previously? Then use the cached data
	if cachedJoin, exists := joinResolver.joinPathCache[joinAlias]; exists {
		return cachedJoin, nil
	}

	var origin string
	var joinSource string

	joinPaths := strings.Split(joinPath, "_")
	if len(joinPaths) > 2 {
		joinSource = util.DescriptorToIdentifier(strings.Join(joinPaths[0:len(joinPaths)-1], "_"))
		origin = joinSource

		if possibleSource, exists := joinResolver.joinPathCache[joinSource]; exists {
			joinSource = possibleSource.TargetTable()
		} else {
			log.Fatal("Unable to figure out table for " + joinSource)
		}
	} else {
		joinSource = util.DescriptorToIdentifier(joinPaths[0])
		origin = util.DescriptorToIdentifier(joinSource)
	}

	joinTargetField := util.DescriptorToIdentifier(joinPaths[len(joinPaths)-1])

	foreignLink, backLink, err := joinResolver.findRelationTarget(TableColumn{Table: joinSource, Column: joinTargetField})
	if err != nil {
		return Join{}, err
	}

	resolvedJoin := NewJoin(
		origin, // Don't use backLink.table, because we might be in a join chain! (e.g. person_organization)
		backLink.Column,
		foreignLink.Table,
		foreignLink.Column,
		joinAlias,
		LEFT,
	)

	// Cache resolved join alias for later retrieval
	if _, exists := joinResolver.joinPathCache[joinSource]; !exists {
		joinResolver.joinPathCache[joinAlias] = resolvedJoin
	}

	return resolvedJoin, nil
}

func (joinResolver *CommonJoinResolver) ResolveCountJoin(path string, schemaMapper config.SchemaMapper, keyResolver KeyResolver) (CountJoin, error) {
	pathParts := strings.Split(path, "_")

	countTargetTable, err := schemaMapper.Schema(strings.ToLower(pathParts[len(pathParts)-1]))
	if err != nil {
		return CountJoin{}, err
	}

	countOriginPaths := pathParts[0 : len(pathParts)-1]
	originTable := strings.Join(countOriginPaths, "_")

	countJoinOriginTable := ""
	if len(countOriginPaths) > 1 {
		// Its a join target, so we resolve the type of the preceding join first
		resolvedJoin, err := joinResolver.ResolvePath(util.DescriptorToIdentifier(strings.Join(pathParts[0:len(pathParts)-1], "_")))
		if err != nil {
			return CountJoin{}, err
		}

		countJoinOriginTable = resolvedJoin.TargetTable()
	} else {
		countJoinOriginTable = countOriginPaths[0]
	}

	originTablePrimaryKey := keyResolver.ResolvePrimaryKey(countTargetTable.Entity)[0]
	countRelation := keyResolver.ResolveRelation(countTargetTable.Entity, countJoinOriginTable)[0]

	return NewCountJoin(
		util.DescriptorToIdentifier(originTable),
		countRelation.PrimaryKey,
		util.DescriptorToIdentifier(countTargetTable.Entity),
		originTablePrimaryKey,
		countRelation.ForeignKey,
		util.DescriptorToIdentifier(path),
	), nil
}

// ExtractCommonJoinForeignKeyCache encapsulates common behavior to extract relations
// from a properly prepared sql.Rows instance. This implementation assumes that columns are
// returned in the following order: tableName, columnName, referencedTableName, referencedColumnName.
func ExtractCommonJoinForeignKeyCache(rows *sql.Rows) (map[TableColumn]TableColumn, error) {
	var (
		tableName, columnName, referencedTableName, referencedColumnName string
	)

	foreignKeyCache := make(map[TableColumn]TableColumn)
	for rows.Next() {
		err := rows.Scan(&tableName, &columnName, &referencedTableName, &referencedColumnName)
		if err != nil {
			return nil, err
		}

		foreignKeyCache[TableColumn{Table: tableName, Column: columnName}] = TableColumn{Table: referencedTableName, Column: referencedColumnName}
	}

	return foreignKeyCache, rows.Err()
}

// ExtractCommonColumnCache encapsulates common behavior to extract column information
// from a properly prepared sql.Rows instance. This implementation assumes that columns are
// returned in the following order: tableName, columnName, isNullable.
func ExtractCommonColumnCache(rows *sql.Rows) (map[TableColumn]ColumnInformation, error) {
	var (
		tableName, columnName, isNullable string
	)

	columnCache := make(map[TableColumn]ColumnInformation)
	for rows.Next() {
		err := rows.Scan(&tableName, &columnName, &isNullable)
		if err != nil {
			return nil, err
		}

		columnCache[TableColumn{Table: tableName, Column: columnName}] = ColumnInformation{Nullable: isNullable == "YES"}
	}

	return columnCache, rows.Err()
}
