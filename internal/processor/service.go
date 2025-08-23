package processor

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	pdf "github.com/dslipak/pdf"
	"github.com/google/uuid"
	"github.com/liliang-cn/rago/internal/config"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/llm"
	"github.com/liliang-cn/rago/internal/tools"
	"github.com/liliang-cn/rago/internal/tools/builtin"
)

type Service struct {
	embedder        domain.Embedder
	generator       domain.Generator
	chunker         domain.Chunker
	vectorStore     domain.VectorStore
	keywordStore    domain.KeywordStore
	documentStore   domain.DocumentStore
	config          *config.Config
	llmService      *llm.OllamaService
	toolsEnabled    bool
	toolRegistry    *tools.Registry
	toolExecutor    *tools.Executor
	toolCoordinator *tools.Coordinator
}

func New(
	embedder domain.Embedder,
	generator domain.Generator,
	chunker domain.Chunker,
	vectorStore domain.VectorStore,
	keywordStore domain.KeywordStore,
	documentStore domain.DocumentStore,
	config *config.Config,
	llmService *llm.OllamaService,
) *Service {
	s := &Service{
		embedder:      embedder,
		generator:     generator,
		chunker:       chunker,
		vectorStore:   vectorStore,
		keywordStore:  keywordStore,
		documentStore: documentStore,
		config:        config,
		llmService:    llmService,
		toolsEnabled:  config.Tools.Enabled,
	}

	// Initialize tools if enabled
	if config.Tools.Enabled {
		s.initializeTools()
	}

	return s
}

// initializeTools sets up the tool system
func (s *Service) initializeTools() {
	// Create tool registry
	s.toolRegistry = tools.NewRegistry(&s.config.Tools)

	// Create tool executor
	executorConfig := &tools.ExecutorConfig{
		MaxConcurrency: s.config.Tools.MaxConcurrency,
		DefaultTimeout: s.config.Tools.CallTimeout,
		EnableLogging:  true,
	}
	s.toolExecutor = tools.NewExecutor(s.toolRegistry, executorConfig)

	// Create tool coordinator
	coordConfig := tools.DefaultCoordinatorConfig()
	s.toolCoordinator = tools.NewCoordinator(s.toolRegistry, s.toolExecutor, coordConfig)

	// Register built-in tools
	s.registerBuiltinTools()
}

// registerBuiltinTools registers the built-in tools
func (s *Service) registerBuiltinTools() {
	// Register datetime tool
	if s.config.Tools.BuiltinTools["datetime"].Enabled {
		datetimeTool := builtin.NewDateTimeTool()
		if err := s.toolRegistry.Register(datetimeTool); err != nil {
			log.Printf("Failed to register datetime tool: %v", err)
		}
	}

	// Register RAG search tool
	if s.config.Tools.BuiltinTools["rag_search"].Enabled {
		ragSearchTool := builtin.NewRAGSearchTool(s)
		if err := s.toolRegistry.Register(ragSearchTool); err != nil {
			log.Printf("Failed to register rag_search tool: %v", err)
		}
	}

	// Register document info tool
	if s.config.Tools.BuiltinTools["document_info"].Enabled {
		docInfoTool := builtin.NewDocumentInfoTool(s)
		if err := s.toolRegistry.Register(docInfoTool); err != nil {
			log.Printf("Failed to register document_info tool: %v", err)
		}
	}

	// Register file operations tool
	if s.config.Tools.BuiltinTools["file_operations"].Enabled {
		// Parse configuration
		allowedPaths := []string{"./knowledge", "./data", "./examples"} // Default paths
		maxFileSize := int64(10 * 1024 * 1024)                         // Default 10MB

		if params := s.config.Tools.BuiltinTools["file_operations"].Parameters; params != nil {
			if pathsStr, ok := params["allowed_paths"]; ok {
				allowedPaths = strings.Split(pathsStr, ",")
				// Trim whitespace from paths
				for i, path := range allowedPaths {
					allowedPaths[i] = strings.TrimSpace(path)
				}
			}
			if sizeStr, ok := params["max_file_size"]; ok {
				if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
					maxFileSize = size
				}
			}
		}

		fileOpsTool := builtin.NewFileOperationTool(allowedPaths, maxFileSize)
		if err := s.toolRegistry.Register(fileOpsTool); err != nil {
			log.Printf("Failed to register file_operations tool: %v", err)
		}
	}

	// Register SQL query tool
	if s.config.Tools.BuiltinTools["sql_query"].Enabled {
		// Parse configuration
		allowedDBs := make(map[string]string)
		maxRows := 1000
		queryTimeout := 30 * time.Second

		if params := s.config.Tools.BuiltinTools["sql_query"].Parameters; params != nil {
			if dbsStr, ok := params["allowed_databases"]; ok {
				// Parse "name:path,name2:path2" format
				pairs := strings.Split(dbsStr, ",")
				for _, pair := range pairs {
					parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
					if len(parts) == 2 {
						allowedDBs[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					}
				}
			}
			if rowsStr, ok := params["max_rows"]; ok {
				if rows, err := strconv.Atoi(rowsStr); err == nil && rows > 0 {
					maxRows = rows
				}
			}
			if timeoutStr, ok := params["query_timeout"]; ok {
				if timeout, err := time.ParseDuration(timeoutStr); err == nil {
					queryTimeout = timeout
				}
			}
		}

		// Only register if we have configured databases
		if len(allowedDBs) > 0 {
			sqlQueryTool := builtin.NewSQLQueryTool(allowedDBs, maxRows, queryTimeout)
			if err := s.toolRegistry.Register(sqlQueryTool); err != nil {
				log.Printf("Failed to register sql_query tool: %v", err)
			}
		} else {
			log.Printf("SQL query tool enabled but no databases configured, skipping registration")
		}
	}

	// Register HTTP request tool
	if s.config.Tools.BuiltinTools["http_request"].Enabled {
		// Parse configuration
		config := builtin.HTTPToolConfig{
			Timeout:      30 * time.Second,
			MaxBodySize:  10 * 1024 * 1024, // 10MB default
			UserAgent:    "RAGO-HTTP-Tool/1.0",
			FollowRedirect: true,
		}

		if params := s.config.Tools.BuiltinTools["http_request"].Parameters; params != nil {
			if timeoutStr, ok := params["timeout"]; ok {
				if timeout, err := time.ParseDuration(timeoutStr); err == nil {
					config.Timeout = timeout
				}
			}
			if sizeStr, ok := params["max_body_size"]; ok {
				if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
					config.MaxBodySize = size
				}
			}
			if userAgent, ok := params["user_agent"]; ok {
				config.UserAgent = userAgent
			}
			if followStr, ok := params["follow_redirect"]; ok {
				if follow, err := strconv.ParseBool(followStr); err == nil {
					config.FollowRedirect = follow
				}
			}
			if allowedHosts, ok := params["allowed_hosts"]; ok {
				config.AllowedHosts = strings.Split(allowedHosts, ",")
				for i, host := range config.AllowedHosts {
					config.AllowedHosts[i] = strings.TrimSpace(host)
				}
			}
			if blockedHosts, ok := params["blocked_hosts"]; ok {
				config.BlockedHosts = strings.Split(blockedHosts, ",")
				for i, host := range config.BlockedHosts {
					config.BlockedHosts[i] = strings.TrimSpace(host)
				}
			}
		}

		httpTool := builtin.NewHTTPTool(config)
		if err := s.toolRegistry.Register(httpTool); err != nil {
			log.Printf("Failed to register http_request tool: %v", err)
		}
	}

	// Register Web request tool
	if s.config.Tools.BuiltinTools["web_request"].Enabled {
		// Parse configuration
		config := builtin.WebToolConfig{
			Timeout:       60 * time.Second,
			MaxContentLen: 100 * 1024, // 100KB default
			UserAgent:     "RAGO-Web-Tool/1.0",
		}

		if params := s.config.Tools.BuiltinTools["web_request"].Parameters; params != nil {
			if timeoutStr, ok := params["timeout"]; ok {
				if timeout, err := time.ParseDuration(timeoutStr); err == nil {
					config.Timeout = timeout
				}
			}
			if lenStr, ok := params["max_content_len"]; ok {
				if length, err := strconv.Atoi(lenStr); err == nil {
					config.MaxContentLen = length
				}
			}
			if userAgent, ok := params["user_agent"]; ok {
				config.UserAgent = userAgent
			}
			if allowedHosts, ok := params["allowed_hosts"]; ok {
				config.AllowedHosts = strings.Split(allowedHosts, ",")
				for i, host := range config.AllowedHosts {
					config.AllowedHosts[i] = strings.TrimSpace(host)
				}
			}
			if blockedHosts, ok := params["blocked_hosts"]; ok {
				config.BlockedHosts = strings.Split(blockedHosts, ",")
				for i, host := range config.BlockedHosts {
					config.BlockedHosts[i] = strings.TrimSpace(host)
				}
			}
		}

		webTool := builtin.NewWebTool(config)
		if err := s.toolRegistry.Register(webTool); err != nil {
			log.Printf("Failed to register web_request tool: %v", err)
		}
	}

	// Register Google Search tool
	if s.config.Tools.BuiltinTools["google_search"].Enabled {
		// Parse configuration
		config := builtin.GoogleSearchConfig{
			MaxResults:    10,
			SearchTimeout: 60 * time.Second,
			UserAgent:     "RAGO-Search-Tool/1.0",
		}

		if params := s.config.Tools.BuiltinTools["google_search"].Parameters; params != nil {
			if maxResultsStr, ok := params["max_results"]; ok {
				if maxResults, err := strconv.Atoi(maxResultsStr); err == nil && maxResults > 0 {
					config.MaxResults = maxResults
				}
			}
			if timeoutStr, ok := params["search_timeout"]; ok {
				if timeout, err := time.ParseDuration(timeoutStr); err == nil {
					config.SearchTimeout = timeout
				}
			}
			if userAgent, ok := params["user_agent"]; ok {
				config.UserAgent = userAgent
			}
		}

		googleSearchTool := builtin.NewGoogleSearchTool(config)
		if err := s.toolRegistry.Register(googleSearchTool); err != nil {
			log.Printf("Failed to register google_search tool: %v", err)
		}
	}

	// Register DuckDuckGo Search tool
	if s.config.Tools.BuiltinTools["duckduckgo_search"].Enabled {
		// Parse configuration
		config := builtin.DuckDuckGoSearchConfig{
			MaxResults:    10,
			SearchTimeout: 30 * time.Second,
			UserAgent:     "RAGO-DuckDuckGo-Tool/1.0",
			SafeSearch:    "moderate",
		}

		if params := s.config.Tools.BuiltinTools["duckduckgo_search"].Parameters; params != nil {
			if maxResultsStr, ok := params["max_results"]; ok {
				if maxResults, err := strconv.Atoi(maxResultsStr); err == nil && maxResults > 0 {
					config.MaxResults = maxResults
				}
			}
			if timeoutStr, ok := params["search_timeout"]; ok {
				if timeout, err := time.ParseDuration(timeoutStr); err == nil {
					config.SearchTimeout = timeout
				}
			}
			if userAgent, ok := params["user_agent"]; ok {
				config.UserAgent = userAgent
			}
			if safeSearch, ok := params["safe_search"]; ok {
				config.SafeSearch = safeSearch
			}
		}

		duckDuckGoTool := builtin.NewDuckDuckGoSearchTool(config)
		if err := s.toolRegistry.Register(duckDuckGoTool); err != nil {
			log.Printf("Failed to register duckduckgo_search tool: %v", err)
		}
	}
}

func (s *Service) Ingest(ctx context.Context, req domain.IngestRequest) (domain.IngestResponse, error) {
	if err := s.validateIngestRequest(req); err != nil {
		return domain.IngestResponse{}, err
	}

	content, err := s.extractContent(req)
	if err != nil {
		return domain.IngestResponse{}, err
	}

	if content == "" {
		return domain.IngestResponse{
				Success: false,
				Message: "no content found",
			},
			nil
	}

	// Initialize metadata map if it's nil
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}

	// Automatic metadata extraction
	if s.config.Ingest.MetadataExtraction.Enable {
		log.Println("Metadata extraction enabled, calling LLM...")
		extracted, err := s.llmService.ExtractMetadata(ctx, content, s.config.Ingest.MetadataExtraction.LLMModel)
		if err != nil {
			log.Printf("Warning: metadata extraction failed, proceeding without it. Error: %v", err)
		} else {
			log.Printf("Successfully extracted metadata: %+v", extracted)
			s.mergeMetadata(req.Metadata, extracted)
		}
	}

	// Fallback for creation_date
	if _, ok := req.Metadata["creation_date"]; !ok || req.Metadata["creation_date"] == nil || req.Metadata["creation_date"] == "" {
		s.addFileCreationDate(req.FilePath, req.Metadata)
	}

	doc := domain.Document{
		ID:       uuid.New().String(),
		Path:     req.FilePath,
		URL:      req.URL,
		Content:  content, // Storing full content might be redundant, consider trade-offs
		Metadata: req.Metadata,
		Created:  time.Now(),
	}

	if err := s.documentStore.Store(ctx, doc); err != nil {
		return domain.IngestResponse{}, fmt.Errorf("failed to store document: %w", err)
	}

	chunkOptions := domain.ChunkOptions{
		Size:    req.ChunkSize,
		Overlap: req.Overlap,
		Method:  "sentence",
	}

	if req.ChunkSize <= 0 {
		chunkOptions.Size = s.config.Chunker.ChunkSize
	}
	if req.Overlap < 0 {
		chunkOptions.Overlap = s.config.Chunker.Overlap
	}

	textChunks, err := s.chunker.Split(content, chunkOptions)
	if err != nil {
		return domain.IngestResponse{}, fmt.Errorf("failed to chunk text: %w", err)
	}

	var chunks []domain.Chunk
	for i, textChunk := range textChunks {
		vector, err := s.embedder.Embed(ctx, textChunk)
		if err != nil {
			return domain.IngestResponse{}, fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
		}

		chunk := domain.Chunk{
			ID:         fmt.Sprintf("%s_%d", doc.ID, i),
			DocumentID: doc.ID,
			Content:    textChunk,
			Vector:     vector,
			Metadata:   doc.Metadata, // Pass down the combined metadata to each chunk
		}
		chunks = append(chunks, chunk)

		// Index the chunk in the keyword store as well.
		if err := s.keywordStore.Index(ctx, chunk); err != nil {
			return domain.IngestResponse{}, fmt.Errorf("failed to index chunk %d in keyword store: %w", i, err)
		}
	}

	if err := s.vectorStore.Store(ctx, chunks); err != nil {
		return domain.IngestResponse{}, fmt.Errorf("failed to store vectors: %w", err)
	}

	return domain.IngestResponse{
			Success:    true,
			DocumentID: doc.ID,
			ChunkCount: len(chunks),
			Message:    fmt.Sprintf("Successfully ingested %d chunks", len(chunks)),
		},
		nil
}

// mergeMetadata merges the extracted metadata into the request's metadata map.
func (s *Service) mergeMetadata(base map[string]interface{}, extracted *domain.ExtractedMetadata) {
	if extracted.Summary != "" {
		base["summary"] = extracted.Summary
	}
	if len(extracted.Keywords) > 0 {
		base["keywords"] = extracted.Keywords
	}
	if extracted.DocumentType != "" {
		base["document_type"] = extracted.DocumentType
	}
	if extracted.CreationDate != "" {
		base["creation_date"] = extracted.CreationDate
	}
}

// addFileCreationDate adds the file's modification time as a fallback creation date.
func (s *Service) addFileCreationDate(filePath string, metadata map[string]interface{}) {
	if filePath == "" {
		return
	}
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		metadata["creation_date"] = fileInfo.ModTime().Format("2006-01-02")
	}
}

func (s *Service) Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
	start := time.Now()

	if req.Query == "" {
		return domain.QueryResponse{}, fmt.Errorf("%w: empty query", domain.ErrInvalidInput)
	}

	chunks, err := s.hybridSearch(ctx, req)
	if err != nil {
		return domain.QueryResponse{}, err
	}

	if len(chunks) == 0 {
		return domain.QueryResponse{
				Answer:  "很抱歉，我在知识库中找不到相关信息来回答您的问题。",
				Sources: []domain.Chunk{},
				Elapsed: time.Since(start).String(),
			},
			nil
	}

	prompt := llm.ComposePrompt(chunks, req.Query)

	genOpts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if genOpts.Temperature <= 0 {
		genOpts.Temperature = 0.7
	}
	if genOpts.MaxTokens <= 0 {
		genOpts.MaxTokens = 500
	}

	answer, err := s.generator.Generate(ctx, prompt, genOpts)
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("failed to generate answer: %w", err)
	}

	// Clean up internal thinking tags from the answer only if ShowThinking is false
	if !req.ShowThinking {
		answer = s.cleanThinkingTags(answer)
	}

	return domain.QueryResponse{
			Answer:  answer,
			Sources: chunks,
			Elapsed: time.Since(start).String(),
		},
		nil
}

// QueryWithTools processes a query with tool calling support
func (s *Service) QueryWithTools(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
	start := time.Now()

	if req.Query == "" {
		return domain.QueryResponse{}, fmt.Errorf("%w: empty query", domain.ErrInvalidInput)
	}

	// If tools are not enabled or not requested, fall back to regular query
	if !s.toolsEnabled || !req.ToolsEnabled {
		return s.Query(ctx, req)
	}

	// Perform hybrid search first to get context
	chunks, err := s.hybridSearch(ctx, req)
	if err != nil {
		return domain.QueryResponse{}, err
	}

	// Build tools list based on allowed tools
	availableTools := s.getAvailableTools(req.AllowedTools)
	if len(availableTools) == 0 {
		// No tools available, fall back to regular query
		return s.Query(ctx, req)
	}

	// Convert to domain.ToolDefinition
	toolDefs := make([]domain.ToolDefinition, 0, len(availableTools))
	for _, tool := range availableTools {
		params := tool.Parameters()
		toolDef := domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters: map[string]interface{}{
					"type":       params.Type,
					"properties": params.Properties,
					"required":   params.Required,
				},
			},
		}
		toolDefs = append(toolDefs, toolDef)
	}

	// Create execution context
	execCtx := &tools.ExecutionContext{
		RequestID: uuid.New().String(),
	}

	// Generate options
	genOpts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if genOpts.Temperature <= 0 {
		genOpts.Temperature = 0.7
	}
	if genOpts.MaxTokens <= 0 {
		genOpts.MaxTokens = 1500  // Increased for tool calling scenarios
	}

	// Build prompt with context
	prompt := s.buildPromptWithContext(req.Query, chunks)

	// Use coordinator to handle tool calling conversation
	response, err := s.toolCoordinator.HandleToolCallingConversation(
		ctx, s.generator, prompt, toolDefs, genOpts, execCtx,
	)
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("tool calling failed: %w", err)
	}

	// Clean up internal thinking tags if needed
	if !req.ShowThinking {
		response.Answer = s.cleanThinkingTags(response.Answer)
	}

	// Add sources from initial search
	response.Sources = chunks
	response.Elapsed = time.Since(start).String()

	return *response, nil
}

// StreamQueryWithTools processes a streaming query with tool calling support
func (s *Service) StreamQueryWithTools(ctx context.Context, req domain.QueryRequest, callback func(string)) error {
	if req.Query == "" {
		return fmt.Errorf("%w: empty query", domain.ErrInvalidInput)
	}

	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	// If tools are not enabled or not requested, fall back to regular stream query
	if !s.toolsEnabled || !req.ToolsEnabled {
		return s.StreamQuery(ctx, req, callback)
	}

	// Perform hybrid search first
	chunks, err := s.hybridSearch(ctx, req)
	if err != nil {
		return err
	}

	// Build tools list
	availableTools := s.getAvailableTools(req.AllowedTools)
	if len(availableTools) == 0 {
		return s.StreamQuery(ctx, req, callback)
	}

	// Convert to domain.ToolDefinition
	toolDefs := make([]domain.ToolDefinition, 0, len(availableTools))
	for _, tool := range availableTools {
		params := tool.Parameters()
		toolDef := domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters: map[string]interface{}{
					"type":       params.Type,
					"properties": params.Properties,
					"required":   params.Required,
				},
			},
		}
		toolDefs = append(toolDefs, toolDef)
	}

	// Create execution context
	execCtx := &tools.ExecutionContext{
		RequestID: uuid.New().String(),
	}

	// Generate options
	genOpts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if genOpts.Temperature <= 0 {
		genOpts.Temperature = 0.7
	}
	if genOpts.MaxTokens <= 0 {
		genOpts.MaxTokens = 500
	}

	// Build prompt with context
	prompt := s.buildPromptWithContext(req.Query, chunks)

	// Wrap callback to handle thinking tags
	wrappedCallback := s.wrapCallbackForThinking(callback, req.ShowThinking)

	// Create a callback that handles tool execution results
	toolCallback := func(chunk string, toolCalls []domain.ExecutedToolCall, finished bool) error {
		if chunk != "" {
			wrappedCallback(chunk)
		}

		// Optionally send tool execution info
		if len(toolCalls) > 0 && req.ShowThinking {
			for _, call := range toolCalls {
				info := fmt.Sprintf("\n[Tool: %s - %s]\n", call.Function.Name,
					map[bool]string{true: "Success", false: "Failed"}[call.Success])
				wrappedCallback(info)
			}
		}

		return nil
	}

	// Use coordinator for streaming with tools
	return s.toolCoordinator.StreamToolCallingConversation(
		ctx, s.generator, prompt, toolDefs, genOpts, execCtx, toolCallback,
	)
}

// getAvailableTools returns the list of available tools based on allowed list
func (s *Service) getAvailableTools(allowedTools []string) []tools.Tool {
	if s.toolRegistry == nil {
		return nil
	}

	allTools := s.toolRegistry.ListEnabled()
	if len(allowedTools) == 0 {
		// Return all enabled tools
		result := make([]tools.Tool, 0, len(allTools))
		for _, info := range allTools {
			if tool, exists := s.toolRegistry.Get(info.Name); exists {
				result = append(result, tool)
			}
		}
		return result
	}

	// Filter by allowed list
	allowedMap := make(map[string]bool)
	for _, name := range allowedTools {
		allowedMap[name] = true
	}

	result := make([]tools.Tool, 0, len(allowedTools))
	for _, info := range allTools {
		if allowedMap[info.Name] {
			if tool, exists := s.toolRegistry.Get(info.Name); exists {
				result = append(result, tool)
			}
		}
	}

	return result
}

// buildPromptWithContext builds a prompt with RAG context
func (s *Service) buildPromptWithContext(query string, chunks []domain.Chunk) string {
	if len(chunks) == 0 {
		return query
	}

	var contextParts []string
	for i, chunk := range chunks {
		contextParts = append(contextParts, fmt.Sprintf("[Document %d]\n%s", i+1, chunk.Content))
	}

	context := strings.Join(contextParts, "\n\n")

	return fmt.Sprintf(`Based on the following context, please answer the user's question. 
If the context doesn't contain relevant information, you may use tools to get additional information.

Context:
%s

User Question: %s`, context, query)
}

func (s *Service) StreamQuery(ctx context.Context, req domain.QueryRequest, callback func(string)) error {
	if req.Query == "" {
		return fmt.Errorf("%w: empty query", domain.ErrInvalidInput)
	}

	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	chunks, err := s.hybridSearch(ctx, req)
	if err != nil {
		return err
	}

	if len(chunks) == 0 {
		callback("很抱歉，我在知识库中找不到相关信息来回答您的问题。")
		return nil
	}

	prompt := llm.ComposePrompt(chunks, req.Query)

	genOpts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if genOpts.Temperature <= 0 {
		genOpts.Temperature = 0.7
	}
	if genOpts.MaxTokens <= 0 {
		genOpts.MaxTokens = 500
	}

	return s.generator.Stream(ctx, prompt, genOpts, s.wrapCallbackForThinking(callback, req.ShowThinking))
}

func (s *Service) hybridSearch(ctx context.Context, req domain.QueryRequest) ([]domain.Chunk, error) {
	if req.TopK <= 0 {
		req.TopK = 5
	}

	var wg sync.WaitGroup
	var vectorErr, keywordErr error
	var vectorChunks, keywordChunks []domain.Chunk

	wg.Add(1)
	go func() {
		defer wg.Done()
		queryVector, err := s.embedder.Embed(ctx, req.Query)
		if err != nil {
			vectorErr = fmt.Errorf("failed to generate query embedding: %w", err)
			return
		}

		if len(req.Filters) > 0 {
			vectorChunks, vectorErr = s.vectorStore.SearchWithFilters(ctx, queryVector, req.TopK, req.Filters)
		} else {
			vectorChunks, vectorErr = s.vectorStore.Search(ctx, queryVector, req.TopK)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		keywordChunks, keywordErr = s.keywordStore.Search(ctx, req.Query, req.TopK)
	}()

	wg.Wait()

	if vectorErr != nil {
		log.Printf("Warning: vector search failed: %v", vectorErr)
		// Do not return error, proceed with keyword results if available
	}
	if keywordErr != nil {
		log.Printf("Warning: keyword search failed: %v", keywordErr)
		// Do not return error, proceed with vector results if available
	}

	fusedChunks := s.fuseResults(vectorChunks, keywordChunks)

	return s.deduplicateChunks(fusedChunks), nil
}

// fuseResults combines and re-ranks search results using Reciprocal Rank Fusion (RRF).
func (s *Service) fuseResults(listA, listB []domain.Chunk) []domain.Chunk {
	const k = 60 // RRF constant

	scores := make(map[string]float64)
	chunksMap := make(map[string]domain.Chunk)

	// Process first list
	for i, chunk := range listA {
		rank := i + 1
		scores[chunk.ID] += 1.0 / float64(k+rank)
		if _, exists := chunksMap[chunk.ID]; !exists {
			chunksMap[chunk.ID] = chunk
		}
	}

	// Process second list
	for i, chunk := range listB {
		rank := i + 1
		scores[chunk.ID] += 1.0 / float64(k+rank)
		if _, exists := chunksMap[chunk.ID]; !exists {
			chunksMap[chunk.ID] = chunk
		}
	}

	// Create a slice of unique chunks
	var fused []domain.Chunk
	for id := range chunksMap {
		chunk := chunksMap[id]
		chunk.Score = scores[id] // Assign the fused RRF score
		fused = append(fused, chunk)
	}

	// Sort by the new RRF score in descending order
	sort.Slice(fused, func(i, j int) bool {
		return fused[i].Score > fused[j].Score
	})

	return fused
}

func (s *Service) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	return s.documentStore.List(ctx)
}

func (s *Service) DeleteDocument(ctx context.Context, documentID string) error {
	if documentID == "" {
		return fmt.Errorf("%w: empty document ID", domain.ErrInvalidInput)
	}

	if err := s.vectorStore.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("failed to delete from vector store: %w", err)
	}

	if err := s.keywordStore.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("failed to delete from keyword store: %w", err)
	}

	if err := s.documentStore.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("failed to delete from document store: %w", err)
	}

	return nil
}

func (s *Service) Reset(ctx context.Context) error {
	if err := s.vectorStore.Reset(ctx); err != nil {
		return fmt.Errorf("failed to reset vector store: %w", err)
	}

	if err := s.keywordStore.Reset(ctx); err != nil {
		return fmt.Errorf("failed to reset keyword store: %w", err)
	}

	return nil
}

func (s *Service) validateIngestRequest(req domain.IngestRequest) error {
	hasContent := req.Content != ""
	hasFilePath := req.FilePath != ""
	hasURL := req.URL != ""

	contentSources := 0
	if hasContent {
		contentSources++
	}
	if hasFilePath {
		contentSources++
	}
	if hasURL {
		contentSources++
	}

	if contentSources == 0 {
		return fmt.Errorf("%w: no content source provided", domain.ErrInvalidInput)
	}

	if contentSources > 1 {
		return fmt.Errorf("%w: multiple content sources provided", domain.ErrInvalidInput)
	}

	return nil
}

func (s *Service) extractContent(req domain.IngestRequest) (string, error) {
	if req.Content != "" {
		return req.Content, nil
	}

	if req.FilePath != "" {
		return s.readFile(req.FilePath)
	}

	if req.URL != "" {
		return "", fmt.Errorf("URL content extraction not yet implemented")
	}

	return "", fmt.Errorf("%w: no content source", domain.ErrInvalidInput)
}

func (s *Service) readFile(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".txt", ".md", ".markdown":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		return string(data), nil

	case ".pdf":
		r, err := pdf.Open(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to open PDF %s: %w", filePath, err)
		}
		var buf strings.Builder
		for i := 1; i <= r.NumPage(); i++ {
			p := r.Page(i)
			if p.V.IsNull() {
				continue
			}
			text, err := p.GetPlainText(nil)
			if err != nil {
				// Log a warning but continue processing other pages
				log.Printf("Warning: failed to get text from page %d of %s: %v", i, filePath, err)
				continue
			}
			buf.WriteString(text)
			buf.WriteString("\n") // Add a newline between pages
		}
		return buf.String(), nil

	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// deduplicateChunks removes duplicate chunks by content to avoid confusion
func (s *Service) deduplicateChunks(chunks []domain.Chunk) []domain.Chunk {
	seen := make(map[string]bool)
	result := make([]domain.Chunk, 0, len(chunks))

	for _, chunk := range chunks {
		// Use content as the key for deduplication
		if !seen[chunk.Content] {
			seen[chunk.Content] = true
			result = append(result, chunk)
		}
	}

	return result
}

// cleanThinkingTags removes internal thinking tags from LLM responses
func (s *Service) cleanThinkingTags(answer string) string {
	// Remove <think>...</think> blocks and their contents
	re := strings.NewReplacer("<think>", "", "</think>", "")
	cleaned := re.Replace(answer)

	// Also handle the case where thinking tags might span multiple lines
	if strings.Contains(cleaned, "<think") || strings.Contains(cleaned, "</think") {
		// Use regex for more complex cases
		lines := strings.Split(cleaned, "\n")
		var filtered []string
		inThinking := false

		for _, line := range lines {
			if strings.Contains(line, "<think") {
				inThinking = true
				continue
			}
			if strings.Contains(line, "</think") {
				inThinking = false
				continue
			}
			if !inThinking {
				filtered = append(filtered, line)
			}
		}
		cleaned = strings.Join(filtered, "\n")
	}

	// Trim any extra whitespace
	return strings.TrimSpace(cleaned)
}

// wrapCallbackForThinking wraps the callback to filter thinking tags in streaming mode
func (s *Service) wrapCallbackForThinking(callback func(string), showThinking bool) func(string) {
	if showThinking {
		// If showing thinking, just pass through
		return callback
	}

	// If not showing thinking, filter out thinking content
	var buffer strings.Builder
	inThinking := false

	return func(token string) {
		buffer.WriteString(token)
		content := buffer.String()

		// Process complete thinking blocks
		for {
			if !inThinking {
				// Look for start of thinking block
				if idx := strings.Index(content, "<think>"); idx != -1 {
					// Send content before thinking block
					if idx > 0 {
						callback(content[:idx])
					}
					inThinking = true
					content = content[idx+7:] // Skip "<think>"
					buffer.Reset()
					buffer.WriteString(content)
					continue
				} else {
					// No thinking block start, send everything
					if content != "" {
						callback(content)
						buffer.Reset()
					}
					break
				}
			} else {
				// Look for end of thinking block
				if idx := strings.Index(content, "</think>"); idx != -1 {
					inThinking = false
					content = content[idx+8:] // Skip "</think>"
					buffer.Reset()
					buffer.WriteString(content)
					continue
				} else {
					// Still in thinking block, don't send anything
					break
				}
			}
		}
	}
}

// GetToolRegistry returns the tool registry if tools are enabled
func (s *Service) GetToolRegistry() *tools.Registry {
	return s.toolRegistry
}

// GetToolExecutor returns the tool executor if tools are enabled
func (s *Service) GetToolExecutor() *tools.Executor {
	return s.toolExecutor
}
