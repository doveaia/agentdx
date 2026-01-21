package localsetup

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateComposeYAML returns the Docker Compose file content.
func GenerateComposeYAML() string {
	return fmt.Sprintf(`services:
  postgres:
    image: doveaia/timescaledb:latest-pg17-ts
    container_name: %s
    environment:
      POSTGRES_USER: %s
      POSTGRES_PASSWORD: %s
    ports:
      - "%d:5432"
    volumes:
      - agentdx-pgdata:/var/lib/postgresql/data
    restart: always
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U %s"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  agentdx-pgdata:
`, containerName, defaultPostgresUser, defaultPostgresPassword, defaultPostgresPort, defaultPostgresUser)
}

// WriteComposeFile writes the compose.yaml file to the .agentdx directory.
func WriteComposeFile(projectRoot string) error {
	agentdxDir := filepath.Join(projectRoot, ".agentdx")

	// Ensure .agentdx directory exists
	if err := os.MkdirAll(agentdxDir, 0755); err != nil {
		return fmt.Errorf("failed to create .agentdx directory: %w", err)
	}

	composePath := filepath.Join(agentdxDir, "compose.yaml")
	content := GenerateComposeYAML()

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write compose.yaml: %w", err)
	}

	return nil
}
