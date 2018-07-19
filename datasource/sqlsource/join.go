package sqlsource

// JoinType is the type of join to be applied, either LEFT or INNER.
type JoinType string

const (
	// LEFT describes an LEFT JOIN.
	LEFT JoinType = "LEFT"

	// INNER describes an INNER JOIN.
	INNER JoinType = "INNER"
)

// Join is an abstract description of a join to be applied while constructing the query.
type Join struct {
	// e.g. person
	sourceTable string

	// The foreign key for the join source
	// e.g. organization_id
	sourceColumn string

	// e.g. organization
	targetTable string

	// The primary key of the target table
	// e.g. organization_uuid
	targetColumn string

	// e.g. person_organization
	joinAlias string

	// The type of the join, either LEFT or INNER
	joinType JoinType
}

// NewJoin creates a new Join instance.
func NewJoin(sourceTable, sourceColumn, targetTable, targetColumn, joinAlias string, joinType JoinType) Join {
	return Join{
		sourceTable:  sourceTable,
		sourceColumn: sourceColumn,
		targetTable:  targetTable,
		targetColumn: targetColumn,
		joinAlias:    joinAlias,
		joinType:     joinType,
	}
}

// SourceTable returns the source table from which the join originates.
func (Resolved Join) SourceTable() string {
	return Resolved.sourceTable
}

// SourceColumn returns the foreign key of the table from which the join
// originates.
func (Resolved Join) SourceColumn() string {
	return Resolved.sourceColumn
}

// TargetTable returns the target table, which should be joined with the
// source table.
func (Resolved Join) TargetTable() string {
	return Resolved.targetTable
}

// TargetColumn returns the primary key of the target table.
func (Resolved Join) TargetColumn() string {
	return Resolved.targetColumn
}

// JoinAlias returns alias to be used for the join.
func (Resolved Join) JoinAlias() string {
	return Resolved.joinAlias
}

// JoinType returns the type of the join, either LEFT or INNER
func (Resolved Join) JoinType() JoinType {
	return Resolved.joinType
}
