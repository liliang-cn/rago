package rago

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liliang-cn/rago/internal/chunker"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/embedder"
	"github.com/liliang-cn/rago/internal/llm"
	"github.com/liliang-cn/rago/internal/processor"
	"github.com/liliang-cn/rago/internal/store"
	"github.com/spf13/cobra"
)

var (
	topK         int
	temperature  float64
	maxTokens    int
	stream       bool
	showThinking bool
	interactive  bool
	queryFile    string
	filterBy     []string
	enableTools  bool
	allowedTools []string
	maxToolCalls int
)

var queryCmd = &cobra.Command{
	Use:   "query [question]",
	Short: "Query knowledge base",
	Long: `Perform semantic search and Q&A based on imported documents.
You can provide a question as an argument or use interactive mode.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize stores
		vectorStore, err := store.NewSQLiteStore(
			cfg.Sqvect.DBPath,
			cfg.Sqvect.VectorDim,
			cfg.Sqvect.MaxConns,
			cfg.Sqvect.BatchSize,
		)
		if err != nil {
			return fmt.Errorf("failed to create vector store: %w", err)
		}
		defer vectorStore.Close()

		keywordStore, err := store.NewKeywordStore(cfg.Keyword.IndexPath)
		if err != nil {
			return fmt.Errorf("failed to create keyword store: %w", err)
		}
		defer keywordStore.Close()

		docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

		// Initialize services
		embedService, err := embedder.NewOllamaService(
			cfg.Ollama.BaseURL,
			cfg.Ollama.EmbeddingModel,
		)
		if err != nil {
			return fmt.Errorf("failed to create embedder: %w", err)
		}

		llmService, err := llm.NewOllamaService(
			cfg.Ollama.BaseURL,
			cfg.Ollama.LLMModel,
		)
		if err != nil {
			return fmt.Errorf("failed to create LLM service: %w", err)
		}

		chunkerService := chunker.New()

		processor := processor.New(
			embedService,
			llmService,
			chunkerService,
			vectorStore,
			keywordStore,
			docStore,
			cfg,
			llmService,
		)

		ctx := context.Background()

		if interactive || len(args) == 0 {
			return runInteractive(ctx, processor)
		}

		if queryFile != "" {
			return processQueryFile(ctx, processor)
		}

		query := strings.Join(args, " ")
		return processQuery(ctx, processor, query)
	},
}

func runInteractive(ctx context.Context, p *processor.Service) error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("RAGO Interactive Query Mode")
	fmt.Println("Type 'exit' or 'quit' to exit, 'help' for commands")
	fmt.Println()

	for {
		fmt.Print("rago> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println("Goodbye!")
			return nil
		case "help", "h":
			printHelp()
			continue
		case "clear", "cls":
			fmt.Print("\033[2J\033[H")
			continue
		}

		if err := processQuery(ctx, p, input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		fmt.Println()
	}

	return scanner.Err()
}

func processQueryFile(ctx context.Context, p *processor.Service) error {
	file, err := os.Open(queryFile)
	if err != nil {
		return fmt.Errorf("failed to open query file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close query file: %v\n", closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		query := strings.TrimSpace(scanner.Text())
		if query == "" || strings.HasPrefix(query, "#") {
			continue
		}

		fmt.Printf("Query %d: %s\n", lineNum, query)
		if err := processQuery(ctx, p, query); err != nil {
			fmt.Printf("Error processing query %d: %v\n", lineNum, err)
		}
		fmt.Println(strings.Repeat("-", 50))
	}

	return scanner.Err()
}

func processQuery(ctx context.Context, p *processor.Service, query string) error {
	// Parse filters from filterBy flag
	filters := make(map[string]interface{})
	for _, filter := range filterBy {
		parts := strings.SplitN(filter, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			filters[key] = value
		}
	}

	req := domain.QueryRequest{
		Query:        query,
		TopK:         topK,
		Temperature:  temperature,
		MaxTokens:    maxTokens,
		Stream:       stream && !enableTools, // Disable streaming when tools are enabled
		ShowThinking: showThinking,
		Filters:      filters,
		ToolsEnabled: enableTools,
		AllowedTools: allowedTools,
		MaxToolCalls: maxToolCalls,
	}

	// Inform user if streaming was disabled due to tools
	if stream && enableTools && verbose {
		fmt.Println("Note: Streaming disabled when tools are enabled")
	}

	if req.Stream {
		return processStreamQuery(ctx, p, req)
	}

	var resp domain.QueryResponse
	var err error
	
	// Use QueryWithTools if tools are enabled
	if enableTools {
		resp, err = p.QueryWithTools(ctx, req)
	} else {
		resp, err = p.Query(ctx, req)
	}
	
	if err != nil {
		return err
	}

	fmt.Printf("Answer: %s\n", resp.Answer)

	// Show tool execution information if available
	if enableTools && len(resp.ToolCalls) > 0 {
		fmt.Printf("\nTool Executions (%d):\n", len(resp.ToolCalls))
		for i, toolCall := range resp.ToolCalls {
			status := "Success"
			if !toolCall.Success {
				status = "Failed"
			}
			fmt.Printf("  [%d] %s - %s (%s)\n", i+1, toolCall.Function.Name, status, toolCall.Elapsed)
			if toolCall.Error != "" {
				fmt.Printf("      Error: %s\n", toolCall.Error)
			}
		}
	}

	if verbose && len(resp.Sources) > 0 {
		fmt.Printf("\nSources (%d):\n", len(resp.Sources))
		for i, source := range resp.Sources {
			fmt.Printf("  [%d] Score: %.4f\n", i+1, source.Score)
			fmt.Printf("      Content: %s...\n", truncateText(source.Content, 100))
			if len(source.Metadata) > 0 {
				fmt.Printf("      Metadata: %v\n", source.Metadata)
			}
		}
	}

	if verbose {
		fmt.Printf("\nElapsed: %s\n", resp.Elapsed)
	}

	return nil
}

func processStreamQuery(ctx context.Context, p *processor.Service, req domain.QueryRequest) error {
	fmt.Print("Answer: ")

	var err error
	if enableTools {
		err = p.StreamQueryWithTools(ctx, req, func(token string) {
			fmt.Print(token)
		})
	} else {
		err = p.StreamQuery(ctx, req, func(token string) {
			fmt.Print(token)
		})
	}

	fmt.Println()
	return err
}

func printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help, h     - Show this help message")
	fmt.Println("  clear, cls  - Clear the screen")
	fmt.Println("  exit, quit  - Exit the program")
	fmt.Println("  <question>  - Ask a question to the knowledge base")
	fmt.Println()
	fmt.Println("Available flags:")
	fmt.Println("  --filter key=value    - Filter results by metadata")
	fmt.Println("  --tools               - Enable tool calling")
	fmt.Println("  --allowed-tools tool1,tool2 - Specify allowed tools")
	fmt.Println("  --max-tool-calls N    - Maximum tool calls per query")
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func init() {
	queryCmd.Flags().IntVar(&topK, "top-k", 5, "number of documents to retrieve")
	queryCmd.Flags().Float64Var(&temperature, "temperature", 0.7, "generation temperature")
	queryCmd.Flags().IntVar(&maxTokens, "max-tokens", 500, "maximum generation length")
	queryCmd.Flags().BoolVar(&stream, "stream", true, "streaming output")
	queryCmd.Flags().BoolVar(&showThinking, "show-thinking", true, "show AI thinking process")
	queryCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "interactive mode")
	queryCmd.Flags().StringVar(&queryFile, "file", "", "batch query from file")
	queryCmd.Flags().StringSliceVar(&filterBy, "filter", []string{}, "filter by metadata (key=value format, can be used multiple times)")
	queryCmd.Flags().BoolVar(&enableTools, "tools", false, "enable tool calling capabilities")
	queryCmd.Flags().StringSliceVar(&allowedTools, "allowed-tools", []string{}, "comma-separated list of allowed tools (empty means all enabled tools)")
	queryCmd.Flags().IntVar(&maxToolCalls, "max-tool-calls", 5, "maximum number of tool calls per query")
}
