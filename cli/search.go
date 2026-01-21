package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/search"
	"github.com/doveaia/agentdx/store"
	"github.com/spf13/cobra"
)

var (
	searchLimit   int
	searchJSON    bool
	searchCompact bool
)

// SearchResultJSON is a lightweight struct for JSON output (excludes vector, hash, updated_at)
type SearchResultJSON struct {
	FilePath  string  `json:"file_path"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
	Content   string  `json:"content"`
}

// SearchResultCompactJSON is a minimal struct for compact JSON output (no content field)
type SearchResultCompactJSON struct {
	FilePath  string  `json:"file_path"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search codebase with full text search",
	Long: `Search your codebase using full text search queries.

The search will:
- Query the documents_fts table with your search terms
- Return the most relevant results with file path, line numbers, and score`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 10, "Maximum number of results to return")
	searchCmd.Flags().BoolVarP(&searchJSON, "json", "j", false, "Output results in JSON format (for AI agents)")
	searchCmd.Flags().BoolVarP(&searchCompact, "compact", "c", false, "Output minimal JSON without content (requires --json)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	ctx := context.Background()

	// Validate flag combination
	if searchCompact && !searchJSON {
		return fmt.Errorf("--compact flag requires --json flag")
	}

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Load configuration
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize PostgreSQL FTS store
	ftsStore, err := store.NewPostgresFTSStore(ctx, cfg.Index.Store.Postgres.DSN, projectRoot)
	if err != nil {
		if searchJSON {
			return outputSearchError(err)
		}
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer ftsStore.Close()

	// Search using FTS
	results, err := ftsStore.SearchFTS(ctx, query, searchLimit*2)
	if err != nil {
		if searchJSON {
			return outputSearchError(err)
		}
		return fmt.Errorf("search failed: %w", err)
	}

	// Apply structural boosting
	results = search.ApplyBoost(results, cfg.Index.Search.Boost)

	// Trim to requested limit
	if len(results) > searchLimit {
		results = results[:searchLimit]
	}

	// JSON output mode
	if searchJSON {
		if searchCompact {
			return outputSearchCompactJSON(results)
		}
		return outputSearchJSON(results)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Display results
	fmt.Printf("Found %d results for: %q\n\n", len(results), query)

	for i, result := range results {
		fmt.Printf("─── Result %d (score: %.4f) ───\n", i+1, result.Score)
		fmt.Printf("File: %s:%d-%d\n", result.Chunk.FilePath, result.Chunk.StartLine, result.Chunk.EndLine)
		fmt.Println()

		// Display content with line numbers
		lines := strings.Split(result.Chunk.Content, "\n")
		// Skip the "File: xxx" prefix line if present
		startIdx := 0
		if len(lines) > 0 && strings.HasPrefix(lines[0], "File: ") {
			startIdx = 2 // Skip "File: xxx" and empty line
		}

		lineNum := result.Chunk.StartLine
		for j := startIdx; j < len(lines) && j < startIdx+15; j++ {
			fmt.Printf("%4d │ %s\n", lineNum, lines[j])
			lineNum++
		}
		if len(lines)-startIdx > 15 {
			fmt.Printf("     │ ... (%d more lines)\n", len(lines)-startIdx-15)
		}
		fmt.Println()
	}

	return nil
}

// outputSearchJSON outputs results in JSON format for AI agents
func outputSearchJSON(results []store.SearchResult) error {
	jsonResults := make([]SearchResultJSON, len(results))
	for i, r := range results {
		jsonResults[i] = SearchResultJSON{
			FilePath:  r.Chunk.FilePath,
			StartLine: r.Chunk.StartLine,
			EndLine:   r.Chunk.EndLine,
			Score:     r.Score,
			Content:   r.Chunk.Content,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonResults)
}

// outputSearchCompactJSON outputs results in minimal JSON format (without content)
func outputSearchCompactJSON(results []store.SearchResult) error {
	jsonResults := make([]SearchResultCompactJSON, len(results))
	for i, r := range results {
		jsonResults[i] = SearchResultCompactJSON{
			FilePath:  r.Chunk.FilePath,
			StartLine: r.Chunk.StartLine,
			EndLine:   r.Chunk.EndLine,
			Score:     r.Score,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonResults)
}

// outputSearchError outputs an error in JSON format
func outputSearchError(err error) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(map[string]string{"error": err.Error()})
	return nil
}

// SearchJSON returns results in JSON format for AI agents
func SearchJSON(projectRoot string, query string, limit int) ([]store.SearchResult, error) {
	ctx := context.Background()

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	// Initialize PostgreSQL FTS store
	ftsStore, err := store.NewPostgresFTSStore(ctx, cfg.Index.Store.Postgres.DSN, projectRoot)
	if err != nil {
		return nil, err
	}
	defer ftsStore.Close()

	// Search using FTS
	results, err := ftsStore.SearchFTS(ctx, query, limit*2)
	if err != nil {
		return nil, err
	}

	// Apply structural boosting
	results = search.ApplyBoost(results, cfg.Index.Search.Boost)

	// Trim to requested limit
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func init() {
	// Ensure the search command is registered
	_ = os.Getenv("GREPAI_DEBUG")
}
