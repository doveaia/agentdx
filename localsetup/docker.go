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
	hostPort             = "55432"
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

	for key, value := range cfg.EnvVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, cfg.Image)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
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
		HostPort:      hostPort,
		ContainerPort: containerPort,
		RestartPolicy: "always",
		EnvVars: map[string]string{
			"POSTGRES_USER":     "agentdx",
			"POSTGRES_PASSWORD": "agentdx",
		},
	}
}
