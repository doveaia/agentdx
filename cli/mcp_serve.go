package cli

import (
	"fmt"

	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/mcp"
	"github.com/spf13/cobra"
)

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start agentdx as an MCP server",
	Long: `Start agentdx as an MCP (Model Context Protocol) server.

This allows AI agents to use agentdx as a native tool through the MCP protocol.
The server communicates via stdio and exposes the following tools:

  - agentdx_search: Semantic code search with natural language
  - agentdx_files: List indexed files matching a glob pattern
  - agentdx_trace_callers: Find all functions that call a symbol
  - agentdx_trace_callees: Find all functions called by a symbol
  - agentdx_trace_graph: Build a call graph around a symbol
  - agentdx_index_status: Check index health and statistics

Configuration for Claude Code:
  claude mcp add agentdx --scope project agentdx serve

Configuration for Cursor (.cursor/mcp.json):
  {
    "mcpServers": {
      "agentdx": {
        "command": "agentdx",
        "args": ["serve"]
      }
    }
  }`,
	RunE: runMCPServe,
}

func init() {
	rootCmd.AddCommand(mcpServeCmd)
}

func runMCPServe(_ *cobra.Command, _ []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Create and start MCP server
	server, err := mcp.NewServer(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	return server.Serve()
}
