package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	ConfigDir           = ".agentdx"
	ConfigFileName      = "config.yaml"
	SymbolIndexFileName = "symbols.gob"
)

// Config holds the agentdx configuration.
type Config struct {
	Version int          `yaml:"version"`
	Mode    string       `yaml:"mode"` // "local" or "remote" - local uses embedded PostgreSQL, remote uses configured backend
	Index   IndexSection `yaml:"index"`
}
type IndexSection struct {
	Store    StoreConfig    `yaml:"store"`
	Chunking ChunkingConfig `yaml:"chunking"`
	Watch    WatchConfig    `yaml:"watch"`
	Search   SearchConfig   `yaml:"search"`
	Trace    TraceConfig    `yaml:"trace"`
	Update   UpdateConfig   `yaml:"update"`
	Ignore   []string       `yaml:"ignore"`
}

// UpdateConfig holds auto-update settings
type UpdateConfig struct {
	CheckOnStartup bool `yaml:"check_on_startup"` // Check for updates when running commands
}

type SearchConfig struct {
	Boost BoostConfig `yaml:"boost"`
}

type BoostConfig struct {
	Enabled   bool        `yaml:"enabled"`
	Penalties []BoostRule `yaml:"penalties"`
	Bonuses   []BoostRule `yaml:"bonuses"`
}

type BoostRule struct {
	Pattern string  `yaml:"pattern"`
	Factor  float32 `yaml:"factor"`
}

type StoreConfig struct {
	Postgres PostgresConfig `yaml:"postgres,omitempty"`
}

type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

type ChunkingConfig struct {
	Size    int `yaml:"size"`
	Overlap int `yaml:"overlap"`
}

type WatchConfig struct {
	DebounceMs int `yaml:"debounce_ms"`
}

type TraceConfig struct {
	Mode             string   `yaml:"mode"`              // fast or precise
	EnabledLanguages []string `yaml:"enabled_languages"` // File extensions to index
	ExcludePatterns  []string `yaml:"exclude_patterns"`  // Patterns to exclude
}

func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Mode:    "local",
		Index: IndexSection{
			Store: StoreConfig{},
			Chunking: ChunkingConfig{
				Size:    512,
				Overlap: 50,
			},
			Watch: WatchConfig{
				DebounceMs: 500,
			},
			Search: SearchConfig{
				Boost: BoostConfig{
					Enabled: true,
					Penalties: []BoostRule{
						// Test files (multi-language)
						{Pattern: "/tests/", Factor: 0.5},
						{Pattern: "/test/", Factor: 0.5},
						{Pattern: "__tests__", Factor: 0.5},
						{Pattern: "_test.", Factor: 0.5},
						{Pattern: ".test.", Factor: 0.5},
						{Pattern: ".spec.", Factor: 0.5},
						{Pattern: "test_", Factor: 0.5},
						// Mocks
						{Pattern: "/mocks/", Factor: 0.4},
						{Pattern: "/mock/", Factor: 0.4},
						{Pattern: ".mock.", Factor: 0.4},
						// Fixtures & test data
						{Pattern: "/fixtures/", Factor: 0.4},
						{Pattern: "/testdata/", Factor: 0.4},
						// Generated code
						{Pattern: "/generated/", Factor: 0.4},
						{Pattern: ".generated.", Factor: 0.4},
						{Pattern: ".gen.", Factor: 0.4},
						// Documentation
						{Pattern: ".md", Factor: 0.6},
						{Pattern: "/docs/", Factor: 0.6},
					},
					Bonuses: []BoostRule{
						// Entry points (multi-language)
						{Pattern: "/src/", Factor: 1.1},
						{Pattern: "/lib/", Factor: 1.1},
						{Pattern: "/app/", Factor: 1.1},
					},
				},
			},
			Trace: TraceConfig{
				Mode: "fast",
				EnabledLanguages: []string{
					".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".php",
					".c", ".h", ".cpp", ".hpp", ".cc", ".cxx",
					".rs", ".zig",
				},
				ExcludePatterns: []string{
					"*_test.go",
					"*.spec.ts",
					"*.spec.js",
					"*.test.ts",
					"*.test.js",
					"__tests__/*",
				},
			},
			Update: UpdateConfig{
				CheckOnStartup: false, // Opt-in by default for privacy
			},
			Ignore: []string{
				".git",
				".agentdx",
				"node_modules",
				"vendor",
				"bin",
				"dist",
				"__pycache__",
				".venv",
				"venv",
				".idea",
				".vscode",
				"target",
				".zig-cache",
				"zig-out",
			},
		},
	}
}

func GetConfigDir(projectRoot string) string {
	return filepath.Join(projectRoot, ConfigDir)
}

func GetConfigPath(projectRoot string) string {
	return filepath.Join(GetConfigDir(projectRoot), ConfigFileName)
}

func GetSymbolIndexPath(projectRoot string) string {
	return filepath.Join(GetConfigDir(projectRoot), SymbolIndexFileName)
}

func Load(projectRoot string) (*Config, error) {
	configPath := GetConfigPath(projectRoot)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for missing values (backward compatibility)
	cfg.applyDefaults()

	return &cfg, nil
}

// applyDefaults fills in missing configuration values with sensible defaults.
func (c *Config) applyDefaults() {
	defaults := DefaultConfig()

	// Chunking defaults
	if c.Index.Chunking.Size == 0 {
		c.Index.Chunking.Size = defaults.Index.Chunking.Size
	}
	if c.Index.Chunking.Overlap == 0 {
		c.Index.Chunking.Overlap = defaults.Index.Chunking.Overlap
	}

	// Watch defaults
	if c.Index.Watch.DebounceMs == 0 {
		c.Index.Watch.DebounceMs = defaults.Index.Watch.DebounceMs
	}
}

func (c *Config) Save(projectRoot string) error {
	configDir := GetConfigDir(projectRoot)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := GetConfigPath(projectRoot)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func Exists(projectRoot string) bool {
	configPath := GetConfigPath(projectRoot)
	_, err := os.Stat(configPath)
	return err == nil
}

func FindProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	dir := cwd
	for {
		if Exists(dir) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no agentdx project found (run 'agentdx init' first)")
}
