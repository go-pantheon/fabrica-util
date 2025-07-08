package migrate

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pantheon/fabrica-util/data/db/pg"
)

// Migration represents a database migration with up and down operations
type Migration struct {
	ID      string
	Up      func(ctx context.Context, db *pg.DB) error
	Down    func(ctx context.Context, db *pg.DB) error
	Comment string
}

// MigrationRecord represents a migration record in the database
type MigrationRecord struct {
	ID        string    `pgname:"id" pgtype:"VARCHAR(255) PRIMARY KEY"`
	AppliedAt time.Time `pgname:"applied_at" pgtype:"TIMESTAMPTZ DEFAULT NOW()"`
	Comment   string    `pgname:"comment" pgtype:"TEXT"`
}

// Migrator handles database migrations
type Migrator struct {
	db         *pg.DB
	tableName  string
	migrations map[string]*Migration
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *pg.DB, tableName string) *Migrator {
	if tableName == "" {
		tableName = "schema_migrations"
	}

	return &Migrator{
		db:         db,
		tableName:  tableName,
		migrations: make(map[string]*Migration),
	}
}

// Add adds a migration to the migrator
func (m *Migrator) Add(migration *Migration) {
	m.migrations[migration.ID] = migration
}

// AddFunc creates and adds a migration with functions
func (m *Migrator) AddFunc(id, comment string, up, down func(ctx context.Context, db *pg.DB) error) {
	m.Add(&Migration{
		ID:      id,
		Up:      up,
		Down:    down,
		Comment: comment,
	})
}

// ensureMigrationTable creates the migration tracking table if it doesn't exist
func (m *Migrator) ensureMigrationTable(ctx context.Context) error {
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS "%s" (
			"id" VARCHAR(255) PRIMARY KEY,
			"applied_at" TIMESTAMPTZ DEFAULT NOW(),
			"comment" TEXT
		);
	`, m.tableName)

	_, err := m.db.ExecContext(ctx, createSQL)

	return err
}

// getAppliedMigrations returns a set of applied migration IDs
func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	query := fmt.Sprintf(`SELECT "id" FROM "%s"`, m.tableName)

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	applied := make(map[string]bool)

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		applied[id] = true
	}

	return applied, nil
}

// recordMigration records a migration as applied
func (m *Migrator) recordMigration(ctx context.Context, migration *Migration) error {
	insertSQL := fmt.Sprintf(`
		INSERT INTO "%s" ("id", "comment") 
		VALUES ($1, $2)
	`, m.tableName)

	_, err := m.db.ExecContext(ctx, insertSQL, migration.ID, migration.Comment)

	return err
}

// removeMigrationRecord removes a migration record
func (m *Migrator) removeMigrationRecord(ctx context.Context, migrationID string) error {
	deleteSQL := fmt.Sprintf(`DELETE FROM "%s" WHERE "id" = $1`, m.tableName)
	_, err := m.db.ExecContext(ctx, deleteSQL, migrationID)

	return err
}

// Up runs all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration table: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	var executed int

	for id, migration := range m.migrations {
		if !applied[id] {
			if err := migration.Up(ctx, m.db); err != nil {
				return fmt.Errorf("failed to run migration %s: %w", id, err)
			}

			if err := m.recordMigration(ctx, migration); err != nil {
				return fmt.Errorf("failed to record migration %s: %w", id, err)
			}

			executed++
		}
	}

	return nil
}

// Down rolls back the last applied migration
func (m *Migrator) Down(ctx context.Context) error {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration table: %w", err)
	}

	// Get the last applied migration
	query := fmt.Sprintf(`
		SELECT "id" FROM "%s" 
		ORDER BY "applied_at" DESC 
		LIMIT 1
	`, m.tableName)

	row := m.db.QueryRowContext(ctx, query)

	var lastMigrationID string

	if err := row.Scan(&lastMigrationID); err != nil {
		return fmt.Errorf("no migrations to rollback")
	}

	migration, exists := m.migrations[lastMigrationID]
	if !exists {
		return fmt.Errorf("migration %s not found in registered migrations", lastMigrationID)
	}

	if migration.Down == nil {
		return fmt.Errorf("migration %s has no down function", lastMigrationID)
	}

	if err := migration.Down(ctx, m.db); err != nil {
		return fmt.Errorf("failed to rollback migration %s: %w", lastMigrationID, err)
	}

	if err := m.removeMigrationRecord(ctx, lastMigrationID); err != nil {
		return fmt.Errorf("failed to remove migration record %s: %w", lastMigrationID, err)
	}

	return nil
}

// DownTo rolls back migrations to a specific migration ID
func (m *Migrator) DownTo(ctx context.Context, targetID string) error {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration table: %w", err)
	}

	// Get all applied migrations ordered by applied_at DESC
	query := fmt.Sprintf(`
		SELECT "id" FROM "%s" 
		ORDER BY "applied_at" DESC
	`, m.tableName)

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var toRollback []string

	var foundTarget bool

	for rows.Next() {
		var migrationID string
		if err := rows.Scan(&migrationID); err != nil {
			return fmt.Errorf("failed to scan migration ID: %w", err)
		}

		if migrationID == targetID {
			foundTarget = true
			break
		}

		toRollback = append(toRollback, migrationID)
	}

	if !foundTarget {
		return fmt.Errorf("target migration %s not found in applied migrations", targetID)
	}

	// Rollback migrations in reverse order
	for _, migrationID := range toRollback {
		migration, exists := m.migrations[migrationID]
		if !exists {
			return fmt.Errorf("migration %s not found in registered migrations", migrationID)
		}

		if migration.Down == nil {
			return fmt.Errorf("migration %s has no down function", migrationID)
		}

		if err := migration.Down(ctx, m.db); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migrationID, err)
		}

		if err := m.removeMigrationRecord(ctx, migrationID); err != nil {
			return fmt.Errorf("failed to remove migration record %s: %w", migrationID, err)
		}
	}

	return nil
}

// Reset rolls back all migrations
func (m *Migrator) Reset(ctx context.Context) error {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration table: %w", err)
	}

	// Get all applied migrations ordered by applied_at DESC
	query := fmt.Sprintf(`
		SELECT "id" FROM "%s" 
		ORDER BY "applied_at" DESC
	`, m.tableName)

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var toRollback []string

	for rows.Next() {
		var migrationID string
		if err := rows.Scan(&migrationID); err != nil {
			return fmt.Errorf("failed to scan migration ID: %w", err)
		}

		toRollback = append(toRollback, migrationID)
	}

	// Rollback all migrations in reverse order
	for _, migrationID := range toRollback {
		migration, exists := m.migrations[migrationID]
		if !exists {
			return fmt.Errorf("migration %s not found in registered migrations", migrationID)
		}

		if migration.Down == nil {
			return fmt.Errorf("migration %s has no down function", migrationID)
		}

		if err := migration.Down(ctx, m.db); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migrationID, err)
		}

		if err := m.removeMigrationRecord(ctx, migrationID); err != nil {
			return fmt.Errorf("failed to remove migration record %s: %w", migrationID, err)
		}
	}

	return nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	ID        string
	Applied   bool
	AppliedAt *time.Time
	Comment   string
}

// Status returns the status of all migrations
func (m *Migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure migration table: %w", err)
	}

	// Get applied migrations with timestamps
	query := fmt.Sprintf(`
		SELECT "id", "applied_at", "comment" FROM "%s"
	`, m.tableName)

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	appliedMap := make(map[string]MigrationStatus)

	for rows.Next() {
		var id, comment string

		var appliedAt time.Time
		if err := rows.Scan(&id, &appliedAt, &comment); err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}

		appliedMap[id] = MigrationStatus{
			ID:        id,
			Applied:   true,
			AppliedAt: &appliedAt,
			Comment:   comment,
		}
	}

	// Build complete status list
	var statuses []MigrationStatus

	for id, migration := range m.migrations {
		if status, exists := appliedMap[id]; exists {
			statuses = append(statuses, status)
		} else {
			statuses = append(statuses, MigrationStatus{
				ID:        id,
				Applied:   false,
				AppliedAt: nil,
				Comment:   migration.Comment,
			})
		}
	}

	return statuses, nil
}
