package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/store"
	"github.com/spf13/cobra"
)

var (
	filesLimit   int
	filesJSON    bool
	filesCompact bool
)

// FileResultJSON is the full output struct for JSON mode
type FileResultJSON struct {
	Path    string `json:"path"`
	ModTime string `json:"mod_time"`
}

// FileResultCompactJSON is the minimal output struct for compact mode
type FileResultCompactJSON struct {
	Path string `json:"path"`
}

var filesCmd = &cobra.Command{
	Use:   "files <glob>",
	Short: "List indexed files matching a glob pattern",
	Long: `List files in the index that match a glob pattern.

Patterns without path separators are matched recursively by default:
  *.go          - All Go files recursively (same as **/*.go)
  *.test.ts     - All test files recursively

Use explicit paths to limit scope:
  internal/**   - All files under internal/
  cli/*.go      - Go files only in cli/ directory`,
	Args: cobra.ExactArgs(1),
	RunE: runFiles,
}

func init() {
	filesCmd.Flags().IntVarP(&filesLimit, "limit", "n", 0, "Maximum number of results (0 = unlimited)")
	filesCmd.Flags().BoolVarP(&filesJSON, "json", "j", false, "Output results in JSON format")
	filesCmd.Flags().BoolVarP(&filesCompact, "compact", "c", false, "Output minimal JSON (requires --json)")
}

func runFiles(cmd *cobra.Command, args []string) error {
	pattern := args[0]
	ctx := context.Background()

	// Validate flag combination
	if filesCompact && !filesJSON {
		return fmt.Errorf("--compact flag requires --json flag")
	}

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		if filesJSON {
			return outputFilesError(err)
		}
		return err
	}

	// Load configuration
	cfg, err := config.Load(projectRoot)
	if err != nil {
		if filesJSON {
			return outputFilesError(fmt.Errorf("failed to load configuration: %w", err))
		}
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize PostgreSQL FTS store
	st, err := store.NewPostgresFTSStore(ctx, cfg.Index.Store.Postgres.DSN, projectRoot)
	if err != nil {
		if filesJSON {
			return outputFilesError(fmt.Errorf("failed to connect to postgres: %w", err))
		}
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer st.Close()

	// Get all files with stats
	allFiles, err := st.ListFilesWithStats(ctx)
	if err != nil {
		if filesJSON {
			return outputFilesError(fmt.Errorf("failed to list files: %w", err))
		}
		return fmt.Errorf("failed to list files: %w", err)
	}

	// Filter by glob pattern
	matched, err := filterByGlob(allFiles, pattern)
	if err != nil {
		if filesJSON {
			return outputFilesError(err)
		}
		return err
	}

	// Sort alphabetically by path
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Path < matched[j].Path
	})

	// Apply limit if specified
	if filesLimit > 0 && len(matched) > filesLimit {
		matched = matched[:filesLimit]
	}

	// Output results
	if filesJSON {
		if filesCompact {
			return outputFilesCompactJSON(matched)
		}
		return outputFilesJSON(matched)
	}

	outputFilesText(matched, pattern)
	return nil
}

// normalizeGlobPattern makes patterns without path separators recursive by default.
// "*.go" becomes "**/*.go" to match all Go files recursively.
// Patterns with "/" or "**" are left unchanged.
func normalizeGlobPattern(pattern string) string {
	// If pattern already has path separator or **, leave it as-is
	if strings.Contains(pattern, "/") || strings.Contains(pattern, "**") {
		return pattern
	}
	// Make simple patterns like "*.go" recursive
	return "**/" + pattern
}

// filterByGlob filters files by glob pattern using doublestar
func filterByGlob(files []store.FileStats, pattern string) ([]store.FileStats, error) {
	// Normalize pattern to be recursive by default
	normalizedPattern := normalizeGlobPattern(pattern)

	var matched []store.FileStats
	for _, f := range files {
		ok, err := doublestar.Match(normalizedPattern, f.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern: %w", err)
		}
		if ok {
			matched = append(matched, f)
		}
	}
	return matched, nil
}

// outputFilesText outputs files in plain text format
func outputFilesText(files []store.FileStats, pattern string) {
	if len(files) == 0 {
		fmt.Println("No files found matching pattern.")
		return
	}
	fmt.Printf("Found %d files matching %q:\n\n", len(files), pattern)
	for _, f := range files {
		fmt.Println(f.Path)
	}
}

// outputFilesJSON outputs files in full JSON format
func outputFilesJSON(files []store.FileStats) error {
	results := make([]FileResultJSON, len(files))
	for i, f := range files {
		results[i] = FileResultJSON{
			Path:    f.Path,
			ModTime: f.ModTime.Format("2006-01-02T15:04:05Z"),
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// outputFilesCompactJSON outputs files in minimal JSON format
func outputFilesCompactJSON(files []store.FileStats) error {
	results := make([]FileResultCompactJSON, len(files))
	for i, f := range files {
		results[i] = FileResultCompactJSON{
			Path: f.Path,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// outputFilesError outputs an error in JSON format
func outputFilesError(err error) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(map[string]string{"error": err.Error()})
	return nil
}
