package dashboard

import (
	"html/template"
	"net/http"
)

// PageData holds common data for all pages.
type PageData struct {
	Title       string
	CurrentPage string
	ProjectRoot string
}

// IndexData holds data for the index page.
type IndexData struct {
	PageData
	Status *StatusResponse
}

// SearchPageData holds data for the search page.
type SearchPageData struct {
	PageData
	Query   string
	Results []SearchResult
}

// FilesPageData holds data for the files page.
type FilesPageData struct {
	PageData
	Pattern string
	Files   []FileResult
}

// TracePageData holds data for the trace page.
type TracePageData struct {
	PageData
	Symbol string
	Mode   string
	Result *TraceResponse
}

// MCPPageData holds data for the MCP page.
type MCPPageData struct {
	PageData
	Tools        []MCPTool
	DebugCommand string
}

// ProjectsPageData holds data for the projects page.
type ProjectsPageData struct {
	PageData
	Projects       []ProjectResult
	CurrentProject string
}

// MCPTool describes an MCP tool.
type MCPTool struct {
	Name        string
	Description string
	Parameters  []MCPParameter
}

// MCPParameter describes an MCP tool parameter.
type MCPParameter struct {
	Name        string
	Type        string
	Required    bool
	Description string
}

// handleIndex renders the dashboard home page.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := s.getStatus(ctx)

	data := IndexData{
		PageData: PageData{
			Title:       "Dashboard",
			CurrentPage: "index",
			ProjectRoot: s.projectRoot,
		},
		Status: status,
	}

	s.renderTemplate(w, "index.html", data)
}

// handleSearchPage renders the search page.
func (s *Server) handleSearchPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	data := SearchPageData{
		PageData: PageData{
			Title:       "Search",
			CurrentPage: "search",
			ProjectRoot: s.projectRoot,
		},
		Query: query,
	}

	// If query provided, perform search
	if query != "" {
		ctx := r.Context()
		results, err := s.performSearch(ctx, query, 20)
		if err == nil {
			data.Results = results
		}
	}

	s.renderTemplate(w, "search.html", data)
}

// handleFilesPage renders the files page.
func (s *Server) handleFilesPage(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("pattern")

	data := FilesPageData{
		PageData: PageData{
			Title:       "Files",
			CurrentPage: "files",
			ProjectRoot: s.projectRoot,
		},
		Pattern: pattern,
	}

	// If pattern provided, list files
	if pattern != "" {
		ctx := r.Context()
		files, err := s.listFiles(ctx, pattern, 100)
		if err == nil {
			data.Files = files
		}
	}

	s.renderTemplate(w, "files.html", data)
}

// handleTracePage renders the trace page.
func (s *Server) handleTracePage(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "callers"
	}

	data := TracePageData{
		PageData: PageData{
			Title:       "Trace",
			CurrentPage: "trace",
			ProjectRoot: s.projectRoot,
		},
		Symbol: symbol,
		Mode:   mode,
	}

	// If symbol provided, perform trace
	if symbol != "" {
		ctx := r.Context()
		result, err := s.performTrace(ctx, mode, symbol)
		if err == nil {
			data.Result = result
		}
	}

	s.renderTemplate(w, "trace.html", data)
}

// handleMCPPage renders the MCP tools documentation page.
func (s *Server) handleMCPPage(w http.ResponseWriter, r *http.Request) {
	data := MCPPageData{
		PageData: PageData{
			Title:       "MCP Tools",
			CurrentPage: "mcp",
			ProjectRoot: s.projectRoot,
		},
		Tools:        getMCPTools(),
		DebugCommand: "npx @modelcontextprotocol/inspector agentdx serve",
	}

	s.renderTemplate(w, "mcp.html", data)
}

// handleProjectsPage renders the projects page.
func (s *Server) handleProjectsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	projects, _ := s.listProjects(ctx)

	data := ProjectsPageData{
		PageData: PageData{
			Title:       "Projects",
			CurrentPage: "projects",
			ProjectRoot: s.projectRoot,
		},
		Projects:       projects,
		CurrentProject: s.projectRoot,
	}

	s.renderTemplate(w, "projects.html", data)
}

// renderTemplate renders a template with the given data.
func (s *Server) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	tmpl, err := template.ParseFS(templatesFS, "templates/base.html", "templates/"+name, "templates/partials/*.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Template render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// getMCPTools returns the list of available MCP tools.
func getMCPTools() []MCPTool {
	return []MCPTool{
		{
			Name:        "agentdx_search",
			Description: "Semantic code search. Search your codebase using natural language queries.",
			Parameters: []MCPParameter{
				{Name: "query", Type: "string", Required: true, Description: "Natural language search query"},
				{Name: "limit", Type: "number", Required: false, Description: "Maximum results (default: 10)"},
			},
		},
		{
			Name:        "agentdx_trace_callers",
			Description: "Find all functions that call the specified symbol.",
			Parameters: []MCPParameter{
				{Name: "symbol", Type: "string", Required: true, Description: "Function/method name to find callers for"},
			},
		},
		{
			Name:        "agentdx_trace_callees",
			Description: "Find all functions called by the specified symbol.",
			Parameters: []MCPParameter{
				{Name: "symbol", Type: "string", Required: true, Description: "Function/method name to find callees for"},
			},
		},
		{
			Name:        "agentdx_trace_graph",
			Description: "Build a complete call graph around a symbol.",
			Parameters: []MCPParameter{
				{Name: "symbol", Type: "string", Required: true, Description: "Function/method to build graph for"},
				{Name: "depth", Type: "number", Required: false, Description: "Max depth (default: 2)"},
			},
		},
		{
			Name:        "agentdx_index_status",
			Description: "Check the health and status of the agentdx index.",
			Parameters:  []MCPParameter{},
		},
		{
			Name:        "agentdx_files",
			Description: "List indexed files matching a glob pattern.",
			Parameters: []MCPParameter{
				{Name: "pattern", Type: "string", Required: true, Description: "Glob pattern (e.g., '*.go', '**/*.ts')"},
				{Name: "limit", Type: "number", Required: false, Description: "Maximum results (default: unlimited)"},
			},
		},
	}
}
