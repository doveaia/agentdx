// Package mcp provides an MCP (Model Context Protocol) server for agentdx.
// This allows AI agents to use agentdx as a native tool.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/search"
	"github.com/doveaia/agentdx/store"
	"github.com/doveaia/agentdx/trace"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with agentdx functionality.
type Server struct {
	mcpServer   *server.MCPServer
	projectRoot string
}

// SearchResult is a lightweight struct for MCP output.
type SearchResult struct {
	FilePath  string  `json:"file_path"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
	Content   string  `json:"content"`
}

// IndexStatus represents the current state of the index.
type IndexStatus struct {
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

// FileResult is the output struct for the files tool.
type FileResult struct {
	Path    string `json:"path"`
	ModTime string `json:"mod_time,omitempty"`
}

// NewServer creates a new MCP server for agentdx.
func NewServer(projectRoot string) (*Server, error) {
	s := &Server{
		projectRoot: projectRoot,
	}

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"agentdx",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Register tools
	s.registerTools()

	return s, nil
}

// registerTools registers all agentdx tools with the MCP server.
func (s *Server) registerTools() {
	// agentdx_search tool
	searchTool := mcp.NewTool("agentdx_search",
		mcp.WithDescription("Semantic code search. Search your codebase using natural language queries. Returns the most relevant code chunks with file paths, line numbers, and similarity scores."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Natural language search query (e.g., 'user authentication flow', 'error handling middleware')"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 10)"),
		),
	)
	s.mcpServer.AddTool(searchTool, s.handleSearch)

	// agentdx_trace_callers tool
	traceCallersTool := mcp.NewTool("agentdx_trace_callers",
		mcp.WithDescription("Find all functions that call the specified symbol. Useful for understanding code dependencies before modifying a function."),
		mcp.WithString("symbol",
			mcp.Required(),
			mcp.Description("Name of the function/method to find callers for"),
		),
	)
	s.mcpServer.AddTool(traceCallersTool, s.handleTraceCallers)

	// agentdx_trace_callees tool
	traceCalleesTool := mcp.NewTool("agentdx_trace_callees",
		mcp.WithDescription("Find all functions called by the specified symbol. Useful for understanding what a function depends on."),
		mcp.WithString("symbol",
			mcp.Required(),
			mcp.Description("Name of the function/method to find callees for"),
		),
	)
	s.mcpServer.AddTool(traceCalleesTool, s.handleTraceCallees)

	// agentdx_trace_graph tool
	traceGraphTool := mcp.NewTool("agentdx_trace_graph",
		mcp.WithDescription("Build a complete call graph around a symbol showing both callers and callees up to a specified depth."),
		mcp.WithString("symbol",
			mcp.Required(),
			mcp.Description("Name of the function/method to build graph for"),
		),
		mcp.WithNumber("depth",
			mcp.Description("Maximum depth for graph traversal (default: 2)"),
		),
	)
	s.mcpServer.AddTool(traceGraphTool, s.handleTraceGraph)

	// agentdx_index_status tool
	indexStatusTool := mcp.NewTool("agentdx_index_status",
		mcp.WithDescription("Check the health and status of the agentdx index. Returns statistics about indexed files, chunks, and configuration."),
	)
	s.mcpServer.AddTool(indexStatusTool, s.handleIndexStatus)

	// agentdx_files tool
	filesTool := mcp.NewTool("agentdx_files",
		mcp.WithDescription("List indexed files matching a glob pattern. Patterns without path separators are matched recursively by default (e.g., '*.go' matches all Go files). Use explicit paths to limit scope (e.g., 'internal/**', 'cli/*.go')."),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Glob pattern to match files (e.g., '*.go', '**/*.test.ts', 'internal/**')"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 0 = unlimited)"),
		),
	)
	s.mcpServer.AddTool(filesTool, s.handleFiles)
}

// handleSearch handles the agentdx_search tool call.
func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	limit := request.GetInt("limit", 10)
	if limit <= 0 {
		limit = 10
	}

	// Load configuration
	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load configuration: %v", err)), nil
	}

	// Initialize PostgreSQL FTS store
	ftsStore, err := store.NewPostgresFTSStore(ctx, cfg.Index.Store.Postgres.DSN, s.projectRoot)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to initialize store: %v", err)), nil
	}
	defer ftsStore.Close()

	// Search using FTS
	results, err := ftsStore.SearchFTS(ctx, query, limit*2)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// Apply structural boosting
	results = search.ApplyBoost(results, cfg.Index.Search.Boost)

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

	// Return JSON result
	jsonBytes, err := json.MarshalIndent(searchResults, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleTraceCallers handles the agentdx_trace_callers tool call.
func (s *Server) handleTraceCallers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbolName, err := request.RequireString("symbol")
	if err != nil {
		return mcp.NewToolResultError("symbol parameter is required"), nil
	}

	// Initialize symbol store
	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(s.projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load symbol index: %v. Run 'agentdx watch' first", err)), nil
	}
	defer symbolStore.Close()

	// Check if index exists
	stats, err := symbolStore.GetStats(ctx)
	if err != nil || stats.TotalSymbols == 0 {
		return mcp.NewToolResultError("symbol index is empty. Run 'agentdx watch' first to build the index"), nil
	}

	// Lookup symbol
	symbols, err := symbolStore.LookupSymbol(ctx, symbolName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to lookup symbol: %v", err)), nil
	}

	if len(symbols) == 0 {
		result := trace.TraceResult{Query: symbolName, Mode: "fast"}
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}

	// Find callers
	refs, err := symbolStore.LookupCallers(ctx, symbolName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to lookup callers: %v", err)), nil
	}

	result := trace.TraceResult{
		Query:  symbolName,
		Mode:   "fast",
		Symbol: &symbols[0],
	}

	// Convert refs to CallerInfo
	for _, ref := range refs {
		callerSyms, _ := symbolStore.LookupSymbol(ctx, ref.CallerName)
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

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleTraceCallees handles the agentdx_trace_callees tool call.
func (s *Server) handleTraceCallees(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbolName, err := request.RequireString("symbol")
	if err != nil {
		return mcp.NewToolResultError("symbol parameter is required"), nil
	}

	// Initialize symbol store
	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(s.projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load symbol index: %v. Run 'agentdx watch' first", err)), nil
	}
	defer symbolStore.Close()

	// Check if index exists
	stats, err := symbolStore.GetStats(ctx)
	if err != nil || stats.TotalSymbols == 0 {
		return mcp.NewToolResultError("symbol index is empty. Run 'agentdx watch' first to build the index"), nil
	}

	// Lookup symbol
	symbols, err := symbolStore.LookupSymbol(ctx, symbolName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to lookup symbol: %v", err)), nil
	}

	if len(symbols) == 0 {
		result := trace.TraceResult{Query: symbolName, Mode: "fast"}
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}

	// Find callees
	refs, err := symbolStore.LookupCallees(ctx, symbolName, symbols[0].File)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to lookup callees: %v", err)), nil
	}

	result := trace.TraceResult{
		Query:  symbolName,
		Mode:   "fast",
		Symbol: &symbols[0],
	}

	for _, ref := range refs {
		calleeSyms, _ := symbolStore.LookupSymbol(ctx, ref.SymbolName)
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

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleTraceGraph handles the agentdx_trace_graph tool call.
func (s *Server) handleTraceGraph(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbolName, err := request.RequireString("symbol")
	if err != nil {
		return mcp.NewToolResultError("symbol parameter is required"), nil
	}

	depth := request.GetInt("depth", 2)
	if depth <= 0 {
		depth = 2
	}

	// Initialize symbol store
	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(s.projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load symbol index: %v. Run 'agentdx watch' first", err)), nil
	}
	defer symbolStore.Close()

	// Check if index exists
	stats, err := symbolStore.GetStats(ctx)
	if err != nil || stats.TotalSymbols == 0 {
		return mcp.NewToolResultError("symbol index is empty. Run 'agentdx watch' first to build the index"), nil
	}

	graph, err := symbolStore.GetCallGraph(ctx, symbolName, depth)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build call graph: %v", err)), nil
	}

	result := trace.TraceResult{
		Query: symbolName,
		Mode:  "fast",
		Graph: graph,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleIndexStatus handles the agentdx_index_status tool call.
func (s *Server) handleIndexStatus(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Load configuration
	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load configuration: %v", err)), nil
	}

	// Initialize PostgreSQL FTS store
	st, err := store.NewPostgresFTSStore(ctx, cfg.Index.Store.Postgres.DSN, s.projectRoot)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to initialize store: %v", err)), nil
	}
	defer st.Close()

	// Get stats
	stats, err := st.GetStats(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	// Check symbol index
	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(s.projectRoot))
	symbolsReady := false
	if err := symbolStore.Load(ctx); err == nil {
		if symbolStats, err := symbolStore.GetStats(ctx); err == nil && symbolStats.TotalSymbols > 0 {
			symbolsReady = true
		}
		symbolStore.Close()
	}

	// Get backend status
	var backendType, backendHost, backendName string
	var backendOK bool
	if status := st.BackendStatus(ctx); status != nil {
		backendType = status.Type
		backendHost = status.Host
		backendName = status.Name
		backendOK = status.Healthy
	}

	status := IndexStatus{
		TotalFiles:   stats.TotalFiles,
		TotalChunks:  stats.TotalChunks,
		IndexSize:    formatBytes(stats.IndexSize),
		LastUpdated:  stats.LastUpdated.Format("2006-01-02 15:04:05"),
		Search:       "PostgreSQL FTS",
		SymbolsReady: symbolsReady,
		BackendType:  backendType,
		BackendHost:  backendHost,
		BackendName:  backendName,
		BackendOK:    backendOK,
	}

	jsonBytes, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal status: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleFiles handles the agentdx_files tool call.
func (s *Server) handleFiles(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern, err := request.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError("pattern parameter is required"), nil
	}

	limit := request.GetInt("limit", 0)

	// Load configuration
	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load configuration: %v", err)), nil
	}

	// Initialize PostgreSQL FTS store
	st, err := store.NewPostgresFTSStore(ctx, cfg.Index.Store.Postgres.DSN, s.projectRoot)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to connect to postgres: %v", err)), nil
	}
	defer st.Close()

	// Get all files with stats
	allFiles, err := st.ListFilesWithStats(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list files: %v", err)), nil
	}

	// Filter by glob pattern
	matched, err := filterFilesByGlob(allFiles, pattern)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid glob pattern: %v", err)), nil
	}

	// Sort alphabetically by path
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Path < matched[j].Path
	})

	// Apply limit if specified
	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}

	// Convert to FileResult
	results := make([]FileResult, len(matched))
	for i, f := range matched {
		results[i] = FileResult{
			Path:    f.Path,
			ModTime: f.ModTime.Format("2006-01-02T15:04:05Z"),
		}
	}

	jsonBytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// normalizeGlobPattern makes patterns without path separators recursive by default.
// "*.go" becomes "**/*.go" to match all Go files recursively.
// Patterns with "/" or "**" are left unchanged.
func normalizeGlobPattern(pattern string) string {
	if strings.Contains(pattern, "/") || strings.Contains(pattern, "**") {
		return pattern
	}
	return "**/" + pattern
}

// filterFilesByGlob filters files by glob pattern using doublestar.
func filterFilesByGlob(files []store.FileStats, pattern string) ([]store.FileStats, error) {
	normalizedPattern := normalizeGlobPattern(pattern)

	var matched []store.FileStats
	for _, f := range files {
		ok, err := doublestar.Match(normalizedPattern, f.Path)
		if err != nil {
			return nil, err
		}
		if ok {
			matched = append(matched, f)
		}
	}
	return matched, nil
}

func formatBytes(b int64) string {
	if b == 0 {
		return "N/A"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Serve starts the MCP server using stdio transport.
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcpServer)
}
