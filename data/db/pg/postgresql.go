package pg

import (
	"context"
	"database/sql"

	"github.com/go-pantheon/fabrica-util/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is a wrapper around pgxpool.Pool that provides a sql.DB-like interface
// while maintaining the performance benefits of pgx.
// This allows for easier migration from database/sql to pgx without changing
// the application interface significantly.
type DB struct {
	pool *pgxpool.Pool
}

// NewDB creates a new DB instance that wraps a pgxpool.Pool
func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{
		pool: pool,
	}
}

// NewDBFromConfig creates a new DB instance from a Config
func NewDBFromConfig(config Config) (*DB, func(), error) {
	pool, cleanup, err := NewPool(config)
	if err != nil {
		return nil, nil, err
	}

	return NewDB(pool), cleanup, nil
}

// NewDBSimple creates a new DB instance with simple parameters
func NewDBSimple(dsn, dbname string) (*DB, func(), error) {
	pool, cleanup, err := NewPoolSimple(dsn, dbname)
	if err != nil {
		return nil, nil, err
	}

	return NewDB(pool), cleanup, nil
}

// Exec executes a query without returning any rows
func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.ExecContext(context.Background(), query, args...)
}

// ExecContext executes a query without returning any rows with context
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if db.pool == nil {
		return nil, errors.New("connection pool is nil")
	}

	cmdTag, err := db.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query")
	}

	return &pgxResult{cmdTag: cmdTag}, nil
}

// Query executes a query that returns pgx.Rows (native pgx interface)
func (db *DB) Query(query string, args ...any) (pgx.Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}

// QueryContext executes a query that returns pgx.Rows with context
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if db.pool == nil {
		return nil, errors.New("connection pool is nil")
	}

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query")
	}

	return rows, nil
}

// QueryRow executes a query that is expected to return at most one row
func (db *DB) QueryRow(query string, args ...any) pgx.Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row with context
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) pgx.Row {
	if db.pool == nil {
		return nil
	}

	return db.pool.QueryRow(ctx, query, args...)
}

// Begin starts a transaction
func (db *DB) Begin() (pgx.Tx, error) {
	return db.BeginTx(context.Background(), pgx.TxOptions{})
}

// BeginTx starts a transaction with options
func (db *DB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	if db.pool == nil {
		return nil, errors.New("connection pool is nil")
	}

	pgxTx, err := db.pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}

	return pgxTx, nil
}

// Close closes the database connection pool
func (db *DB) Close() error {
	if db.pool == nil {
		return nil
	}

	db.pool.Close()

	return nil
}

// Ping verifies a connection to the database is still alive
func (db *DB) Ping() error {
	return db.PingContext(context.Background())
}

// PingContext verifies a connection to the database is still alive with context
func (db *DB) PingContext(ctx context.Context) error {
	if db.pool == nil {
		return errors.New("connection pool is nil")
	}

	return db.pool.Ping(ctx)
}

// Stats returns database statistics
func (db *DB) Stats() sql.DBStats {
	if db.pool == nil {
		return sql.DBStats{}
	}

	stat := db.pool.Stat()

	return sql.DBStats{
		MaxOpenConnections: int(stat.MaxConns()),
		OpenConnections:    int(stat.TotalConns()),
		InUse:              int(stat.AcquiredConns()),
		Idle:               int(stat.IdleConns()),
	}
}

// GetPool returns the underlying pgxpool.Pool
func (db *DB) GetPool() *pgxpool.Pool {
	return db.pool
}

// pgxResult implements sql.Result interface for pgx CommandTag
type pgxResult struct {
	cmdTag pgconn.CommandTag
}

// LastInsertId returns the last insert ID (not supported by PostgreSQL)
func (r *pgxResult) LastInsertId() (int64, error) {
	return 0, errors.New("LastInsertId is not supported by PostgreSQL")
}

// RowsAffected returns the number of rows affected by the query
func (r *pgxResult) RowsAffected() (int64, error) {
	return r.cmdTag.RowsAffected(), nil
}
