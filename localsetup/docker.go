package localsetup

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	dockerCommandTimeout = 30 * time.Second
	containerName        = "agentdx-postgres"
	containerImage       = "doveaia/timescaledb:latest-pg17-ts"
	containerPort        = "5432"
)

// IsDockerAvailable checks if the docker CLI is available in PATH.
func IsDockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// ContainerExists checks if a container with the given name exists.
func ContainerExists(name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "inspect", name)
	err := cmd.Run()
	if err != nil {
		// docker inspect returns exit code 1 if container doesn't exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check container: %w", err)
	}
	return true, nil
}

// ContainerRunning checks if a container is currently running.
func ContainerRunning(name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}", name)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check container state: %w", err)
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

// CreateContainer creates a new Docker container with the specified configuration.
func CreateContainer(cfg ContainerConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), dockerCommandTimeout)
	defer cancel()

	args := []string{
		"run", "-d",
		"--name", cfg.Name,
		"--restart", cfg.RestartPolicy,
		"-p", fmt.Sprintf("%s:%s", cfg.HostPort, cfg.ContainerPort),
	}

	// Add volume if specified
	if cfg.VolumeName != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/var/lib/postgresql/data", cfg.VolumeName))
	}

	for key, value := range cfg.EnvVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, cfg.Image)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "Conflict. The container name") {
			exists, existsErr := ContainerExists(cfg.Name)
			if existsErr == nil && exists {
				return nil
			}
		}
		return fmt.Errorf("failed to create container: %s: %w", string(output), err)
	}
	return nil
}

// StartContainer starts a stopped container.
func StartContainer(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dockerCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "start", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %s: %w", string(output), err)
	}
	return nil
}

// DefaultContainerConfig returns the default configuration for the agentdx-postgres container.
func DefaultContainerConfig() ContainerConfig {
	return ContainerConfig{
		Name:          containerName,
		Image:         containerImage,
		HostPort:      fmt.Sprintf("%d", defaultPostgresPort),
		ContainerPort: containerPort,
		RestartPolicy: "always",
		EnvVars: map[string]string{
			"POSTGRES_USER":     defaultPostgresUser,
			"POSTGRES_PASSWORD": defaultPostgresPassword,
		},
	}
}

// RemoveContainer removes a Docker container.
// If the container is running, it will be stopped first.
// If the container doesn't exist, no error is returned.
// Waits up to 10 seconds for the container to be fully removed.
func RemoveContainer(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dockerCommandTimeout)
	defer cancel()

	// First, try to stop the container if it's running
	// Ignore errors if container isn't running
	stopCmd := exec.CommandContext(ctx, "docker", "stop", name)
	_ = stopCmd.Run()

	// Remove the container
	rmCmd := exec.CommandContext(ctx, "docker", "rm", name)
	output, err := rmCmd.CombinedOutput()
	if err != nil {
		// If container doesn't exist, that's fine
		if strings.Contains(string(output), "No such container") {
			return nil
		}
		return fmt.Errorf("failed to remove container: %s: %w", string(output), err)
	}

	// Wait for the container to be fully removed (Docker removal is async)
	// This prevents race conditions where we try to recreate immediately
	removeCtx, removeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer removeCancel()
	for {
		exists, err := ContainerExists(name)
		if err != nil || !exists {
			return nil
		}
		select {
		case <-removeCtx.Done():
			return fmt.Errorf("timeout waiting for container %s to be removed", name)
		case <-time.After(100 * time.Millisecond):
			// Continue waiting
		}
	}
}
