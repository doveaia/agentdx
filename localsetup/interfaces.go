// Package localsetup provides local development environment setup functionality,
// including Docker container management and PostgreSQL database initialization.
package localsetup

import "time"

// DockerClient defines operations for Docker container management.
type DockerClient interface {
	// IsAvailable checks if the Docker CLI is available in the system PATH.
	IsAvailable() bool
	// ContainerExists checks if a container with the given name exists.
	ContainerExists(name string) (bool, error)
	// ContainerRunning checks if a container with the given name is currently running.
	ContainerRunning(name string) (bool, error)
	// CreateContainer creates a new Docker container with the specified configuration.
	CreateContainer(cfg ContainerConfig) error
	// StartContainer starts an existing but stopped container.
	StartContainer(name string) error
}

// ContainerConfig specifies Docker container settings for local development.
type ContainerConfig struct {
	// Name is the container name (e.g., "agentdx-postgres")
	Name string
	// Image is the Docker image to use (e.g., "doveaia/timescaledb:latest-pg17-ts")
	Image string
	// EnvVars are environment variables to pass to the container
	EnvVars map[string]string
	// HostPort is the port on the host machine (e.g., "55432")
	HostPort string
	// ContainerPort is the port inside the container (e.g., "5432")
	ContainerPort string
	// RestartPolicy defines when the container should restart (e.g., "always", "unless-stopped")
	RestartPolicy string
}

// DatabaseClient defines operations for PostgreSQL database management.
type DatabaseClient interface {
	// WaitForReady waits for PostgreSQL to be ready to accept connections.
	WaitForReady(dsn string, timeout time.Duration) error
	// CreateDatabase creates a new database if it doesn't already exist.
	CreateDatabase(dsn, dbName string) error
}
