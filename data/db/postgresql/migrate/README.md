# PostgreSQL Migration Library

A comprehensive PostgreSQL database migration library that supports version control, auto-migration, file-based migrations, and ORM tag parsing.

## Features

- ✅ Version-controlled database migrations
- ✅ Support for up/down migrations
- ✅ Migration state tracking and querying
- ✅ Programmatic migration definition
- ✅ File-based migration system
- ✅ Auto-migration with version control
- ✅ GORM-like ORM tag support
- ✅ Rollback to specific versions
- ✅ Reset all migrations
- ✅ Comprehensive test coverage

## Installation

```go
import "github.com/go-pantheon/fabrica-util/data/db/postgresql/migrate"
```

## Core Components

### 1. Core Migration System

The core migration system provides basic migration management functionality:

```go
package main

import (
    "context"
    "log"
    
    "github.com/go-pantheon/fabrica-util/data/db/postgresql"
    "github.com/go-pantheon/fabrica-util/data/db/postgresql/migrate"
)

func main() {
    // Create database connection
    config := postgresql.NewConfig("postgres://user:password@localhost/dbname", "mydb")
    db, cleanup, err := postgresql.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup()

    // Create migrator
    migrator := migrate.NewMigrator(db, "schema_migrations")

    // Add migration
    migrator.AddFunc("001_create_users_table", "Create users table",
        // Up migration
        func(ctx context.Context, db postgresql.DBPool) error {
            _, err := db.Exec(ctx, `
                CREATE TABLE users (
                    id SERIAL PRIMARY KEY,
                    email VARCHAR(255) UNIQUE NOT NULL,
                    name VARCHAR(255) NOT NULL,
                    created_at TIMESTAMPTZ DEFAULT NOW()
                );
            `)
            return err
        },
        // Down migration
        func(ctx context.Context, db postgresql.DBPool) error {
            _, err := db.Exec(ctx, "DROP TABLE IF EXISTS users;")
            return err
        },
    )

    ctx := context.Background()

    // Execute all pending migrations
    if err := migrator.Up(ctx); err != nil {
        log.Fatal("Migration failed:", err)
    }

    // View migration status
    statuses, _ := migrator.Status(ctx)
    for _, status := range statuses {
        if status.Applied {
            log.Printf("✓ %s: %s", status.ID, status.Comment)
        } else {
            log.Printf("✗ %s: %s", status.ID, status.Comment)
        }
    }

    // Rollback the last migration
    if err := migrator.Down(ctx); err != nil {
        log.Printf("Rollback failed: %v", err)
    }

    // Rollback to specific migration
    if err := migrator.DownTo(ctx, "001_create_users_table"); err != nil {
        log.Printf("Rollback to specific migration failed: %v", err)
    }

    // Reset all migrations
    if err := migrator.Reset(ctx); err != nil {
        log.Printf("Reset failed: %v", err)
    }
}
```

### 2. Auto Migration System

The auto migration system provides struct-based auto-migration functionality:

```go
import (
    "time"
    "github.com/go-pantheon/fabrica-util/data/db/postgresql/migrate"
)

func main() {
    config := postgresql.NewConfig("postgres://user:password@localhost/dbname", "mydb")
    db, cleanup, err := postgresql.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup()

    // Define model
    type User struct {
        ID        int       `orm:"primaryKey;autoIncrement"`
        Email     string    `orm:"size:255;unique;notNull"`
        Name      string    `orm:"size:100;notNull"`
        Age       int       `orm:"default:0"`
        CreatedAt time.Time `orm:"default:NOW()"`
    }

    ctx := context.Background()
    
    // Method 1: Using backward-compatible API
    if err := migrate.Migrate(ctx, db, "users", User{}, nil); err != nil {
        log.Fatal("Migration failed:", err)
    }

    // Method 2: Using version-controlled auto migration
    if err := migrate.MigrateWithVersionControl(ctx, db, "users", User{}, nil, "auto_migrations"); err != nil {
        log.Fatal("Migration failed:", err)
    }

    // Method 3: Using AutoMigrator to manage multiple models
    autoMigrator := migrate.NewAutoMigrator(db, "auto_migrations")
    
    // Register models
    autoMigrator.RegisterModel("users", User{}, nil)
    autoMigrator.RegisterModel("posts", Post{}, map[string]string{
        "search_vector": "TSVECTOR",
    })
    
    // Migrate all registered models
    if err := autoMigrator.MigrateAll(ctx); err != nil {
        log.Fatal("Auto migration failed:", err)
    }
}
```

### 3. File Migration System

The file migration system supports SQL file-based migrations:

```go
func main() {
    config := postgresql.NewConfig("postgres://user:password@localhost/dbname", "mydb")
    db, cleanup, err := postgresql.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup()

    // Create file migrator
    migrator := migrate.NewFileMigrator(db, "./migrations", "schema_migrations")

    // Create new migration file
    if err := migrator.CreateMigration("create_posts_table"); err != nil {
        log.Fatal("Failed to create migration:", err)
    }

    // Load migrations from filesystem
    if err := migrator.LoadMigrations(); err != nil {
        log.Fatal("Failed to load migrations:", err)
    }

    ctx := context.Background()

    // Execute migrations
    if err := migrator.Up(ctx); err != nil {
        log.Fatal("Migration failed:", err)
    }
}
```

### 4. ORM Tag Parser

The ORM tag parser supports GORM-like tag syntax:

```go
import "github.com/go-pantheon/fabrica-util/data/db/postgresql/migrate"

func main() {
    parser := migrate.NewTagParser()
    
    // Parse struct field tags
    field := reflect.StructField{
        Name: "Email",
        Type: reflect.TypeOf(""),
        Tag:  `orm:"size:255;unique;notNull;default:example@email.com"`,
    }
    
    columnTag, err := parser.ParseTag(field)
    if err != nil {
        log.Fatal("Failed to parse tag:", err)
    }
    
    // Generate column definition
    definition := parser.BuildColumnDefinition(columnTag)
    fmt.Printf("Column definition: %s\n", definition)
}
```

## Migration File Format

When using `CreateMigration` to create migrations, two files are generated:

```
migrations/
├── 1635123456_create_posts_table.up.sql
└── 1635123456_create_posts_table.down.sql
```

### Up Migration File Example (`*.up.sql`)
```sql
-- Migration: create posts table
-- Created at: 2023-10-25 10:30:00

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_posts_title ON posts(title);
```

### Down Migration File Example (`*.down.sql`)
```sql
-- Rollback for: create posts table
-- Created at: 2023-10-25 10:30:00

DROP INDEX IF EXISTS idx_posts_title;
DROP TABLE IF EXISTS posts;
```

## ORM Tag Syntax

### Supported Tag Parameters

#### Simple Flag Parameters
- `primaryKey` - Primary key
- `autoIncrement` - Auto-increment (only used with primary key)
- `notNull` - Not null constraint
- `unique` - Unique constraint
- `index` - Index

#### Key-Value Parameters
- `column:name` - Custom column name
- `type:TYPE` - Specify data type
- `size:N` - String length (VARCHAR, CHAR)
- `precision:N` - Number precision
- `scale:N` - Number decimal places
- `default:value` - Default value
- `comment:text` - Column comment

### Type Mapping

| Go Type            | Default PostgreSQL Type | Example                           |
| ------------------ | ----------------------- | --------------------------------- |
| `int`, `int32`     | `INTEGER`               | `ID int`                          |
| `int64`            | `BIGINT`                | `UserID int64`                    |
| `float32`          | `REAL`                  | `Price float32`                   |
| `float64`          | `DOUBLE PRECISION`      | `Amount float64`                  |
| `bool`             | `BOOLEAN`               | `IsActive bool`                   |
| `string`           | `TEXT`                  | `Description string`              |
| `string` + `size`  | `VARCHAR(N)`            | `Name string \`orm:"size:100"\``  |
| `time.Time`        | `TIMESTAMPTZ`           | `CreatedAt time.Time`             |
| `[]byte`           | `BYTEA`                 | `Data []byte`                     |
| `[]string`         | `TEXT[]`                | `Tags []string`                   |
| `struct/map/slice` | `JSONB`                 | `Metadata map[string]interface{}` |

### Tag Examples

```go
type User struct {
    ID        int       `orm:"primaryKey;autoIncrement"`
    Email     string    `orm:"size:255;unique;notNull"`
    Name      string    `orm:"size:100;notNull"`
    Age       int       `orm:"default:0"`
    Bio       string    `orm:"type:TEXT"`
    Salary    float64   `orm:"precision:10;scale:2"`
    IsActive  bool      `orm:"default:true"`
    CreatedAt time.Time `orm:"default:NOW()"`
    UpdatedAt time.Time  // Uses default type inference when no tag
}

type Product struct {
    ID           int    `orm:"primaryKey;autoIncrement"`
    UUID         string `orm:"type:UUID;unique;notNull"`
    SearchVector string `orm:"type:TSVECTOR;column:search_vector"`
    IPAddress    string `orm:"type:INET"`
    MacAddress   string `orm:"type:MACADDR"`
    JSONData     string `orm:"type:JSONB"`
    Point        string `orm:"type:POINT"`
    ByteArray    []byte `orm:"type:BYTEA"`
    TextArray    []string `orm:"type:TEXT[]"`
    CreatedAt    time.Time `orm:"default:NOW()"`
}
```

## API Reference

### Migrator

#### Basic Methods
- `NewMigrator(db DBPool, tableName string) *Migrator` - Create new migrator
- `Add(migration *Migration)` - Add migration
- `AddFunc(id, comment string, up, down func) error` - Add function-based migration

#### Migration Operations
- `Up(ctx context.Context) error` - Execute all pending migrations
- `Down(ctx context.Context) error` - Rollback last migration
- `DownTo(ctx context.Context, targetID string) error` - Rollback to specific migration
- `Reset(ctx context.Context) error` - Rollback all migrations
- `Status(ctx context.Context) ([]MigrationStatus, error)` - Get migration status

### FileMigrator

- `NewFileMigrator(db DBPool, migrationsDir, tableName string) *FileMigrator` - Create file migrator
- `LoadMigrations() error` - Load migrations from filesystem
- `CreateMigration(name string) error` - Create new migration file

### AutoMigrator

- `NewAutoMigrator(db DBPool, migrationTable string) *AutoMigrator` - Create auto migrator
- `RegisterModel(tableName string, model any, extracols map[string]string)` - Register model
- `MigrateAll(ctx context.Context) error` - Migrate all registered models

### TagParser

- `NewTagParser() *TagParser` - Create tag parser
- `ParseTag(field reflect.StructField) (*ColumnTag, error)` - Parse field tag
- `BuildColumnDefinition(tag *ColumnTag) string` - Generate column definition

## Advanced Usage

### Complex Migration Example

```go
migrator.AddFunc("003_create_complex_schema", "Create complex schema",
    func(ctx context.Context, db postgresql.DBPool) error {
        statements := []string{
            `CREATE TABLE categories (
                id SERIAL PRIMARY KEY,
                name VARCHAR(100) UNIQUE NOT NULL
            );`,
            `CREATE TABLE posts (
                id SERIAL PRIMARY KEY,
                title VARCHAR(255) NOT NULL,
                category_id INTEGER REFERENCES categories(id)
            );`,
            `CREATE INDEX idx_posts_category_id ON posts(category_id);`,
        }

        for _, stmt := range statements {
            if _, err := db.Exec(ctx, stmt); err != nil {
                return err
            }
        }
        return nil
    },
    func(ctx context.Context, db postgresql.DBPool) error {
        statements := []string{
            "DROP TABLE IF EXISTS posts;",
            "DROP TABLE IF EXISTS categories;",
        }
        for _, stmt := range statements {
            db.Exec(ctx, stmt)
        }
        return nil
    },
)
```

### Conditional Migration

```go
migrator.AddFunc("004_conditional_migration", "Add column if not exists",
    func(ctx context.Context, db postgresql.DBPool) error {
        // Check if column exists
        var exists bool
        err := db.QueryRow(ctx, `
            SELECT EXISTS (
                SELECT 1 FROM information_schema.columns 
                WHERE table_name = 'users' AND column_name = 'phone'
            )
        `).Scan(&exists)
        
        if err != nil {
            return err
        }
        
        if !exists {
            _, err = db.Exec(ctx, "ALTER TABLE users ADD COLUMN phone VARCHAR(20);")
        }
        
        return err
    },
    nil, // Optional down migration
)
```

### Incremental Change Example

```go
// Demonstrates how to handle incremental changes
func incremental() {
    ctx := context.Background()
    
    // Version 1: Basic user model
    type UserV1 struct {
        ID    int    `orm:"primaryKey;autoIncrement"`
        Email string `orm:"size:255;unique;notNull"`
        Name  string `orm:"size:100;notNull"`
    }
    
    // Version 2: Add timestamps
    type UserV2 struct {
        ID        int       `orm:"primaryKey;autoIncrement"`
        Email     string    `orm:"size:255;unique;notNull"`
        Name      string    `orm:"size:100;notNull"`
        CreatedAt time.Time `orm:"default:NOW()"`
        UpdatedAt time.Time `orm:"default:NOW()"`
    }
    
    // Version 3: Add profile fields
    type UserV3 struct {
        ID        int       `orm:"primaryKey;autoIncrement"`
        Email     string    `orm:"size:255;unique;notNull"`
        Name      string    `orm:"size:100;notNull"`
        Bio       string    `orm:"type:TEXT"`
        AvatarURL string    `orm:"size:255"`
        CreatedAt time.Time `orm:"default:NOW()"`
        UpdatedAt time.Time `orm:"default:NOW()"`
    }
    
    // Progressive migration
    migrate.MigrateWithVersionControl(ctx, db, "users", UserV1{}, nil, "incremental")
    migrate.MigrateWithVersionControl(ctx, db, "users", UserV2{}, nil, "incremental")
    migrate.MigrateWithVersionControl(ctx, db, "users", UserV3{}, nil, "incremental")
}
```

## Best Practices

1. **Migration Naming**: Use timestamp prefixes to ensure order, descriptive naming
   ```
   20231025103000_create_users_table
   20231025104500_add_user_indexes
   ```

2. **Atomic Operations**: Each migration should be atomic, either fully successful or fully failed

3. **Backward Compatibility**: Try to maintain backward compatibility in migrations, avoid dropping columns or tables in use

4. **Test Migrations**: Test all migration up and down operations before production

5. **Backup**: Backup production database before running migrations

6. **Version Control**: Use version-controlled migration system for tracking and rollback

## Backward Compatibility

The new `orm` tags are fully compatible with existing `pgname`/`pgtype`/`primarykey` tags:

```go
type MixedModel struct {
    // New syntax (preferred)
    ID   int    `orm:"primaryKey;autoIncrement"`
    Name string `orm:"size:255;notNull"`
    
    // Old syntax (still supported)
    Email string `pgname:"email" pgtype:"VARCHAR(255) UNIQUE NOT NULL"`
    
    // No tags (use type inference)
    CreatedAt time.Time
}
```

## Error Handling

```go
if err := migrator.Up(ctx); err != nil {
    log.Printf("Migration failed: %v", err)
    // Handle migration failure
}

// Check migration status
statuses, err := migrator.Status(ctx)
if err != nil {
    log.Printf("Failed to get migration status: %v", err)
}

for _, status := range statuses {
    if status.Applied {
        log.Printf("✓ %s: %s (applied at %s)", status.ID, status.Comment, status.AppliedAt.Format(time.RFC3339))
    } else {
        log.Printf("✗ %s: %s (not applied)", status.ID, status.Comment)
    }
}
```
