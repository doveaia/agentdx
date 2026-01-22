package cli

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/agents/*
var agentTemplates embed.FS

// AgentConfig represents a coding agent configuration
type AgentConfig struct {
	Name        string
	Description string
	Files       []AgentFile
	Directories []string
}

// AgentFile represents a file to be generated for an agent
type AgentFile struct {
	TemplateName string // Name in embedded templates
	DestPath     string // Destination path relative to project root
	Description  string // Human-readable description
}

// SupportedAgentConfigs returns all supported coding agent configurations
func SupportedAgentConfigs() []AgentConfig {
	return []AgentConfig{
		{
			Name:        "Claude Code",
			Description: "Anthropic's CLI coding assistant",
			Directories: []string{
				".claude",
				".claude/rules",
				".claude/agents",
				".claude/hooks/agentdx",
				".claude/hooks/agentdx/start",
				".claude/hooks/agentdx/stop",
			},
			Files: []AgentFile{
				{TemplateName: "CLAUDE.md", DestPath: "CLAUDE.md", Description: "Main instructions"},
				{TemplateName: "claude_settings.json", DestPath: ".claude/settings.json", Description: "Hook configuration"},
				{TemplateName: "claude_rules_agentdx.md", DestPath: ".claude/rules/agentdx.md", Description: "Search rules"},
				{TemplateName: "claude_agents_deep-explore.md", DestPath: ".claude/agents/deep-explore.md", Description: "Deep explore subagent"},
			},
		},
		{
			Name:        "Cursor",
			Description: "Cursor AI editor",
			Directories: []string{
				".cursor",
				".cursor/rules",
			},
			Files: []AgentFile{
				{TemplateName: "cursorrules", DestPath: ".cursorrules", Description: "Legacy rules (deprecated)"},
				{TemplateName: "cursor_rules_agentdx.mdc", DestPath: ".cursor/rules/agentdx.mdc", Description: "MDC rules (recommended)"},
			},
		},
		{
			Name:        "Windsurf",
			Description: "Codeium Windsurf editor",
			Directories: []string{
				".windsurf",
				".windsurf/rules",
			},
			Files: []AgentFile{
				{TemplateName: "windsurfrules", DestPath: ".windsurfrules", Description: "Main rules"},
				{TemplateName: "windsurf_rules_agentdx.md", DestPath: ".windsurf/rules/agentdx.md", Description: "Workspace rules"},
			},
		},
		{
			Name:        "Codex CLI / GitHub Copilot",
			Description: "OpenAI Codex CLI and GitHub Copilot",
			Directories: []string{
				".github",
				".github/instructions",
			},
			Files: []AgentFile{
				{TemplateName: "AGENTS.md", DestPath: "AGENTS.md", Description: "Agent instructions"},
				{TemplateName: "copilot-instructions.md", DestPath: ".github/copilot-instructions.md", Description: "Copilot instructions"},
				{TemplateName: "github_instructions_agentdx.md", DestPath: ".github/instructions/agentdx.instructions.md", Description: "Additional instructions"},
			},
		},
		{
			Name:        "Gemini",
			Description: "Google Gemini CLI and Code Assist",
			Directories: []string{
				".gemini",
			},
			Files: []AgentFile{
				{TemplateName: "GEMINI.md", DestPath: "GEMINI.md", Description: "Main instructions"},
			},
		},
	}
}

// GenerateAgentConfigs creates configuration files for all supported coding agents
func GenerateAgentConfigs(cwd string) error {
	fmt.Println("\nGenerating coding agent configurations...")

	agents := SupportedAgentConfigs()
	totalFiles := 0
	createdFiles := 0
	skippedFiles := 0

	for _, agent := range agents {
		fmt.Printf("\n%s:\n", agent.Name)

		// Create directories
		for _, dir := range agent.Directories {
			dirPath := filepath.Join(cwd, dir)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}

		// Create files
		for _, file := range agent.Files {
			totalFiles++
			destPath := filepath.Join(cwd, file.DestPath)

			// Check if file already exists
			if _, err := os.Stat(destPath); err == nil {
				// File exists - check if it already has agentdx content
				content, readErr := os.ReadFile(destPath)
				if readErr == nil && strings.Contains(string(content), "agentdx") {
					fmt.Printf("  [skip] %s (already configured)\n", file.DestPath)
					skippedFiles++
					continue
				}

				// File exists but doesn't have agentdx - we'll update it
				if err := updateAgentFile(destPath, file.TemplateName); err != nil {
					fmt.Printf("  [warn] %s: %v\n", file.DestPath, err)
					continue
				}
				fmt.Printf("  [update] %s\n", file.DestPath)
				createdFiles++
				continue
			}

			// File doesn't exist - create it
			if err := createAgentFile(destPath, file.TemplateName); err != nil {
				fmt.Printf("  [warn] %s: %v\n", file.DestPath, err)
				continue
			}
			fmt.Printf("  [create] %s\n", file.DestPath)
			createdFiles++
		}
	}

	// Install Claude Code session hooks
	if err := installClaudeSessionHooks(cwd); err != nil {
		fmt.Printf("\n[warn] Could not install session hooks: %v\n", err)
	}

	fmt.Printf("\nAgent configurations: %d created, %d skipped, %d total\n", createdFiles, skippedFiles, totalFiles)
	return nil
}

// createAgentFile creates a new agent configuration file from a template
func createAgentFile(destPath, templateName string) error {
	content, err := agentTemplates.ReadFile("templates/agents/" + templateName)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Determine file permissions based on extension
	perm := os.FileMode(0644)
	if strings.HasSuffix(destPath, ".sh") {
		perm = 0755
	}

	return os.WriteFile(destPath, content, perm)
}

// updateAgentFile appends or prepends agentdx content to an existing file
func updateAgentFile(destPath, templateName string) error {
	// Read existing content
	existing, err := os.ReadFile(destPath)
	if err != nil {
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	// Get template content
	template, err := agentTemplates.ReadFile("templates/agents/" + templateName)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// For markdown files that are primary docs (CLAUDE.md, AGENTS.md, GEMINI.md),
	// prepend the agentdx instructions
	var newContent []byte
	baseName := filepath.Base(destPath)
	if baseName == "CLAUDE.md" || baseName == "AGENTS.md" || baseName == "GEMINI.md" {
		// Prepend: template first, then existing
		newContent = append(template, '\n')
		newContent = append(newContent, existing...)
	} else {
		// Append: existing first, then template
		newContent = existing
		if len(newContent) > 0 && newContent[len(newContent)-1] != '\n' {
			newContent = append(newContent, '\n')
		}
		newContent = append(newContent, '\n')
		newContent = append(newContent, template...)
	}

	return os.WriteFile(destPath, newContent, 0644)
}

// installClaudeSessionHooks installs the session management hooks for Claude Code
func installClaudeSessionHooks(cwd string) error {
	// Define hook paths
	startHookDir := filepath.Join(cwd, ".claude", "hooks", "agentdx", "start")
	stopHookDir := filepath.Join(cwd, ".claude", "hooks", "agentdx", "stop")

	// Create directories
	if err := os.MkdirAll(startHookDir, 0755); err != nil {
		return fmt.Errorf("failed to create start hook dir: %w", err)
	}
	if err := os.MkdirAll(stopHookDir, 0755); err != nil {
		return fmt.Errorf("failed to create stop hook dir: %w", err)
	}

	// Write start hook
	startHook := `#!/bin/sh
# agentdx session hook - starts watch daemon when coding agent session begins
# Installed by: agentdx init

# Only run if this is an agentdx-initialized project
if [ ! -f ".agentdx/config.yaml" ]; then
    exit 0
fi

# Start the session daemon (idempotent - does nothing if already running)
agentdx session start --quiet 2>/dev/null || true

# Always exit 0 to not block the coding agent
exit 0
`

	startHookPath := filepath.Join(startHookDir, "claude-code.sh")
	if err := os.WriteFile(startHookPath, []byte(startHook), 0755); err != nil {
		return fmt.Errorf("failed to write start hook: %w", err)
	}

	// Write stop hook
	stopHook := `#!/bin/sh
# agentdx session hook - stops watch daemon when coding agent session ends
# Installed by: agentdx init

# Only run if there's a session PID file
if [ ! -f ".agentdx/session.pid" ]; then
    exit 0
fi

# Stop the session daemon
agentdx session stop --quiet 2>/dev/null || true

# Always exit 0 to not block the coding agent
exit 0
`

	stopHookPath := filepath.Join(stopHookDir, "claude-code.sh")
	if err := os.WriteFile(stopHookPath, []byte(stopHook), 0755); err != nil {
		return fmt.Errorf("failed to write stop hook: %w", err)
	}

	fmt.Println("\nInstalled Claude Code session hooks")
	return nil
}
