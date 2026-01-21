package localsetup

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	// PostgreSQL connection constants
	defaultPostgresUser     = "agentdx"
	defaultPostgresPassword = "agentdx"
	defaultPostgresHost     = "localhost"
	defaultPostgresPort     = 55432
	defaultPostgresDB       = "postgres"

	// Retry constants
	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 5 * time.Second
)

// WaitForPostgres polls the PostgreSQL server until it's ready or timeout expires.
// Uses exponential backoff starting at 500ms, maxing at 5s between attempts.
func WaitForPostgres(dsn string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := initialBackoff
	ctx := context.Background()

	for time.Now().Before(deadline) {
		conn, err := pgx.Connect(ctx, dsn)
		if err == nil {
			conn.Close(ctx)
			return nil
		}

		// Sleep with exponential backoff
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return fmt.Errorf("timeout waiting for PostgreSQL to be ready after %v", timeout)
}

// CreateDatabase creates a new database if it doesn't already exist.
// Connects to the 'postgres' default database to execute CREATE DATABASE.
func CreateDatabase(dsn, dbName string) error {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer conn.Close(ctx)

	// Check if database already exists
	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		return nil // Database already exists, nothing to do
	}

	// Create the database (can't use parameterized query for CREATE DATABASE)
	// dbName is derived from ToSlug() which only allows alphanumeric and underscore
	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}

	return nil
}

// PostgresDSN returns a DSN for connecting to the postgres default database.
func PostgresDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		defaultPostgresUser, defaultPostgresPassword,
		defaultPostgresHost, defaultPostgresPort,
		defaultPostgresDB)
}

// ProjectDSN returns a DSN for connecting to the project-specific database.
func ProjectDSN(dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		defaultPostgresUser, defaultPostgresPassword,
		defaultPostgresHost, defaultPostgresPort,
		dbName)
}
