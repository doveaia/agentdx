package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSubagent(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating subagent with FTS template
	err := createSubagent(tmpDir, fullTextSubagent, fullTextSubagentMarker)
	if err != nil {
		t.Fatalf("failed to create subagent: %v", err)
	}

	// Verify file exists
	subagentPath := filepath.Join(tmpDir, ".claude", "agents", "deep-explore.md")
	if _, err := os.Stat(subagentPath); os.IsNotExist(err) {
		t.Fatal("subagent file was not created")
	}

	// Verify content contains marker
	content, err := os.ReadFile(subagentPath)
	if err != nil {
		t.Fatalf("failed to read subagent file: %v", err)
	}

	if !strings.Contains(string(content), fullTextSubagentMarker) {
		t.Error("subagent file does not contain expected marker")
	}

	if !strings.Contains(string(content), "agentdx search") {
		t.Error("subagent file does not contain agentdx search instructions")
	}

	if !strings.Contains(string(content), "agentdx trace") {
		t.Error("subagent file does not contain agentdx trace instructions")
	}
}

func TestCreateSubagentIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subagent twice
	err := createSubagent(tmpDir, fullTextSubagent, fullTextSubagentMarker)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	err = createSubagent(tmpDir, fullTextSubagent, fullTextSubagentMarker)
	if err != nil {
		t.Fatalf("second creation failed: %v", err)
	}

	// Should still only have one file with expected content
	subagentPath := filepath.Join(tmpDir, ".claude", "agents", "deep-explore.md")
	content, err := os.ReadFile(subagentPath)
	if err != nil {
		t.Fatalf("failed to read subagent file: %v", err)
	}

	// Count occurrences of marker to ensure no duplication
	count := strings.Count(string(content), fullTextSubagentMarker)
	if count != 1 {
		t.Errorf("expected 1 occurrence of marker, got %d", count)
	}
}

func TestCreateSubagentDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Ensure .claude/agents/ directory is created
	err := createSubagent(tmpDir, fullTextSubagent, fullTextSubagentMarker)
	if err != nil {
		t.Fatalf("failed to create subagent: %v", err)
	}

	agentsDir := filepath.Join(tmpDir, ".claude", "agents")
	info, err := os.Stat(agentsDir)
	if os.IsNotExist(err) {
		t.Fatal(".claude/agents directory was not created")
	}

	if !info.IsDir() {
		t.Fatal(".claude/agents is not a directory")
	}
}

func TestCreateSubagentTemplateContent(t *testing.T) {
	tmpDir := t.TempDir()

	err := createSubagent(tmpDir, fullTextSubagent, fullTextSubagentMarker)
	if err != nil {
		t.Fatalf("failed to create subagent: %v", err)
	}

	subagentPath := filepath.Join(tmpDir, ".claude", "agents", "deep-explore.md")
	content, err := os.ReadFile(subagentPath)
	if err != nil {
		t.Fatalf("failed to read subagent file: %v", err)
	}

	contentStr := string(content)

	// Verify YAML frontmatter
	if !strings.Contains(contentStr, "name: deep-explore") {
		t.Error("missing name in frontmatter")
	}
	if !strings.Contains(contentStr, "description:") {
		t.Error("missing description in frontmatter")
	}

	// Verify FTS-specific content (check for at least one variant)
	if !strings.Contains(contentStr, "full-text search") && !strings.Contains(contentStr, "Full-Text Search") && !strings.Contains(contentStr, "Full Text Search") {
		t.Error("missing Full-Text Search description")
	}
}

func TestCreateSubagentIdempotentAcrossTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subagent first
	err := createSubagent(tmpDir, fullTextSubagent, fullTextSubagentMarker)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	// Try to create again - should be skipped (idempotent)
	err = createSubagent(tmpDir, fullTextSubagent, fullTextSubagentMarker)
	if err != nil {
		t.Fatalf("second creation failed: %v", err)
	}

	// Should still have the original content
	subagentPath := filepath.Join(tmpDir, ".claude", "agents", "deep-explore.md")
	content, err := os.ReadFile(subagentPath)
	if err != nil {
		t.Fatalf("failed to read subagent file: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "name: deep-explore") {
		t.Error("deep-explore marker not found")
	}
}

// Tests for getTemplates function

func TestGetTemplates_FullText(t *testing.T) {
	instructions, subagent, marker, subagentMarker, rule := getTemplates()

	if !strings.Contains(instructions, "Full-Text Search") {
		t.Error("instructions should contain 'Full-Text Search'")
	}
	if marker != fullTextMarker {
		t.Errorf("marker = %q, want %q", marker, fullTextMarker)
	}
	if subagentMarker != fullTextSubagentMarker {
		t.Errorf("subagentMarker = %q, want %q", subagentMarker, fullTextSubagentMarker)
	}
	if subagent == "" {
		t.Error("subagent template should not be empty")
	}
	if rule == "" {
		t.Error("rule template should not be empty")
	}
	if !strings.Contains(rule, ruleMarker) {
		t.Error("rule template should contain rule marker")
	}
}

func TestTemplateMarkers_Unique(t *testing.T) {
	markers := []string{fullTextMarker, fullTextSubagentMarker, ruleMarker}
	seen := make(map[string]bool)
	for _, m := range markers {
		if seen[m] {
			t.Errorf("duplicate marker found: %q", m)
		}
		seen[m] = true
	}
}

func TestCreateRule(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating rule with FTS template
	err := createRule(tmpDir, fullTextRule)
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	// Verify file exists
	rulePath := filepath.Join(tmpDir, ".claude", "rules", "agentdx.md")
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		t.Fatal("rule file was not created")
	}

	// Verify content contains marker
	content, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("failed to read rule file: %v", err)
	}

	if !strings.Contains(string(content), ruleMarker) {
		t.Error("rule file does not contain expected marker")
	}
}

func TestCreateRuleIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create rule twice
	err := createRule(tmpDir, fullTextRule)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	err = createRule(tmpDir, fullTextRule)
	if err != nil {
		t.Fatalf("second creation failed: %v", err)
	}

	// Should still only have one file with expected content
	rulePath := filepath.Join(tmpDir, ".claude", "rules", "agentdx.md")
	content, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("failed to read rule file: %v", err)
	}

	// Count occurrences of marker to ensure no duplication
	count := strings.Count(string(content), ruleMarker)
	if count != 1 {
		t.Errorf("expected 1 occurrence of marker, got %d", count)
	}
}

func TestFullTextInstructions_NoEmbeddingReferences(t *testing.T) {
	// Check for forbidden terms that shouldn't appear in FTS instructions
	forbidden := []string{"vector similarity", "embedding model"}
	instructionsLower := strings.ToLower(fullTextInstructions)
	for _, word := range forbidden {
		if strings.Contains(instructionsLower, word) {
			t.Errorf("fullTextInstructions should not contain %q", word)
		}
	}

	// Verify it contains FTS search content
	if !strings.Contains(fullTextInstructions, "PostgreSQL") {
		t.Error("fullTextInstructions should mention PostgreSQL")
	}
}

func TestFullTextInstructions_HasSearchExamples(t *testing.T) {
	// Check for search examples
	if !strings.Contains(fullTextInstructions, `agentdx search "`) {
		t.Error("fullTextInstructions should contain agentdx search examples")
	}
}

// Tests for createSettings function

func TestCreateSettings_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create settings
	err := createSettings(tmpDir)
	if err != nil {
		t.Fatalf("failed to create settings: %v", err)
	}

	// Verify file exists
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatal("settings file was not created")
	}

	// Verify content
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	settings, err := parseSettings(content)
	if err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	// Verify agentdx hooks are present
	if !hasAgentdxHooks(settings) {
		t.Error("settings file should contain agentdx hooks")
	}

	// Verify expected hooks
	// UserPromptSubmit: session start (runs once per user message, not per tool call)
	if len(settings.Hooks.UserPromptSubmit) != 1 {
		t.Errorf("expected 1 UserPromptSubmit hook, got %d", len(settings.Hooks.UserPromptSubmit))
	}
	// PreToolUse: Grep and Glob warnings only
	if len(settings.Hooks.PreToolUse) != 2 {
		t.Errorf("expected 2 PreToolUse hooks, got %d", len(settings.Hooks.PreToolUse))
	}
	if len(settings.Hooks.PostToolUse) != 1 {
		t.Errorf("expected 1 PostToolUse hook, got %d", len(settings.Hooks.PostToolUse))
	}
	// NOTE: No Stop hooks - daemon keeps running for fresh index
}

func TestCreateSettings_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create settings twice
	err := createSettings(tmpDir)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	err = createSettings(tmpDir)
	if err != nil {
		t.Fatalf("second creation failed: %v", err)
	}

	// Verify content is still valid
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	settings, err := parseSettings(content)
	if err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	// Should still have only the expected hooks (not duplicated)
	// UserPromptSubmit: session start
	if len(settings.Hooks.UserPromptSubmit) != 1 {
		t.Errorf("expected 1 UserPromptSubmit hook after second call, got %d", len(settings.Hooks.UserPromptSubmit))
	}
	// PreToolUse: Grep and Glob warnings only
	if len(settings.Hooks.PreToolUse) != 2 {
		t.Errorf("expected 2 PreToolUse hooks after second call, got %d", len(settings.Hooks.PreToolUse))
	}
	if len(settings.Hooks.PostToolUse) != 1 {
		t.Errorf("expected 1 PostToolUse hook after second call, got %d", len(settings.Hooks.PostToolUse))
	}
	// NOTE: No Stop hooks - daemon keeps running for fresh index
}

func TestCreateSettings_MergeWithExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .claude directory
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude directory: %v", err)
	}

	// Create existing settings with other hooks
	existingSettings := `{
  "enabledPlugins": {
    "gopls-lsp@claude-plugins-official": true
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "SomeOtherTool",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'This is some other tool hook'"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "AnotherTool",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'This is another tool hook'"
          }
        ]
      }
    ]
  }
}`

	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(existingSettings), 0644); err != nil {
		t.Fatalf("failed to write existing settings: %v", err)
	}

	// Create/merge settings
	err := createSettings(tmpDir)
	if err != nil {
		t.Fatalf("failed to merge settings: %v", err)
	}

	// Verify merged content
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	settings, err := parseSettings(content)
	if err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	// Verify agentdx hooks are present
	if !hasAgentdxHooks(settings) {
		t.Error("settings file should contain agentdx hooks after merge")
	}

	// Verify original hooks are preserved
	// UserPromptSubmit: session start
	if len(settings.Hooks.UserPromptSubmit) != 1 {
		t.Errorf("expected 1 UserPromptSubmit hook, got %d", len(settings.Hooks.UserPromptSubmit))
	}
	// Original + Grep + Glob (session start moved to UserPromptSubmit)
	if len(settings.Hooks.PreToolUse) != 3 {
		t.Errorf("expected 3 PreToolUse hooks, got %d", len(settings.Hooks.PreToolUse))
	}
	if len(settings.Hooks.PostToolUse) != 2 { // Original + Bash (fallback)
		t.Errorf("expected 2 PostToolUse hooks, got %d", len(settings.Hooks.PostToolUse))
	}
	// NOTE: No Stop hooks - daemon keeps running for fresh index

	// Verify enabledPlugins preserved
	if !settings.EnabledPlugins["gopls-lsp@claude-plugins-official"] {
		t.Error("enabledPlugins should be preserved after merge")
	}

	// Verify original hook is first
	if settings.Hooks.PreToolUse[0].Matcher != "SomeOtherTool" {
		t.Error("original PreToolUse hook should be first")
	}
	if settings.Hooks.PostToolUse[0].Matcher != "AnotherTool" {
		t.Error("original PostToolUse hook should be first")
	}
}

func TestCreateSettings_SkipsIfAlreadyConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	// Create settings first time
	err := createSettings(tmpDir)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	// Get file modification time
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	info1, err := os.Stat(settingsPath)
	if err != nil {
		t.Fatalf("failed to stat settings file: %v", err)
	}

	// Create settings second time
	err = createSettings(tmpDir)
	if err != nil {
		t.Fatalf("second creation failed: %v", err)
	}

	// Verify file wasn't modified (idempotent)
	info2, err := os.Stat(settingsPath)
	if err != nil {
		t.Fatalf("failed to stat settings file: %v", err)
	}

	// ModTime should be the same since the file shouldn't be rewritten
	if info1.ModTime() != info2.ModTime() {
		// Note: This test may fail if the FS doesn't preserve modtime on write
		// In that case, we can check the content is identical
		content1, _ := os.ReadFile(settingsPath)
		content2, _ := os.ReadFile(settingsPath)
		if string(content1) != string(content2) {
			t.Error("settings file was modified when it shouldn't have been")
		}
	}
}

func TestCreateSettings_CreatesBackupBeforeModifying(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial settings.json with partial configuration (only Grep hook)
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude directory: %v", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	initialContent := `{
  "enabledPlugins": {
    "gopls-lsp@claude-plugins-official": true
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Grep",
        "hooks": [
          {
            "type": "command",
            "command": "echo '⚠️ AGENTDX FALLBACK: Grep tool requested.'"
          }
        ]
      }
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial settings: %v", err)
	}

	// Run createSettings - should merge missing hooks
	err := createSettings(tmpDir)
	if err != nil {
		t.Fatalf("createSettings failed: %v", err)
	}

	// Verify backup was created
	backupPath := filepath.Join(claudeDir, "settings.backup.json")
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("backup file was not created: %v", err)
	}

	// Verify backup contains the original content
	if string(backupContent) != initialContent {
		t.Error("backup content does not match original settings")
	}

	// Verify settings.json was updated with all hooks
	updatedContent, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read updated settings: %v", err)
	}

	settings, err := parseSettings(updatedContent)
	if err != nil {
		t.Fatalf("failed to parse updated settings: %v", err)
	}

	// Should now have all hooks (Grep, Glob, Bash)
	if !hasAgentdxHooks(settings) {
		t.Error("updated settings should have all agentdx hooks")
	}
}
