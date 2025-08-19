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
	"github.com/liliang-cn/rago/internal/llm"
	"github.com/liliang-cn/rago/internal/processor"
	"github.com/liliang-cn/rago/internal/store"
	"github.com/spf13/cobra"
)

var (
	chunkSize         int
	overlap           int
	batchSize         int
	recursive         bool
	textInput         string
	source            string
	extractMetadata   bool
)

var ingestCmd = &cobra.Command{
	Use:   "ingest [file/directory]",
	Short: "Import documents into vector database",
	Long: `Chunk document content, vectorize and store into local vector database.
Supports text format files like .txt, .md, etc.
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
		// Override config if flag is set
		if extractMetadata {
			cfg.Ingest.MetadataExtraction.Enable = true
		}

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
		)
		if err != nil {
			return fmt.Errorf("failed to create embedder: %w", err)
		}

		llmService, err := llm.NewOllamaService(
			cfg.Ollama.BaseURL,
			cfg.Ollama.LLMModel, // Default model for other tasks
		)
		if err != nil {
			return fmt.Errorf("failed to create llm service: %w", err)
		}

		chunkerService := chunker.New()

		processor := processor.New(
			embedService,
			nil, // generator not needed for ingest
			chunkerService,
			vectorStore,
			docStore,
			cfg,
			llmService,
		)

		ctx := context.Background()

		// Handle text input
		if textInput != "" {
			return processText(ctx, processor, textInput)
		}

		// Handle file path
		if len(args) == 0 {
			return fmt.Errorf("no file path provided")
		}
		path := args[0]

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

func processText(ctx context.Context, p *processor.Service, text string) error {
	sourceValue := source
	if sourceValue == "" {
		sourceValue = "text-input"
	}

	req := domain.IngestRequest{
		Content:   text,
		ChunkSize: chunkSize,
		Overlap:   overlap,
		Metadata: map[string]interface{}{
			"source": sourceValue,
			"type":   "text",
		},
	}

	resp, err := p.Ingest(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to ingest text: %w", err)
	}

	fmt.Printf("Successfully ingested text: %d chunks (ID: %s)\n", resp.ChunkCount, resp.DocumentID)
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
}
