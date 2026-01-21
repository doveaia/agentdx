package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/indexer"
	"github.com/doveaia/agentdx/localsetup"
	"github.com/spf13/cobra"
)

var (
	initNonInteractive bool
	initLocal          bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize agentdx in the current directory",
	Long: `Initialize agentdx by creating a .agentdx directory with configuration.

This command will:
- Create .agentdx/config.yaml with PostgreSQL Full Text Search settings
- Auto-configure PostgreSQL via Docker if available
- Prompt for PostgreSQL DSN if Docker is not available
- Add .agentdx/ to .gitignore if present`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initNonInteractive, "yes", false, "Use defaults without prompting")
	initCmd.Flags().BoolVarP(&initLocal, "local", "l", false, "Non-interactive local setup with PostgreSQL FTS")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Handle --local flag
	if initLocal {
		return runLocalInit(cwd)
	}

	// Check if already initialized
	if config.Exists(cwd) {
		fmt.Println("agentdx is already initialized in this directory.")
		fmt.Printf("Configuration: %s\n", config.GetConfigPath(cwd))
		return nil
	}

	cfg := config.DefaultConfig()

	// Always use PostgreSQL FTS (configured in DefaultConfig)

	// Interactive mode
	if !initNonInteractive {
		reader := bufio.NewReader(os.Stdin)

		// Attempt PostgreSQL auto-setup
		result, err := setupPostgresBackend(cwd)
		if err != nil {
			fmt.Printf("Warning: auto PostgreSQL setup failed: %v\n", err)
			fmt.Println("Falling back to manual DSN configuration.")
		}

		if result != nil {
			cfg.Index.Store.Postgres.DSN = result.DSN
			fmt.Printf("\nAuto-configured PostgreSQL FTS (container: %s)\n", result.ContainerName)
			fmt.Printf("  DSN: %s\n", result.DSN)
		} else {
			// Docker unavailable - prompt for DSN
			fmt.Print("\nPostgreSQL DSN (required for FTS): ")
			dsn, _ := reader.ReadString('\n')
			cfg.Index.Store.Postgres.DSN = strings.TrimSpace(dsn)
		}
	} else {
		// Non-interactive mode - require Docker
		result, err := setupPostgresBackend(cwd)
		if err != nil {
			return fmt.Errorf("PostgreSQL auto-setup failed: %w", err)
		}
		if result == nil {
			return fmt.Errorf(`PostgreSQL backend requires Docker for automatic setup.

Options:
  1. Install Docker and ensure it's running
  2. Use interactive mode: agentdx init
  3. Use local setup: agentdx init -l`)
		}
		cfg.Index.Store.Postgres.DSN = result.DSN
		fmt.Printf("Auto-configured PostgreSQL FTS (container: %s)\n", result.ContainerName)
	}

	// Save configuration
	if err := cfg.Save(cwd); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("\nCreated configuration at %s\n", config.GetConfigPath(cwd))

	// Add .agentdx/ to .gitignore
	gitignorePath := cwd + "/.gitignore"
	if _, err := os.Stat(gitignorePath); err == nil {
		if err := indexer.AddToGitignore(cwd, ".agentdx/"); err != nil {
			fmt.Printf("Warning: could not update .gitignore: %v\n", err)
		} else {
			fmt.Println("Added .agentdx/ to .gitignore")
		}
	}

	fmt.Println("\nagentdx initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Start the indexing daemon: agentdx watch")
	fmt.Println("  2. Search your code: agentdx search \"your query\"")
	fmt.Println("  3. Configure your AI agent: agentdx setup")

	fmt.Println("\nUsing PostgreSQL Full Text Search (no external embedding service needed).")

	return nil
}

// setupPostgresBackend attempts to set up PostgreSQL using Docker.
// Returns SetupResult with setup details (compose.yaml always generated).
// Returns nil, error if setup fails.
func setupPostgresBackend(cwd string) (*localsetup.SetupResult, error) {
	result, err := localsetup.RunLocalSetup(cwd)
	if err != nil {
		return nil, fmt.Errorf("auto PostgreSQL setup failed: %w", err)
	}

	// If Docker was not available, return nil to signal caller should prompt for DSN
	// (compose.yaml has already been generated for manual setup)
	if !result.DockerUsed {
		return nil, nil
	}

	return result, nil
}

// runLocalInit handles the --local flag for non-interactive local PostgreSQL setup.
func runLocalInit(cwd string) error {
	// Check if already initialized (same check as interactive mode)
	if config.Exists(cwd) {
		fmt.Println("agentdx is already initialized in this directory.")
		fmt.Printf("Configuration: %s\n", config.GetConfigPath(cwd))
		return nil
	}

	fmt.Println("Initializing agentdx with local PostgreSQL setup...")

	// Run the local setup
	result, err := localsetup.RunLocalSetup(cwd)
	if err != nil {
		return fmt.Errorf("local setup failed: %w", err)
	}

	// Create and configure the config
	cfg := config.DefaultConfig()
	cfg.Mode = "local"
	cfg.Index.Store.Postgres.DSN = result.DSN

	// Save configuration
	if err := cfg.Save(cwd); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("\nCreated configuration at %s\n", config.GetConfigPath(cwd))

	// Add .agentdx/ to .gitignore
	gitignorePath := cwd + "/.gitignore"
	if _, err := os.Stat(gitignorePath); err == nil {
		if err := indexer.AddToGitignore(cwd, ".agentdx/"); err != nil {
			fmt.Printf("Warning: could not update .gitignore: %v\n", err)
		} else {
			fmt.Println("Added .agentdx/ to .gitignore")
		}
	}

	// Print results
	if result.DockerUsed {
		fmt.Println("\nagentdx initialized successfully!")
		fmt.Printf("  Container: %s (running)\n", result.ContainerName)
		fmt.Printf("  Database:  %s\n", result.DatabaseName)
		fmt.Printf("  DSN:       %s\n", result.DSN)
	} else {
		fmt.Println("\nagentdx initialized (Docker not available).")
		fmt.Printf("  Database:  %s (needs manual creation)\n", result.DatabaseName)
		fmt.Printf("  DSN:       %s\n", result.DSN)
		fmt.Println("\nTo set up the database manually:")
		fmt.Println("  1. Install PostgreSQL 17 with pg_search extensions")
		fmt.Println("     See: https://github.com/timescale/pg_textsearch")
		fmt.Println("  2. Or install Docker and run:")
		fmt.Printf("     docker compose -f %s up -d\n", result.ComposeFilePath)
		fmt.Printf("  3. Create database: CREATE DATABASE %s;\n", result.DatabaseName)
	}

	if result.ComposeGenerated {
		fmt.Printf("\nDocker Compose file: %s\n", result.ComposeFilePath)
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Start the indexing daemon: agentdx watch")
	fmt.Println("  2. Search your code: agentdx search \"your query\"")
	fmt.Println("  3. Configure your AI agent: agentdx setup --with-subagent")

	return nil
}
