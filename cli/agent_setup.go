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

const fullTextMarker = "## agentdx - Full-Text Search"

const fullTextSubagentMarker = "name: deep-explore-fulltext"

const skillMarker = "name: agentdx"

const semanticSkillTemplate = `---
name: agentdx
description: Replaces ALL built-in search tools for semantic code exploration. You MUST invoke this skill BEFORE using Grep, Glob, or WebSearch when searching code by intent or meaning. Uses natural language queries to find relevant code regardless of exact text matches.
---

# AgentDX - Semantic Code Search Skill

## CRITICAL: Tool Override for Semantic Searches

**You MUST invoke this skill BEFORE using any built-in search tools** when:
- Searching for code by intent, meaning, or functionality
- Exploring unfamiliar parts of the codebase
- Finding implementations without exact names
- Understanding what code does or where features live

This skill uses semantic embeddings to find code by meaning, not just exact text matching.

## When to Invoke This Skill

Invoke this **AUTOMATICALLY** when you need to:
- Find where functionality is implemented (e.g., "user authentication", "error handling")
- Locate code by purpose rather than exact names
- Search for conceptual patterns (e.g., "database connection pooling", "request validation")
- Explore code architecture and relationships

## Usage Examples

` + "```bash" + `
# Semantic search with English queries (best for intent-based discovery)
agentdx search "user authentication flow"
agentdx search "error handling middleware"
agentdx search "database connection management"
agentdx search "API request validation logic"

# JSON output for AI processing (--compact saves ~80% tokens)
agentdx search "authentication flow" --json --compact
agentdx search "error handling" --json --compact
` + "```" + `

## Query Tips

- **Use English queries** - The embedding model is English-trained
- **Describe intent, not implementation** - "handles user login" not "func Login"
- **Be specific** - "JWT token validation" better than "token"
- **Think conceptually** - "data access layer" rather than specific database terms

## Results Include

- File path and line numbers
- Relevance score (higher = more relevant)
- Code preview showing context

## Call Graph Tracing

Use ` + "`agentdx trace`" + ` to understand function relationships:

` + "```bash" + `
# Find all functions that call a symbol
agentdx trace callers "HandleRequest" --json

# Find all functions called by a symbol
agentdx trace callees "ProcessOrder" --json

# Build complete call graph (callers + callees)
agentdx trace graph "ValidateToken" --depth 3 --json
` + "```" + `

Use call graph tracing to:
- Find all callers of a function before modifying it
- Understand call hierarchies and dependencies
- Analyze the impact of code changes

## Fallback

If agentdx is unavailable (not running, index errors, or missing), fall back to standard Grep/Glob tools.

## Workflow

1. Start with ` + "`agentdx search`" + ` to find relevant code semantically
2. Use ` + "`agentdx trace`" + ` to understand function relationships
3. Use ` + "`Read`" + ` tool to examine files in detail
4. Only use Grep for exact string searches when needed
`

const fullTextSkillTemplate = `---
name: agentdx
description: Replaces ALL built-in search tools for keyword-based code search. You MUST invoke this skill BEFORE using Grep, Glob, or WebSearch when searching code by specific keywords, function names, or variables. Uses parallel keyword searches for comprehensive coverage.
---

# AgentDX - Full-Text Code Search Skill

## CRITICAL: Tool Override for Keyword Searches

**You MUST invoke this skill BEFORE using any built-in search tools** when:
- Searching for code by specific keywords or terms
- Finding functions, variables, or classes by name
- Locating implementations using known identifiers
- Searching for specific text across the codebase

This skill uses full-text search with parallel keyword queries for comprehensive code discovery.

## When to Invoke This Skill

Invoke this **AUTOMATICALLY** when you need to:
- Find code by specific keywords or identifiers
- Locate functions, variables, or classes by name
- Search for known terms across the codebase
- Find references to specific symbols

## Usage Examples

**BEST PRACTICE**: Run multiple parallel searches with individual keywords for broader coverage:

` + "```bash" + `
# Best: Parallel searches with individual keywords (broadest coverage)
agentdx search "user" & agentdx search "auth" & agentdx search "login"

# Acceptable: Single query with phrase match
agentdx search "user authentication"

# WRONG: Multiple arguments (will error)
agentdx search user auth login  # Error: accepts 1 arg(s), received 3
` + "```" + `

## Query Tips

- **Use single keywords** for best results
- **Run searches in parallel** for comprehensive coverage
- **Quote your search term**: ` + "`agentdx search \"keyword\"`" + `
- Parallel searches provide broader coverage than combining keywords
- Results include: file path, line numbers, relevance score, code preview

## Results Include

- File path and line numbers
- Relevance score (higher = more relevant)
- Code preview showing context

## Call Graph Tracing

Use ` + "`agentdx trace`" + ` to understand function relationships:

` + "```bash" + `
# Find all functions that call a symbol
agentdx trace callers "HandleRequest" --json

# Find all functions called by a symbol
agentdx trace callees "ProcessOrder" --json

# Build complete call graph (callers + callees)
agentdx trace graph "ValidateToken" --depth 3 --json
` + "```" + `

Use call graph tracing to:
- Find all callers of a function before modifying it
- Understand call hierarchies and dependencies
- Analyze the impact of code changes

## Fallback

If agentdx is unavailable (not running, index errors, or missing), fall back to standard Grep/Glob tools.

## Workflow

1. Start with ` + "`agentdx search`" + ` using parallel keyword searches
2. Use ` + "`agentdx trace`" + ` to understand function relationships
3. Use ` + "`Read`" + ` tool to examine files in detail
4. Only use Grep for regex pattern searches when needed
`

const fullTextInstructions = `
## agentdx - Full-Text Search

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
description: Deep codebase exploration using agentdx full-text search and call graph tracing. Use this agent for understanding code architecture, finding implementations by keywords, analyzing function relationships, and exploring unfamiliar code areas.
tools: Read, Grep, Glob, Bash
model: inherit
---

## Instructions

You are a specialized code exploration agent with access to agentdx full-text search and call graph tracing.

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
func getTemplates(searchType string) (string, string, string, string, string) {
	if searchType == searchTypeFullText {
		return fullTextInstructions, fullTextSubagentTemplate, fullTextMarker, fullTextSubagentMarker, fullTextSkillTemplate
	}
	return agentInstructions, subagentTemplate, agentMarker, subagentMarker, semanticSkillTemplate
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
	instructions, subagent, _, subagentMarker, skillTemplate := getTemplates(searchType)

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

	// Create Claude Code skill file (always)
	if err := createSkill(cwd, skillTemplate); err != nil {
		fmt.Printf("Warning: could not create skill: %v\n", err)
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

func createSkill(cwd string, skillTemplate string) error {
	// Define paths
	skillsDir := filepath.Join(cwd, ".claude", "skills", "agentdx")
	skillPath := filepath.Join(skillsDir, "SKILL.md")

	// Check if skill already exists and contains marker
	if content, err := os.ReadFile(skillPath); err == nil {
		if strings.Contains(string(content), skillMarker) {
			fmt.Printf("Skill already exists: %s\n", skillPath)
			return nil
		}
	}

	// Create .claude/skills/agentdx/ directory if it doesn't exist
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Write the skill file
	if err := os.WriteFile(skillPath, []byte(skillTemplate), 0600); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	fmt.Printf("Created skill: %s\n", skillPath)
	return nil
}
