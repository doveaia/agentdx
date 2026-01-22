package cli

import (
	"testing"

	"github.com/doveaia/agentdx/config"
)

func TestBuildContainerOptions(t *testing.T) {
	tests := []struct {
		name     string
		cfgName  string
		cfgPort  int
		flagName string
		flagPort int
		wantName string
		wantPort int
	}{
		{
			name:     "all defaults",
			wantName: "agentdx-postgres",
			wantPort: 55432,
		},
		{
			name:     "config only",
			cfgName:  "my-config-db",
			cfgPort:  5433,
			wantName: "my-config-db",
			wantPort: 5433,
		},
		{
			name:     "flags override config",
			cfgName:  "config-db",
			cfgPort:  5433,
			flagName: "flag-db",
			flagPort: 5434,
			wantName: "flag-db",
			wantPort: 5434,
		},
		{
			name:     "partial override - flag name only",
			cfgName:  "config-db",
			cfgPort:  5433,
			flagName: "flag-db",
			wantName: "flag-db",
			wantPort: 5433,
		},
		{
			name:     "partial override - flag port only",
			cfgName:  "config-db",
			cfgPort:  5433,
			flagPort: 5434,
			wantName: "config-db",
			wantPort: 5434,
		},
		{
			name:     "partial override - config name only",
			cfgName:  "config-db",
			flagPort: 5434,
			wantName: "config-db",
			wantPort: 5434,
		},
		{
			name:     "partial override - config port only",
			cfgPort:  5433,
			flagName: "flag-db",
			wantName: "flag-db",
			wantPort: 5433,
		},
		{
			name:     "zero values in config are ignored",
			cfgPort:  0, // zero should be ignored
			flagPort: 5434,
			wantName: "agentdx-postgres",
			wantPort: 5434,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Index: config.IndexSection{
					Store: config.StoreConfig{
						Postgres: config.PostgresConfig{
							ContainerName: tt.cfgName,
							Port:          tt.cfgPort,
						},
					},
				},
			}

			got := buildContainerOptions(cfg, tt.flagName, tt.flagPort)

			if got.Name != tt.wantName {
				t.Errorf("buildContainerOptions().Name = %s, want %s", got.Name, tt.wantName)
			}
			if got.Port != tt.wantPort {
				t.Errorf("buildContainerOptions().Port = %d, want %d", got.Port, tt.wantPort)
			}
		})
	}
}

func TestBuildSessionContainerOptions(t *testing.T) {
	tests := []struct {
		name     string
		cfgName  string
		cfgPort  int
		flagName string
		flagPort int
		wantName string
		wantPort int
	}{
		{
			name:     "all defaults",
			wantName: "agentdx-postgres",
			wantPort: 55432,
		},
		{
			name:     "config only",
			cfgName:  "my-session-db",
			cfgPort:  5435,
			wantName: "my-session-db",
			wantPort: 5435,
		},
		{
			name:     "flags override config",
			cfgName:  "config-db",
			cfgPort:  5433,
			flagName: "flag-db",
			flagPort: 5434,
			wantName: "flag-db",
			wantPort: 5434,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Index: config.IndexSection{
					Store: config.StoreConfig{
						Postgres: config.PostgresConfig{
							ContainerName: tt.cfgName,
							Port:          tt.cfgPort,
						},
					},
				},
			}

			got := buildSessionContainerOptions(cfg, tt.flagName, tt.flagPort)

			if got.Name != tt.wantName {
				t.Errorf("buildSessionContainerOptions().Name = %s, want %s", got.Name, tt.wantName)
			}
			if got.Port != tt.wantPort {
				t.Errorf("buildSessionContainerOptions().Port = %d, want %d", got.Port, tt.wantPort)
			}
		})
	}
}
