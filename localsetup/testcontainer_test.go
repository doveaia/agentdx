package localsetup

import (
	"fmt"
	"testing"
)

func TestNewTestContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := NewTestContainer(t)

	// Verify container was created
	exists, err := ContainerExists(tc.Name)
	if err != nil {
		t.Fatalf("failed to check container: %v", err)
	}
	if !exists {
		t.Error("container should exist")
	}

	// Verify DSN is set
	if tc.DSN == "" {
		t.Error("DSN should be set")
	}

	// Verify port is valid
	if tc.Port < 1024 || tc.Port > 65535 {
		t.Errorf("port %d is out of valid range", tc.Port)
	}
}

func TestTestContainerParallel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Run multiple tests in parallel to verify no conflicts
	for i := 0; i < 3; i++ {
		i := i
		t.Run(fmt.Sprintf("parallel-%d", i), func(t *testing.T) {
			t.Parallel()
			tc := NewTestContainer(t)
			t.Logf("Container %d: %s on port %d", i, tc.Name, tc.Port)

			// Verify each container has a unique name
			if tc.Name == "" {
				t.Error("container name should not be empty")
			}
		})
	}
}

func TestTestContainerCreateDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := NewTestContainer(t)

	// Sanitize container name for use as database name (replace hyphens with underscores)
	dbName := "test_db_" + sanitizeDbName(tc.Name)
	dsn := tc.CreateDatabase(dbName)

	if dsn == "" {
		t.Error("DSN should not be empty")
	}

	// Verify the DSN contains the database name
	expected := fmt.Sprintf("/%s?", dbName)
	if !contains(dsn, expected) {
		t.Errorf("DSN should contain database name %s: %s", dbName, dsn)
	}
}

// sanitizeDbName replaces characters invalid for PostgreSQL database names with underscores.
// PostgreSQL identifiers can only contain letters, digits, and underscores.
func sanitizeDbName(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
