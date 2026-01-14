package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/doveaia/agentdx/config"
)

func TestCreateSubagent(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating subagent with semantic template (default)
	err := createSubagent(tmpDir, subagentTemplate, subagentMarker)
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

	if !strings.Contains(string(content), subagentMarker) {
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
	err := createSubagent(tmpDir, subagentTemplate, subagentMarker)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	err = createSubagent(tmpDir, subagentTemplate, subagentMarker)
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
	count := strings.Count(string(content), subagentMarker)
	if count != 1 {
		t.Errorf("expected 1 occurrence of marker, got %d", count)
	}
}

func TestCreateSubagentDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Ensure .claude/agents/ directory is created
	err := createSubagent(tmpDir, subagentTemplate, subagentMarker)
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

	err := createSubagent(tmpDir, subagentTemplate, subagentMarker)
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
	if !strings.Contains(contentStr, "tools: Read, Grep, Glob, Bash") {
		t.Error("missing or incorrect tools in frontmatter")
	}
	if !strings.Contains(contentStr, "model: inherit") {
		t.Error("missing or incorrect model in frontmatter")
	}

	// Verify instructions content
	if !strings.Contains(contentStr, "Semantic Search") {
		t.Error("missing Semantic Search section")
	}
	if !strings.Contains(contentStr, "Call Graph Tracing") {
		t.Error("missing Call Graph Tracing section")
	}
	if !strings.Contains(contentStr, "Workflow") {
		t.Error("missing Workflow section")
	}
}

func TestCreateSubagentFullTextTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating subagent with full-text template
	err := createSubagent(tmpDir, fullTextSubagentTemplate, fullTextSubagentMarker)
	if err != nil {
		t.Fatalf("failed to create full-text subagent: %v", err)
	}

	subagentPath := filepath.Join(tmpDir, ".claude", "agents", "deep-explore.md")
	content, err := os.ReadFile(subagentPath)
	if err != nil {
		t.Fatalf("failed to read subagent file: %v", err)
	}

	contentStr := string(content)

	// Verify YAML frontmatter for full-text
	if !strings.Contains(contentStr, "name: deep-explore-fulltext") {
		t.Error("missing name: deep-explore-fulltext in frontmatter")
	}

	// Verify full-text specific content
	if !strings.Contains(contentStr, "PostgreSQL full-text search") {
		t.Error("missing PostgreSQL full-text search description")
	}
	if !strings.Contains(contentStr, "parallel keyword searches") {
		t.Error("missing parallel keyword search instruction")
	}
}

func TestCreateSubagentIdempotentAcrossTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create semantic subagent first
	err := createSubagent(tmpDir, subagentTemplate, subagentMarker)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	// Try to create full-text subagent - should be skipped (idempotent)
	err = createSubagent(tmpDir, fullTextSubagentTemplate, fullTextSubagentMarker)
	if err != nil {
		t.Fatalf("second creation failed: %v", err)
	}

	// Should still have the original semantic content
	subagentPath := filepath.Join(tmpDir, ".claude", "agents", "deep-explore.md")
	content, err := os.ReadFile(subagentPath)
	if err != nil {
		t.Fatalf("failed to read subagent file: %v", err)
	}

	contentStr := string(content)

	// Should have semantic marker (not full-text)
	if !strings.Contains(contentStr, "name: deep-explore") {
		t.Error("semantic marker not found")
	}
	if strings.Contains(contentStr, "name: deep-explore-fulltext") {
		t.Error("full-text marker should not be present (first created was semantic)")
	}
}

// Tests for detectSearchType function

func TestDetectSearchType_Postgres(t *testing.T) {
	cfg := &config.Config{
		Index: config.IndexSection{
			Embedder: config.EmbedderConfig{Provider: "postgres"},
		},
	}
	got := detectSearchType(cfg)
	if got != searchTypeFullText {
		t.Errorf("detectSearchType() = %q, want %q", got, searchTypeFullText)
	}
}

func TestDetectSearchType_Ollama(t *testing.T) {
	cfg := &config.Config{
		Index: config.IndexSection{
			Embedder: config.EmbedderConfig{Provider: "ollama"},
		},
	}
	got := detectSearchType(cfg)
	if got != searchTypeSemantic {
		t.Errorf("detectSearchType() = %q, want %q", got, searchTypeSemantic)
	}
}

func TestDetectSearchType_OpenAI(t *testing.T) {
	cfg := &config.Config{
		Index: config.IndexSection{
			Embedder: config.EmbedderConfig{Provider: "openai"},
		},
	}
	got := detectSearchType(cfg)
	if got != searchTypeSemantic {
		t.Errorf("detectSearchType() = %q, want %q", got, searchTypeSemantic)
	}
}

func TestDetectSearchType_LMStudio(t *testing.T) {
	cfg := &config.Config{
		Index: config.IndexSection{
			Embedder: config.EmbedderConfig{Provider: "lmstudio"},
		},
	}
	got := detectSearchType(cfg)
	if got != searchTypeSemantic {
		t.Errorf("detectSearchType() = %q, want %q", got, searchTypeSemantic)
	}
}

func TestDetectSearchType_UnknownProvider(t *testing.T) {
	cfg := &config.Config{
		Index: config.IndexSection{
			Embedder: config.EmbedderConfig{Provider: "unknown-provider"},
		},
	}
	got := detectSearchType(cfg)
	if got != searchTypeSemantic {
		t.Errorf("detectSearchType() = %q, want %q for unknown provider", got, searchTypeSemantic)
	}
}

// Tests for getTemplates function

func TestGetTemplates_Semantic(t *testing.T) {
	instructions, subagent, marker, subagentMarker := getTemplates(searchTypeSemantic)

	if !strings.Contains(instructions, "Semantic Code Search") {
		t.Error("semantic instructions should contain 'Semantic Code Search'")
	}
	if marker != agentMarker {
		t.Errorf("marker = %q, want %q", marker, agentMarker)
	}
	if subagentMarker != subagentMarker {
		t.Errorf("subagentMarker mismatch")
	}
	if subagent == "" {
		t.Error("subagent template should not be empty")
	}
}

func TestGetTemplates_FullText(t *testing.T) {
	instructions, subagent, marker, subagentMarker := getTemplates(searchTypeFullText)

	if !strings.Contains(instructions, "PostgreSQL Full-Text Search") {
		t.Error("full-text instructions should contain 'PostgreSQL Full-Text Search'")
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
}

func TestTemplateMarkers_Unique(t *testing.T) {
	markers := []string{agentMarker, subagentMarker, fullTextMarker, fullTextSubagentMarker}
	seen := make(map[string]bool)
	for _, m := range markers {
		if seen[m] {
			t.Errorf("duplicate marker found: %q", m)
		}
		seen[m] = true
	}
}

func TestFullTextInstructions_NoEmbeddingReferences(t *testing.T) {
	// Check for forbidden terms that shouldn't appear in full-text instructions
	forbidden := []string{"vector similarity", "embedding model"}
	instructionsLower := strings.ToLower(fullTextInstructions)
	for _, word := range forbidden {
		if strings.Contains(instructionsLower, word) {
			t.Errorf("fullTextInstructions should not contain %q", word)
		}
	}

	// Verify it contains the expected full-text search content
	if !strings.Contains(fullTextInstructions, "parallel") {
		t.Error("fullTextInstructions should mention parallel searches")
	}
}

func TestFullTextInstructions_HasParallelExamples(t *testing.T) {
	// Check for parallel search pattern (multiple searches with &)
	if !strings.Contains(fullTextInstructions, "&") {
		t.Error("fullTextInstructions should contain parallel search examples (with &)")
	}
	// Check for individual keyword examples
	if !strings.Contains(fullTextInstructions, `agentdx search "`) {
		t.Error("fullTextInstructions should contain agentdx search examples")
	}
}
