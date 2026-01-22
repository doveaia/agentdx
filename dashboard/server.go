// Package dashboard provides a web dashboard for agentdx.
package dashboard

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/store"
	"github.com/doveaia/agentdx/trace"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed templates/*.html templates/partials/*.html
var templatesFS embed.FS

// Server is the web dashboard server.
type Server struct {
	config      *config.Config
	projectRoot string
	store       *store.PostgresFTSStore
	symbolStore *trace.GOBSymbolStore
	httpServer  *http.Server
	router      *chi.Mux
	sseHub      *SSEHub
	mu          sync.RWMutex
	running     bool
}

// NewServer creates a new dashboard server.
func NewServer(cfg *config.Config, projectRoot string, st *store.PostgresFTSStore, symbolStore *trace.GOBSymbolStore) *Server {
	s := &Server{
		config:      cfg,
		projectRoot: projectRoot,
		store:       st,
		symbolStore: symbolStore,
		sseHub:      NewSSEHub(),
	}

	s.router = s.setupRouter()
	return s
}

// setupRouter configures all routes.
func (s *Server) setupRouter() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Page routes
	r.Get("/", s.handleIndex)
	r.Get("/search", s.handleSearchPage)
	r.Get("/files", s.handleFilesPage)
	r.Get("/trace", s.handleTracePage)
	r.Get("/mcp", s.handleMCPPage)
	r.Get("/projects", s.handleProjectsPage)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/search", s.handleAPISearch)
		r.Get("/files", s.handleAPIFiles)
		r.Get("/status", s.handleAPIStatus)
		r.Get("/trace/{mode}/{symbol}", s.handleAPITrace)
		r.Get("/projects", s.handleAPIProjects)
	})

	// SSE route
	r.Get("/events/status", s.handleSSEStatus)

	// Static assets (htmx, css)
	r.Get("/static/*", s.handleStatic)

	return r
}

// Start starts the dashboard server.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.config.Dashboard.Host, s.config.Dashboard.Port)

	// Check if port is available
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to bind to %s: %w", addr, err)
	}
	ln.Close()

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.running = true
	s.mu.Unlock()

	// Start SSE hub
	go s.sseHub.Run(ctx)

	// Start status broadcaster
	go s.broadcastStatus(ctx)

	log.Printf("Dashboard started at http://%s", addr)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Dashboard server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the dashboard server gracefully.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown dashboard: %w", err)
		}
	}

	log.Println("Dashboard stopped")
	return nil
}

// URL returns the dashboard URL.
func (s *Server) URL() string {
	return fmt.Sprintf("http://%s:%d", s.config.Dashboard.Host, s.config.Dashboard.Port)
}

// broadcastStatus periodically sends status updates via SSE.
func (s *Server) broadcastStatus(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			status := s.getStatus(ctx)
			s.sseHub.Broadcast("status", status)
		}
	}
}

// handleStatic serves static assets.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Inline htmx and basic CSS for simplicity
	path := chi.URLParam(r, "*")

	switch path {
	case "htmx.min.js":
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(htmxMinJS))
	case "style.css":
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(dashboardCSS))
	default:
		http.NotFound(w, r)
	}
}

// Inline htmx.min.js (minified version for embedding)
const htmxMinJS = `!function(){var e,t;e=window,t=function(){return function(){"use strict";var e={onLoad:null,process:null,on:null,off:null,trigger:null,ajax:null,find:null,findAll:null,closest:null,values:function(e,t){var r=Er(e,t||"post");return r.values},remove:null,addClass:null,removeClass:null,toggleClass:null,takeClass:null,swap:null,defineExtension:null,removeExtension:null,logAll:null,logNone:null,logger:null,config:{historyEnabled:!0,historyCacheSize:10,refreshOnHistoryMiss:!1,defaultSwapStyle:"innerHTML",defaultSwapDelay:0,defaultSettleDelay:20,includeIndicatorStyles:!0,indicatorClass:"htmx-indicator",requestClass:"htmx-request",addedClass:"htmx-added",settlingClass:"htmx-settling",swappingClass:"htmx-swapping",allowEval:!0,allowScriptTags:!0,inlineScriptNonce:"",inlineStyleNonce:"",attributesToSettle:["class","style","width","height"],withCredentials:!1,timeout:0,wsReconnectDelay:"full-jitter",wsBinaryType:"blob",disableSelector:"[hx-disable], [data-hx-disable]",scrollBehavior:"instant",defaultFocusScroll:!1,getCacheBusterParam:!1,globalViewTransitions:!1,methodsThatUseUrlParams:["get","delete"],selfRequestsOnly:!0,ignoreTitle:!1,scrollIntoViewOnBoost:!0,triggerSpecsCache:null,disableInheritance:!1,responseHandling:[{code:"204",swap:!1},{code:"[23]..",swap:!0},{code:"[45]..",swap:!1,error:!0}],allowNestedOobSwaps:!0},parseInterval:function(e){if(null!=e)return"ms"==e.slice(-2)?parseFloat(e.slice(0,-2)):"s"==e.slice(-1)?1e3*parseFloat(e.slice(0,-1)):"m"==e.slice(-1)?6e4*parseFloat(e.slice(0,-1)):parseFloat(e)},_:t,createEventSource:function(e){return new EventSource(e,{withCredentials:!0})},createWebSocket:function(e){var t=new WebSocket(e,[]);return t.binaryType=e.config.wsBinaryType,t},version:"2.0.4"};return e}()},e.htmx=t(),e.htmx}();`

// Dashboard CSS styles
const dashboardCSS = `
:root {
  --bg-primary: #0f172a;
  --bg-secondary: #1e293b;
  --bg-tertiary: #334155;
  --text-primary: #f8fafc;
  --text-secondary: #94a3b8;
  --accent: #3b82f6;
  --accent-hover: #2563eb;
  --success: #22c55e;
  --warning: #f59e0b;
  --error: #ef4444;
  --border: #475569;
}

* { box-sizing: border-box; margin: 0; padding: 0; }

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  line-height: 1.6;
}

.container { max-width: 1200px; margin: 0 auto; padding: 1rem; }

nav {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  padding: 1rem;
}

nav ul {
  list-style: none;
  display: flex;
  gap: 1.5rem;
  align-items: center;
}

nav a {
  color: var(--text-secondary);
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s;
}

nav a:hover, nav a.active { color: var(--accent); }

.logo { font-weight: 700; font-size: 1.25rem; color: var(--text-primary) !important; }

.card {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  padding: 1.5rem;
  margin-bottom: 1rem;
}

.card h2 { font-size: 1.25rem; margin-bottom: 1rem; color: var(--text-primary); }

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.stat-item {
  background: var(--bg-tertiary);
  padding: 1rem;
  border-radius: 0.375rem;
  text-align: center;
}

.stat-value { font-size: 2rem; font-weight: 700; color: var(--accent); }
.stat-label { font-size: 0.875rem; color: var(--text-secondary); }

input, select, textarea {
  width: 100%;
  padding: 0.75rem 1rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 0.375rem;
  color: var(--text-primary);
  font-size: 1rem;
}

input:focus, select:focus, textarea:focus {
  outline: none;
  border-color: var(--accent);
}

button, .btn {
  display: inline-block;
  padding: 0.75rem 1.5rem;
  background: var(--accent);
  color: white;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-weight: 500;
  text-decoration: none;
  transition: background 0.2s;
}

button:hover, .btn:hover { background: var(--accent-hover); }

.search-form { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
.search-form input { flex: 1; }

.result-item {
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 0.375rem;
  padding: 1rem;
  margin-bottom: 0.5rem;
}

.result-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.result-path { color: var(--accent); font-weight: 500; }
.result-score { color: var(--text-secondary); font-size: 0.875rem; }
.result-lines { color: var(--text-secondary); font-size: 0.875rem; }

pre, code {
  font-family: 'SF Mono', Monaco, 'Consolas', monospace;
  background: var(--bg-primary);
  border-radius: 0.25rem;
  font-size: 0.875rem;
}

pre {
  padding: 1rem;
  overflow-x: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}

code { padding: 0.125rem 0.25rem; }

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-healthy { background: var(--success); }
.status-unhealthy { background: var(--error); }

.table { width: 100%; border-collapse: collapse; }
.table th, .table td {
  padding: 0.75rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
}
.table th { color: var(--text-secondary); font-weight: 500; }
.table tr:hover { background: var(--bg-tertiary); }

.mcp-command {
  background: var(--bg-primary);
  padding: 1rem;
  border-radius: 0.375rem;
  font-family: monospace;
  position: relative;
}

.copy-btn {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.htmx-indicator { display: none; }
.htmx-request .htmx-indicator { display: inline-block; }
.htmx-request.htmx-indicator { display: inline-block; }

.loading { color: var(--text-secondary); padding: 2rem; text-align: center; }

@media (max-width: 768px) {
  nav ul { flex-wrap: wrap; gap: 0.75rem; }
  .stats-grid { grid-template-columns: 1fr 1fr; }
}
`
