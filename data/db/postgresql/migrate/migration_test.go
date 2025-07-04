package migrate

import (
	"context"
	"testing"
	"time"

	xpg "github.com/go-pantheon/fabrica-util/data/db/postgresql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration_Struct(t *testing.T) {
	t.Parallel()

	// Test creating a migration
	migration := &Migration{
		ID:      "20240101120000_create_users_table",
		Comment: "Create users table",
		Up: func(ctx context.Context, db xpg.DBPool) error {
			return nil
		},
		Down: func(ctx context.Context, db xpg.DBPool) error {
			return nil
		},
	}

	assert.Equal(t, "20240101120000_create_users_table", migration.ID)
	assert.Equal(t, "Create users table", migration.Comment)
	assert.NotNil(t, migration.Up)
	assert.NotNil(t, migration.Down)
}

func TestMigrationRecord_Struct(t *testing.T) {
	t.Parallel()

	// Test MigrationRecord creation
	record := MigrationRecord{
		ID:        "20240101120000_test",
		Comment:   "Test migration",
		AppliedAt: time.Now(),
	}

	assert.Equal(t, "20240101120000_test", record.ID)
	assert.Equal(t, "Test migration", record.Comment)
	assert.False(t, record.AppliedAt.IsZero())
}

func TestMigrator_CreateWithCustomTableName(t *testing.T) {
	t.Parallel()

	// Test creating migrator with custom table name
	migrator := NewMigrator(nil, "custom_migrations")

	assert.Equal(t, "custom_migrations", migrator.tableName)
}

func TestMigrator_CreateWithDefaultTableName(t *testing.T) {
	t.Parallel()

	// Test creating migrator with default table name
	migrator := NewMigrator(nil, "")

	assert.Equal(t, "schema_migrations", migrator.tableName)
}

func TestMigrator_AddMigration(t *testing.T) {
	t.Parallel()

	migrator := NewMigrator(nil, "test_migrations")

	migration := &Migration{
		ID:      "20240101120000_test",
		Comment: "Test migration",
		Up:      func(ctx context.Context, db xpg.DBPool) error { return nil },
		Down:    func(ctx context.Context, db xpg.DBPool) error { return nil },
	}

	migrator.Add(migration)

	// Check that migration was added
	assert.Len(t, migrator.migrations, 1)

	// Check that migration can be retrieved
	retrievedMigration, exists := migrator.migrations["20240101120000_test"]
	require.True(t, exists, "Expected migration to be added to migrations map")

	assert.Equal(t, migration.ID, retrievedMigration.ID)
	assert.Equal(t, migration.Comment, retrievedMigration.Comment)
}

func TestMigrator_AddFunc(t *testing.T) {
	t.Parallel()

	migrator := NewMigrator(nil, "test_migrations")

	upCalled := false
	downCalled := false

	upFunc := func(ctx context.Context, db xpg.DBPool) error {
		upCalled = true
		return nil
	}

	downFunc := func(ctx context.Context, db xpg.DBPool) error {
		downCalled = true
		return nil
	}

	migrator.AddFunc("20240101120000_test", "Test migration", upFunc, downFunc)

	// Check that migration was added
	assert.Len(t, migrator.migrations, 1)

	// Check that migration can be retrieved
	retrievedMigration, exists := migrator.migrations["20240101120000_test"]
	require.True(t, exists, "Expected migration to be added to migrations map")

	assert.Equal(t, "20240101120000_test", retrievedMigration.ID)
	assert.Equal(t, "Test migration", retrievedMigration.Comment)

	// Test that functions work
	err := retrievedMigration.Up(context.Background(), nil)
	assert.NoError(t, err)
	assert.True(t, upCalled, "Expected Up function to be called")

	err = retrievedMigration.Down(context.Background(), nil)
	assert.NoError(t, err)
	assert.True(t, downCalled, "Expected Down function to be called")
}

func TestMigrationStatus_Struct(t *testing.T) {
	t.Parallel()

	// Test MigrationStatus creation
	now := time.Now()
	status := MigrationStatus{
		ID:        "20240101120000_test",
		Applied:   true,
		AppliedAt: &now,
		Comment:   "Test migration",
	}

	assert.Equal(t, "20240101120000_test", status.ID)
	assert.True(t, status.Applied)
	assert.NotNil(t, status.AppliedAt)
	assert.Equal(t, "Test migration", status.Comment)
}
