package localsetup

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

const (
	maxRetries     = 20
	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 5 * time.Second
)

// WaitForPostgres polls the PostgreSQL server until it's ready or timeout expires.
// Uses exponential backoff starting at 500ms, maxing at 5s between attempts.
func WaitForPostgres(dsn string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := initialBackoff

	for time.Now().Before(deadline) {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return fmt.Errorf("failed to open database connection: %w", err)
		}

		err = db.Ping()
		db.Close()

		if err == nil {
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
	// Connect to default postgres database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	// Check if database already exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		return nil // Database already exists, nothing to do
	}

	// Create the database (can't use parameterized query for CREATE DATABASE)
	// dbName is derived from ToSlug() which only allows alphanumeric and underscore
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}

	return nil
}

// PostgresDSN returns a DSN for connecting to the postgres default database.
func PostgresDSN() string {
	return "postgres://agentdx:agentdx@localhost:55432/postgres?sslmode=disable"
}

// ProjectDSN returns a DSN for connecting to the project-specific database.
func ProjectDSN(dbName string) string {
	return fmt.Sprintf("postgres://agentdx:agentdx@localhost:55432/%s?sslmode=disable", dbName)
}
