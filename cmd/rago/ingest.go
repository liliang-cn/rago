package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/spf13/cobra"
)

var (
	chunkSize       int
	overlap         int
	batchSize       int
	recursive       bool
	textInput       string
	source          string
	extractMetadata bool
	concurrency     int
)

var ingestCmd = &cobra.Command{
	Use:   "ingest [file/directory]",
	Short: "Import documents into vector database",
	Long: `Chunk document content, vectorize and store into local vector database.
Supports text format files like .txt, .md, .pdf, etc.
You can also use --text flag to ingest text directly.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if textInput != "" {
			if len(args) > 0 {
				return fmt.Errorf("cannot specify both file path and --text flag")
			}
			return nil
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create client
		client, err := client.NewWithConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer client.Close()

		if client.RAG() == nil {
			return fmt.Errorf("RAG service not available")
		}

		ctx := context.Background()

		// Handle text input (not concurrent)
		if textInput != "" {
			return processText(ctx, client, textInput)
		}

		// Handle file path
		if len(args) == 0 {
			return fmt.Errorf("no file path provided")
		}
		path := args[0]

		// Setup for concurrent processing
		var wg sync.WaitGroup
		jobs := make(chan string, 100) // Buffered channel for file paths

		// Start workers
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for filePath := range jobs {
					if !quiet {
						fmt.Printf("Processing: %s\n", filePath)
					}
					err := processFile(ctx, client, filePath)
					if err != nil {
						if !quiet {
							log.Printf("Warning: failed to process %s: %v", filePath, err)
						}
					}
				}
			}()
		}

		// Start producer
		err = processPath(ctx, jobs, path)
		if err != nil {
			// Close channel to unblock workers before returning error
			close(jobs)
			return err
		}

		// Close channel and wait for workers to finish
		close(jobs)
		wg.Wait()

		fmt.Printf("Successfully ingested documents from: %s\n", path)
		return nil
	},
}

func processPath(ctx context.Context, jobs chan<- string, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path %s: %w", path, err)
	}

	if info.IsDir() {
		return processDirectory(ctx, jobs, path)
	}

	// For a single file, just send it to the jobs channel
	jobs <- path
	return nil
}

func processDirectory(ctx context.Context, jobs chan<- string, dirPath string) error {
	if !recursive {
		return fmt.Errorf("directory processing requires --recursive flag")
	}

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if !quiet {
				log.Printf("Warning: failed to access %s: %v", path, err)
			}
			return nil // Continue walking
		}

		if info.IsDir() {
			return nil // Continue walking
		}

		ext := filepath.Ext(path)
		switch ext {
		case ".txt", ".md", ".markdown", ".pdf":
			jobs <- path
		default:
			if verbose {
				log.Printf("Skipping unsupported file: %s", path)
			}
		}

		return nil
	})
}

func processFile(ctx context.Context, c *client.Client, filePath string) error {
	req := core.IngestRequest{
		DocumentID: filePath, // Use file path as document ID
		FilePath:   filePath,
		Metadata: map[string]interface{}{
			"file_path":    filePath,
			"file_ext":     filepath.Ext(filePath),
			"chunk_size":   chunkSize,
			"chunk_overlap": overlap,
		},
	}

	resp, err := c.RAG().IngestDocument(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to ingest file %s: %w", filePath, err)
	}

	if verbose {
		fmt.Printf("Ingested %s: %d chunks (ID: %s)\n", filePath, resp.ChunksCount, resp.DocumentID)
	}

	return nil
}

func processText(ctx context.Context, c *client.Client, text string) error {
	sourceValue := source
	if sourceValue == "" {
		sourceValue = "text-input"
	}

	req := core.IngestRequest{
		DocumentID:   sourceValue,
		Content:      text,
		Metadata: map[string]interface{}{
			"source":       sourceValue,
			"type":         "text",
			"chunk_size":   chunkSize,
			"chunk_overlap": overlap,
		},
	}

	resp, err := c.RAG().IngestDocument(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to ingest text: %w", err)
	}

	fmt.Printf("Successfully ingested text: %d chunks (ID: %s)\n", resp.ChunksCount, resp.DocumentID)
	return nil
}

func init() {
	ingestCmd.Flags().IntVarP(&chunkSize, "chunk-size", "c", 300, "text chunk size")
	ingestCmd.Flags().IntVarP(&overlap, "overlap", "o", 50, "chunk overlap size")
	ingestCmd.Flags().IntVarP(&batchSize, "batch-size", "b", 10, "batch processing size")
	ingestCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "process directory recursively")
	ingestCmd.Flags().StringVar(&textInput, "text", "", "ingest text directly instead of from file")
	ingestCmd.Flags().StringVar(&source, "source", "", "source name for text input (default: text-input)")
	ingestCmd.Flags().BoolVarP(&extractMetadata, "extract-metadata", "e", false, "enable automatic metadata extraction via LLM")
	ingestCmd.Flags().IntVar(&concurrency, "concurrency", runtime.NumCPU(), "number of concurrent workers for ingestion")
}
