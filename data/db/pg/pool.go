package pg

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-pantheon/fabrica-util/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds the configuration for PostgreSQL connection pool
type Config struct {
	DSN               string
	DBName            string
	MaxConns          int32
	MinConns          int32
	ConnMaxIdleTime   time.Duration
	ConnMaxLifetime   time.Duration
	ConnectTimeout    time.Duration
	HealthCheckPeriod time.Duration
	// Tracer is an optional pgx tracer for OpenTelemetry instrumentation
	Tracer pgx.QueryTracer
}

func NewConfig(dsn, dbname string) Config {
	config := DefaultConfig()
	config.DSN = dsn
	config.DBName = dbname

	return config
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		MaxConns:          20,
		MinConns:          5,
		ConnMaxIdleTime:   15 * time.Minute,
		ConnMaxLifetime:   30 * time.Minute,
		ConnectTimeout:    5 * time.Second,
		HealthCheckPeriod: 1 * time.Minute,
		Tracer:            nil, // No tracing by default
	}
}

// DBPool defines the interface for database operations, allowing for mocking.
type DBPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Close()
}

// NewPool creates a new PostgreSQL connection pool with the given configuration
func NewPool(config Config) (pool *pgxpool.Pool, cleanup func(), err error) {
	if config.DSN == "" {
		return nil, nil, errors.New("dsn is empty")
	}

	if config.DBName == "" {
		return nil, nil, errors.New("dbname is empty")
	}

	// Parse config from DSN
	poolConfig, err := pgxpool.ParseConfig(config.DSN)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse DSN")
	}

	// Configure connection pool
	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinConns = config.MinConns
	poolConfig.MaxConnIdleTime = config.ConnMaxIdleTime
	poolConfig.MaxConnLifetime = config.ConnMaxLifetime
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod

	// Add tracer if provided
	if config.Tracer != nil {
		poolConfig.ConnConfig.Tracer = config.Tracer
	}

	// Create connection pool with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create connection pool")
	}

	// Test connection
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, errors.Wrap(err, "failed to ping database")
	}

	cleanup = func() {
		pool.Close()
		slog.Info("database connection pool closed")
	}

	return pool, cleanup, nil
}

// NewPoolSimple creates a PostgreSQL connection pool with simple parameters (backward compatibility)
func NewPoolSimple(dsn, dbname string) (pool *pgxpool.Pool, cleanup func(), err error) {
	config := DefaultConfig()
	config.DSN = dsn
	config.DBName = dbname

	pool, cleanup, err = NewPool(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create connection pool")
	}

	return pool, cleanup, nil
}

// HealthCheck performs a health check on the database connection pool
func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("connection pool is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return errors.Wrap(err, "database health check failed")
	}

	return nil
}

// GetConnectionStats returns connection pool statistics
func GetConnectionStats(pool *pgxpool.Pool) *pgxpool.Stat {
	if pool == nil {
		return nil
	}

	return pool.Stat()
}
