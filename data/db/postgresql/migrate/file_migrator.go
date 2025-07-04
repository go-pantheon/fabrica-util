package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	xpg "github.com/go-pantheon/fabrica-util/data/db/postgresql"
	"github.com/go-pantheon/fabrica-util/errors"
)

// FileMigrator handles file-based migrations
type FileMigrator struct {
	*Migrator
	migrationsDir string
}

// NewFileMigrator creates a new file-based migrator
func NewFileMigrator(db xpg.DBPool, migrationsDir string, tableName string) *FileMigrator {
	return &FileMigrator{
		Migrator:      NewMigrator(db, tableName),
		migrationsDir: migrationsDir,
	}
}

// MigrationFile represents a migration file
type MigrationFile struct {
	ID       string
	UpPath   string
	DownPath string
	Comment  string
}

// LoadMigrations loads migrations from the filesystem
func (fm *FileMigrator) LoadMigrations() error {
	if err := fm.ensureMigrationsDir(); err != nil {
		return err
	}

	files, err := fm.findMigrationFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		migration := &Migration{
			ID:      file.ID,
			Comment: file.Comment,
			Up: func(ctx context.Context, db xpg.DBPool) error {
				return fm.executeSQLFile(ctx, db, file.UpPath)
			},
		}

		if file.DownPath != "" {
			migration.Down = func(ctx context.Context, db xpg.DBPool) error {
				return fm.executeSQLFile(ctx, db, file.DownPath)
			}
		}

		fm.Add(migration)
	}

	return nil
}

// ensureMigrationsDir creates the migrations directory if it doesn't exist
func (fm *FileMigrator) ensureMigrationsDir() error {
	if _, err := os.Stat(fm.migrationsDir); os.IsNotExist(err) {
		return os.MkdirAll(fm.migrationsDir, 0750)
	}

	return nil
}

// findMigrationFiles scans the migration directory for migration files
func (fm *FileMigrator) findMigrationFiles() ([]MigrationFile, error) {
	var files []MigrationFile

	migrationMap := make(map[string]*MigrationFile)

	// Pattern: {timestamp}_{name}.up.sql or {timestamp}_{name}.down.sql
	pattern := regexp.MustCompile(`^(\d+)_([^.]+)\.(up|down)\.sql$`)

	err := filepath.WalkDir(fm.migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		matches := pattern.FindStringSubmatch(d.Name())
		if len(matches) != 4 {
			return nil // Skip non-migration files
		}

		timestamp := matches[1]
		name := matches[2]
		direction := matches[3]
		id := timestamp + "_" + name

		if migrationMap[id] == nil {
			migrationMap[id] = &MigrationFile{
				ID:      id,
				Comment: strings.ReplaceAll(name, "_", " "),
			}
		}

		if direction == "up" {
			migrationMap[id].UpPath = path
		} else {
			migrationMap[id].DownPath = path
		}

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to scan migration directory")
	}

	// Convert map to slice and sort by ID
	for _, file := range migrationMap {
		if file.UpPath != "" { // Only include migrations with up files
			files = append(files, *file)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ID < files[j].ID
	})

	return files, nil
}

// executeSQLFile reads and executes SQL from a file
func (fm *FileMigrator) executeSQLFile(ctx context.Context, db xpg.DBPool, filePath string) error {
	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return errors.Wrapf(err, "failed to read migration file %s", filePath)
	}

	sql := strings.TrimSpace(string(content))
	if sql == "" {
		return nil // Empty file
	}

	// Split SQL by semicolons and execute each statement
	statements := strings.Split(sql, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := db.Exec(ctx, stmt); err != nil {
			return errors.Wrapf(err, "failed to execute SQL statement in %s: %s", filePath, stmt)
		}
	}

	return nil
}

// CreateMigration creates a new migration file pair
func (fm *FileMigrator) CreateMigration(name string) error {
	if err := fm.ensureMigrationsDir(); err != nil {
		return err
	}

	timestamp := fmt.Sprintf("%d", getCurrentTimestamp())
	filename := fmt.Sprintf("%s_%s", timestamp, name)

	upFile := filepath.Join(fm.migrationsDir, filename+".up.sql")
	downFile := filepath.Join(fm.migrationsDir, filename+".down.sql")

	// Create up migration file
	upContent := fmt.Sprintf("-- Migration: %s\n-- Created at: %s\n\n-- Add your up migration here\n",
		strings.ReplaceAll(name, "_", " "), getCurrentTime())

	if err := os.WriteFile(upFile, []byte(upContent), 0600); err != nil {
		return errors.Wrapf(err, "failed to create up migration file %s", upFile)
	}

	// Create down migration file
	downContent := fmt.Sprintf("-- Rollback for: %s\n-- Created at: %s\n\n-- Add your down migration here\n",
		strings.ReplaceAll(name, "_", " "), getCurrentTime())

	if err := os.WriteFile(downFile, []byte(downContent), 0600); err != nil {
		return errors.Wrapf(err, "failed to create down migration file %s", downFile)
	}

	fmt.Printf("Created migration files:\n  %s\n  %s\n", upFile, downFile)

	return nil
}

// getCurrentTimestamp returns current unix timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// getCurrentTime returns current time as string
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
