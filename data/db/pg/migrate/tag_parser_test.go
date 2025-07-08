package migrate

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagParser_New(t *testing.T) {
	t.Parallel()

	// Test creating new tag parser
	parser := NewTagParser()

	assert.NotNil(t, parser)
}

func TestTagParser_ParseTag_Simple(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test simple tag parsing
	field := reflect.StructField{
		Name: "ID",
		Type: reflect.TypeOf(int(0)),
		Tag:  `orm:"primaryKey"`,
	}

	tag, err := parser.ParseTag(field)
	require.NoError(t, err)
	require.NotNil(t, tag)

	assert.True(t, tag.IsPrimaryKey)
	assert.False(t, tag.IsUnique)
	assert.False(t, tag.IsNotNull)
	assert.Equal(t, "id", tag.Column)
	assert.NotEmpty(t, tag.Type)
}

func TestTagParser_ParseTag_Complex(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test complex tag parsing
	field := reflect.StructField{
		Name: "Email",
		Type: reflect.TypeOf(""),
		Tag:  `orm:"size:255;unique;notNull;default:test@example.com"`,
	}

	tag, err := parser.ParseTag(field)
	require.NoError(t, err)
	require.NotNil(t, tag)

	assert.Equal(t, "email", tag.Column)
	assert.Equal(t, 255, tag.Size)
	assert.True(t, tag.IsUnique)
	assert.True(t, tag.IsNotNull)
	assert.Equal(t, "test@example.com", tag.DefaultValue)
}

func TestTagParser_ParseTag_ColumnName(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test parsing tag with column name
	field := reflect.StructField{
		Name: "UserName",
		Type: reflect.TypeOf(""),
		Tag:  `orm:"column:user_name;size:100"`,
	}

	tag, err := parser.ParseTag(field)
	require.NoError(t, err)
	require.NotNil(t, tag)

	assert.Equal(t, "user_name", tag.Column)
	assert.Equal(t, 100, tag.Size)
	assert.False(t, tag.IsPrimaryKey)
}

func TestTagParser_ParseTag_Type(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test parsing tag with type
	field := reflect.StructField{
		Name: "Content",
		Type: reflect.TypeOf(""),
		Tag:  `orm:"type:TEXT;notNull"`,
	}

	tag, err := parser.ParseTag(field)
	require.NoError(t, err)
	require.NotNil(t, tag)

	assert.Equal(t, "TEXT", tag.Type)
	assert.True(t, tag.IsNotNull)
}

func TestTagParser_ParseTag_PrecisionScale(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test parsing tag with precision and scale
	field := reflect.StructField{
		Name: "Price",
		Type: reflect.TypeOf(float64(0)),
		Tag:  `orm:"precision:10;scale:2"`,
	}

	tag, err := parser.ParseTag(field)
	require.NoError(t, err)
	require.NotNil(t, tag)

	assert.Equal(t, 10, tag.Precision)
	assert.Equal(t, 2, tag.Scale)
}

func TestTagParser_ParseTag_NoTag(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test field without orm tag
	field := reflect.StructField{
		Name: "InternalField",
		Type: reflect.TypeOf(""),
		Tag:  `json:"internal_field"`,
	}

	tag, err := parser.ParseTag(field)
	require.NoError(t, err)

	// String fields are auto-included even without orm tag
	require.NotNil(t, tag)
	assert.Equal(t, "internal_field", tag.Column)
	assert.Equal(t, "TEXT", tag.Type)
}

func TestTagParser_ParseTag_TimeField(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test time.Time field without orm tag (should be auto-included)
	field := reflect.StructField{
		Name: "CreatedAt",
		Type: reflect.TypeOf(time.Time{}),
		Tag:  ``,
	}

	tag, err := parser.ParseTag(field)
	require.NoError(t, err)
	require.NotNil(t, tag)

	assert.Equal(t, "created_at", tag.Column)
	assert.NotEmpty(t, tag.Type)
}

func TestTagParser_ParseTag_TypeInference(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test type inference for different Go types
	testCases := []struct {
		name         string
		goType       reflect.Type
		expectedType string
	}{
		{"string", reflect.TypeOf(""), "TEXT"},
		{"int", reflect.TypeOf(int(0)), "INTEGER"},
		{"int64", reflect.TypeOf(int64(0)), "BIGINT"},
		{"float64", reflect.TypeOf(float64(0)), "DOUBLE PRECISION"},
		{"bool", reflect.TypeOf(true), "BOOLEAN"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			field := reflect.StructField{
				Name: "TestField",
				Type: tc.goType,
				Tag:  `orm:"notNull"`,
			}

			tag, err := parser.ParseTag(field)
			require.NoError(t, err)
			require.NotNil(t, tag)

			assert.Equal(t, tc.expectedType, tag.Type)
		})
	}
}

func TestTagParser_BuildColumnDefinition(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test building column definition
	tag := &ColumnTag{
		Column:       "email",
		Type:         "VARCHAR",
		Size:         255,
		IsUnique:     true,
		IsNotNull:    true,
		DefaultValue: "''",
	}

	definition := parser.BuildColumnDefinition(tag)

	assert.Contains(t, definition, "VARCHAR(255)")
	assert.Contains(t, definition, "UNIQUE")
	assert.Contains(t, definition, "NOT NULL")
	assert.Contains(t, definition, "DEFAULT ''")
}

func TestTagParser_BuildColumnDefinition_WithPrecisionScale(t *testing.T) {
	t.Parallel()

	parser := NewTagParser()

	// Test building column definition with precision and scale
	tag := &ColumnTag{
		Column:    "price",
		Type:      "NUMERIC",
		Precision: 10,
		Scale:     2,
		IsNotNull: true,
	}

	definition := parser.BuildColumnDefinition(tag)

	assert.Contains(t, definition, "NUMERIC(10,2)")
	assert.Contains(t, definition, "NOT NULL")
}

func TestColumnTag_DefaultValues(t *testing.T) {
	t.Parallel()

	// Test default values of ColumnTag
	tag := ColumnTag{}

	assert.False(t, tag.IsPrimaryKey)
	assert.False(t, tag.IsUnique)
	assert.False(t, tag.IsNotNull)
	assert.False(t, tag.IsAutoIncr)
	assert.False(t, tag.IsIndex)
	assert.Empty(t, tag.Column)
	assert.Empty(t, tag.Type)
	assert.Zero(t, tag.Size)
	assert.Zero(t, tag.Precision)
	assert.Zero(t, tag.Scale)
	assert.Empty(t, tag.DefaultValue)
	assert.Empty(t, tag.Comment)
}

func TestColumnTag_AllFields(t *testing.T) {
	t.Parallel()

	// Test all fields of ColumnTag
	extra := make(map[string]string)
	extra["custom"] = "value"

	tag := ColumnTag{
		Column:       "test_column",
		Type:         "VARCHAR",
		Size:         100,
		Precision:    10,
		Scale:        2,
		IsPrimaryKey: true,
		IsAutoIncr:   true,
		IsUnique:     true,
		IsNotNull:    true,
		IsIndex:      true,
		DefaultValue: "default_value",
		Comment:      "test comment",
		Extra:        extra,
	}

	assert.Equal(t, "test_column", tag.Column)
	assert.Equal(t, "VARCHAR", tag.Type)
	assert.Equal(t, 100, tag.Size)
	assert.Equal(t, 10, tag.Precision)
	assert.Equal(t, 2, tag.Scale)
	assert.True(t, tag.IsPrimaryKey)
	assert.True(t, tag.IsAutoIncr)
	assert.True(t, tag.IsUnique)
	assert.True(t, tag.IsNotNull)
	assert.True(t, tag.IsIndex)
	assert.Equal(t, "default_value", tag.DefaultValue)
	assert.Equal(t, "test comment", tag.Comment)
	assert.Equal(t, "value", tag.Extra["custom"])
}
