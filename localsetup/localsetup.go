package localsetup

import (
	"fmt"
	"path/filepath"
	"time"
)

const (
	postgresReadyTimeout = 30 * time.Second
)

// SetupResult contains the outcome of a local setup operation.
type SetupResult struct {
	Mode             string // "local"
	DSN              string // Full PostgreSQL connection string
	DatabaseName     string // e.g., "agentdx_my_project"
	ContainerName    string // "agentdx-postgres"
	DockerUsed       bool   // Whether Docker was available and used
	ComposeGenerated bool   // Whether compose.yaml was generated
	ComposeFilePath  string // Path to generated compose.yaml
}

// RunLocalSetup orchestrates the complete local development setup.
// It creates/starts the Docker container if Docker is available,
// waits for PostgreSQL to be ready, creates the project database,
// and always generates the compose.yaml file.
func RunLocalSetup(projectRoot string) (*SetupResult, error) {
	// Get project folder name and convert to slug
	projectName := filepath.Base(projectRoot)
	dbName := "agentdx_" + ToSlug(projectName)

	if dbName == "agentdx_" {
		return nil, fmt.Errorf("project folder name '%s' produces empty slug", projectName)
	}

	result := &SetupResult{
		Mode:          "local",
		DatabaseName:  dbName,
		ContainerName: "agentdx-postgres",
		DSN:           ProjectDSN(dbName),
	}

	// Always generate compose.yaml
	if err := WriteComposeFile(projectRoot); err != nil {
		return nil, fmt.Errorf("failed to generate compose.yaml: %w", err)
	}
	result.ComposeGenerated = true
	result.ComposeFilePath = filepath.Join(projectRoot, ".agentdx", "compose.yaml")

	// Check if Docker is available
	if !IsDockerAvailable() {
		// Docker not available - compose.yaml generated, return with instructions
		result.DockerUsed = false
		return result, nil
	}

	result.DockerUsed = true

	// Check if container exists
	exists, err := ContainerExists(result.ContainerName)
	if err != nil {
		return nil, fmt.Errorf("failed to check container: %w", err)
	}

	if !exists {
		// Create the container
		cfg := DefaultContainerConfig()
		if err := CreateContainer(cfg); err != nil {
			// Race condition: another test may have created the container
			// Check again and if it now exists, continue
			exists, retryErr := ContainerExists(result.ContainerName)
			if retryErr == nil && exists {
				// Container was created by another test/goroutine, continue
			} else {
				return nil, fmt.Errorf("failed to create container: %w", err)
			}
		}
	}

	// At this point, container should exist (either we created it or it was already there)
	// Check if it's running and start if needed
	running, err := ContainerRunning(result.ContainerName)
	if err != nil {
		return nil, fmt.Errorf("failed to check container state: %w", err)
	}
	if !running {
		// Start the stopped container
		if err := StartContainer(result.ContainerName); err != nil {
			return nil, fmt.Errorf("failed to start container: %w", err)
		}
	}

	// Wait for PostgreSQL to be ready
	if err := WaitForPostgres(PostgresDSN(), postgresReadyTimeout); err != nil {
		return nil, fmt.Errorf("PostgreSQL not ready: %w", err)
	}

	// Create the project database
	if err := CreateDatabase(PostgresDSN(), dbName); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return result, nil
}
