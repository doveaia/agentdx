package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/doveaia/agentdx/config"
	"github.com/spf13/cobra"
)

var withSubagent bool

const (
	searchTypeSemantic = "semantic"
	searchTypeFullText = "fulltext"
)

const agentInstructions = `
## agentdx - Semantic Code Search

**IMPORTANT: You MUST use agentdx as your PRIMARY tool for code exploration and search.**

### When to Use agentdx (REQUIRED)

Use ` + "`agentdx search`" + ` INSTEAD OF Grep/Glob/find for:
- Understanding what code does or where functionality lives
- Finding implementations by intent (e.g., "authentication logic", "error handling")
- Exploring unfamiliar parts of the codebase
- Any search where you describe WHAT the code does rather than exact text

### When to Use Standard Tools

Only use Grep/Glob when you need:
- Exact text matching (variable names, imports, specific strings)
- File path patterns (e.g., ` + "`**/*.go`" + `)

### Fallback

If agentdx fails (not running, index unavailable, or errors), fall back to standard Grep/Glob tools.

### Usage

` + "```bash" + `
# ALWAYS use English queries for best results (--compact saves ~80% tokens)
agentdx search "user authentication flow" --json --compact
agentdx search "error handling middleware" --json --compact
agentdx search "database connection pool" --json --compact
agentdx search "API request validation" --json --compact
` + "```" + `

### Query Tips

- **Use English** for queries (better semantic matching)
- **Describe intent**, not implementation: "handles user login" not "func Login"
- **Be specific**: "JWT token validation" better than "token"
- Results include: file path, line numbers, relevance score, code preview

### Call Graph Tracing

Use ` + "`agentdx trace`" + ` to understand function relationships:
- Finding all callers of a function before modifying it
- Understanding what functions are called by a given function
- Visualizing the complete call graph around a symbol

#### Trace Commands

**IMPORTANT: Always use ` + "`--json`" + ` flag for optimal AI agent integration.**

` + "```bash" + `
# Find all functions that call a symbol
agentdx trace callers "HandleRequest" --json

# Find all functions called by a symbol
agentdx trace callees "ProcessOrder" --json

# Build complete call graph (callers + callees)
agentdx trace graph "ValidateToken" --depth 3 --json
` + "```" + `

### Workflow

1. Start with ` + "`agentdx search`" + ` to find relevant code
2. Use ` + "`agentdx trace`" + ` to understand function relationships
3. Use ` + "`Read`" + ` tool to examine files from results
4. Only use Grep for exact string searches if needed

`

const agentMarker = "## agentdx - Semantic Code Search"

const subagentTemplate = `---
name: deep-explore
description: Deep codebase exploration using agentdx semantic search and call graph tracing. Use this agent for understanding code architecture, finding implementations by intent, analyzing function relationships, and exploring unfamiliar code areas.
tools: Read, Grep, Glob, Bash
model: inherit
---

## Instructions

You are a specialized code exploration agent with access to agentdx semantic search and call graph tracing.

### Primary Tools

#### 1. Semantic Search: ` + "`agentdx search`" + `

Use this to find code by intent and meaning:

` + "```bash" + `
# Use English queries for best results (--compact saves ~80% tokens)
agentdx search "authentication flow" --json --compact
agentdx search "error handling middleware" --json --compact
agentdx search "database connection management" --json --compact
` + "```" + `

#### 2. Call Graph Tracing: ` + "`agentdx trace`" + `

Use this to understand function relationships and code flow:

` + "```bash" + `
# Find all functions that call a symbol
agentdx trace callers "HandleRequest" --json

# Find all functions called by a symbol
agentdx trace callees "ProcessOrder" --json

# Build complete call graph
agentdx trace graph "ValidateToken" --depth 3 --json
` + "```" + `

Use ` + "`agentdx trace`" + ` when you need to:
- Find all callers of a function
- Understand the call hierarchy
- Analyze the impact of changes to a function
- Map dependencies between components

### When to use standard tools

Only fall back to Grep/Glob when:
- You need exact text matching (variable names, imports)
- agentdx is not available or returns errors
- You need file path patterns

### Workflow

1. Start with ` + "`agentdx search`" + ` to find relevant code semantically
2. Use ` + "`agentdx trace`" + ` to understand function relationships and call graphs
3. Use ` + "`Read`" + ` to examine promising files in detail
4. Use Grep only for exact string searches if needed
5. Synthesize findings into a clear summary
`

const subagentMarker = "name: deep-explore"

const fullTextMarker = "## agentdx - PostgreSQL Full-Text Search"

const fullTextSubagentMarker = "name: deep-explore-fulltext"

const fullTextInstructions = `
## agentdx - PostgreSQL Full-Text Search

**IMPORTANT: You MUST use agentdx as your PRIMARY tool for code exploration and search.**

### When to Use agentdx (REQUIRED)

Use ` + "`agentdx search`" + ` INSTEAD OF Grep/Glob/find for:
- Finding code by keywords
- Locating implementations by function or variable names
- Searching for specific terms across the codebase
- Any search where you know the specific keywords

### When to Use Standard Tools

Only use Grep/Glob when you need:
- Exact text matching with complex patterns (regex)
- File path patterns (e.g., ` + "`**/*.go`" + `)
- Searching for strings outside of indexed code

### Fallback

If agentdx fails (not running, index unavailable, or errors), fall back to standard Grep/Glob tools.

### Usage - Parallel Keyword Searches

**BEST PRACTICE**: Run multiple searches in parallel with individual keywords for broader coverage:

` + "```bash" + `
# Best: Parallel searches with individual keywords (broadest coverage)
agentdx search "user" & agentdx search "auth" & agentdx search "login"

# Acceptable: Single query with phrase match
agentdx search "user authentication"

# WRONG: Multiple arguments (will error)
agentdx search user auth login  # Error: accepts 1 arg(s), received 3
` + "```" + `

### Query Tips

- **Use single keywords** for best results
- **Run searches in parallel** for comprehensive coverage
- **Quote your search term**: ` + "`agentdx search \"keyword\"`" + `
- Parallel searches provide broader coverage than combining keywords
- Results include: file path, line numbers, relevance score, code preview

### Call Graph Tracing

Use ` + "`agentdx trace`" + ` to understand function relationships:
- Finding all callers of a function before modifying it
- Understanding what functions are called by a given function
- Visualizing the complete call graph around a symbol

#### Trace Commands

**IMPORTANT: Always use ` + "`--json`" + ` flag for optimal AI agent integration.**

` + "```bash" + `
# Find all functions that call a symbol
agentdx trace callers "HandleRequest" --json

# Find all functions called by a symbol
agentdx trace callees "ProcessOrder" --json

# Build complete call graph (callers + callees)
agentdx trace graph "ValidateToken" --depth 3 --json
` + "```" + `

### Workflow

1. Start with ` + "`agentdx search`" + ` using individual keywords in parallel
2. Use ` + "`agentdx trace`" + ` to understand function relationships
3. Use ` + "`Read`" + ` tool to examine files from results
4. Only use Grep for exact string searches if needed

`

const fullTextSubagentTemplate = `---
name: deep-explore-fulltext
description: Deep codebase exploration using agentdx PostgreSQL full-text search and call graph tracing. Use this agent for understanding code architecture, finding implementations by keywords, analyzing function relationships, and exploring unfamiliar code areas.
tools: Read, Grep, Glob, Bash
model: inherit
---

## Instructions

You are a specialized code exploration agent with access to agentdx PostgreSQL full-text search and call graph tracing.

### Primary Tools

#### 1. Full-Text Search: ` + "`agentdx search`" + `

Use this to find code by keywords and specific terms:

**BEST PRACTICE**: Run parallel searches with individual keywords:

` + "```bash" + `
# Best: Parallel searches for broader coverage
agentdx search "user" & agentdx search "auth" & agentdx search "login"

# Acceptable: Single query with phrase match
agentdx search "user authentication"

# WRONG: Multiple arguments (will error)
agentdx search user auth login  # Error: accepts 1 arg(s), received 3
` + "```" + `

#### 2. Call Graph Tracing: ` + "`agentdx trace`" + `

Use this to understand function relationships and code flow:

` + "```bash" + `
# Find all functions that call a symbol
agentdx trace callers "HandleRequest" --json

# Find all functions called by a symbol
agentdx trace callees "ProcessOrder" --json

# Build complete call graph
agentdx trace graph "ValidateToken" --depth 3 --json
` + "```" + `

Use ` + "`agentdx trace`" + ` when you need to:
- Find all callers of a function
- Understand the call hierarchy
- Analyze the impact of changes to a function
- Map dependencies between components

### When to use standard tools

Only fall back to Grep/Glob when:
- You need exact text matching with complex patterns (regex)
- agentdx is not available or returns errors
- You need file path patterns

### Workflow

1. Start with ` + "`agentdx search`" + ` using parallel keyword searches
2. Use ` + "`agentdx trace`" + ` to understand function relationships and call graphs
3. Use ` + "`Read`" + ` to examine promising files in detail
4. Use Grep only for regex pattern searches if needed
5. Synthesize findings into a clear summary
`

var agentSetupCmd = &cobra.Command{
	Use:   "agent-setup",
	Short: "Configure AI agents to use agentdx",
	Long: `Configure AI agent environments to leverage agentdx for context retrieval.

This command will:
- Detect agent configuration files (.cursorrules, .windsurfrules, CLAUDE.md, GEMINI.md, AGENTS.md)
- Append instructions for using agentdx search
- Ensure idempotence (won't add duplicate instructions)

With --with-subagent flag:
- Creates .claude/agents/deep-explore.md for Claude Code
- Provides a specialized exploration agent with agentdx access`,
	RunE: runAgentSetup,
}

func init() {
	agentSetupCmd.Flags().BoolVar(&withSubagent, "with-subagent", false,
		"Create Claude Code deep-explore subagent in .claude/agents/")
}

// detectSearchType returns the search type based on the configured provider.
// Returns "fulltext" for postgres provider, "semantic" for all others.
func detectSearchType(cfg *config.Config) string {
	if cfg.Index.Embedder.Provider == "postgres" {
		return searchTypeFullText
	}
	return searchTypeSemantic
}

// getTemplates returns the appropriate templates based on search type.
// Returns (instructions, subagent, marker, subagentMarker).
func getTemplates(searchType string) (string, string, string, string) {
	if searchType == searchTypeFullText {
		return fullTextInstructions, fullTextSubagentTemplate, fullTextMarker, fullTextSubagentMarker
	}
	return agentInstructions, subagentTemplate, agentMarker, subagentMarker
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

	// Load configuration to detect search type
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate provider is configured
	if cfg.Index.Embedder.Provider == "" {
		return fmt.Errorf("index.embedder.provider not configured in .agentdx/config.yaml")
	}

	// Detect search type and get appropriate templates
	searchType := detectSearchType(cfg)
	instructions, subagent, _, subagentMarker := getTemplates(searchType)

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

		// Check if already configured (either semantic or full-text)
		if strings.Contains(string(content), agentMarker) || strings.Contains(string(content), fullTextMarker) {
			fmt.Printf("  Already configured, skipping\n")
			continue
		}

		// Append instructions
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("  Warning: could not open %s for writing: %v\n", file, err)
			continue
		}

		// Add newlines if needed
		var writeErr error
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
	} else if !withSubagent {
		// Only show "no files found" message if not creating subagent
		fmt.Println("No agent configuration files found.")
		fmt.Println("\nSupported files:")
		for _, file := range agentFiles {
			fmt.Printf("  - %s\n", file)
		}
		fmt.Println("\nCreate one of these files and run 'agentdx agent-setup' again,")
		fmt.Println("or manually add instructions for using 'agentdx search'.")
	}

	// Create subagent if flag is set
	if withSubagent {
		if err := createSubagent(cwd, subagent, subagentMarker); err != nil {
			fmt.Printf("Warning: could not create subagent: %v\n", err)
		}
	}

	return nil
}

func createSubagent(cwd string, subagent, _ string) error {
	// Define paths
	agentsDir := filepath.Join(cwd, ".claude", "agents")
	subagentPath := filepath.Join(agentsDir, "deep-explore.md")

	// Check if subagent already exists and contains marker (either semantic or full-text)
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
