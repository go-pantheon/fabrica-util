package migrate

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/go-pantheon/fabrica-util/camelcase"
	xpg "github.com/go-pantheon/fabrica-util/data/db/postgresql"
	"github.com/go-pantheon/fabrica-util/errors"
)

func Migrate(ctx context.Context, db xpg.DBPool, tableName string, model any, extracols map[string]string) error {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	// Create table if not exists
	if err := createTableIfNotExists(ctx, db, tableName, modelType); err != nil {
		return err
	}

	// Add missing columns
	return addMissingColumns(ctx, db, tableName, modelType, extracols)
}

func createTableIfNotExists(ctx context.Context, db xpg.DBPool, tableName string, modelType reflect.Type) error {
	var columns []string

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		columnName, columnType := getColumnInfo(field)

		if columnName != "" {
			columns = append(columns, fmt.Sprintf(`"%s" %s`, columnName, columnType))
		}
	}

	if len(columns) == 0 {
		return nil
	}

	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (%s);`, tableName, strings.Join(columns, ", "))
	_, err := db.Exec(ctx, createSQL)

	return errors.Wrapf(err, "failed to create table %s", tableName)
}

func addMissingColumns(ctx context.Context, db xpg.DBPool, table string, t reflect.Type, extracols map[string]string) error {
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
			if _, err := db.Exec(ctx, alterSQL); err != nil {
				return errors.Wrapf(err, "failed to add column %s to table %s", columnName, table)
			}
		}
	}

	return nil
}

func getExistingColumns(ctx context.Context, db xpg.DBPool, tableName string) (map[string]bool, error) {
	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1;
	`
	rows, err := db.Query(ctx, query, tableName)

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

func getColumnInfo(field reflect.StructField) (string, string) {
	if !field.IsExported() {
		return "", ""
	}

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
