package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/localsetup"
	"github.com/doveaia/agentdx/session"
	"github.com/spf13/cobra"
)

var (
	quietMode     bool
	forceStop     bool
	jsonOutput    bool
	sessionPgName string
	sessionPgPort int
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage the agentdx watch daemon session",
	Long: `Control the background watch daemon that keeps your code index up-to-date.

The session daemon runs 'agentdx watch' as a background process, automatically
indexing file changes while you work. It's typically managed via coding agent
hooks, but you can also control it manually.

Session State:
  - PID file: .agentdx/session.pid
  - Log file: .agentdx/session.log

The daemon starts automatically when hooks are installed. For manual control,
use the start/stop/status subcommands.`,
}

var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the watch daemon",
	Long: `Start the agentdx watch daemon as a background process.

If the daemon is already running, this command does nothing (idempotent).
If PostgreSQL is not running, it will be started automatically (requires Docker).

Container Options:
  --pg-name, -n    Custom container name (default: agentdx-postgres)
  --pg-port, -p    Custom host port (default: 55432)`,
	Example: `  # Start daemon (typical usage)
  agentdx session start

  # Start with custom container settings
  agentdx session start --pg-name my-project-pg --pg-port 5433

  # Start silently (for scripts/hooks)
  agentdx session start --quiet`,
	RunE: runSessionStart,
}

var sessionStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the watch daemon",
	Long:  `Stop the agentdx watch daemon gracefully. Uses SIGTERM and waits up to 5 seconds. Use --force to send SIGKILL immediately.`,
	Example: `  # Stop gracefully
  agentdx session stop

  # Force stop immediately
  agentdx session stop --force

  # Silent operation
  agentdx session stop --quiet`,
	RunE: runSessionStop,
}

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long:  `Display the current status of the watch daemon, including whether it's running, its PID, uptime, and log file location.`,
	Example: `  # Human-readable status
  agentdx session status

  # JSON output for scripts
  agentdx session status --json`,
	RunE: runSessionStatus,
}

func init() {
	// session start flags
	sessionStartCmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "Suppress output")
	sessionStartCmd.Flags().StringVarP(&sessionPgName, "pg-name", "n", "", "PostgreSQL container name (default: agentdx-postgres)")
	sessionStartCmd.Flags().IntVarP(&sessionPgPort, "pg-port", "p", 0, "PostgreSQL host port (default: 55432)")

	// session stop flags
	sessionStopCmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "Suppress output")
	sessionStopCmd.Flags().BoolVarP(&forceStop, "force", "f", false, "Force kill with SIGKILL")

	// session status flags
	sessionStatusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	// Register subcommands
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStopCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
}

// buildSessionContainerOptions builds container options from flags and config.
// Priority: flags > config > defaults
func buildSessionContainerOptions(cfg *config.Config, flagName string, flagPort int) localsetup.ContainerOptions {
	// Start with defaults
	opts := localsetup.DefaultContainerOptions()

	// Apply config values (if set)
	if cfg.Index.Store.Postgres.ContainerName != "" {
		opts.Name = cfg.Index.Store.Postgres.ContainerName
	}
	if cfg.Index.Store.Postgres.Port != 0 {
		opts.Port = cfg.Index.Store.Postgres.Port
	}

	// Apply flag values (highest priority)
	if flagName != "" {
		opts.Name = flagName
	}
	if flagPort != 0 {
		opts.Port = flagPort
	}

	return opts
}

func runSessionStart(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		if !quietMode {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return err
	}

	// Load configuration
	cfg, err := config.Load(projectRoot)
	if err != nil {
		if !quietMode {
			fmt.Fprintf(os.Stderr, "Error: failed to load configuration: %v\n", err)
		}
		return err
	}

	// Build container options: flags > config > defaults
	opts := buildSessionContainerOptions(cfg, sessionPgName, sessionPgPort)

	// Ensure PostgreSQL is running BEFORE starting daemon
	_, err = localsetup.EnsurePostgresRunning(ctx, projectRoot, opts)
	if err != nil {
		if !quietMode {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return err
	}

	// Create daemon manager with container options
	dm := session.NewDaemonManagerWithOptions(projectRoot, session.DaemonOptions{
		PgName: opts.Name,
		PgPort: opts.Port,
	})

	// Check if already running
	wasRunning, err := dm.IsRunning()
	if err != nil && !quietMode {
		fmt.Fprintf(os.Stderr, "Warning: failed to check daemon status: %v\n", err)
	}

	// Start the daemon
	if err := dm.Start(ctx); err != nil {
		if !quietMode {
			fmt.Fprintf(os.Stderr, "Error: failed to start daemon: %v\n", err)
		}
		return err
	}

	// Print status message unless quiet
	if !quietMode {
		if wasRunning {
			status, _ := dm.Status()
			fmt.Printf("Session daemon already running (PID: %d)\n", status.PID)
		} else {
			status, _ := dm.Status()
			fmt.Printf("Session daemon started (PID: %d)\n", status.PID)
		}
	}

	return nil
}

func runSessionStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		if !quietMode {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return err
	}

	// Create daemon manager
	dm := session.NewDaemonManager(projectRoot)

	// Check if running before stopping
	wasRunning, err := dm.IsRunning()
	if err != nil && !quietMode {
		fmt.Fprintf(os.Stderr, "Warning: failed to check daemon status: %v\n", err)
	}

	// Stop the daemon
	if err := dm.Stop(ctx, forceStop); err != nil {
		if !quietMode {
			fmt.Fprintf(os.Stderr, "Error: failed to stop daemon: %v\n", err)
		}
		return err
	}

	// Print status message unless quiet
	if !quietMode {
		if wasRunning {
			fmt.Println("Session daemon stopped")
		} else {
			fmt.Println("Session daemon not running")
		}
	}

	return nil
}

func runSessionStatus(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not an agentdx project: %w", err)
	}

	// Get daemon status
	dm := session.NewDaemonManager(projectRoot)
	status, err := dm.Status()
	if err != nil {
		return fmt.Errorf("failed to get daemon status: %w", err)
	}

	// Output based on format flag
	if jsonOutput {
		return outputStatusJSON(status)
	}

	return outputStatusHuman(status)
}

func outputStatusHuman(status session.DaemonStatus) error {
	if status.Running {
		relativePath := relativeLogPath(status.LogFile)
		fmt.Printf("agentdx session daemon: running\n")
		fmt.Printf("PID: %d\n", status.PID)
		if !status.StartTime.IsZero() {
			uptime := time.Since(status.StartTime)
			fmt.Printf("Uptime: %s\n", formatUptime(uptime))
		}
		fmt.Printf("Log: %s\n", relativePath)
		return nil
	}

	fmt.Println("agentdx session daemon: not running")
	return nil
}

func outputStatusJSON(status session.DaemonStatus) error {
	// Create a simplified JSON output
	output := map[string]any{
		"running": status.Running,
	}

	if status.Running {
		output["pid"] = status.PID
		if !status.StartTime.IsZero() {
			output["start_time"] = status.StartTime.Format(time.RFC3339)
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "")
	return encoder.Encode(output)
}

// relativeLogPath converts absolute log path to relative path for display
func relativeLogPath(logPath string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return logPath
	}

	// Try to make it relative
	rel, err := relativePath(cwd, logPath)
	if err != nil {
		return logPath
	}
	return rel
}

// relativePath returns a relative path from base to target
func relativePath(base, target string) (string, error) {
	// Simple implementation - if target starts with base, return the relative part
	if len(target) > len(base) && target[:len(base)] == base {
		if target[len(base)] == '/' {
			return "." + target[len(base):], nil
		}
	}
	return target, nil
}

// formatUptime formats a duration for human display
func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int((d - time.Duration(hours)*time.Hour).Minutes())
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%d days, %d hours", days, hours)
}
