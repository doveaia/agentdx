package localsetup

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"testing"
	"time"
)

// TestContainer represents an ephemeral PostgreSQL container for testing.
type TestContainer struct {
	Name    string
	Port    int
	DSN     string
	t       testing.TB
	cleanup func()
}

// NewTestContainer creates a new PostgreSQL container with a random name and port.
// The container is automatically cleaned up when the test completes.
func NewTestContainer(t testing.TB) *TestContainer {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if !IsDockerAvailable() {
		t.Skip("Docker not available")
	}

	// Generate random name and find available port
	name := generateRandomName()
	port := findAvailablePort()

	tc := &TestContainer{
		Name: name,
		Port: port,
		t:    t,
	}

	// Create the container
	cfg := ContainerConfig{
		Name:          name,
		Image:         containerImage,
		HostPort:      fmt.Sprintf("%d", port),
		ContainerPort: containerPort,
		RestartPolicy: "no", // Don't restart test containers
		VolumeName:    "",   // No volume for test containers
		EnvVars: map[string]string{
			"POSTGRES_USER":     defaultPostgresUser,
			"POSTGRES_PASSWORD": defaultPostgresPassword,
		},
	}

	if err := CreateContainer(cfg); err != nil {
		t.Fatalf("failed to create test container: %v", err)
	}

	// Register cleanup
	tc.cleanup = func() {
		_ = RemoveContainer(name)
	}
	t.Cleanup(tc.cleanup)

	// Wait for PostgreSQL to be ready
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%d/postgres?sslmode=disable",
		defaultPostgresUser, defaultPostgresPassword, port)

	if err := WaitForPostgres(dsn, 30*time.Second); err != nil {
		tc.cleanup()
		t.Fatalf("test PostgreSQL not ready: %v", err)
	}

	tc.DSN = dsn
	return tc
}

// Close explicitly removes the container. Usually called automatically via t.Cleanup.
func (tc *TestContainer) Close() {
	if tc.cleanup != nil {
		tc.cleanup()
	}
}

// CreateDatabase creates a database in the test container and returns the DSN.
func (tc *TestContainer) CreateDatabase(dbName string) string {
	if err := CreateDatabase(tc.DSN, dbName); err != nil {
		tc.t.Fatalf("failed to create test database: %v", err)
	}
	return fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable",
		defaultPostgresUser, defaultPostgresPassword, tc.Port, dbName)
}

// generateRandomName creates a unique container name for testing.
func generateRandomName() string {
	b := make([]byte, 4) // 8 hex characters
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if crypto/rand fails
		return fmt.Sprintf("agentdx-test-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("agentdx-test-%s", hex.EncodeToString(b))
}

// findAvailablePort finds an available TCP port.
func findAvailablePort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		// Fallback to a random port in a high range
		return 50000 + int(time.Now().UnixNano()%10000)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
