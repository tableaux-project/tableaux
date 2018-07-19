package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/birkirb/loggers.v1/log"
)

var (
	// ErrUnknownSchema indicates that a requested schema is not
	// known to a SchemaMapper.
	ErrUnknownSchema = errors.New("unknown schema")

	// ErrUnknownColumn indicates that a requested column is
	// not known to a TableSchema.
	ErrUnknownColumn = errors.New("unknown column")
)

// UnresolvableSchemaError indicates that a schema that was required to be
// resolved during schema loading could not be found.
type UnresolvableSchemaError struct {
	schema string
}

func (e UnresolvableSchemaError) Error() string {
	return fmt.Sprintf("cannot resolve table schema %s", e.schema)
}

// UnknownColumnTypeError indicates that a unknown column type (neither
// primitive nor known enum type) was found during integrity checking of
// a TableSchema.
type UnknownColumnTypeError struct {
	schema     string
	column     string
	columnType string
}

func (e UnknownColumnTypeError) Error() string {
	return fmt.Sprintf("Unknown column type %s in column %s of schema %s", e.columnType, e.column, e.schema)
}

// TableSchemaExclusion is a wrapper to describe a column path prefix that
// is to be eliminated after a table schema was resolved.
type TableSchemaExclusion string

// TableSchema describes the schema for a single table, with all its
// meta data.
type TableSchema struct {
	Entity     string                      `json:"entity"`
	Extensions []TableSchemaExtensionTable `json:"extensions"`
	Exclusions []TableSchemaExclusion      `json:"exclusions"`
	Columns    []TableSchemaColumn         `json:"columns"`
}

var validColumnTypes = map[string]struct{}{
	"boolean":  {},
	"integer":  {},
	"long":     {},
	"string":   {},
	"date":     {},
	"datetime": {},
}

// ValidateIntegrity checks that the schema is valid. The given EnumMapper is
// used to check that referenced enums exist.
func (schema TableSchema) ValidateIntegrity(mapper EnumMapper) error {
	for _, column := range schema.Columns {
		columnType := column.Type
		if _, exists := validColumnTypes[strings.ToLower(columnType)]; !exists {
			if _, err := mapper.Enum(columnType); err != nil {
				return &UnknownColumnTypeError{
					schema:     schema.Entity,
					column:     column.Path,
					columnType: columnType,
				}
			}
		}
	}

	return nil
}

// ResolvedTableSchema describes the resolved schema for a table, that is
// resolving all the extensions of a TableSchema, and assembling its columns
// (while deleting columns applying to the exclusions).
type ResolvedTableSchema struct {
	originalSchema TableSchema
	columns        []TableSchemaColumn
	columnsMap     map[string]TableSchemaColumn
}

// OriginalSchema returns the original TableSchema without extended columns.
func (resolvedTableSchema ResolvedTableSchema) OriginalSchema() TableSchema {
	return resolvedTableSchema.originalSchema
}

// Column retrieves the TableSchemaColumn a single column key, or
// returns an ErrUnknownColumn, if the column does not exist.
func (resolvedTableSchema ResolvedTableSchema) Column(key string) (TableSchemaColumn, error) {
	if _, exists := resolvedTableSchema.columnsMap[key]; !exists {
		return TableSchemaColumn{}, ErrUnknownColumn
	}

	return resolvedTableSchema.columnsMap[key], nil
}

// Columns returns all columns in resolved order.
func (resolvedTableSchema ResolvedTableSchema) Columns() []TableSchemaColumn {
	columns := make([]TableSchemaColumn, len(resolvedTableSchema.columns))
	copy(columns, resolvedTableSchema.columns)

	return columns
}

// TableSchemaColumn is a single column of a TableSchema, defining all the
// required (and optional) properties for working with a single column.
type TableSchemaColumn struct {
	Title         string                 `json:"title"`
	Path          string                 `json:"path"`
	Type          string                 `json:"type"`
	Filter        string                 `json:"filter"`
	Order         string                 `json:"order"`
	PathResolver  string                 `json:"pathResolver"`
	FrontendHints map[string]interface{} `json:"frontendHints"`
}

// TableSchemaExtensionTable describes an extension for one TableSchema
// to another.
type TableSchemaExtensionTable struct {
	Title string `json:"title"`
	Table string `json:"table"`
	Key   string `json:"key"`
}

// SchemaMapper is a mapper which maps schema names to resolved schemas.
type SchemaMapper struct {
	schemas         map[string]TableSchema
	resolvedSchemas map[string]ResolvedTableSchema
}

func readFromPath(schemaPath string) (TableSchema, error) {
	file, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return TableSchema{}, err
	}

	dat := TableSchema{}
	if err := json.Unmarshal(file, &dat); err != nil {
		return TableSchema{}, err
	}

	return dat, nil
}

// NewSchemaMapperFromFolder builds a new schema mapper from a given folder,
// recursively loading all enum jsons which are found in there.
func NewSchemaMapperFromFolder(schemaRoot string) (SchemaMapper, error) {
	// Normalize the path, and eliminate separator inconsistencies
	normalizedRoot, err := filepath.Abs(schemaRoot)
	if err != nil {
		return SchemaMapper{}, err
	}

	schemas := make(map[string]TableSchema)
	if walkErr := filepath.Walk(normalizedRoot, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == dotJSON {
			schema, err := readFromPath(path)
			if err != nil {
				return err
			}

			schemas[normalizeSchemaKey(path, normalizedRoot)] = schema
		} else if !f.IsDir() {
			log.WithField("file", path).Debug("Ignoring file, as not a json file!")
		}

		return nil
	}); walkErr != nil {
		return SchemaMapper{}, walkErr
	}

	log.WithField("count", len(schemas)).Info("Successfully loaded schemas")

	// ----------

	resolvedSchemas, err := mapSchemasToResolvedSchemas(schemas)
	if err != nil {
		return SchemaMapper{}, err
	}

	return SchemaMapper{
		schemas:         schemas,
		resolvedSchemas: resolvedSchemas,
	}, nil
}

// normalizeSchemaKey calculates the name of a schema by its path relative
// to the root of all schemas. This method also normalizes system specific
// path separators (e.g. / or \) to "/".
func normalizeSchemaKey(schemaPath, schemaRoot string) string {
	normalizedPath := strings.TrimSuffix(
		// Remove Extension
		strings.TrimPrefix(
			// Remove Separator
			strings.TrimPrefix(
				// Remove schemaRoot
				schemaPath,
				schemaRoot,
			),
			string(os.PathSeparator),
		),
		filepath.Ext(schemaPath),
	)

	return strings.Replace(strings.ToLower(normalizedPath), string(os.PathSeparator), "/", -1)
}

func mapSchemasToResolvedSchemas(schemas map[string]TableSchema) (map[string]ResolvedTableSchema, error) {
	resolvedSchemas := make(map[string]ResolvedTableSchema, len(schemas))

	for table, schema := range schemas {
		resolvedColumns, err := resolveColumns(schema, schemas)
		if err != nil {
			return nil, err
		}

		resolvedColumnsMaps := make(map[string]TableSchemaColumn)
		for _, column := range resolvedColumns {
			resolvedColumnsMaps[column.Path] = column
		}

		resolvedSchemas[table] = ResolvedTableSchema{
			originalSchema: schema,
			columns:        resolvedColumns,
			columnsMap:     resolvedColumnsMaps,
		}
	}

	return resolvedSchemas, nil
}

// Schema retrieves a specific schema from the mapper if existing, or returns
// a ErrUnknownSchema otherwise.
func (schemaMapper SchemaMapper) Schema(schema string) (TableSchema, error) {
	if _, exists := schemaMapper.schemas[schema]; !exists {
		return TableSchema{}, ErrUnknownSchema
	}

	return schemaMapper.schemas[schema], nil
}

// Schemas returns all schemas which the mapper knows, in no particular order.
func (schemaMapper SchemaMapper) Schemas() []TableSchema {
	schemas := make([]TableSchema, len(schemaMapper.schemas))

	i := 0
	for _, v := range schemaMapper.schemas {
		schemas[i] = v
		i++
	}

	return schemas
}

// ResolvedSchema retrieves a specific resolved schema from the mapper if existing,
// or returns a ErrUnknownSchema otherwise.
func (schemaMapper SchemaMapper) ResolvedSchema(schema string) (ResolvedTableSchema, error) {
	if _, exists := schemaMapper.resolvedSchemas[schema]; !exists {
		return ResolvedTableSchema{}, ErrUnknownSchema
	}

	return schemaMapper.resolvedSchemas[schema], nil
}

// ResolvedSchemas returns all resolved schemas which the mapper knows, mapped by
// their path.
func (schemaMapper SchemaMapper) ResolvedSchemas() map[string]ResolvedTableSchema {
	schemas := make(map[string]ResolvedTableSchema, len(schemaMapper.resolvedSchemas))

	for k, v := range schemaMapper.resolvedSchemas {
		schemas[k] = v
	}

	return schemas
}

// ValidateIntegrity iteratively checks all schemas known to the mapper for integrity.
// The given EnumMapper is used to check that all referenced enums exist.
func (schemaMapper SchemaMapper) ValidateIntegrity(mapper EnumMapper) error {
	for _, schema := range schemaMapper.schemas {
		if err := schema.ValidateIntegrity(mapper); err != nil {
			return err
		}
	}

	return nil
}

func resolveColumns(schema TableSchema, allSchemas map[string]TableSchema) ([]TableSchemaColumn, error) {
	newColumns, err := resolveColumnsWithPrefix(schema, allSchemas, "")
	if err != nil {
		return nil, err
	}

	if schema.Exclusions != nil {
		deletableColumns := resolveDeletableColumns(newColumns, schema)

		log.WithFields(
			"columns", len(deletableColumns),
			"schema", schema.Entity,
		).Debug("Removing columns from schema")

		finalColumns := make([]TableSchemaColumn, len(newColumns)-len(deletableColumns))

		i := 0
		for _, column := range newColumns {
			key := column.Path

			if _, exists := deletableColumns[key]; !exists {
				finalColumns[i] = column
				i++
			}
		}

		newColumns = finalColumns
	}

	return newColumns, nil
}

func resolveDeletableColumns(newColumns []TableSchemaColumn, schema TableSchema) map[string]struct{} {
	deletableColumns := make(map[string]struct{})

	for _, column := range newColumns {
		key := column.Path
		for _, exclusion := range schema.Exclusions {
			if strings.TrimPrefix(key, string(exclusion)) != key {
				deletableColumns[key] = struct{}{}
				break
			}
		}
	}

	return deletableColumns
}

func resolveColumnsWithPrefix(resolvableSchema TableSchema, allSchemas map[string]TableSchema, prefix string) ([]TableSchemaColumn, error) {
	newColumns := make([]TableSchemaColumn, len(resolvableSchema.Columns))

	for i := 0; i < len(resolvableSchema.Columns); i++ {
		column := resolvableSchema.Columns[i]

		resolvedColumn := resolveColumnWithPrefix(column, prefix)
		newColumns[i] = resolvedColumn
	}

	for i := 0; i < len(resolvableSchema.Extensions); i++ {
		table := resolvableSchema.Extensions[i]
		targetExtensionTable, exists := allSchemas[table.Table]
		if !exists {
			return nil, &UnresolvableSchemaError{schema: table.Table}
		}

		extensionString := resolvableSchema.Entity
		if prefix != "" {
			extensionString = prefix
		}

		// Only prefix if we have a PathPrefix and/or a table key
		if extensionString != "" && table.Key != "" {
			extensionString += "_" + table.Key
		} else if table.Key != "" {
			extensionString = table.Key
		}

		resolvedColumns, err := resolveColumnsWithPrefix(targetExtensionTable, allSchemas, extensionString)
		if err != nil {
			return nil, err
		}

		newColumns = append(newColumns, resolvedColumns...)
	}

	return newColumns, nil
}

func resolveColumnWithPrefix(column TableSchemaColumn, prefix string) TableSchemaColumn {
	var path string
	if prefix != "" {
		pathStart := strings.Index(column.Path, "_") + 1

		path = column.Path[pathStart:len(column.Path)]
		if prefix != "" {
			path = prefix + "_" + path
		}
	} else {
		path = column.Path
	}

	return TableSchemaColumn{
		Title:         column.Title,
		Path:          path,
		Type:          column.Type,
		Filter:        column.Filter,
		Order:         column.Order,
		PathResolver:  column.PathResolver,
		FrontendHints: column.FrontendHints,
	}
}
