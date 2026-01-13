package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/indexer"
)

var (
	initProvider       string
	initBackend        string
	initNonInteractive bool
)

const (
	openAI3SmallDimensions      = 1536
	lmStudioEmbeddingDimensions = 768
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize agentdx in the current directory",
	Long: `Initialize agentdx by creating a .agentdx directory with configuration.

This command will:
- Create .agentdx/config.yaml with default settings
- Prompt for embedding provider (Ollama or OpenAI)
- Prompt for storage backend (GOB file or PostgreSQL)
- Add .agentdx/ to .gitignore if present`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVarP(&initProvider, "provider", "p", "", "Embedding provider (ollama, lmstudio, openai, or postgres)")
	initCmd.Flags().StringVarP(&initBackend, "backend", "b", "", "Storage backend (gob or postgres)")
	initCmd.Flags().BoolVar(&initNonInteractive, "yes", false, "Use defaults without prompting")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if already initialized
	if config.Exists(cwd) {
		fmt.Println("agentdx is already initialized in this directory.")
		fmt.Printf("Configuration: %s\n", config.GetConfigPath(cwd))
		return nil
	}

	cfg := config.DefaultConfig()

	// Interactive mode
	if !initNonInteractive {
		reader := bufio.NewReader(os.Stdin)

		// Provider selection
		if initProvider == "" {
			fmt.Println("\nSelect embedding provider:")
			fmt.Println("  1) ollama (local, privacy-first, requires Ollama running)")
			fmt.Println("  2) lmstudio (local, OpenAI-compatible, requires LM Studio running)")
			fmt.Println("  3) openai (cloud, requires API key)")
			fmt.Println("  4) postgres (PostgreSQL 15+ Full Text Search, no ML model needed)")
			fmt.Print("Choice [1]: ")

			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch input {
			case "2", "lmstudio":
				cfg.Index.Embedder.Provider = "lmstudio"
				cfg.Index.Embedder.Model = "text-embedding-nomic-embed-text-v1.5"
				cfg.Index.Embedder.Endpoint = "http://127.0.0.1:1234"
				cfg.Index.Embedder.Dimensions = lmStudioEmbeddingDimensions
			case "3", "openai":
				cfg.Index.Embedder.Provider = "openai"
				cfg.Index.Embedder.Model = "text-embedding-3-small"
				cfg.Index.Embedder.Endpoint = "https://api.openai.com/v1"
			case "4", "postgres":
				cfg.Index.Embedder.Provider = "postgres"
				cfg.Index.Embedder.Model = "none"
				cfg.Index.Embedder.Endpoint = "none"
				cfg.Index.Store.Backend = "postgres"
				cfg.Index.Embedder.Dimensions = openAI3SmallDimensions
			default:
				cfg.Index.Embedder.Provider = "ollama"
			}
		} else {
			cfg.Index.Embedder.Provider = initProvider
			switch initProvider {
			case "lmstudio":
				cfg.Index.Embedder.Model = "text-embedding-nomic-embed-text-v1.5"
				cfg.Index.Embedder.Endpoint = "http://127.0.0.1:1234"
				cfg.Index.Embedder.Dimensions = lmStudioEmbeddingDimensions
			case "openai":
				cfg.Index.Embedder.Model = "text-embedding-3-small"
				cfg.Index.Embedder.Endpoint = "https://api.openai.com/v1"
			case "postgres":
				cfg.Index.Embedder.Model = "none"
				cfg.Index.Embedder.Endpoint = "none"
				cfg.Index.Store.Backend = "postgres"
				cfg.Index.Embedder.Dimensions = openAI3SmallDimensions
			}
		}

		// Backend selection (skip if postgres provider was selected - it forces postgres backend)
		if initBackend == "" && cfg.Index.Embedder.Provider != "postgres" {
			fmt.Println("\nSelect storage backend:")
			fmt.Println("  1) gob (local file, recommended for most projects)")
			fmt.Println("  2) postgres (pgvector, for large monorepos or shared index)")
			fmt.Print("Choice [1]: ")

			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch input {
			case "2", "postgres":
				cfg.Index.Store.Backend = "postgres"
				fmt.Print("PostgreSQL DSN: ")
				dsn, _ := reader.ReadString('\n')
				cfg.Index.Store.Postgres.DSN = strings.TrimSpace(dsn)
			default:
				cfg.Index.Store.Backend = "gob"
			}
		} else if cfg.Index.Embedder.Provider == "postgres" {
			// PostgreSQL FTS requires PostgreSQL backend
			fmt.Print("\nPostgreSQL DSN (required for FTS): ")
			dsn, _ := reader.ReadString('\n')
			cfg.Index.Store.Postgres.DSN = strings.TrimSpace(dsn)
		} else {
			cfg.Index.Store.Backend = initBackend
		}
	} else {
		// Non-interactive with flags
		if initProvider != "" {
			cfg.Index.Embedder.Provider = initProvider
			if initProvider == "postgres" {
				cfg.Index.Embedder.Model = "none"
				cfg.Index.Embedder.Endpoint = "none"
				cfg.Index.Store.Backend = "postgres"
			}
		}
		if initBackend != "" {
			cfg.Index.Store.Backend = initBackend
		}
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

	switch cfg.Index.Embedder.Provider {
	case "ollama":
		fmt.Println("\nMake sure Ollama is running with the nomic-embed-text model:")
		fmt.Println("  ollama pull nomic-embed-text")
	case "lmstudio":
		fmt.Println("\nMake sure LM Studio is running with an embedding model loaded.")
		fmt.Printf("  Model: %s\n", cfg.Index.Embedder.Model)
		fmt.Printf("  Endpoint: %s\n", cfg.Index.Embedder.Endpoint)
	case "openai":
		fmt.Println("\nMake sure OPENAI_API_KEY is set in your environment.")
	case "postgres":
		fmt.Println("\nUsing PostgreSQL Full Text Search (no external embedding service needed).")
		fmt.Println("Make sure PostgreSQL 15+ is running and accessible.")
	}

	return nil
}
