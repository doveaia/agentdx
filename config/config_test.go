package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}

	if cfg.Mode != "local" {
		t.Errorf("expected mode local, got %s", cfg.Mode)
	}

	if cfg.Index.Chunking.Size != 512 {
		t.Errorf("expected chunk size 512, got %d", cfg.Index.Chunking.Size)
	}

	if cfg.Index.Chunking.Overlap != 50 {
		t.Errorf("expected chunk overlap 50, got %d", cfg.Index.Chunking.Overlap)
	}

	if cfg.Index.Watch.DebounceMs != 500 {
		t.Errorf("expected debounce 500ms, got %d", cfg.Index.Watch.DebounceMs)
	}

	if !cfg.Index.Search.Boost.Enabled {
		t.Error("expected boost enabled")
	}

	if cfg.Index.Trace.Mode != "fast" {
		t.Errorf("expected trace mode fast, got %s", cfg.Index.Trace.Mode)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Mode = "remote"
	cfg.Index.Store.Postgres.DSN = "postgres://localhost/test?sslmode=disable"

	err := cfg.Save(tmpDir)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Check file exists
	configPath := GetConfigPath(tmpDir)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load config
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Mode != "remote" {
		t.Errorf("expected mode remote, got %s", loaded.Mode)
	}

	if loaded.Index.Store.Postgres.DSN != "postgres://localhost/test?sslmode=disable" {
		t.Errorf("expected postgres DSN, got %s", loaded.Index.Store.Postgres.DSN)
	}
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	if Exists(tmpDir) {
		t.Error("config should not exist initially")
	}

	// Create config
	cfg := DefaultConfig()
	if err := cfg.Save(tmpDir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Should exist now
	if !Exists(tmpDir) {
		t.Error("config should exist after saving")
	}
}

func TestGetConfigDir(t *testing.T) {
	result := GetConfigDir("/test/path")
	expected := filepath.Join("/test/path", ConfigDir)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetConfigPath(t *testing.T) {
	result := GetConfigPath("/test/path")
	expected := filepath.Join("/test/path", ConfigDir, ConfigFileName)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetSymbolIndexPath(t *testing.T) {
	result := GetSymbolIndexPath("/test/path")
	expected := filepath.Join("/test/path", ConfigDir, SymbolIndexFileName)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
	}{
		{
			name: "minimal config gets defaults",
			configYAML: `version: 1
mode: local
`,
		},
		{
			name: "empty chunking gets defaults",
			configYAML: `version: 1
mode: local
index:
  chunking: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, ConfigDir)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatalf("failed to create config dir: %v", err)
			}

			configPath := filepath.Join(configDir, ConfigFileName)
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0600); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			loaded, err := Load(tmpDir)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			defaults := DefaultConfig()

			if loaded.Index.Chunking.Size != defaults.Index.Chunking.Size {
				t.Errorf("expected chunk size %d, got %d", defaults.Index.Chunking.Size, loaded.Index.Chunking.Size)
			}

			if loaded.Index.Chunking.Overlap != defaults.Index.Chunking.Overlap {
				t.Errorf("expected chunk overlap %d, got %d", defaults.Index.Chunking.Overlap, loaded.Index.Chunking.Overlap)
			}

			if loaded.Index.Watch.DebounceMs != defaults.Index.Watch.DebounceMs {
				t.Errorf("expected debounce %d, got %d", defaults.Index.Watch.DebounceMs, loaded.Index.Watch.DebounceMs)
			}
		})
	}
}
