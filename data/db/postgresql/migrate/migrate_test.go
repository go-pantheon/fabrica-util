//go:build integration

// DOCKER_HOST=unix://{{dockerpath}}/docker.sock TESTCONTAINERS_RYUK_DISABLED=true go test ./app/player/internal/data/migrate/...
package migrate_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-pantheon/fabrica-util/data/db/postgresql/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Test Models
type TestModelComplete struct {
	ID          int       `pgname:"id" pgtype:"integer"`
	Name        string    `pgname:"full_name" pgtype:"text"`
	Age         int       `pgname:"age" pgtype:"integer"`
	CreatedAt   time.Time `pgname:"created_at"`
	NoDB        string
	Details     *Nested `pgtype:"jsonb"`
	Description string  `pgname:"description" pgtype:"varchar(255)"`
}

type Nested struct {
	Value string `json:"value"`
}

type TestModelOnlyColname struct {
	ID   int    `pgname:"id"`
	Name string `pgname:"name"`
}

type TestModelOnlyColtype struct {
	ID   int    `pgtype:"serial primary key"`
	Name string `pgtype:"text"`
}

type TestModelDefaults struct {
	ID   int `pgname:"id"`
	Data Nested
}

type TestModelWithCustomTableName struct {
	ID int `pgname:"id"`
}

func (TestModelWithCustomTableName) TableName() string {
	return "my_custom_table_name"
}

// Test Suite
type MigratorSuite struct {
	suite.Suite
	pgContainer *postgres.PostgresContainer
	db          *pgxpool.Pool
}

func (s *MigratorSuite) SetupSuite() {
	ctx := context.Background()

	var err error
	s.pgContainer, err = postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").
				WithStartupTimeout(5*time.Minute),
		),
	)
	require.NoError(s.T(), err)

	connStr, err := s.pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(s.T(), err)

	s.db, err = pgxpool.New(ctx, connStr)
	require.NoError(s.T(), err)
}

func (s *MigratorSuite) TearDownSuite() {
	ctx := context.Background()

	s.db.Close()
	err := s.pgContainer.Terminate(ctx)
	require.NoError(s.T(), err)
}

func (s *MigratorSuite) BeforeTest(suiteName, testName string) {
	// Clean up tables before each test
	rows, err := s.db.Query(context.Background(), `
        SELECT tablename FROM pg_tables WHERE schemaname = 'public'
    `)
	require.NoError(s.T(), err)
	defer rows.Close()

	for rows.Next() {
		var tableName string

		require.NoError(s.T(), rows.Scan(&tableName))
		_, err := s.db.Exec(context.Background(), fmt.Sprintf(`DROP TABLE IF EXISTS "%s" CASCADE`, tableName))
		require.NoError(s.T(), err)
	}
}

func TestMigratorSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(MigratorSuite))
}

func (s *MigratorSuite) Test01_Migrate_CompleteModel() {
	t := s.T()
	ctx := context.Background()

	err := migrate.Migrate(ctx, s.db, &TestModelComplete{}, nil)
	require.NoError(t, err)

	s.assertColumnExists("test_model_complete", "id", "integer")
	s.assertColumnExists("test_model_complete", "full_name", "text")
	s.assertColumnExists("test_model_complete", "age", "integer")
	s.assertColumnExists("test_model_complete", "created_at", "timestamp with time zone")
	s.assertColumnExists("test_model_complete", "details", "jsonb")
	s.assertColumnExists("test_model_complete", "description", "character varying")
	s.assertColumnDoesNotExist("test_model_complete", "no_db")
	s.assertColumnDoesNotExist("test_model_complete", "unexported")
}

func (s *MigratorSuite) Test02_Migrate_Idempotency() {
	t := s.T()
	ctx := context.Background()

	// First migration
	err := migrate.Migrate(ctx, s.db, &TestModelComplete{}, nil)
	require.NoError(t, err)

	// Second migration should not fail
	err = migrate.Migrate(ctx, s.db, &TestModelComplete{}, nil)
	require.NoError(t, err)

	s.assertColumnExists("test_model_complete", "id", "integer")
}

func (s *MigratorSuite) Test03_Migrate_AddColumn() {
	t := s.T()
	ctx := context.Background()

	// Initial model
	type InitialModel struct {
		ID int `colname:"id"`
	}

	err := migrate.Migrate(ctx, s.db, &InitialModel{}, nil)
	require.NoError(t, err)
	s.assertColumnExists("initial_model", "id", "integer")
	s.assertColumnDoesNotExist("initial_model", "new_field")

	// Model with added column
	type UpdatedModel struct {
		ID       int    `colname:"id"`
		NewField string `colname:"new_field"`
	}

	err = migrate.Migrate(ctx, s.db, &UpdatedModel{}, nil)
	require.NoError(t, err)
	s.assertColumnExists("updated_model", "new_field", "text")
}

func (s *MigratorSuite) Test04_Migrate_OnlyColname() {
	t := s.T()
	ctx := context.Background()
	err := migrate.Migrate(ctx, s.db, &TestModelOnlyColname{}, nil)
	require.NoError(t, err)
	s.assertColumnExists("test_model_only_colname", "id", "integer")
	s.assertColumnExists("test_model_only_colname", "name", "text")
}

func (s *MigratorSuite) Test05_Migrate_OnlyColtype() {
	t := s.T()
	ctx := context.Background()
	err := migrate.Migrate(ctx, s.db, &TestModelOnlyColtype{}, nil)
	require.NoError(t, err)
	s.assertColumnExists("test_model_only_coltype", "id", "integer") // serial becomes integer
	s.assertColumnExists("test_model_only_coltype", "name", "text")
}

func (s *MigratorSuite) Test06_Migrate_Defaults() {
	t := s.T()
	ctx := context.Background()
	err := migrate.Migrate(ctx, s.db, &TestModelDefaults{}, nil)
	require.NoError(t, err)
	s.assertColumnExists("test_model_defaults", "id", "integer")
	s.assertColumnExists("test_model_defaults", "data", "jsonb")
}

func (s *MigratorSuite) Test07_Migrate_CustomTableName() {
	t := s.T()
	ctx := context.Background()
	err := migrate.Migrate(ctx, s.db, &TestModelWithCustomTableName{}, nil)
	require.NoError(t, err)
	s.assertColumnExists("my_custom_table_name", "id", "integer")
	s.assertTableDoesNotExist("test_model_with_custom_table_name")
}

func (s *MigratorSuite) Test08_Migrate_WithExtraColumns() {
	t := s.T()
	ctx := context.Background()

	extraCols := map[string]string{
		"extra_col_1": "TEXT",
		"extra_col_2": "BOOLEAN NOT NULL",
	}

	err := migrate.Migrate(ctx, s.db, &TestModelComplete{}, extraCols)
	require.NoError(t, err)

	// Check model columns
	s.assertColumnExists("test_model_complete", "id", "integer")
	// Check extra columns
	s.assertColumnExists("test_model_complete", "extra_col_1", "text")
	s.assertColumnExists("test_model_complete", "extra_col_2", "boolean")

	// Run again to ensure idempotency
	err = migrate.Migrate(ctx, s.db, &TestModelComplete{}, extraCols)
	require.NoError(t, err)
	s.assertColumnExists("test_model_complete", "extra_col_1", "text")
}

// Helper Assertions
func (s *MigratorSuite) assertColumnExists(tableName, columnName, expectedType string) {
	t := s.T()

	var dataType string

	query := `
        SELECT data_type FROM information_schema.columns
        WHERE table_name = $1 AND column_name = $2
    `
	err := s.db.QueryRow(context.Background(), query, tableName, columnName).Scan(&dataType)
	require.NoError(t, err, "Column '%s' on table '%s' should exist", columnName, tableName)

	// Normalize types for comparison
	if expectedType == "integer" && dataType == "serial" {
		dataType = "integer"
	}

	if strings.HasPrefix(expectedType, "varchar") && dataType == "character varying" {
		return
	}

	if expectedType == "boolean" && dataType == "boolean" {
		return
	}

	require.Equal(t, expectedType, dataType, "Column '%s' on table '%s' has wrong type", columnName, tableName)
}

func (s *MigratorSuite) assertColumnDoesNotExist(tableName, columnName string) {
	t := s.T()

	var exists bool

	query := `
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = $1 AND column_name = $2
        )
    `
	err := s.db.QueryRow(context.Background(), query, tableName, columnName).Scan(&exists)
	require.NoError(t, err)
	require.False(t, exists, "Column '%s' on table '%s' should not exist", columnName, tableName)
}

func (s *MigratorSuite) assertTableDoesNotExist(tableName string) {
	t := s.T()

	var exists bool

	query := `
        SELECT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_name = $1
        )
    `
	err := s.db.QueryRow(context.Background(), query, tableName).Scan(&exists)
	require.NoError(t, err)
	require.False(t, exists, "Table '%s' should not exist", tableName)
}
