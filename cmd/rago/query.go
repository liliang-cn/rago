package rago

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liliang-cn/rago/internal/chunker"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/mcp"
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
	showSources  bool
	interactive  bool
	queryFile    string
	filterBy     []string
	enableTools  bool
	allowedTools []string
	maxToolCalls int
	useMCP       bool
)

var queryCmd = &cobra.Command{
	Use:   "query [question]",
	Short: "Query knowledge base with automatic tool calling or MCP tools",
	Long: `Perform semantic search and Q&A based on imported documents, or call MCP tools.
You can provide a question as an argument or use interactive mode.

Tool calling is enabled by default based on your configuration file.
Use --tools to override the configuration setting.
Use --mcp to enable MCP tool integration for the query.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle MCP mode
		if useMCP {
			return processMCPQuery(cmd, args)
		}

		// Initialize stores
		vectorStore, err := store.NewSQLiteStore(
			cfg.Sqvect.DBPath,
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

		// Initialize services using shared provider system
		ctx := context.Background()
		embedService, llmService, metadataExtractor, err := initializeProviders(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize providers: %w", err)
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
			metadataExtractor,
		)

		// Determine if tools should be enabled based on config or flag
		toolsEnabled := enableTools
		if !cmd.Flags().Changed("tools") {
			toolsEnabled = cfg.Tools.Enabled
		}

		if interactive || len(args) == 0 {
			return runInteractive(ctx, processor, toolsEnabled)
		}

		if queryFile != "" {
			return processQueryFile(ctx, processor, toolsEnabled)
		}

		query := strings.Join(args, " ")
		return processQuery(ctx, processor, query, toolsEnabled)
	},
}

func runInteractive(ctx context.Context, p *processor.Service, toolsEnabled bool) error {
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

		if err := processQuery(ctx, p, input, toolsEnabled); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		fmt.Println()
	}

	return scanner.Err()
}

func processQueryFile(ctx context.Context, p *processor.Service, toolsEnabled bool) error {
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
		if err := processQuery(ctx, p, query, toolsEnabled); err != nil {
			fmt.Printf("Error processing query %d: %v\n", lineNum, err)
		}
		fmt.Println(strings.Repeat("-", 50))
	}

	return scanner.Err()
}

func processQuery(ctx context.Context, p *processor.Service, query string, toolsEnabled bool) error {
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
		Stream:       stream, // Streaming now works with tools
		ShowThinking: showThinking,
		ShowSources:  showSources,
		Filters:      filters,
		ToolsEnabled: toolsEnabled,
		AllowedTools: allowedTools,
		MaxToolCalls: maxToolCalls,
	}

	if req.Stream {
		return processStreamQuery(ctx, p, req)
	}

	var resp domain.QueryResponse
	var err error
	
	// Use QueryWithTools if tools are enabled
	if toolsEnabled {
		resp, err = p.QueryWithTools(ctx, req)
	} else {
		resp, err = p.Query(ctx, req)
	}
	
	if err != nil {
		return err
	}

	fmt.Printf("Answer: %s\n", resp.Answer)

	// Show tool execution information if available
	if toolsEnabled && len(resp.ToolCalls) > 0 {
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

	// Show sources if requested
	if (showSources || verbose) && len(resp.Sources) > 0 {
		fmt.Printf("\nSources (%d):\n", len(resp.Sources))
		for i, source := range resp.Sources {
			fmt.Printf("  [%d] Score: %.4f\n", i+1, source.Score)
			
			// Extract filename from metadata if available
			var sourceInfo string
			if source.Metadata != nil {
				if filename, ok := source.Metadata["filename"].(string); ok && filename != "" {
					sourceInfo = filename
				} else if source_path, ok := source.Metadata["source"].(string); ok && source_path != "" {
					sourceInfo = source_path
				} else {
					sourceInfo = "Unknown source"
				}
			} else {
				sourceInfo = "Unknown source"
			}
			
			fmt.Printf("      Source: %s\n", sourceInfo)
			if verbose {
				fmt.Printf("      Content: %s...\n", truncateText(source.Content, 100))
				if len(source.Metadata) > 0 {
					fmt.Printf("      Metadata: %v\n", source.Metadata)
				}
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
	
	var sources []domain.Chunk
	var err error
	
	// If sources are requested, we need to get them first
	if req.ShowSources || verbose {
		// First get sources by doing a quick non-streaming query
		tempReq := req
		tempReq.Stream = false
		tempReq.ShowSources = true
		
		var tempResp domain.QueryResponse
		if req.ToolsEnabled {
			tempResp, err = p.QueryWithTools(ctx, tempReq)
		} else {
			tempResp, err = p.Query(ctx, tempReq)
		}
		if err == nil {
			sources = tempResp.Sources
		}
	}

	// Now do the streaming
	if req.ToolsEnabled {
		err = p.StreamQueryWithTools(ctx, req, func(token string) {
			fmt.Print(token)
		})
	} else {
		err = p.StreamQuery(ctx, req, func(token string) {
			fmt.Print(token)
		})
	}

	fmt.Println()
	
	// Show sources after streaming is complete
	if (req.ShowSources || verbose) && len(sources) > 0 {
		fmt.Printf("\nSources (%d):\n", len(sources))
		for i, source := range sources {
			fmt.Printf("  [%d] Score: %.4f\n", i+1, source.Score)
			
			// Extract filename from metadata if available
			var sourceInfo string
			if source.Metadata != nil {
				if filename, ok := source.Metadata["filename"].(string); ok && filename != "" {
					sourceInfo = filename
				} else if source_path, ok := source.Metadata["source"].(string); ok && source_path != "" {
					sourceInfo = source_path
				} else {
					sourceInfo = "Unknown source"
				}
			} else {
				sourceInfo = "Unknown source"
			}
			
			fmt.Printf("      Source: %s\n", sourceInfo)
			if verbose {
				fmt.Printf("      Content: %s...\n", truncateText(source.Content, 100))
				if len(source.Metadata) > 0 {
					fmt.Printf("      Metadata: %v\n", source.Metadata)
				}
			}
		}
	}
	
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
	fmt.Println("  --tools               - Enable/disable tool calling (overrides config)")
	fmt.Println("  --allowed-tools tool1,tool2 - Specify allowed tools")
	fmt.Println("  --max-tool-calls N    - Maximum tool calls per query")
	fmt.Println()
	fmt.Println("Note: Tool calling is enabled by default based on config.toml settings.")
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func processMCPQuery(cmd *cobra.Command, args []string) error {
	if !cfg.MCP.Enabled {
		return fmt.Errorf("MCP is disabled in configuration")
	}

	// Get the query
	var query string
	if len(args) == 0 {
		return fmt.Errorf("please provide a question when using --mcp flag")
	}
	query = strings.Join(args, " ")

	fmt.Printf("MCP Query: %s\n\n", query)

	// Initialize MCP service
	mcpService := mcp.NewMCPService(&cfg.MCP)
	ctx := context.Background()
	
	if err := mcpService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to start MCP service: %w", err)
	}
	defer mcpService.Close()

	// Get available tools
	toolsMap := mcpService.GetAvailableTools()
	if len(toolsMap) == 0 {
		return fmt.Errorf("no MCP tools available")
	}

	// Initialize processors for LLM functionality
	vectorStore, err := store.NewSQLiteStore(cfg.Sqvect.DBPath)
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
	embedService, llmService, metadataExtractor, err := initializeProviders(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize providers: %w", err)
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
		metadataExtractor,
	)

	// Register MCP tools with the processor
	if err := processor.RegisterMCPTools(mcpService); err != nil {
		return fmt.Errorf("failed to register MCP tools: %w", err)
	}

	// Use regular query with tools enabled - the MCP tools are now registered
	req := domain.QueryRequest{
		Query:        query,
		TopK:         5,
		Temperature:  0.7,
		MaxTokens:    1000,
		Stream:       false,
		ShowSources:  false,
		ToolsEnabled: true,
		MaxToolCalls: 3,
	}

	resp, err := processor.QueryWithTools(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to process MCP query: %w", err)
	}

	fmt.Printf("Answer: %s\n", resp.Answer)

	// Show tool execution information if available
	if len(resp.ToolCalls) > 0 {
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

	return nil
}

func formatToolsForPrompt(tools []mcp.ToolSummary) string {
	var result []string
	for _, tool := range tools {
		result = append(result, fmt.Sprintf("- %s (%s): %s", tool.Name, tool.ServerName, tool.Description))
	}
	return strings.Join(result, "\n")
}

func init() {
	queryCmd.Flags().IntVar(&topK, "top-k", 5, "number of documents to retrieve")
	queryCmd.Flags().Float64Var(&temperature, "temperature", 0.7, "generation temperature")
	queryCmd.Flags().IntVar(&maxTokens, "max-tokens", 500, "maximum generation length")
	queryCmd.Flags().BoolVar(&stream, "stream", true, "streaming output")
	queryCmd.Flags().BoolVar(&showThinking, "show-thinking", true, "show AI thinking process")
	queryCmd.Flags().BoolVarP(&showSources, "show-sources", "s", false, "show source documents used for answering")
	queryCmd.Flags().BoolVar(&verbose, "verbose", false, "show verbose output including sources")
	queryCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "interactive mode")
	queryCmd.Flags().StringVar(&queryFile, "file", "", "batch query from file")
	queryCmd.Flags().StringSliceVar(&filterBy, "filter", []string{}, "filter by metadata (key=value format, can be used multiple times)")
	queryCmd.Flags().BoolVar(&enableTools, "tools", false, "enable tool calling capabilities (overrides config file setting)")
	queryCmd.Flags().StringSliceVar(&allowedTools, "allowed-tools", []string{}, "comma-separated list of allowed tools (empty means all enabled tools)")
	queryCmd.Flags().IntVar(&maxToolCalls, "max-tool-calls", 5, "maximum number of tool calls per query")
	queryCmd.Flags().BoolVar(&useMCP, "mcp", false, "use MCP tools for query processing")
}
