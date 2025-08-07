package rago

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/internal/chunker"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/embedder"
	"github.com/liliang-cn/rago/internal/processor"
	"github.com/liliang-cn/rago/internal/store"
	"github.com/spf13/cobra"
)

var (
	chunkSize int
	overlap   int
	batchSize int
	recursive bool
)

var ingestCmd = &cobra.Command{
	Use:   "ingest [file/directory]",
	Short: "Import documents into vector database",
	Long: `Chunk document content, vectorize and store into local vector database.
Supports text format files like .txt, .md, etc.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		vectorStore, err := store.NewSQLiteStore(
			cfg.Sqvect.DBPath,
			cfg.Sqvect.VectorDim,
			cfg.Sqvect.MaxConns,
			cfg.Sqvect.BatchSize,
		)
		if err != nil {
			return fmt.Errorf("failed to create vector store: %w", err)
		}
		defer func() {
			if closeErr := vectorStore.Close(); closeErr != nil {
				fmt.Printf("Warning: failed to close vector store: %v\n", closeErr)
			}
		}()

		docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

		embedService, err := embedder.NewOllamaService(
			cfg.Ollama.BaseURL,
			cfg.Ollama.EmbeddingModel,
			cfg.Ollama.Timeout,
		)
		if err != nil {
			return fmt.Errorf("failed to create embedder: %w", err)
		}

		chunkerService := chunker.New()

		processor := processor.New(
			embedService,
			nil, // generator not needed for ingest
			chunkerService,
			vectorStore,
			docStore,
		)

		ctx := context.Background()

		if err := processPath(ctx, processor, path); err != nil {
			return err
		}

		fmt.Printf("Successfully ingested documents from: %s\n", path)
		return nil
	},
}

func processPath(ctx context.Context, p *processor.Service, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path %s: %w", path, err)
	}

	if info.IsDir() {
		return processDirectory(ctx, p, path)
	}

	return processFile(ctx, p, path)
}

func processDirectory(ctx context.Context, p *processor.Service, dirPath string) error {
	if !recursive {
		return fmt.Errorf("directory processing requires --recursive flag")
	}

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if !quiet {
				log.Printf("Warning: failed to access %s: %v", path, err)
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		switch ext {
		case ".txt", ".md", ".markdown":
			if !quiet {
				fmt.Printf("Processing: %s\n", path)
			}

			if err := processFile(ctx, p, path); err != nil {
				if !quiet {
					log.Printf("Warning: failed to process %s: %v", path, err)
				}
			}
		default:
			if verbose {
				log.Printf("Skipping unsupported file: %s", path)
			}
		}

		return nil
	})
}

func processFile(ctx context.Context, p *processor.Service, filePath string) error {
	req := domain.IngestRequest{
		FilePath:  filePath,
		ChunkSize: chunkSize,
		Overlap:   overlap,
		Metadata: map[string]interface{}{
			"file_path": filePath,
			"file_ext":  filepath.Ext(filePath),
		},
	}

	resp, err := p.Ingest(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to ingest file %s: %w", filePath, err)
	}

	if verbose {
		fmt.Printf("Ingested %s: %d chunks (ID: %s)\n", filePath, resp.ChunkCount, resp.DocumentID)
	}

	return nil
}

func init() {
	ingestCmd.Flags().IntVar(&chunkSize, "chunk-size", 300, "text chunk size")
	ingestCmd.Flags().IntVar(&overlap, "overlap", 50, "chunk overlap size")
	ingestCmd.Flags().IntVar(&batchSize, "batch-size", 10, "batch processing size")
	ingestCmd.Flags().BoolVar(&recursive, "recursive", false, "process directory recursively")
}
