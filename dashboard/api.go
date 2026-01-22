package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/doveaia/agentdx/search"
	"github.com/doveaia/agentdx/trace"
	"github.com/go-chi/chi/v5"
)

// API Response Types

// StatusResponse is the API response for index status.
type StatusResponse struct {
	TotalFiles   int    `json:"total_files"`
	TotalChunks  int    `json:"total_chunks"`
	IndexSize    string `json:"index_size"`
	LastUpdated  string `json:"last_updated"`
	Search       string `json:"search"`
	SymbolsReady bool   `json:"symbols_ready"`
	BackendType  string `json:"backend_type,omitempty"`
	BackendHost  string `json:"backend_host,omitempty"`
	BackendName  string `json:"backend_name,omitempty"`
	BackendOK    bool   `json:"backend_ok,omitempty"`
}

// SearchResult represents a search result.
type SearchResult struct {
	FilePath  string  `json:"file_path"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
	Content   string  `json:"content"`
}

// FileResult represents a file in the index.
type FileResult struct {
	Path    string `json:"path"`
	ModTime string `json:"mod_time,omitempty"`
}

// TraceResponse is the API response for trace queries.
type TraceResponse struct {
	Query   string             `json:"query"`
	Mode    string             `json:"mode"`
	Symbol  *trace.Symbol      `json:"symbol,omitempty"`
	Callers []trace.CallerInfo `json:"callers,omitempty"`
	Callees []trace.CalleeInfo `json:"callees,omitempty"`
	Graph   *trace.CallGraph   `json:"graph,omitempty"`
}

// ProjectResult represents a project in the index.
type ProjectResult struct {
	ID        string `json:"id"`
	FileCount int    `json:"file_count"`
	IsCurrent bool   `json:"is_current"`
}

// API Handlers

// handleAPISearch handles GET /api/search
func (s *Server) handleAPISearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter 'q' is required"})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	ctx := r.Context()
	results, err := s.performSearch(ctx, query, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, results)
}

// handleAPIFiles handles GET /api/files
func (s *Server) handleAPIFiles(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter 'pattern' is required"})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	ctx := r.Context()
	files, err := s.listFiles(ctx, pattern, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, files)
}

// handleAPIStatus handles GET /api/status
func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := s.getStatus(ctx)
	writeJSON(w, http.StatusOK, status)
}

// handleAPITrace handles GET /api/trace/{mode}/{symbol}
func (s *Server) handleAPITrace(w http.ResponseWriter, r *http.Request) {
	mode := chi.URLParam(r, "mode")
	symbol := chi.URLParam(r, "symbol")

	if symbol == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "symbol parameter is required"})
		return
	}

	ctx := r.Context()
	result, err := s.performTrace(ctx, mode, symbol)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleAPIProjects handles GET /api/projects
func (s *Server) handleAPIProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projects, err := s.listProjects(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, projects)
}

// Business Logic

// getStatus returns the current index status.
func (s *Server) getStatus(ctx context.Context) *StatusResponse {
	status := &StatusResponse{
		Search: "PostgreSQL FTS",
	}

	// Get store stats
	if s.store != nil {
		stats, err := s.store.GetStats(ctx)
		if err == nil {
			status.TotalFiles = stats.TotalFiles
			status.TotalChunks = stats.TotalChunks
			status.IndexSize = formatBytes(stats.IndexSize)
			status.LastUpdated = stats.LastUpdated.Format("2006-01-02 15:04:05")
		}

		// Get backend status
		if bs := s.store.BackendStatus(ctx); bs != nil {
			status.BackendType = bs.Type
			status.BackendHost = bs.Host
			status.BackendName = bs.Name
			status.BackendOK = bs.Healthy
		}
	}

	// Check symbol index
	if s.symbolStore != nil {
		if symbolStats, err := s.symbolStore.GetStats(ctx); err == nil && symbolStats.TotalSymbols > 0 {
			status.SymbolsReady = true
		}
	}

	return status
}

// performSearch performs a search query.
func (s *Server) performSearch(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if s.store == nil {
		return nil, nil
	}

	// Search using FTS
	results, err := s.store.SearchFTS(ctx, query, limit*2)
	if err != nil {
		return nil, err
	}

	// Apply structural boosting
	results = search.ApplyBoost(results, s.config.Index.Search.Boost)

	// Trim to requested limit
	if len(results) > limit {
		results = results[:limit]
	}

	// Convert to lightweight results
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			FilePath:  r.Chunk.FilePath,
			StartLine: r.Chunk.StartLine,
			EndLine:   r.Chunk.EndLine,
			Score:     r.Score,
			Content:   r.Chunk.Content,
		}
	}

	return searchResults, nil
}

// listFiles lists files matching a pattern.
func (s *Server) listFiles(ctx context.Context, pattern string, limit int) ([]FileResult, error) {
	if s.store == nil {
		return nil, nil
	}

	// Get all files with stats
	allFiles, err := s.store.ListFilesWithStats(ctx)
	if err != nil {
		return nil, err
	}

	// Normalize pattern
	normalizedPattern := normalizeGlobPattern(pattern)

	// Filter by glob pattern
	var matched []FileResult
	for _, f := range allFiles {
		ok, err := doublestar.Match(normalizedPattern, f.Path)
		if err != nil {
			return nil, err
		}
		if ok {
			matched = append(matched, FileResult{
				Path:    f.Path,
				ModTime: f.ModTime.Format("2006-01-02T15:04:05Z"),
			})
		}
	}

	// Sort alphabetically
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Path < matched[j].Path
	})

	// Apply limit if specified
	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}

	return matched, nil
}

// performTrace performs a trace query.
func (s *Server) performTrace(ctx context.Context, mode, symbolName string) (*TraceResponse, error) {
	if s.symbolStore == nil {
		return &TraceResponse{Query: symbolName, Mode: mode}, nil
	}

	// Lookup symbol
	symbols, err := s.symbolStore.LookupSymbol(ctx, symbolName)
	if err != nil {
		return nil, err
	}

	result := &TraceResponse{
		Query: symbolName,
		Mode:  mode,
	}

	if len(symbols) > 0 {
		result.Symbol = &symbols[0]
	}

	switch mode {
	case "callers":
		refs, err := s.symbolStore.LookupCallers(ctx, symbolName)
		if err != nil {
			return nil, err
		}
		for _, ref := range refs {
			callerSyms, _ := s.symbolStore.LookupSymbol(ctx, ref.CallerName)
			var callerSym trace.Symbol
			if len(callerSyms) > 0 {
				callerSym = callerSyms[0]
			} else {
				callerSym = trace.Symbol{Name: ref.CallerName, File: ref.CallerFile, Line: ref.CallerLine}
			}
			result.Callers = append(result.Callers, trace.CallerInfo{
				Symbol: callerSym,
				CallSite: trace.CallSite{
					File:    ref.File,
					Line:    ref.Line,
					Context: ref.Context,
				},
			})
		}

	case "callees":
		if len(symbols) > 0 {
			refs, err := s.symbolStore.LookupCallees(ctx, symbolName, symbols[0].File)
			if err != nil {
				return nil, err
			}
			for _, ref := range refs {
				calleeSyms, _ := s.symbolStore.LookupSymbol(ctx, ref.SymbolName)
				var calleeSym trace.Symbol
				if len(calleeSyms) > 0 {
					calleeSym = calleeSyms[0]
				} else {
					calleeSym = trace.Symbol{Name: ref.SymbolName}
				}
				result.Callees = append(result.Callees, trace.CalleeInfo{
					Symbol: calleeSym,
					CallSite: trace.CallSite{
						File:    ref.File,
						Line:    ref.Line,
						Context: ref.Context,
					},
				})
			}
		}

	case "graph":
		graph, err := s.symbolStore.GetCallGraph(ctx, symbolName, 2)
		if err != nil {
			return nil, err
		}
		result.Graph = graph
	}

	return result, nil
}

// listProjects lists all indexed projects.
func (s *Server) listProjects(ctx context.Context) ([]ProjectResult, error) {
	if s.store == nil {
		return nil, nil
	}

	projects, err := s.store.GetAllProjects(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]ProjectResult, len(projects))
	currentProject := s.store.ProjectID()

	for i, p := range projects {
		results[i] = ProjectResult{
			ID:        p.ID,
			FileCount: p.FileCount,
			IsCurrent: p.ID == currentProject,
		}
	}

	return results, nil
}

// Helper functions

// normalizeGlobPattern makes patterns without path separators recursive by default.
func normalizeGlobPattern(pattern string) string {
	if strings.Contains(pattern, "/") || strings.Contains(pattern, "**") {
		return pattern
	}
	return "**/" + pattern
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(b int64) string {
	if b == 0 {
		return "N/A"
	}
	const unit = 1024
	if b < unit {
		return strconv.FormatInt(b, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(b)/float64(div), 'f', 1, 64) + " " + string("KMGTPE"[exp]) + "B"
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
