package migrate

import (
	"crypto/md5"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test models for auto migration
type TestUser struct {
	ID       int       `orm:"primaryKey;autoIncrement" pgname:"id" pgtype:"SERIAL PRIMARY KEY"`
	Email    string    `orm:"size:255;unique;notNull" pgname:"email" pgtype:"VARCHAR(255) UNIQUE NOT NULL"`
	Name     string    `orm:"size:100;notNull" pgname:"name" pgtype:"VARCHAR(100) NOT NULL"`
	Age      int       `orm:"default:0" pgname:"age" pgtype:"INTEGER DEFAULT 0"`
	CreateAt time.Time `orm:"default:NOW()" pgname:"create_at" pgtype:"TIMESTAMPTZ DEFAULT NOW()"`
}

type TestPost struct {
	ID       int       `orm:"primaryKey;autoIncrement" pgname:"id" pgtype:"SERIAL PRIMARY KEY"`
	Title    string    `orm:"size:255;notNull" pgname:"title" pgtype:"VARCHAR(255) NOT NULL"`
	Content  string    `orm:"type:TEXT" pgname:"content" pgtype:"TEXT"`
	UserID   int       `orm:"notNull" pgname:"user_id" pgtype:"INTEGER NOT NULL"`
	CreateAt time.Time `orm:"default:NOW()" pgname:"create_at" pgtype:"TIMESTAMPTZ DEFAULT NOW()"`
}

func TestAutoMigrator_New(t *testing.T) {
	// Test creating AutoMigrator
	autoMigrator := NewAutoMigrator(nil, "auto_migrations")

	assert.Nil(t, autoMigrator.db, "Expected db to be nil in test")
	assert.Equal(t, "auto_migrations", autoMigrator.tableName)
	assert.Empty(t, autoMigrator.registeredModels, "Expected registeredModels to be empty")
}

func TestAutoMigrator_RegisterModel(t *testing.T) {
	autoMigrator := NewAutoMigrator(nil, "auto_migrations")

	// Test registering a model
	autoMigrator.RegisterModel("test_users", &TestUser{}, nil)

	assert.Len(t, autoMigrator.registeredModels, 1)

	// Check model name
	expectedName := "test_users"
	_, exists := autoMigrator.registeredModels[expectedName]
	assert.True(t, exists, "Expected model '%s' to be registered", expectedName)
}

func TestAutoMigrator_RegisterMultipleModels(t *testing.T) {
	autoMigrator := NewAutoMigrator(nil, "auto_migrations")

	// Test registering multiple models
	autoMigrator.RegisterModel("test_users", &TestUser{}, nil)
	autoMigrator.RegisterModel("test_posts", &TestPost{}, nil)

	assert.Len(t, autoMigrator.registeredModels, 2)

	// Check both models are registered
	assert.Contains(t, autoMigrator.registeredModels, "test_users")
	assert.Contains(t, autoMigrator.registeredModels, "test_posts")
}

func TestModelInfo_Struct(t *testing.T) {
	// Test modelInfo creation
	info := modelInfo{
		tableName: "test_users",
		modelType: reflect.TypeOf(TestUser{}),
		extracols: map[string]string{"extra_col": "VARCHAR(100)"},
	}

	assert.Equal(t, "test_users", info.tableName)
	assert.Equal(t, reflect.TypeOf(TestUser{}), info.modelType)
	require.NotNil(t, info.extracols)
	assert.Equal(t, "VARCHAR(100)", info.extracols["extra_col"])
}

func TestGenerateMigrationID(t *testing.T) {
	// Test generating migration ID
	id := generateMigrationID("test_users", reflect.TypeOf(TestUser{}), nil)

	assert.NotEmpty(t, id, "Expected non-empty migration ID")

	// Test consistency
	id2 := generateMigrationID("test_users", reflect.TypeOf(TestUser{}), nil)
	assert.Equal(t, id, id2, "Expected consistent migration ID for same inputs")

	// Test different inputs produce different IDs
	id3 := generateMigrationID("test_posts", reflect.TypeOf(TestPost{}), nil)
	assert.NotEqual(t, id, id3, "Expected different migration IDs for different inputs")
}

func TestGenerateMigrationID_WithExtracols(t *testing.T) {
	// Test generating migration ID with extra columns
	extracols := map[string]string{"extra_col": "VARCHAR(100)"}
	id := generateMigrationID("test_users", reflect.TypeOf(TestUser{}), extracols)

	assert.NotEmpty(t, id, "Expected non-empty migration ID")

	// Test that extracols affect the ID
	id2 := generateMigrationID("test_users", reflect.TypeOf(TestUser{}), nil)
	assert.NotEqual(t, id, id2, "Expected different migration IDs with and without extracols")
}

func TestGetLegacyColumnInfo(t *testing.T) {
	// Test getting legacy column info
	userType := reflect.TypeOf(TestUser{})

	// Test ID field
	idField, found := userType.FieldByName("ID")
	require.True(t, found)

	colName, colType := getLegacyColumnInfo(idField)

	assert.Equal(t, "id", colName)
	assert.Equal(t, "SERIAL PRIMARY KEY", colType)
}

func TestGetLegacyColumnInfo_String(t *testing.T) {
	// Test string field
	userType := reflect.TypeOf(TestUser{})
	nameField, found := userType.FieldByName("Name")
	require.True(t, found)

	colName, colType := getLegacyColumnInfo(nameField)

	assert.Equal(t, "name", colName)
	assert.Equal(t, "VARCHAR(100) NOT NULL", colType)
}

func TestGetLegacyColumnInfo_Time(t *testing.T) {
	// Test time field
	userType := reflect.TypeOf(TestUser{})
	timeField, found := userType.FieldByName("CreateAt")
	require.True(t, found)

	colName, colType := getLegacyColumnInfo(timeField)

	assert.Equal(t, "create_at", colName)
	assert.Equal(t, "TIMESTAMPTZ DEFAULT NOW()", colType)
}

func TestGetColumnInfo_NewORM(t *testing.T) {
	// Test new ORM tag parsing
	userType := reflect.TypeOf(TestUser{})

	// Test email field with new ORM tags
	emailField, found := userType.FieldByName("Email")
	require.True(t, found)

	colName, colType := getColumnInfo(emailField)

	assert.Equal(t, "email", colName)
	assert.NotEmpty(t, colType, "Expected non-empty column type from ORM parser")
}

func TestGetColumnInfo_Fallback(t *testing.T) {
	// Test fallback to legacy tags
	type LegacyModel struct {
		ID   int    `pgname:"id" pgtype:"SERIAL PRIMARY KEY"`
		Name string `pgname:"name" pgtype:"VARCHAR(100)"`
	}

	legacyType := reflect.TypeOf(LegacyModel{})

	// Test ID field
	idField, found := legacyType.FieldByName("ID")
	require.True(t, found)

	colName, colType := getColumnInfo(idField)

	assert.Equal(t, "id", colName)
	assert.Equal(t, "INTEGER", colType)
}

func TestMD5Hash(t *testing.T) {
	// Test MD5 hash generation
	data := "test data"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	assert.Len(t, hash, 32, "Expected MD5 hash length to be 32")

	// Test consistency
	hash2 := fmt.Sprintf("%x", md5.Sum([]byte(data)))
	assert.Equal(t, hash, hash2, "Expected consistent hash for same data")

	// Test different data produces different hash
	hash3 := fmt.Sprintf("%x", md5.Sum([]byte("different data")))
	assert.NotEqual(t, hash, hash3, "Expected different hash for different data")
}
