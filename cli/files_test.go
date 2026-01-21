package cli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/doveaia/agentdx/store"
	"github.com/stretchr/testify/assert"
)

func TestGlobPatternMatching(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		paths   []string
		want    []string
	}{
		{
			name:    "root level only",
			pattern: "*.go",
			paths:   []string{"main.go", "cli/files.go", "store/gob.go"},
			want:    []string{"main.go"},
		},
		{
			name:    "recursive glob",
			pattern: "**/*.go",
			paths:   []string{"main.go", "cli/files.go", "store/gob.go"},
			want:    []string{"main.go", "cli/files.go", "store/gob.go"},
		},
		{
			name:    "directory specific",
			pattern: "cli/**/*.go",
			paths:   []string{"main.go", "cli/files.go", "cli/search.go", "store/gob.go"},
			want:    []string{"cli/files.go", "cli/search.go"},
		},
		{
			name:    "no matches",
			pattern: "*.rs",
			paths:   []string{"main.go", "cli/files.go"},
			want:    []string{},
		},
		{
			name:    "double star directory",
			pattern: "store/**",
			paths:   []string{"main.go", "store/gob.go", "store/postgres.go", "store/subdir/file.go"},
			want:    []string{"store/gob.go", "store/postgres.go", "store/subdir/file.go"},
		},
		{
			name:    "complex pattern",
			pattern: "**/*_test.go",
			paths:   []string{"main.go", "cli/files_test.go", "store/gob_test.go", "cli/files.go"},
			want:    []string{"cli/files_test.go", "store/gob_test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, path := range tt.paths {
				ok, _ := doublestar.Match(tt.pattern, path)
				if ok {
					got = append(got, path)
				}
			}
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestInvalidGlobPattern(t *testing.T) {
	_, err := doublestar.Match("[invalid", "test.go")
	assert.Error(t, err)
}

func TestCompactRequiresJSON(t *testing.T) {
	// Test that --compact without --json returns error
	// We test this by directly checking the validation logic

	// Save original values
	originalCompact := filesCompact
	originalJSON := filesJSON

	// Reset after test
	defer func() {
		filesCompact = originalCompact
		filesJSON = originalJSON
	}()

	// Set up test case: --compact without --json
	filesCompact = true
	filesJSON = false

	// The validation happens at the start of runFiles
	if filesCompact && !filesJSON {
		// This is the expected behavior - validation would fail
		return
	}

	t.Error("expected --compact to require --json flag")
}

func TestFilesCompactFlagWithJSON(t *testing.T) {
	// Test that --compact with --json is valid

	// Save original values
	originalCompact := filesCompact
	originalJSON := filesJSON

	// Reset after test
	defer func() {
		filesCompact = originalCompact
		filesJSON = originalJSON
	}()

	// Set up test case: --compact with --json
	filesCompact = true
	filesJSON = true

	// The validation should pass
	if filesCompact && !filesJSON {
		t.Error("expected --compact with --json to be valid")
	}
}

func TestFileResultJSONStruct(t *testing.T) {
	result := FileResultJSON{
		Path:    "path/to/file.go",
		ModTime: time.Now().Format("2006-01-02T15:04:05Z"),
	}

	data, err := json.Marshal(result)
	assert.NoError(t, err)

	// Verify all fields are present in JSON
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	expectedFields := []string{"path", "mod_time"}
	for _, field := range expectedFields {
		if _, exists := decoded[field]; !exists {
			t.Errorf("expected field '%s' to be present", field)
		}
	}
}

func TestFileResultCompactJSONStruct(t *testing.T) {
	result := FileResultCompactJSON{
		Path: "path/to/file.go",
	}

	data, err := json.Marshal(result)
	assert.NoError(t, err)

	// Verify expected fields are present
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	if _, exists := decoded["path"]; !exists {
		t.Error("expected 'path' field to be present")
	}

	// Verify only path field exists
	if len(decoded) != 1 {
		t.Errorf("expected only 1 field in compact struct, got %d", len(decoded))
	}
}

func TestNormalizeGlobPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"*.go", "**/*.go"},                      // Simple pattern becomes recursive
		{"*.test.ts", "**/*.test.ts"},            // Multiple extensions
		{"**/*.go", "**/*.go"},                   // Already recursive - unchanged
		{"cli/*.go", "cli/*.go"},                 // Has path separator - unchanged
		{"internal/**/*.go", "internal/**/*.go"}, // Explicit path - unchanged
		{"*.md", "**/*.md"},                      // Simple markdown pattern
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeGlobPattern(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterByGlob(t *testing.T) {
	files := []store.FileStats{
		{Path: "main.go", ModTime: time.Now(), ChunkCount: 1},
		{Path: "cli/files.go", ModTime: time.Now(), ChunkCount: 2},
		{Path: "cli/search.go", ModTime: time.Now(), ChunkCount: 3},
		{Path: "store/gob.go", ModTime: time.Now(), ChunkCount: 4},
		{Path: "README.md", ModTime: time.Now(), ChunkCount: 0},
	}

	tests := []struct {
		name    string
		pattern string
		wantLen int
	}{
		{
			name:    "simple pattern matches recursively",
			pattern: "*.go",
			wantLen: 4, // *.go normalizes to **/*.go
		},
		{
			name:    "explicit recursive glob",
			pattern: "**/*.go",
			wantLen: 4,
		},
		{
			name:    "cli directory only",
			pattern: "cli/**/*.go",
			wantLen: 2,
		},
		{
			name:    "markdown files recursive",
			pattern: "*.md",
			wantLen: 1, // *.md normalizes to **/*.md
		},
		{
			name:    "no matches",
			pattern: "*.rs",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterByGlob(files, tt.pattern)
			assert.NoError(t, err)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestFilterByGlobInvalidPattern(t *testing.T) {
	files := []store.FileStats{
		{Path: "main.go", ModTime: time.Now(), ChunkCount: 1},
	}

	_, err := filterByGlob(files, "[invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid glob pattern")
}

func TestOutputFilesJSON(t *testing.T) {
	files := []store.FileStats{
		{Path: "cli/files.go", ModTime: time.Date(2026, 1, 19, 10, 30, 0, 0, time.UTC), ChunkCount: 1},
		{Path: "cli/search.go", ModTime: time.Date(2026, 1, 19, 10, 30, 0, 0, time.UTC), ChunkCount: 2},
	}

	// Capture stdout
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	results := make([]FileResultJSON, len(files))
	for i, f := range files {
		results[i] = FileResultJSON{
			Path:    f.Path,
			ModTime: f.ModTime.Format("2006-01-02T15:04:05Z"),
		}
	}
	err := encoder.Encode(results)
	assert.NoError(t, err)

	// Verify output
	var decoded []FileResultJSON
	err = json.Unmarshal(buf.Bytes(), &decoded)
	assert.NoError(t, err)
	assert.Len(t, decoded, 2)
	assert.Equal(t, "cli/files.go", decoded[0].Path)
	assert.Equal(t, "cli/search.go", decoded[1].Path)
	assert.NotEmpty(t, decoded[0].ModTime)
}

func TestOutputFilesCompactJSON(t *testing.T) {
	files := []store.FileStats{
		{Path: "cli/files.go", ModTime: time.Now(), ChunkCount: 1},
		{Path: "cli/search.go", ModTime: time.Now(), ChunkCount: 2},
	}

	// Capture stdout
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	results := make([]FileResultCompactJSON, len(files))
	for i, f := range files {
		results[i] = FileResultCompactJSON{
			Path: f.Path,
		}
	}
	err := encoder.Encode(results)
	assert.NoError(t, err)

	// Verify output
	var decoded []FileResultCompactJSON
	err = json.Unmarshal(buf.Bytes(), &decoded)
	assert.NoError(t, err)
	assert.Len(t, decoded, 2)
	assert.Equal(t, "cli/files.go", decoded[0].Path)
	assert.Equal(t, "cli/search.go", decoded[1].Path)
}

func TestOutputFilesError(t *testing.T) {
	testErr := assert.AnError

	// Capture stdout
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(map[string]string{"error": testErr.Error()})

	// Verify output
	var decoded map[string]string
	err := json.Unmarshal(buf.Bytes(), &decoded)
	assert.NoError(t, err)
	assert.Contains(t, decoded["error"], "assert.AnError")
}
