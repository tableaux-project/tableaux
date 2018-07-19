package sqlsource

import (
	"database/sql"
)

// DatabaseConnector is the central connector interface for tableaux,
// to access a database implementation. A DatabaseConnector is characterized
// by the fact that it exposes database specific implementation of numerous
// resolvers (which aid in establishing table relations), as well as a database
// specific QueryBuilder.
type DatabaseConnector interface {
	DatabaseVersion() (string, error)
	JoinResolver() JoinResolver
	KeyResolver() KeyResolver
	QueryBuilder() QueryBuilder
	DatabaseObject() *sql.DB

	Close() error

	MakeItemTypeSafe(item []byte, itemType *sql.ColumnType) (interface{}, error)
}

// CommonDatabaseConnector encapsulates the actual database interface and resolvers
// for ease of specific implementations. This way, implementing a DatabaseConnector
// involves encapsulating a CommonDatabaseConnector, and only filling in the missing
// methods for a DatabaseConnector.
type CommonDatabaseConnector struct {
	db           *sql.DB
	joinResolver JoinResolver
	keyResolver  KeyResolver
	queryBuilder QueryBuilder
}

// NewCommonDatabaseConnector constructs a new CommonDatabaseConnector instance,
// encapsulating the given database interface and resolvers.
func NewCommonDatabaseConnector(
	db *sql.DB,
	joinResolver JoinResolver,
	keyResolver KeyResolver,
	queryBuilder QueryBuilder,
) *CommonDatabaseConnector {
	return &CommonDatabaseConnector{
		db:           db,
		joinResolver: joinResolver,
		keyResolver:  keyResolver,
		queryBuilder: queryBuilder,
	}
}

// Close is a shortcut to closing the underlying database of the connector.
func (sqlDatabase CommonDatabaseConnector) Close() error {
	return sqlDatabase.db.Close()
}

// DatabaseObject exposes the raw database interface
func (sqlDatabase *CommonDatabaseConnector) DatabaseObject() *sql.DB {
	return sqlDatabase.db
}

// JoinResolver returns the used JoinResolver
func (sqlDatabase CommonDatabaseConnector) JoinResolver() JoinResolver {
	return sqlDatabase.joinResolver
}

// KeyResolver returns the used KeyResolver
func (sqlDatabase CommonDatabaseConnector) KeyResolver() KeyResolver {
	return sqlDatabase.keyResolver
}

// QueryBuilder returns the used QueryBuilder
func (sqlDatabase CommonDatabaseConnector) QueryBuilder() QueryBuilder {
	return sqlDatabase.queryBuilder
}
