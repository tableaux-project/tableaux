package sqlsource

import (
	"database/sql"

	"github.com/tableaux-project/tableaux/datasource/sqlsource/util"
)

// TableDoublet is a doublet of two tables.
type TableDoublet struct {
	OriginName, TargetName string
}

// TableKeyDoublet is a doublet of a primary and a referencing foreign key.
type TableKeyDoublet struct {
	PrimaryKey, ForeignKey string
}

// KeyResolver is a helping resolver, which resolves primary and
// foreign keys, as well as relations between the two for individual
// tables.
type KeyResolver interface {
	ResolvePrimaryKey(tableName string) []string
	ResolveRelation(originName, targetName string) []TableKeyDoublet
}

// CommonKeyResolver encapsulates common KeyResolver behavior,
// so that implementations only need to feed the objects caches
// for proper usage.
type CommonKeyResolver struct {
	// Cache to return primary keys for tables
	primaryKeyMap map[string][]string

	// Cache to map the relation of two tables to the referencing foreign key
	foreignKeyMap map[TableDoublet][]TableKeyDoublet
}

// NewCommonKeyResolver creates a new CommonKeyResolver instance.
func NewCommonKeyResolver(
	primaryKeyMap map[string][]string,
	foreignKeyMap map[TableDoublet][]TableKeyDoublet,
) *CommonKeyResolver {
	return &CommonKeyResolver{
		primaryKeyMap: primaryKeyMap,
		foreignKeyMap: foreignKeyMap,
	}
}

func (keyResolver *CommonKeyResolver) ResolvePrimaryKey(tableName string) []string {
	return keyResolver.primaryKeyMap[util.DescriptorToIdentifier(tableName)]
}

func (keyResolver *CommonKeyResolver) ResolveRelation(originName, targetName string) []TableKeyDoublet {
	return keyResolver.foreignKeyMap[TableDoublet{
		OriginName: util.DescriptorToIdentifier(originName),
		TargetName: util.DescriptorToIdentifier(targetName),
	}]
}

func ExtractCommonPrimaryKeyCache(rows *sql.Rows) (map[string][]string, error) {
	var (
		tableName, columnName string
	)

	primaryKeyCache := make(map[string][]string)
	for rows.Next() {
		err := rows.Scan(&tableName, &columnName)
		if err != nil {
			return nil, err
		}

		primaryKeyCache[tableName] = append(primaryKeyCache[tableName], columnName)
	}

	return primaryKeyCache, rows.Err()
}

func ExtractCommonForeignKeyCache(rows *sql.Rows) (map[TableDoublet][]TableKeyDoublet, error) {
	var (
		tableName, referencedTableName, columnName, referencedColumnName string
	)

	foreignKeyCache := make(map[TableDoublet][]TableKeyDoublet)
	for rows.Next() {
		err := rows.Scan(&tableName, &referencedTableName, &columnName, &referencedColumnName)
		if err != nil {
			return nil, err
		}

		key := TableDoublet{OriginName: tableName, TargetName: referencedTableName}
		value := TableKeyDoublet{PrimaryKey: referencedColumnName, ForeignKey: columnName}
		foreignKeyCache[key] = append(foreignKeyCache[key], value)
	}

	return foreignKeyCache, rows.Err()
}
