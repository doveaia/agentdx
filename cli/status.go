package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/hooks"
	"github.com/doveaia/agentdx/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display index status and browse indexed files",
	Long: `Display statistics about the index and interactively browse indexed files.

Navigation:
  Enter    - Browse files / View chunks
  Esc      - Go back
  Up/Down  - Navigate
  q        - Quit`,
	RunE: runStatus,
}

type viewState int

const (
	viewStats viewState = iota
	viewFiles
	viewChunks
)

type model struct {
	st             *store.PostgresFTSStore
	cfg            *config.Config
	state          viewState
	stats          *store.IndexStats
	files          []store.FileStats
	chunks         []store.Chunk
	selectedFile   int
	selectedChunk  int
	width          int
	height         int
	err            error
	backendType    string
	backendHost    string
	backendName    string
	backendHealthy bool
	hooksStatus    []hookStatus
	detectedAgent  string
}

// hookStatus represents the installation status of hooks for an agent
type hookStatus struct {
	agentName string
	startHook bool
	stopHook  bool
	startPath string
	stopPath  string
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	statusOKStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	statusErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc":
			switch m.state {
			case viewFiles:
				m.state = viewStats
			case viewChunks:
				m.state = viewFiles
			}

		case "enter":
			switch m.state {
			case viewStats:
				m.state = viewFiles
			case viewFiles:
				if len(m.files) > 0 {
					ctx := context.Background()
					chunks, err := m.st.GetChunksForFile(ctx, m.files[m.selectedFile].Path)
					if err != nil {
						m.err = err
					} else {
						m.chunks = chunks
						m.selectedChunk = 0
						m.state = viewChunks
					}
				}
			}

		case "up", "k":
			switch m.state {
			case viewFiles:
				if m.selectedFile > 0 {
					m.selectedFile--
				}
			case viewChunks:
				if m.selectedChunk > 0 {
					m.selectedChunk--
				}
			}

		case "down", "j":
			switch m.state {
			case viewFiles:
				if m.selectedFile < len(m.files)-1 {
					m.selectedFile++
				}
			case viewChunks:
				if m.selectedChunk < len(m.chunks)-1 {
					m.selectedChunk++
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	switch m.state {
	case viewStats:
		return m.viewStats()
	case viewFiles:
		return m.viewFiles()
	case viewChunks:
		return m.viewChunks()
	}

	return ""
}

func (m model) viewStats() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("agentdx index status"))
	sb.WriteString("\n\n")

	sb.WriteString(normalStyle.Render("Files indexed:    "))
	sb.WriteString(fmt.Sprintf("%d\n", m.stats.TotalFiles))

	sb.WriteString(normalStyle.Render("Total chunks:     "))
	sb.WriteString(fmt.Sprintf("%d\n", m.stats.TotalChunks))

	sb.WriteString(normalStyle.Render("Index size:       "))
	sb.WriteString(fmt.Sprintf("%s\n", formatBytes(m.stats.IndexSize)))

	sb.WriteString(normalStyle.Render("Last updated:     "))
	if m.stats.LastUpdated.IsZero() {
		sb.WriteString("Never\n")
	} else {
		sb.WriteString(fmt.Sprintf("%s\n", m.stats.LastUpdated.Format("2006-01-02 15:04:05")))
	}

	sb.WriteString(normalStyle.Render("Search:           "))
	sb.WriteString("PostgreSQL FTS\n")

	// Show backend status for all backends
	if m.backendType != "" {
		sb.WriteString(normalStyle.Render("Backend:          "))
		sb.WriteString(m.backendType)
		if m.backendHost != "" && m.backendName != "" {
			sb.WriteString(fmt.Sprintf(" (%s/%s)", m.backendHost, m.backendName))
		}
		sb.WriteString("\n")

		sb.WriteString(normalStyle.Render("Status:           "))
		if m.backendHealthy {
			sb.WriteString(statusOKStyle.Render("● Connected"))
		} else {
			sb.WriteString(statusErrStyle.Render("● Disconnected"))
		}
		sb.WriteString("\n")
	}

	// Add hooks status section
	sb.WriteString("\n")
	sb.WriteString(normalStyle.Render("Hooks:            "))
	if len(m.hooksStatus) == 0 {
		sb.WriteString("Not configured\n")
		sb.WriteString(dimStyle.Render("                  → Run: agentdx setup\n"))
	} else {
		// Show detected agent if any
		if m.detectedAgent != "" {
			sb.WriteString(fmt.Sprintf("\n%s Agent: %s (detected)\n",
				dimStyle.Render("                  "), statusOKStyle.Render(m.detectedAgent)))
		}
		for i, hs := range m.hooksStatus {
			if i > 0 || m.detectedAgent != "" {
				sb.WriteString(dimStyle.Render("                  "))
			}
			if hs.startHook && hs.stopHook {
				sb.WriteString(statusOKStyle.Render(fmt.Sprintf("● %s\n", hs.agentName)))
			} else if hs.startHook || hs.stopHook {
				sb.WriteString(statusErrStyle.Render(fmt.Sprintf("◐ %s (partial)\n", hs.agentName)))
			} else {
				sb.WriteString(dimStyle.Render(fmt.Sprintf("○ %s (not installed)\n", hs.agentName)))
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("[Enter] Browse files  [q] Quit"))

	return boxStyle.Render(sb.String())
}

func (m model) viewFiles() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(fmt.Sprintf("Indexed Files (%d)", len(m.files))))
	sb.WriteString("\n\n")

	// Calculate visible range
	maxVisible := 15
	if m.height > 0 {
		maxVisible = m.height - 10
	}
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if m.selectedFile >= maxVisible {
		start = m.selectedFile - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.files) {
		end = len(m.files)
	}

	for i := start; i < end; i++ {
		f := m.files[i]
		line := fmt.Sprintf("%-50s %3d chunks", truncatePath(f.Path, 50), f.ChunkCount)

		if i == m.selectedFile {
			sb.WriteString(selectedStyle.Render("> " + line))
		} else {
			sb.WriteString(normalStyle.Render("  " + line))
		}
		sb.WriteString("\n")
	}

	if len(m.files) > maxVisible {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("\n... showing %d-%d of %d files", start+1, end, len(m.files))))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("[Up/Down] Navigate  [Enter] View chunks  [Esc] Back  [q] Quit"))

	return boxStyle.Render(sb.String())
}

func (m model) viewChunks() string {
	var sb strings.Builder

	if len(m.chunks) == 0 {
		sb.WriteString(titleStyle.Render("No chunks"))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("[Esc] Back  [q] Quit"))
		return boxStyle.Render(sb.String())
	}

	filePath := m.files[m.selectedFile].Path
	sb.WriteString(titleStyle.Render(fmt.Sprintf("%s (%d chunks)", filePath, len(m.chunks))))
	sb.WriteString("\n\n")

	chunk := m.chunks[m.selectedChunk]
	sb.WriteString(normalStyle.Render(fmt.Sprintf("Chunk %d/%d  [Lines %d-%d]",
		m.selectedChunk+1, len(m.chunks), chunk.StartLine, chunk.EndLine)))
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render(strings.Repeat("-", 50)))
	sb.WriteString("\n\n")

	// Show chunk content (truncated)
	content := chunk.Content
	// Remove "File: xxx" prefix if present
	if strings.HasPrefix(content, "File: ") {
		if idx := strings.Index(content, "\n\n"); idx != -1 {
			content = content[idx+2:]
		}
	}

	lines := strings.Split(content, "\n")
	maxLines := 12
	if m.height > 0 {
		maxLines = m.height - 15
	}
	if maxLines < 5 {
		maxLines = 5
	}

	for i, line := range lines {
		if i >= maxLines {
			sb.WriteString(dimStyle.Render("..."))
			sb.WriteString("\n")
			break
		}
		// Truncate long lines
		if len(line) > 70 {
			line = line[:67] + "..."
		}
		sb.WriteString(dimStyle.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("[Up/Down] Navigate chunks  [Esc] Back to files  [q] Quit"))

	return boxStyle.Render(sb.String())
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

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
	st, err := store.NewPostgresFTSStore(ctx, cfg.Index.Store.Postgres.DSN, projectRoot)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer st.Close()

	// Get stats
	stats, err := st.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	// Get files
	files, err := st.ListFilesWithStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	// Sort files by path
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Get backend status
	var backendType, backendHost, backendName string
	var backendHealthy bool
	if status := st.BackendStatus(ctx); status != nil {
		backendType = status.Type
		backendHost = status.Host
		backendName = status.Name
		backendHealthy = status.Healthy
	}

	// Get hooks status and detected agent
	cwd, _ := os.Getwd()
	hooksStatus := getProjectHooksStatus(cwd)
	detectedAgent := detectCurrentAgent()

	// Create model
	m := model{
		st:             st,
		cfg:            cfg,
		state:          viewStats,
		stats:          stats,
		files:          files,
		backendType:    backendType,
		backendHost:    backendHost,
		backendName:    backendName,
		backendHealthy: backendHealthy,
		hooksStatus:    hooksStatus,
		detectedAgent:  detectedAgent,
	}

	// Run TUI
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
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

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

// detectCurrentAgent returns the coding agent currently running this session
// It checks environment variables set by coding agents
func detectCurrentAgent() string {
	// Check for Claude Code - this env var is set when running in Claude Code
	if os.Getenv("CLAUDE_CODE_ENTRY") != "" || os.Getenv("CLAUDE_CODE_SESSION") != "" {
		return "claude-code"
	}
	// Add more agent detection as needed
	// if os.Getenv("CODEX_SESSION") != "" {
	//     return "codex"
	// }

	return "" // No agent detected
}

// getProjectHooksStatus returns the hooks installation status for all agents
func getProjectHooksStatus(cwd string) []hookStatus {
	var statuses []hookStatus
	detectedAgent := detectCurrentAgent()

	for _, agent := range hooks.SupportedAgents() {
		status := hookStatus{agentName: agent.Name}

		// Check start hook
		startPath, err := hooks.GetHookPath(agent, "start")
		if err == nil {
			status.startPath = startPath
			if info, err := os.Stat(startPath); err == nil && info.Mode().IsRegular() {
				status.startHook = true
			}
		}

		// Check stop hook
		stopPath, err := hooks.GetHookPath(agent, "stop")
		if err == nil {
			status.stopPath = stopPath
			if info, err := os.Stat(stopPath); err == nil && info.Mode().IsRegular() {
				status.stopHook = true
			}
		}

		// Only include if at least one hook exists or it's the detected agent
		if status.startHook || status.stopHook || agent.Name == detectedAgent {
			statuses = append(statuses, status)
		}
	}

	return statuses
}
