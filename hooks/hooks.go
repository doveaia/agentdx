package hooks

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// AgentdxHooksDir is the directory containing hook templates
	AgentdxHooksDir = ".claude/hooks/agentdx"
)

//go:embed templates/*.sh
var embeddedTemplates embed.FS

// AgentHookConfig describes where to install hooks for a coding agent
type AgentHookConfig struct {
	Name         string // Agent name (e.g., "claude-code")
	StartHookDir string // Directory for start hooks
	StopHookDir  string // Directory for stop hooks
	StartScript  string // Script filename in agentdx/start/
	StopScript   string // Script filename in agentdx/stop/
}

// SupportedAgents returns configuration for all supported coding agents
// All paths are project-relative (no ~ prefix) to install hooks in project directory
func SupportedAgents() []AgentHookConfig {
	return []AgentHookConfig{
		{
			Name:         "claude-code",
			StartHookDir: ".claude/hooks/UserPromptSubmit",
			StopHookDir:  ".claude/hooks/Stop",
			StartScript:  "claude-code.sh",
			StopScript:   "claude-code.sh",
		},
		{
			Name:         "codex",
			StartHookDir: ".codex/hooks/start",
			StopHookDir:  ".codex/hooks/stop",
			StartScript:  "codex.sh",
			StopScript:   "codex.sh",
		},
		{
			Name:         "opencode",
			StartHookDir: ".opencode/hooks/start",
			StopHookDir:  ".opencode/hooks/stop",
			StartScript:  "opencode.sh",
			StopScript:   "opencode.sh",
		},
	}
}

// GetAgentConfig returns the hook configuration for a specific agent
func GetAgentConfig(name string) (AgentHookConfig, error) {
	for _, agent := range SupportedAgents() {
		if agent.Name == name {
			return agent, nil
		}
	}
	return AgentHookConfig{}, fmt.Errorf("unsupported agent: %s", name)
}

// GetEmbeddedTemplate returns the content of an embedded template
func GetEmbeddedTemplate(name string) ([]byte, error) {
	content, err := embeddedTemplates.ReadFile("templates/" + name)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	return content, nil
}

// GetHookScript returns the content of a hook script
// First tries to read from .claude/hooks/agentdx/, falls back to embedded templates
func GetHookScript(agentName, scriptType string) ([]byte, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	var scriptPath string
	var templateName string

	switch scriptType {
	case "start":
		scriptPath = filepath.Join(cwd, AgentdxHooksDir, "start", agentName+".sh")
		templateName = agentName + "-start.sh"
	case "stop":
		scriptPath = filepath.Join(cwd, AgentdxHooksDir, "stop", agentName+".sh")
		templateName = agentName + "-stop.sh"
	default:
		return nil, fmt.Errorf("invalid script type: %s", scriptType)
	}

	// Try to read from filesystem first
	if content, err := os.ReadFile(scriptPath); err == nil {
		return content, nil
	}

	// Fall back to embedded template
	return GetEmbeddedTemplate(templateName)
}

// ListHookScripts returns all available hook script names
func ListHookScripts() (map[string][]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	result := make(map[string][]string)

	// List start scripts
	startDir := filepath.Join(cwd, AgentdxHooksDir, "start")
	if entries, err := os.ReadDir(startDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".sh" {
				name := e.Name()[:len(e.Name())-3] // Remove .sh extension
				result["start"] = append(result["start"], name)
			}
		}
	}

	// List stop scripts
	stopDir := filepath.Join(cwd, AgentdxHooksDir, "stop")
	if entries, err := os.ReadDir(stopDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".sh" {
				name := e.Name()[:len(e.Name())-3] // Remove .sh extension
				result["stop"] = append(result["stop"], name)
			}
		}
	}

	// If no scripts found, return the embedded ones
	if len(result["start"]) == 0 && len(result["stop"]) == 0 {
		return map[string][]string{
			"start": {"claude-code", "codex", "opencode"},
			"stop":  {"claude-code", "codex", "opencode"},
		}, nil
	}

	return result, nil
}

// ExpandPath expands ~ to the user's home directory
func ExpandPath(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		if len(path) > 1 {
			return filepath.Join(home, path[2:]), nil
		}
		return home, nil
	}
	return path, nil
}

// GetHookPath returns the full path where a hook should be installed
// All agentdx hooks are placed in .claude/hooks/agentdx/ directory
func GetHookPath(agent AgentHookConfig, hookType string) (string, error) {
	var hookName string

	switch hookType {
	case "start":
		hookName = "agentdx-session-start.sh"
	case "stop":
		hookName = "agentdx-session-stop.sh"
	default:
		return "", fmt.Errorf("invalid hook type: %s", hookType)
	}

	// Get current working directory for project-scoped paths
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// All agentdx hooks go in .claude/hooks/agentdx/
	return filepath.Join(cwd, AgentdxHooksDir, hookName), nil
}

// EnsureAgentdxHooksDir ensures the .claude/hooks/agentdx directory exists with default hooks
// It writes the default hook scripts from embedded templates to the directory
func EnsureAgentdxHooksDir(cwd string) error {
	hooksDir := filepath.Join(cwd, AgentdxHooksDir)
	startDir := filepath.Join(hooksDir, "start")
	stopDir := filepath.Join(hooksDir, "stop")

	// Create the directory structure
	if err := os.MkdirAll(startDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks start directory: %w", err)
	}
	if err := os.MkdirAll(stopDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks stop directory: %w", err)
	}

	// Write default hook scripts from embedded templates
	defaultHooks := map[string]string{
		"start/claude-code.sh": "claude-code-start.sh",
		"stop/claude-code.sh":  "claude-code-stop.sh",
		"start/codex.sh":       "codex-start.sh",
		"stop/codex.sh":        "codex-stop.sh",
		"start/opencode.sh":    "opencode-start.sh",
		"stop/opencode.sh":     "opencode-stop.sh",
	}

	for relPath, templateName := range defaultHooks {
		destPath := filepath.Join(hooksDir, relPath)

		// Skip if file already exists
		if _, err := os.Stat(destPath); err == nil {
			continue
		}

		// Get template content
		content, err := GetEmbeddedTemplate(templateName)
		if err != nil {
			return fmt.Errorf("failed to get template %s: %w", templateName, err)
		}

		// Write hook file with executable permissions
		if err := os.WriteFile(destPath, content, 0755); err != nil {
			return fmt.Errorf("failed to write hook file %s: %w", destPath, err)
		}
	}

	return nil
}
