package localsetup

// ContainerOptions holds configuration for the PostgreSQL container.
type ContainerOptions struct {
	Name string // Container name (default: "agentdx-postgres")
	Port int    // Host port (default: 55432)
}

// DefaultContainerOptions returns the default container configuration.
func DefaultContainerOptions() ContainerOptions {
	return ContainerOptions{
		Name: "agentdx-postgres",
		Port: 55432,
	}
}

// VolumeName returns the Docker volume name for this container.
func (o ContainerOptions) VolumeName() string {
	return o.Name + "-data"
}

// Merge returns a new ContainerOptions with non-zero values from other taking precedence.
func (o ContainerOptions) Merge(other ContainerOptions) ContainerOptions {
	result := o
	if other.Name != "" {
		result.Name = other.Name
	}
	if other.Port != 0 {
		result.Port = other.Port
	}
	return result
}
