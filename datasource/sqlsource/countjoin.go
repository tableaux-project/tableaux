package sqlsource

// CountJoin bundles all the attributes, which are required to count the amount
// of instances that are related to another entity.
type CountJoin struct {
	// The entity, from which the counting originates
	// (e.g. user_group)
	originEntity string

	// The primary key of the entity, from which the
	// counting originates
	// (e.g. user_group_uuid)
	originEntityPrimaryKey string

	// The entity that should be counted
	// (e.g. user)
	countEntity string

	// The primary key of the entity to be counted
	// (e.g. user_uuid)
	countEntityPrimaryKey string

	// The foreign key of the entity to be counted,
	// which references the origin entity
	// (e.g. user_group_id)
	countEntityForeignKey string

	// The alias to be used
	alias string
}

// NewCountJoin creates a new CountJoin instance.
func NewCountJoin(originEntity, originEntityPrimaryKey, countEntity, countEntityPrimaryKey,
	countEntityForeignKey, alias string) CountJoin {
	return CountJoin{
		originEntity:           originEntity,
		originEntityPrimaryKey: originEntityPrimaryKey,
		countEntity:            countEntity,
		countEntityPrimaryKey:  countEntityPrimaryKey,
		countEntityForeignKey:  countEntityForeignKey,
		alias: alias,
	}
}

// OriginEntity returns the entity, from which the counting originates.
func (count CountJoin) OriginEntity() string {
	return count.originEntity
}

// OriginEntityPrimaryKey returns primary key of the entity, from which
// the counting originates.
func (count CountJoin) OriginEntityPrimaryKey() string {
	return count.originEntityPrimaryKey
}

// CountEntity returns the entity that should be counted.
func (count CountJoin) CountEntity() string {
	return count.countEntity
}

// CountEntityPrimaryKey returns the primary key of the entity to be counted.
func (count CountJoin) CountEntityPrimaryKey() string {
	return count.countEntityPrimaryKey
}

// CountEntityForeignKey returns the foreign key of the entity to be counted,
// which references the origin entity
func (count CountJoin) CountEntityForeignKey() string {
	return count.countEntityForeignKey
}

// Alias returns alias to be used for the count join.
func (count CountJoin) Alias() string {
	return count.alias
}
