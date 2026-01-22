package localsetup

import "testing"

func TestDefaultContainerOptions(t *testing.T) {
	opts := DefaultContainerOptions()

	if opts.Name != "agentdx-postgres" {
		t.Errorf("expected default name 'agentdx-postgres', got '%s'", opts.Name)
	}

	if opts.Port != 55432 {
		t.Errorf("expected default port 55432, got %d", opts.Port)
	}
}

func TestVolumeName(t *testing.T) {
	tests := []struct {
		name       string
		opts       ContainerOptions
		wantVolume string
	}{
		{
			name:       "default options",
			opts:       DefaultContainerOptions(),
			wantVolume: "agentdx-postgres-data",
		},
		{
			name: "custom name",
			opts: ContainerOptions{
				Name: "my-custom-pg",
				Port: 5433,
			},
			wantVolume: "my-custom-pg-data",
		},
		{
			name: "name with hyphens",
			opts: ContainerOptions{
				Name: "my-project-postgres",
				Port: 55432,
			},
			wantVolume: "my-project-postgres-data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.VolumeName()
			if got != tt.wantVolume {
				t.Errorf("VolumeName() = %s, want %s", got, tt.wantVolume)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     ContainerOptions
		other    ContainerOptions
		wantName string
		wantPort int
	}{
		{
			name: "merge with empty other",
			base: ContainerOptions{
				Name: "base-name",
				Port: 5433,
			},
			other:    ContainerOptions{},
			wantName: "base-name",
			wantPort: 5433,
		},
		{
			name: "merge with name only",
			base: ContainerOptions{
				Name: "base-name",
				Port: 5433,
			},
			other: ContainerOptions{
				Name: "other-name",
			},
			wantName: "other-name",
			wantPort: 5433,
		},
		{
			name: "merge with port only",
			base: ContainerOptions{
				Name: "base-name",
				Port: 5433,
			},
			other: ContainerOptions{
				Port: 5444,
			},
			wantName: "base-name",
			wantPort: 5444,
		},
		{
			name: "merge with both values",
			base: ContainerOptions{
				Name: "base-name",
				Port: 5433,
			},
			other: ContainerOptions{
				Name: "other-name",
				Port: 5444,
			},
			wantName: "other-name",
			wantPort: 5444,
		},
		{
			name: "merge defaults with custom",
			base: ContainerOptions{
				Name: "agentdx-postgres",
				Port: 55432,
			},
			other: ContainerOptions{
				Name: "custom-pg",
				Port: 5433,
			},
			wantName: "custom-pg",
			wantPort: 5433,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.base.Merge(tt.other)
			if got.Name != tt.wantName {
				t.Errorf("Merge().Name = %s, want %s", got.Name, tt.wantName)
			}
			if got.Port != tt.wantPort {
				t.Errorf("Merge().Port = %d, want %d", got.Port, tt.wantPort)
			}
		})
	}
}

func TestMergePreservesOriginal(t *testing.T) {
	base := ContainerOptions{
		Name: "base-name",
		Port: 5433,
	}
	other := ContainerOptions{
		Name: "other-name",
		Port: 5444,
	}

	// Store original values
	originalName := base.Name
	originalPort := base.Port

	_ = base.Merge(other)

	// Original should be unchanged
	if base.Name != originalName {
		t.Errorf("Merge() modified original Name: %s != %s", base.Name, originalName)
	}
	if base.Port != originalPort {
		t.Errorf("Merge() modified original Port: %d != %d", base.Port, originalPort)
	}
}
