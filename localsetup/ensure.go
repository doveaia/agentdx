package localsetup

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"time"
)

// EnsurePostgresRunning ensures a PostgreSQL container is running and ready.
// Returns the DSN for connecting to the project database.
func EnsurePostgresRunning(ctx context.Context, projectRoot string, opts ContainerOptions) (string, error) {
	// Apply defaults
	defaults := DefaultContainerOptions()
	opts = defaults.Merge(opts)

	// Check Docker availability
	if !IsDockerAvailable() {
		return "", fmt.Errorf("Docker is not running. Please start Docker and try again")
	}

	// Check if container exists
	exists, err := ContainerExists(opts.Name)
	if err != nil {
		return "", fmt.Errorf("failed to check container: %w", err)
	}

	if exists {
		// Check if running
		running, err := ContainerRunning(opts.Name)
		if err != nil {
			return "", fmt.Errorf("failed to check container state: %w", err)
		}

		if !running {
			// Start stopped container
			if err := StartContainer(opts.Name); err != nil {
				return "", fmt.Errorf("failed to start container: %w", err)
			}
		}
	} else {
		// Create new container with volume
		cfg := ContainerConfig{
			Name:          opts.Name,
			Image:         containerImage,
			HostPort:      fmt.Sprintf("%d", opts.Port),
			ContainerPort: containerPort,
			RestartPolicy: "always",
			VolumeName:    opts.VolumeName(),
			EnvVars: map[string]string{
				"POSTGRES_USER":     defaultPostgresUser,
				"POSTGRES_PASSWORD": defaultPostgresPassword,
			},
		}

		if err := CreateContainer(cfg); err != nil {
			// Check if port is in use
			if isPortInUse(opts.Port) {
				return "", fmt.Errorf("Port %d is already in use. Try a different port with --pg-port", opts.Port)
			}
			return "", fmt.Errorf("failed to create container: %w", err)
		}
	}

	// Wait for PostgreSQL to be ready
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%d/postgres?sslmode=disable",
		defaultPostgresUser, defaultPostgresPassword, opts.Port)

	if err := WaitForPostgres(dsn, 30*time.Second); err != nil {
		return "", fmt.Errorf("PostgreSQL not ready after 30s. Check container logs: docker logs %s", opts.Name)
	}

	// Return project-specific DSN
	projectName := filepath.Base(projectRoot)
	dbName := "agentdx_" + ToSlug(projectName)

	// Create database if needed
	if err := CreateDatabase(dsn, dbName); err != nil {
		return "", fmt.Errorf("failed to create database: %w", err)
	}

	return fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable",
		defaultPostgresUser, defaultPostgresPassword, opts.Port, dbName), nil
}

// isPortInUse checks if a port is already in use.
func isPortInUse(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	listener.Close()
	return false
}
