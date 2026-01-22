package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/dashboard"
	"github.com/doveaia/agentdx/indexer"
	"github.com/doveaia/agentdx/localsetup"
	"github.com/doveaia/agentdx/store"
	"github.com/doveaia/agentdx/trace"
	"github.com/doveaia/agentdx/watcher"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Start the real-time file watcher daemon",
	Long: `Start a background process that monitors file changes and maintains the index.

The watcher will:
- Start a PostgreSQL container if not already running (requires Docker)
- Perform an initial scan comparing disk state with existing index
- Remove obsolete entries and index new files
- Monitor filesystem events (create, modify, delete, rename)
- Apply debouncing (500ms) to batch rapid changes
- Handle atomic updates to avoid duplicate vectors

Container Options:
  --pg-name, -n    Custom container name (default: agentdx-postgres)
  --pg-port, -p    Custom host port (default: 55432)

The PostgreSQL container persists after agentdx exits to preserve your index.`,
	RunE: runWatch,
}

var (
	daemonMode bool
	pgName     string
	pgPort     int
)

func init() {
	watchCmd.Flags().BoolVar(&daemonMode, "daemon", false, "Run in daemon mode (for session management)")
	watchCmd.Flags().StringVarP(&pgName, "pg-name", "n", "", "PostgreSQL container name (default: agentdx-postgres)")
	watchCmd.Flags().IntVarP(&pgPort, "pg-port", "p", 0, "PostgreSQL host port (default: 55432)")
}

// buildContainerOptions builds container options from flags and config.
// Priority: flags > config > defaults
func buildContainerOptions(cfg *config.Config, flagName string, flagPort int) localsetup.ContainerOptions {
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

func runWatch(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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

	// Build container options: flags > config > defaults
	opts := buildContainerOptions(cfg, pgName, pgPort)

	// Ensure PostgreSQL is running
	dsn, err := localsetup.EnsurePostgresRunning(ctx, projectRoot, opts)
	if err != nil {
		return err
	}

	if !daemonMode {
		fmt.Printf("Starting agentdx watch in %s\n", projectRoot)
		fmt.Printf("Backend: PostgreSQL FTS\n")
	}

	// Initialize PostgreSQL FTS store with the DSN from EnsurePostgresRunning
	st, err := store.NewPostgresFTSStore(ctx, dsn, projectRoot)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer st.Close()

	// Initialize ignore matcher
	ignoreMatcher, err := indexer.NewIgnoreMatcher(projectRoot, cfg.Index.Ignore)
	if err != nil {
		return fmt.Errorf("failed to initialize ignore matcher: %w", err)
	}

	// Initialize scanner
	scanner := indexer.NewScanner(projectRoot, ignoreMatcher)

	// Initialize chunker
	chunker := indexer.NewChunker(cfg.Index.Chunking.Size, cfg.Index.Chunking.Overlap)

	// Initialize indexer
	idx := indexer.NewIndexer(projectRoot, st, chunker, scanner)

	// Initialize symbol store and extractor
	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		log.Printf("Warning: failed to load symbol index: %v", err)
	}
	defer symbolStore.Close()

	extractor, err := trace.NewRegexExtractor()
	if err != nil {
		return fmt.Errorf("failed to create symbol extractor: %w", err)
	}

	// Use default trace languages if not configured
	tracedLanguages := cfg.Index.Trace.EnabledLanguages
	if len(tracedLanguages) == 0 {
		tracedLanguages = []string{".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".php", ".java"}
	}

	// Initial scan with progress
	if !daemonMode {
		fmt.Println("\nPerforming initial scan...")
	}
	stats, err := idx.IndexAllWithProgress(ctx, func(info indexer.ProgressInfo) {
		if !daemonMode {
			printProgress(info.Current, info.Total, info.CurrentFile)
		}
	})
	if !daemonMode {
		// Clear progress line
		fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
	}
	if err != nil {
		return fmt.Errorf("initial indexing failed: %w", err)
	}

	if !daemonMode {
		fmt.Printf("Initial scan complete: %d files indexed, %d chunks created, %d files removed, %d skipped (took %s)\n",
			stats.FilesIndexed, stats.ChunksCreated, stats.FilesRemoved, stats.FilesSkipped, stats.Duration.Round(time.Millisecond))
	} else {
		log.Printf("Initial scan complete: %d files indexed, %d chunks created", stats.FilesIndexed, stats.ChunksCreated)
	}

	// Index symbols for traced languages
	if !daemonMode {
		fmt.Println("Building symbol index...")
	}
	symbolCount := 0
	files, _, _ := scanner.Scan()
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Path))
		if !isTracedLanguage(ext, tracedLanguages) {
			continue
		}
		symbols, refs, err := extractor.ExtractAll(ctx, file.Path, file.Content)
		if err != nil {
			log.Printf("Warning: failed to extract symbols from %s: %v", file.Path, err)
			continue
		}
		if err := symbolStore.SaveFile(ctx, file.Path, symbols, refs); err != nil {
			log.Printf("Warning: failed to save symbols for %s: %v", file.Path, err)
		}
		symbolCount += len(symbols)
	}
	if err := symbolStore.Persist(ctx); err != nil {
		log.Printf("Warning: failed to persist symbol index: %v", err)
	}
	if !daemonMode {
		fmt.Printf("Symbol index built: %d symbols extracted\n", symbolCount)
	} else {
		log.Printf("Symbol index built: %d symbols extracted", symbolCount)
	}

	// Start dashboard if enabled
	var dashboardServer *dashboard.Server
	if cfg.Dashboard.Enabled {
		dashboardServer = dashboard.NewServer(cfg, projectRoot, st, symbolStore)
		if err := dashboardServer.Start(ctx); err != nil {
			log.Printf("Warning: failed to start dashboard: %v", err)
		} else {
			if !daemonMode {
				fmt.Printf("Dashboard started at %s\n", dashboardServer.URL())
			} else {
				log.Printf("Dashboard started at %s", dashboardServer.URL())
			}
		}
	}

	// Initialize watcher
	w, err := watcher.NewWatcher(projectRoot, ignoreMatcher, cfg.Index.Watch.DebounceMs)
	if err != nil {
		return fmt.Errorf("failed to initialize watcher: %w", err)
	}
	defer w.Close()

	if err := w.Start(ctx); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	if !daemonMode {
		fmt.Println("\nWatching for changes... (Press Ctrl+C to stop)")
	} else {
		log.Println("Watching for changes...")
	}

	// Event loop
	for {
		select {
		case <-sigChan:
			if !daemonMode {
				fmt.Println("\nShutting down...")
			} else {
				log.Println("Shutting down...")
			}
			// Stop dashboard
			if dashboardServer != nil {
				if err := dashboardServer.Stop(ctx); err != nil {
					log.Printf("Warning: failed to stop dashboard: %v", err)
				}
			}
			if err := symbolStore.Persist(ctx); err != nil {
				log.Printf("Warning: failed to persist symbol index on shutdown: %v", err)
			}
			return nil

		case event := <-w.Events():
			handleFileEvent(ctx, idx, scanner, extractor, symbolStore, tracedLanguages, event)
		}
	}
}

func handleFileEvent(ctx context.Context, idx *indexer.Indexer, scanner *indexer.Scanner, extractor trace.SymbolExtractor, symbolStore *trace.GOBSymbolStore, enabledLanguages []string, event watcher.FileEvent) {
	log.Printf("[%s] %s", event.Type, event.Path)

	switch event.Type {
	case watcher.EventCreate, watcher.EventModify:
		fileInfo, err := scanner.ScanFile(event.Path)
		if err != nil {
			log.Printf("Failed to scan %s: %v", event.Path, err)
			return
		}
		if fileInfo == nil {
			return // File was skipped (binary, too large, etc.)
		}

		chunks, err := idx.IndexFile(ctx, *fileInfo)
		if err != nil {
			log.Printf("Failed to index %s: %v", event.Path, err)
			return
		}
		log.Printf("Indexed %s (%d chunks)", event.Path, chunks)

		// Extract symbols if language is supported
		ext := strings.ToLower(filepath.Ext(event.Path))
		if isTracedLanguage(ext, enabledLanguages) {
			symbols, refs, err := extractor.ExtractAll(ctx, fileInfo.Path, fileInfo.Content)
			if err != nil {
				log.Printf("Failed to extract symbols from %s: %v", event.Path, err)
			} else if err := symbolStore.SaveFile(ctx, fileInfo.Path, symbols, refs); err != nil {
				log.Printf("Failed to save symbols for %s: %v", event.Path, err)
			} else {
				log.Printf("Extracted %d symbols from %s", len(symbols), event.Path)
			}
		}

	case watcher.EventDelete, watcher.EventRename:
		if err := idx.RemoveFile(ctx, event.Path); err != nil {
			log.Printf("Failed to remove %s from index: %v", event.Path, err)
			return
		}
		// Also remove from symbol index
		if err := symbolStore.DeleteFile(ctx, event.Path); err != nil {
			log.Printf("Failed to remove symbols for %s: %v", event.Path, err)
		}
		log.Printf("Removed %s from index", event.Path)
	}
}

// isTracedLanguage checks if a file extension is in the enabled languages list.
func isTracedLanguage(ext string, enabledLanguages []string) bool {
	for _, lang := range enabledLanguages {
		if ext == lang {
			return true
		}
	}
	return false
}

// printProgress displays a progress bar for indexing
func printProgress(current, total int, filePath string) {
	if total == 0 {
		return
	}

	// Calculate percentage
	percent := float64(current) / float64(total) * 100

	// Build progress bar (20 chars width)
	barWidth := 20
	filled := int(float64(barWidth) * float64(current) / float64(total))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Truncate file path if too long
	maxPathLen := 35
	displayPath := filePath
	if len(filePath) > maxPathLen {
		displayPath = "..." + filePath[len(filePath)-maxPathLen+3:]
	}

	// Print with carriage return to overwrite previous line
	fmt.Printf("\rIndexing [%s] %3.0f%% (%d/%d) %s", bar, percent, current, total, displayPath)
}
