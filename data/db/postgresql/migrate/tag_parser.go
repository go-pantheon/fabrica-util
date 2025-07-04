package migrate

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-pantheon/fabrica-util/camelcase"
	"github.com/go-pantheon/fabrica-util/errors"
)

// ColumnTag represents parsed column configuration from struct tags
type ColumnTag struct {
	Column       string            // Column name
	Type         string            // Column type
	IsPrimaryKey bool              // Primary key flag
	IsAutoIncr   bool              // Auto increment flag
	IsNotNull    bool              // Not null flag
	IsUnique     bool              // Unique flag
	IsIndex      bool              // Index flag
	DefaultValue string            // Default value
	Size         int               // Column size
	Precision    int               // Decimal precision
	Scale        int               // Decimal scale
	Comment      string            // Column comment
	Extra        map[string]string // Extra attributes
}

// TagParser handles parsing of ORM struct tags
type TagParser struct {
	supportedTypes map[string]bool
	validators     map[string]func(string) error
}

// NewTagParser creates a new tag parser instance
func NewTagParser() *TagParser {
	parser := &TagParser{
		supportedTypes: make(map[string]bool),
		validators:     make(map[string]func(string) error),
	}

	parser.initSupportedTypes()
	parser.initValidators()

	return parser
}

// initSupportedTypes initializes supported PostgreSQL data types
func (p *TagParser) initSupportedTypes() {
	types := []string{
		"BOOLEAN", "BOOL",
		"SMALLINT", "INT2", "INTEGER", "INT", "INT4", "BIGINT", "INT8",
		"SMALLSERIAL", "SERIAL2", "SERIAL", "SERIAL4", "BIGSERIAL", "SERIAL8",
		"DECIMAL", "NUMERIC", "REAL", "FLOAT4", "DOUBLE PRECISION", "FLOAT8",
		"CHAR", "CHARACTER", "VARCHAR", "CHARACTER VARYING", "TEXT",
		"BYTEA", "TIMESTAMP", "TIMESTAMPTZ", "DATE", "TIME", "TIMETZ",
		"INTERVAL", "UUID", "JSON", "JSONB", "XML",
		"INET", "CIDR", "MACADDR", "MACADDR8",
		"POINT", "LINE", "LSEG", "BOX", "PATH", "POLYGON", "CIRCLE",
		"TSVECTOR", "TSQUERY",
	}

	for _, t := range types {
		p.supportedTypes[t] = true
	}
}

// initValidators initializes parameter validators
func (p *TagParser) initValidators() {
	p.validators["column"] = p.validateColumnName
	p.validators["type"] = p.validateColumnType
	p.validators["size"] = p.validateSize
	p.validators["precision"] = p.validatePrecision
	p.validators["scale"] = p.validateScale
	p.validators["default"] = p.validateDefault
	p.validators["comment"] = p.validateComment
}

// ParseTag parses the orm tag from a struct field
//
//nolint:gocognit
func (p *TagParser) ParseTag(field reflect.StructField) (*ColumnTag, error) {
	if !field.IsExported() {
		return nil, nil
	}

	tag := field.Tag.Get("orm")
	if tag == "" {
		// Check if this is a time.Time field (auto-include)
		if field.Type == reflect.TypeOf(time.Time{}) ||
			(field.Type.Kind() == reflect.Ptr && field.Type.Elem() == reflect.TypeOf(time.Time{})) {
			return p.createDefaultTimeColumn(field), nil
		}

		// Check if this is a basic type that should be included
		if p.shouldIncludeField(field) {
			return p.createDefaultColumn(field), nil
		}

		return nil, nil
	}

	columnTag := &ColumnTag{
		Extra: make(map[string]string),
	}

	// Parse tag parameters
	params, err := p.parseTagParameters(tag)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse tag for field %s", field.Name)
	}

	// Set column name
	if columnName, exists := params["column"]; exists {
		columnTag.Column = columnName
	} else {
		columnTag.Column = camelcase.ToUnderScore(field.Name)
	}

	// Parse simple flags
	columnTag.IsPrimaryKey = p.hasFlag(params, "primaryKey", "primary_key")
	columnTag.IsAutoIncr = p.hasFlag(params, "autoIncrement", "auto_increment")
	columnTag.IsNotNull = p.hasFlag(params, "notNull", "not_null")
	columnTag.IsUnique = p.hasFlag(params, "unique")
	columnTag.IsIndex = p.hasFlag(params, "index")

	// Parse key-value parameters
	if colType, exists := params["type"]; exists {
		columnTag.Type = colType
	}

	if defaultVal, exists := params["default"]; exists {
		columnTag.DefaultValue = defaultVal
	}

	if comment, exists := params["comment"]; exists {
		columnTag.Comment = comment
	}

	if size, exists := params["size"]; exists {
		if s, err := strconv.Atoi(size); err == nil {
			columnTag.Size = s
		}
	}

	if precision, exists := params["precision"]; exists {
		if p, err := strconv.Atoi(precision); err == nil {
			columnTag.Precision = p
		}
	}

	if scale, exists := params["scale"]; exists {
		if s, err := strconv.Atoi(scale); err == nil {
			columnTag.Scale = s
		}
	}

	// Store extra parameters
	for key, value := range params {
		if !p.isKnownParameter(key) {
			columnTag.Extra[key] = value
		}
	}

	// Infer type if not specified
	if columnTag.Type == "" {
		columnTag.Type = p.inferTypeFromField(field, columnTag)
	}

	// Validate the configuration
	if err := p.validateColumnTag(columnTag); err != nil {
		return nil, errors.Wrapf(err, "validation failed for field %s", field.Name)
	}

	return columnTag, nil
}

// parseTagParameters parses tag parameters like "primaryKey;size:255;default:0"
func (p *TagParser) parseTagParameters(tag string) (map[string]string, error) {
	params := make(map[string]string)

	// Split by semicolon
	parts := strings.SplitSeq(tag, ";")

	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it's a key-value pair
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])

				// Validate parameter
				if validator, exists := p.validators[key]; exists {
					if err := validator(value); err != nil {
						return nil, errors.Wrapf(err, "invalid value for parameter %s", key)
					}
				}

				params[key] = value
			}
		} else {
			// Simple flag
			params[part] = "true"
		}
	}

	return params, nil
}

// hasFlag checks if a flag parameter exists
func (p *TagParser) hasFlag(params map[string]string, names ...string) bool {
	for _, name := range names {
		if value, exists := params[name]; exists && value == "true" {
			return true
		}
	}
	return false
}

// isKnownParameter checks if a parameter is known/reserved
func (p *TagParser) isKnownParameter(key string) bool {
	knownParams := []string{
		"column", "type", "size", "precision", "scale", "default", "comment",
		"primaryKey", "primary_key", "autoIncrement", "auto_increment",
		"notNull", "not_null", "unique", "index",
	}

	for _, param := range knownParams {
		if param == key {
			return true
		}
	}
	return false
}

// shouldIncludeField determines if a field should be included by default
func (p *TagParser) shouldIncludeField(field reflect.StructField) bool {
	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
		return true
	case reflect.Struct:
		return fieldType == reflect.TypeOf(time.Time{})
	default:
		return false
	}
}

// createDefaultColumn creates a default column configuration
func (p *TagParser) createDefaultColumn(field reflect.StructField) *ColumnTag {
	return &ColumnTag{
		Column: camelcase.ToUnderScore(field.Name),
		Type:   p.inferTypeFromField(field, &ColumnTag{}),
		Extra:  make(map[string]string),
	}
}

// createDefaultTimeColumn creates a default time column configuration
func (p *TagParser) createDefaultTimeColumn(field reflect.StructField) *ColumnTag {
	return &ColumnTag{
		Column: camelcase.ToUnderScore(field.Name),
		Type:   "TIMESTAMPTZ",
		Extra:  make(map[string]string),
	}
}

// inferTypeFromField infers PostgreSQL type from Go field type
func (p *TagParser) inferTypeFromField(field reflect.StructField, columnTag *ColumnTag) string {
	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Handle primary key auto increment
	if columnTag.IsPrimaryKey && columnTag.IsAutoIncr {
		switch fieldType.Kind() {
		case reflect.Int, reflect.Int32, reflect.Uint, reflect.Uint32:
			return "SERIAL"
		case reflect.Int64, reflect.Uint64:
			return "BIGSERIAL"
		default:
			return "SERIAL"
		}
	}

	// Handle regular types
	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INTEGER"
	case reflect.Int64, reflect.Uint64:
		return "BIGINT"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		if columnTag.Precision > 0 {
			if columnTag.Scale > 0 {
				return fmt.Sprintf("NUMERIC(%d,%d)", columnTag.Precision, columnTag.Scale)
			}
			return fmt.Sprintf("NUMERIC(%d)", columnTag.Precision)
		}
		return "DOUBLE PRECISION"
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.String:
		if columnTag.Size > 0 {
			return fmt.Sprintf("VARCHAR(%d)", columnTag.Size)
		}
		return "TEXT"
	case reflect.Struct:
		if fieldType == reflect.TypeOf(time.Time{}) {
			return "TIMESTAMPTZ"
		}
		return "JSONB"
	case reflect.Slice:
		if fieldType.Elem().Kind() == reflect.Uint8 {
			return "BYTEA"
		}
		return "JSONB"
	default:
		return "JSONB"
	}
}

// Validators

func (p *TagParser) validateColumnName(name string) error {
	if name == "" {
		return errors.New("column name cannot be empty")
	}

	// Check for valid identifier
	matched, err := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	if err != nil {
		return err
	}
	if !matched {
		return errors.New("column name must be a valid identifier")
	}

	// Check for reserved words (basic check)
	reservedWords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "FROM", "WHERE", "ORDER", "BY",
		"GROUP", "HAVING", "JOIN", "INNER", "LEFT", "RIGHT", "FULL", "OUTER",
		"UNION", "ALL", "DISTINCT", "AS", "ON", "IN", "EXISTS", "BETWEEN",
		"LIKE", "IS", "NULL", "NOT", "AND", "OR", "TRUE", "FALSE",
		"CREATE", "ALTER", "DROP", "TABLE", "INDEX", "VIEW", "DATABASE",
		"SCHEMA", "GRANT", "REVOKE", "COMMIT", "ROLLBACK", "TRANSACTION",
	}

	upperName := strings.ToUpper(name)
	for _, word := range reservedWords {
		if upperName == word {
			return fmt.Errorf("column name '%s' is a reserved word", name)
		}
	}

	return nil
}

func (p *TagParser) validateColumnType(colType string) error {
	if colType == "" {
		return errors.New("column type cannot be empty")
	}

	// Extract base type (handle cases like VARCHAR(255), NUMERIC(10,2))
	baseType := strings.ToUpper(colType)

	// Remove parameters in parentheses
	if idx := strings.Index(baseType, "("); idx != -1 {
		baseType = baseType[:idx]
	}

	// Remove array notation
	baseType = strings.TrimSuffix(baseType, "[]")

	// Check if it's a supported type
	if !p.supportedTypes[baseType] {
		return fmt.Errorf("unsupported column type: %s", colType)
	}

	return nil
}

func (p *TagParser) validateSize(size string) error {
	s, err := strconv.Atoi(size)
	if err != nil {
		return errors.New("size must be a valid integer")
	}
	if s <= 0 {
		return errors.New("size must be positive")
	}
	if s > 65535 {
		return errors.New("size too large (max 65535)")
	}
	return nil
}

func (p *TagParser) validatePrecision(precision string) error {
	prec, err := strconv.Atoi(precision)
	if err != nil {
		return errors.New("precision must be a valid integer")
	}
	if prec <= 0 {
		return errors.New("precision must be positive")
	}
	if prec > 1000 {
		return errors.New("precision too large (max 1000)")
	}
	return nil
}

func (p *TagParser) validateScale(scale string) error {
	s, err := strconv.Atoi(scale)
	if err != nil {
		return errors.New("scale must be a valid integer")
	}
	if s < 0 {
		return errors.New("scale must be non-negative")
	}
	if s > 1000 {
		return errors.New("scale too large (max 1000)")
	}
	return nil
}

func (p *TagParser) validateDefault(defaultVal string) error {
	// Basic validation - could be expanded based on column type
	if defaultVal == "" {
		return errors.New("default value cannot be empty")
	}

	// Check for SQL injection patterns (basic check)
	dangerous := []string{";", "--", "/*", "*/", "xp_", "sp_"}
	lowerVal := strings.ToLower(defaultVal)
	for _, pattern := range dangerous {
		if strings.Contains(lowerVal, pattern) {
			return errors.Errorf("default value contains potentially dangerous pattern: %s", pattern)
		}
	}

	return nil
}

func (p *TagParser) validateComment(comment string) error {
	if len(comment) > 1024 {
		return errors.New("comment too long (max 1024 characters)")
	}
	return nil
}

func (p *TagParser) validateColumnTag(tag *ColumnTag) error {
	// Validate column name
	if err := p.validateColumnName(tag.Column); err != nil {
		return err
	}

	// Validate column type
	if err := p.validateColumnType(tag.Type); err != nil {
		return err
	}

	// Validate scale vs precision
	if tag.Scale > 0 && tag.Precision > 0 {
		if tag.Scale > tag.Precision {
			return errors.New("scale cannot be greater than precision")
		}
	}

	// Validate auto increment only with primary key
	if tag.IsAutoIncr && !tag.IsPrimaryKey {
		return errors.New("autoIncrement can only be used with primaryKey")
	}

	return nil
}

// BuildColumnDefinition builds the SQL column definition from the parsed tag
func (p *TagParser) BuildColumnDefinition(tag *ColumnTag) string {
	var parts []string

	// Column name
	parts = append(parts, fmt.Sprintf(`"%s"`, tag.Column))

	// Column type
	columnType := tag.Type

	// Handle size for VARCHAR and CHAR
	if tag.Size > 0 {
		upperType := strings.ToUpper(columnType)
		switch upperType {
		case "VARCHAR":
			columnType = fmt.Sprintf("VARCHAR(%d)", tag.Size)
		case "CHAR", "CHARACTER":
			columnType = fmt.Sprintf("CHAR(%d)", tag.Size)
		}
	}

	// Handle precision and scale for NUMERIC
	if tag.Precision > 0 && (strings.ToUpper(columnType) == "NUMERIC" || strings.ToUpper(columnType) == "DECIMAL") {
		if tag.Scale > 0 {
			columnType = fmt.Sprintf("NUMERIC(%d,%d)", tag.Precision, tag.Scale)
		} else {
			columnType = fmt.Sprintf("NUMERIC(%d)", tag.Precision)
		}
	}

	parts = append(parts, columnType)

	// Constraints
	if tag.IsPrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}

	if tag.IsNotNull {
		parts = append(parts, "NOT NULL")
	}

	if tag.IsUnique {
		parts = append(parts, "UNIQUE")
	}

	if tag.DefaultValue != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", tag.DefaultValue))
	}

	return strings.Join(parts, " ")
}
