package migrate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/go-pantheon/fabrica-util/camelcase"
	"github.com/go-pantheon/fabrica-util/data/db/pg"
	"github.com/go-pantheon/fabrica-util/errors"
)

// Migrate migrates the database schema based on the model struct.
// This function maintains backward compatibility with the original API.
func Migrate(ctx context.Context, db *pg.DB, tableName string, model any, extracols map[string]string) error {
	return MigrateWithVersionControl(ctx, db, tableName, model, extracols, "auto_migrations")
}

// MigrateWithVersionControl migrates the database schema with version control.
// It uses the new migration system to track changes and provide rollback capabilities.
func MigrateWithVersionControl(ctx context.Context, db *pg.DB, tableName string, model any, extracols map[string]string, migrationTable string) error {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	// Create migrator instance
	migrator := NewMigrator(db, migrationTable)

	// Generate migration ID based on table name and model structure
	migrationID, err := generateMigrationID(tableName, modelType, extracols)
	if err != nil {
		return errors.Wrapf(err, "failed to generate migration ID for table %s", tableName)
	}

	// Check if this migration already exists and is applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check migration status")
	}

	// Check if migration is already applied
	for _, status := range statuses {
		if status.ID == migrationID && status.Applied {
			// Migration already applied, check if we need to add new columns
			return addMissingColumnsWithVersionControl(ctx, db, migrator, tableName, modelType, extracols, migrationID)
		}
	}

	// Create migration for table creation and initial columns
	migrator.AddFunc(migrationID, fmt.Sprintf("Auto-migrate table %s", tableName),
		func(ctx context.Context, db *pg.DB) error {
			return createTableAndColumns(ctx, db, tableName, modelType, extracols)
		},
		func(ctx context.Context, db *pg.DB) error {
			return dropTable(ctx, db, tableName)
		},
	)

	// Execute the migration
	return migrator.Up(ctx)
}

// AutoMigrator provides a convenient way to manage auto-migrations with version control
type AutoMigrator struct {
	*Migrator
	registeredModels map[string]modelInfo
}

type modelInfo struct {
	tableName string
	modelType reflect.Type
	extracols map[string]string
}

// NewAutoMigrator creates a new auto-migrator instance
func NewAutoMigrator(db *pg.DB, migrationTable string) *AutoMigrator {
	if migrationTable == "" {
		migrationTable = "auto_migrations"
	}

	return &AutoMigrator{
		Migrator:         NewMigrator(db, migrationTable),
		registeredModels: make(map[string]modelInfo),
	}
}

// RegisterModel registers a model for auto-migration
func (am *AutoMigrator) RegisterModel(tableName string, model any, extracols map[string]string) {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	am.registeredModels[tableName] = modelInfo{
		tableName: tableName,
		modelType: modelType,
		extracols: extracols,
	}
}

// MigrateAll migrates all registered models
func (am *AutoMigrator) MigrateAll(ctx context.Context) error {
	for tableName, info := range am.registeredModels {
		if err := am.migrateModel(ctx, tableName, info); err != nil {
			return errors.Wrapf(err, "failed to migrate model %s", tableName)
		}
	}

	return am.Up(ctx)
}

// migrateModel creates migrations for a specific model
func (am *AutoMigrator) migrateModel(_ context.Context, tableName string, info modelInfo) error {
	migrationID, err := generateMigrationID(tableName, info.modelType, info.extracols)
	if err != nil {
		return errors.Wrapf(err, "failed to generate migration ID for table %s", tableName)
	}

	// Check if migration already exists
	if _, exists := am.migrations[migrationID]; exists {
		return nil
	}

	// Add migration
	am.AddFunc(migrationID, fmt.Sprintf("Auto-migrate table %s", tableName),
		func(ctx context.Context, db *pg.DB) error {
			return createTableAndColumns(ctx, db, tableName, info.modelType, info.extracols)
		},
		func(ctx context.Context, db *pg.DB) error {
			return dropTable(ctx, db, tableName)
		},
	)

	return nil
}

// generateMigrationID generates a unique migration ID based on table name and model structure
func generateMigrationID(tableName string, modelType reflect.Type, extracols map[string]string) (string, error) {
	// Create a hash of the table structure to detect changes
	hash := sha256.New()

	// Include table name
	if _, err := fmt.Fprintf(hash, "%s", tableName); err != nil {
		return "", errors.Wrapf(err, "failed to write to hash")
	}

	// Include model fields
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		columnName, columnType := getColumnInfo(field)
		if columnName != "" {
			if _, err := fmt.Fprintf(hash, "%s:%s", columnName, columnType); err != nil {
				return "", errors.Wrapf(err, "failed to write to hash")
			}
		}
	}

	// Include extra columns
	for name, colType := range extracols {
		if _, err := fmt.Fprintf(hash, "%s:%s", name, colType); err != nil {
			return "", errors.Wrapf(err, "failed to write to hash")
		}
	}

	return fmt.Sprintf("auto_%s_%x", tableName, hash.Sum(nil)[:8]), nil
}

// createTableAndColumns creates table and adds all columns
func createTableAndColumns(ctx context.Context, db *pg.DB, tableName string, modelType reflect.Type, extracols map[string]string) error {
	// Create table if not exists
	if err := createTableIfNotExists(ctx, db, tableName, modelType); err != nil {
		return err
	}

	// Add missing columns
	return addMissingColumns(ctx, db, tableName, modelType, extracols)
}

// addMissingColumnsWithVersionControl adds missing columns and creates new migrations for them
//
//nolint:gocognit
func addMissingColumnsWithVersionControl(ctx context.Context, db *pg.DB, migrator *Migrator, tableName string, modelType reflect.Type, extracols map[string]string, baseMigrationID string) error {
	existingColumns, err := getExistingColumns(ctx, db, tableName)
	if err != nil {
		return err
	}

	expectedColumns := make(map[string]string)
	if extracols != nil {
		expectedColumns = maps.Clone(extracols)
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		columnName, columnType := getColumnInfo(field)

		if columnName == "" {
			continue
		}

		expectedColumns[columnName] = columnType
	}

	var newColumns []string

	for columnName, columnType := range expectedColumns {
		if _, exists := existingColumns[columnName]; !exists {
			newColumns = append(newColumns, fmt.Sprintf(`"%s" %s`, columnName, columnType))
		}
	}

	// If there are new columns, create a new migration
	if len(newColumns) > 0 {
		hashSum := sha256.Sum256([]byte(strings.Join(newColumns, ",")))
		newMigrationID := fmt.Sprintf("%s_add_columns_%x", baseMigrationID, hashSum[:4])

		migrator.AddFunc(newMigrationID, fmt.Sprintf("Add columns to table %s", tableName),
			func(ctx context.Context, db *pg.DB) error {
				for columnName, columnType := range expectedColumns {
					if _, exists := existingColumns[columnName]; !exists {
						alterSQL := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s;`, tableName, columnName, columnType)
						if _, err := db.ExecContext(ctx, alterSQL); err != nil {
							return errors.Wrapf(err, "failed to add column %s to table %s", columnName, tableName)
						}
					}
				}

				return nil
			},
			func(ctx context.Context, db *pg.DB) error {
				// Drop the newly added columns
				for columnName := range expectedColumns {
					if _, exists := existingColumns[columnName]; !exists {
						alterSQL := fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN IF EXISTS "%s";`, tableName, columnName)
						if _, err := db.ExecContext(ctx, alterSQL); err != nil {
							return errors.Wrapf(err, "failed to drop column %s from table %s", columnName, tableName)
						}
					}
				}

				return nil
			},
		)

		return migrator.Up(ctx)
	}

	return nil
}

// dropTable drops a table (used for rollback)
func dropTable(ctx context.Context, db *pg.DB, tableName string) error {
	dropSQL := fmt.Sprintf(`DROP TABLE IF EXISTS "%s";`, tableName)
	_, err := db.ExecContext(ctx, dropSQL)

	return errors.Wrapf(err, "failed to drop table %s", tableName)
}

// Below are the original functions, kept for internal use and backward compatibility

func createTableIfNotExists(ctx context.Context, db *pg.DB, tableName string, modelType reflect.Type) error {
	var columns []string

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		// Try new orm tag first
		if columnTag, err := defaultTagParser.ParseTag(field); err == nil && columnTag != nil {
			columns = append(columns, defaultTagParser.BuildColumnDefinition(columnTag))
		} else {
			// Fallback to legacy method
			columnName, columnType := getLegacyColumnInfo(field)
			if columnName != "" {
				columns = append(columns, fmt.Sprintf(`"%s" %s`, columnName, columnType))
			}
		}
	}

	if len(columns) == 0 {
		return nil
	}

	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (%s);`, tableName, strings.Join(columns, ", "))
	_, err := db.ExecContext(ctx, createSQL)

	return errors.Wrapf(err, "failed to create table %s", tableName)
}

func addMissingColumns(ctx context.Context, db *pg.DB, table string, t reflect.Type, extracols map[string]string) error {
	existingColumns, err := getExistingColumns(ctx, db, table)
	if err != nil {
		return err
	}

	expectedColumns := make(map[string]string)
	if extracols != nil {
		expectedColumns = maps.Clone(extracols)
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		columnName, columnType := getColumnInfo(field)

		if columnName == "" {
			continue
		}

		expectedColumns[columnName] = columnType
	}

	for columnName, columnType := range expectedColumns {
		if _, exists := existingColumns[columnName]; !exists {
			alterSQL := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s;`, table, columnName, columnType)
			if _, err := db.ExecContext(ctx, alterSQL); err != nil {
				return errors.Wrapf(err, "failed to add column %s to table %s", columnName, table)
			}
		}
	}

	return nil
}

func getExistingColumns(ctx context.Context, db *pg.DB, tableName string) (map[string]bool, error) {
	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1;
	`
	rows, err := db.QueryContext(ctx, query, tableName)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to query existing columns for table %s", tableName)
	}

	defer rows.Close()

	columns := make(map[string]bool)

	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, errors.Wrap(err, "failed to scan column name")
		}

		columns[columnName] = true
	}

	return columns, nil
}

// Global tag parser instance
var defaultTagParser = NewTagParser()

func getColumnInfo(field reflect.StructField) (string, string) {
	if !field.IsExported() {
		return "", ""
	}

	// Try new orm tag first
	if columnTag, err := defaultTagParser.ParseTag(field); err == nil && columnTag != nil {
		return columnTag.Column, columnTag.Type
	}

	// Fallback to legacy tags for backward compatibility
	return getLegacyColumnInfo(field)
}

// getLegacyColumnInfo handles the old pgname/pgtype/primarykey tags for backward compatibility
func getLegacyColumnInfo(field reflect.StructField) (string, string) {
	tag := field.Tag
	colName := tag.Get("pgname")
	colType := tag.Get("pgtype")
	primaryKey := tag.Get("primarykey")

	// If neither colname nor coltype is specified, it's not a column, unless it's a time.Time struct.
	if colName == "" && colType == "" && primaryKey == "" && field.Type.Kind() != reflect.Struct {
		return "", ""
	}

	// If colname is not specified, use the snake_case of the field name.
	if colName == "" {
		colName = camelcase.ToUnderScore(field.Name)
	}

	switch primaryKey {
	case "auto":
		colType = "SERIAL PRIMARY KEY"
	case "":
	default:
		colType = fmt.Sprintf("%s PRIMARY KEY", primaryKey)
	}

	// If coltype is not specified, infer it from the field type.
	if colType == "" {
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			colType = "INTEGER"
		case reflect.Int64, reflect.Uint64:
			colType = "BIGINT"
		case reflect.Float32:
			colType = "REAL"
		case reflect.Float64:
			colType = "DOUBLE PRECISION"
		case reflect.Bool:
			colType = "BOOLEAN"
		case reflect.String:
			colType = "TEXT"
		case reflect.Struct:
			if fieldType == reflect.TypeOf(time.Time{}) {
				colType = "TIMESTAMPTZ"
			} else {
				colType = "JSONB"
			}
		}
	}

	return colName, colType
}
