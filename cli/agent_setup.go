package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/hooks"
	"github.com/spf13/cobra"
)

// Embedded template files
var (
	//go:embed templates/agentdx-fallback.sh
	fallbackHook string
)

// Marker strings for detecting existing configuration
const (
	fullTextMarker         = "## agentdx - Full-Text Search"
	fullTextSubagentMarker = "name: deep-explore"
	ruleMarker             = "# AgentDX Rule"
	hookMarker             = "PostToolUse hook for Bash tool"
)

// FTS-only templates
const (
	fullTextInstructions = `
## agentdx - PostgreSQL Full-Text Search

This project uses agentdx for fast full-text code search optimized for AI agents.

### Quick Reference

agentdx search "pattern" --json --compact
agentdx files "*.go" --json --compact
agentdx trace callers "FunctionName" --json
agentdx trace callees "FunctionName" --json

### Search Tips

- Use exact code identifiers for best results
- FTS works well with symbol names, function names, and string literals
- Combine with trace commands for deeper code understanding
- Add --json --compact for AI-friendly output

agentdx uses PostgreSQL Full Text Search with structural boosting for fast, relevant results.
`

	fullTextSubagent = `name: deep-explore
description: Full-text code search specialist using agentdx

You are a code exploration specialist with access to agentdx's PostgreSQL Full-Text Search index.

### First Step: Start Session

Before any search or trace command, ensure the agentdx session is running:

` + "```" + `bash
agentdx session start
` + "```" + `

This command is idempotent - safe to run multiple times.

### Search Strategy

1. **Use exact identifiers**: Function names, variable names, type names search best
2. **Combine with trace**: Use trace commands to understand call relationships
3. **Leverage file patterns**: Narrow scope by file type or directory
4. **Use parallel searches**: For multiple terms, run separate searches in parallel

### Available Commands

agentdx search "func Login" --json --compact
agentdx files "**/*.go" --json --compact
agentdx trace callers "FunctionName" --json
agentdx trace callees "FunctionName" --json
agentdx trace graph "SymbolName" --depth 2 --json

### IMPORTANT: No Regex OR Patterns

agentdx does NOT support regex patterns. For multiple terms, use parallel searches:

CORRECT: Run parallel searches
  agentdx search "Login" --json --compact &
  agentdx search "Auth" --json --compact &
  agentdx search "Session" --json --compact

WRONG: Regex OR syntax (will not work)
  agentdx search "Login\|Auth\|Session"

### Key Difference

This mode uses **PostgreSQL Full Text Search** optimized for code:
- Fast text-based search on indexed code
- Structural boosting for relevant results
- No vector embeddings required
- Lower token usage for AI interactions
`

	fullTextRule = `# AgentDX Rule
- When user asks about code structure: Use agentdx trace commands
- When searching for specific functions: Use exact function names with agentdx search
- Always use --json --compact for AI-friendly output
- Combine search + trace for complete understanding
`
)

var agentSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure AI agents to use agentdx",
	Long: `Configure AI agent environments to leverage agentdx for context retrieval.

This command will:
- Detect agent configuration files (.cursorrules, .windsurfrules, CLAUDE.md, GEMINI.md, AGENTS.md)
- Append instructions for using agentdx search
- Create .claude/rules/agentdx.md for Claude Code rules
- Create .claude/hooks/agentdx-fallback.sh for empty result handling
- Create/update .claude/settings.json with agentdx hooks
- Create .claude/agents/deep-explore.md for Claude Code
- Install session management hooks for automatic daemon start/stop
- Ensure idempotence (won't add duplicate instructions)

All configurations are project-scoped (installed in current directory).`,
	RunE: runAgentSetup,
}

// getTemplates returns the FTS search templates.
// Returns (instructions, subagent, marker, subagentMarker, rule).
func getTemplates() (string, string, string, string, string) {
	return fullTextInstructions, fullTextSubagent, fullTextMarker, fullTextSubagentMarker, fullTextRule
}

func runAgentSetup(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root (walks up parent directories to find .agentdx/config.yaml)
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("agentdx configuration not found. Run 'agentdx init' first")
	}

	// Load configuration
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	_ = cfg // Config is loaded to verify project is initialized

	// Always use FTS search
	instructions, subagent, _, subagentMarker, rule := getTemplates()

	agentFiles := []string{
		".cursorrules",
		".windsurfrules",
		"CLAUDE.md",
		".claude/settings.md",
		"GEMINI.md",
		"AGENTS.md",
	}

	found := false
	modified := 0

	for _, file := range agentFiles {
		path := filepath.Join(cwd, file)

		// Check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		found = true
		fmt.Printf("Found: %s\n", file)

		// Read existing content
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("  Warning: could not read %s: %v\n", file, err)
			continue
		}

		// Check if already configured (full-text marker)
		if strings.Contains(string(content), fullTextMarker) {
			fmt.Printf("  Already configured, skipping\n")
			continue
		}

		// Prepend instructions for CLAUDE.md, append for others
		var writeErr error
		if file == "CLAUDE.md" {
			// Prepend: instructions first, then existing content
			var newContent strings.Builder
			newContent.WriteString(instructions)
			newContent.WriteString("\n")
			if len(content) > 0 {
				newContent.Write(content)
			}
			writeErr = os.WriteFile(path, []byte(newContent.String()), 0644)
		} else {
			// Append instructions
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("  Warning: could not open %s for writing: %v\n", file, err)
				continue
			}

			// Add newlines if needed
			if len(content) > 0 && content[len(content)-1] != '\n' {
				_, writeErr = f.WriteString("\n")
			}
			if writeErr == nil {
				_, writeErr = f.WriteString("\n")
			}
			if writeErr == nil {
				_, writeErr = f.WriteString(instructions)
			}
			f.Close()
		}

		if writeErr != nil {
			fmt.Printf("  Warning: failed to write to %s: %v\n", file, writeErr)
			continue
		}

		fmt.Printf("  Added agentdx instructions\n")
		modified++
	}

	if modified > 0 {
		fmt.Printf("\nUpdated %d file(s).\n", modified)
	} else if found {
		fmt.Println("\nAll files already configured.")
	} else {
		fmt.Println("No agent configuration files found.")
		fmt.Println("\nSupported files:")
		for _, file := range agentFiles {
			fmt.Printf("  - %s\n", file)
		}
		fmt.Println("\nCreate one of these files and run 'agentdx setup' again,")
		fmt.Println("or manually add instructions for using 'agentdx search'.")
	}

	// Create Claude Code subagent (always)
	if err := createSubagent(cwd, subagent, subagentMarker); err != nil {
		fmt.Printf("Warning: could not create subagent: %v\n", err)
	}

	// Create Claude Code rule (always)
	if err := createRule(cwd, rule); err != nil {
		fmt.Printf("Warning: could not create rule: %v\n", err)
	}

	// Create Claude Code hook for fallback behavior (always)
	if err := createHook(cwd); err != nil {
		fmt.Printf("Warning: could not create hook: %v\n", err)
	}

	// Create or update Claude Code settings.json with agentdx hooks (always)
	if err := createSettings(cwd); err != nil {
		fmt.Printf("Warning: could not create/update settings: %v\n", err)
	}

	// Install session management hooks (always)
	if err := installSessionHooks(cwd); err != nil {
		fmt.Printf("Warning: could not install session hooks: %v\n", err)
	}

	return nil
}

func createSubagent(cwd string, subagent, _ string) error {
	// Define paths
	agentsDir := filepath.Join(cwd, ".claude", "agents")
	subagentPath := filepath.Join(agentsDir, "deep-explore.md")

	// Check if subagent already exists and contains marker
	if content, err := os.ReadFile(subagentPath); err == nil {
		if strings.Contains(string(content), "name: deep-explore") {
			fmt.Printf("Subagent already exists: %s\n", subagentPath)
			return nil
		}
	}

	// Create .claude/agents/ directory if it doesn't exist
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Write the subagent file
	if err := os.WriteFile(subagentPath, []byte(subagent), 0600); err != nil {
		return fmt.Errorf("failed to write subagent file: %w", err)
	}

	fmt.Printf("Created subagent: %s\n", subagentPath)
	return nil
}

func createRule(cwd string, rule string) error {
	// Define paths
	rulesDir := filepath.Join(cwd, ".claude", "rules")
	rulePath := filepath.Join(rulesDir, "agentdx.md")

	// Check if rule already exists and contains marker
	if content, err := os.ReadFile(rulePath); err == nil {
		if strings.Contains(string(content), ruleMarker) {
			fmt.Printf("Rule already exists: %s\n", rulePath)
			return nil
		}
	}

	// Create .claude/rules/ directory if it doesn't exist
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create rules directory: %w", err)
	}

	// Write the rule file
	if err := os.WriteFile(rulePath, []byte(rule), 0600); err != nil {
		return fmt.Errorf("failed to write rule file: %w", err)
	}

	fmt.Printf("Created rule: %s\n", rulePath)
	return nil
}

func createHook(cwd string) error {
	// Define paths - all agentdx hooks go in .claude/hooks/agentdx/
	hooksDir := filepath.Join(cwd, ".claude", "hooks", "agentdx")
	hookPath := filepath.Join(hooksDir, "agentdx-fallback.sh")

	// Check if hook already exists and contains marker
	if content, err := os.ReadFile(hookPath); err == nil {
		if strings.Contains(string(content), hookMarker) {
			fmt.Printf("Hook already exists: %s\n", hookPath)
			return nil
		}
	}

	// Create .claude/hooks/ directory if it doesn't exist
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Write the hook file with executable permissions
	if err := os.WriteFile(hookPath, []byte(fallbackHook), 0755); err != nil {
		return fmt.Errorf("failed to write hook file: %w", err)
	}

	fmt.Printf("Created hook: %s\n", hookPath)
	return nil
}

func createSettings(cwd string) error {
	// Define paths
	claudeDir := filepath.Join(cwd, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Check if settings.json exists
	existingData, err := os.ReadFile(settingsPath)
	if err == nil {
		// File exists - check if agentdx hooks are already present
		settings, parseErr := parseSettings(existingData)
		if parseErr != nil {
			return fmt.Errorf("failed to parse existing settings.json: %w", parseErr)
		}

		// Check if agentdx hooks are already present
		if hasAgentdxHooks(settings) {
			fmt.Printf("Settings already configured: %s\n", settingsPath)
			return nil
		}

		// Create backup before modifying
		backupPath := filepath.Join(claudeDir, "settings.backup.json")
		if writeErr := os.WriteFile(backupPath, existingData, 0644); writeErr != nil {
			return fmt.Errorf("failed to create backup: %w", writeErr)
		}
		fmt.Printf("Created backup: %s\n", backupPath)

		// Merge agentdx hooks into existing settings
		merged := mergeAgentdxHooks(settings)

		// Serialize and validate
		output, serErr := serializeSettings(merged)
		if serErr != nil {
			return fmt.Errorf("failed to serialize merged settings: %w", serErr)
		}

		if valErr := validateSettingsJSON(output); valErr != nil {
			return fmt.Errorf("merged settings JSON is invalid: %w", valErr)
		}

		// Write back
		if writeErr := os.WriteFile(settingsPath, output, 0644); writeErr != nil {
			return fmt.Errorf("failed to write settings file: %w", writeErr)
		}

		fmt.Printf("Updated settings: %s\n", settingsPath)
		return nil
	}

	// File doesn't exist - create it with default agentdx settings
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read settings file: %w", err)
	}

	// Create .claude/ directory if it doesn't exist
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Create default settings with agentdx hooks
	settings := createDefaultSettings()

	// Serialize and validate
	output, err := serializeSettings(settings)
	if err != nil {
		return fmt.Errorf("failed to serialize default settings: %w", err)
	}

	if valErr := validateSettingsJSON(output); valErr != nil {
		return fmt.Errorf("default settings JSON is invalid: %w", valErr)
	}

	// Write the file
	if err := os.WriteFile(settingsPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	fmt.Printf("Created settings: %s\n", settingsPath)
	return nil
}

// installSessionHooks installs session management hooks for the current project
// It copies hooks from .claude/hooks/agentdx/ to the agent's hook directories
func installSessionHooks(cwd string) error {
	// First ensure the agentdx hooks directory exists
	if err := hooks.EnsureAgentdxHooksDir(cwd); err != nil {
		return fmt.Errorf("failed to ensure hooks directory: %w", err)
	}

	// Get the agent configuration for claude-code
	// Since we're setting up .claude/, we always install claude-code hooks
	config, err := hooks.GetAgentConfig("claude-code")
	if err != nil {
		return fmt.Errorf("failed to get agent config: %w", err)
	}

	// Check if session hooks are already installed (idempotency)
	startPath, err := hooks.GetHookPath(config, "start")
	if err != nil {
		return fmt.Errorf("failed to get start hook path: %w", err)
	}

	if hookFileContains(startPath, "agentdx-session") {
		fmt.Println("Session hooks already installed")
		return nil
	}

	// Install start hook - copy from agentdx hooks directory
	if err := installSessionHookFile(config.Name, "start", startPath); err != nil {
		return fmt.Errorf("failed to install start hook: %w", err)
	}
	fmt.Printf("Created hook: %s\n", startPath)

	// Install stop hook
	stopPath, err := hooks.GetHookPath(config, "stop")
	if err != nil {
		return fmt.Errorf("failed to get stop hook path: %w", err)
	}

	if err := installSessionHookFile(config.Name, "stop", stopPath); err != nil {
		return fmt.Errorf("failed to install stop hook: %w", err)
	}
	fmt.Printf("Created hook: %s\n", stopPath)

	return nil
}

// hookFileContains checks if a hook file contains a specific marker string
func hookFileContains(path, marker string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), marker)
}

// installSessionHookFile copies a hook script from the agentdx hooks directory to the destination path
func installSessionHookFile(agentName, hookType, destPath string) error {
	// Get script content from the agentdx hooks directory
	content, err := hooks.GetHookScript(agentName, hookType)
	if err != nil {
		return fmt.Errorf("failed to get hook script: %w", err)
	}

	// Create directory if needed
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create hook directory: %w", err)
	}

	// Write hook file with executable permissions
	if err := os.WriteFile(destPath, content, 0755); err != nil {
		return fmt.Errorf("failed to write hook file: %w", err)
	}

	return nil
}
