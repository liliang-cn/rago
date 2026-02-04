package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/internal/api/handlers"
	chatHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/chat"
	conversationHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/conversation"
	llmHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/llm"
	mcpHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/mcp"
	// platformHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/platform"
	ragHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/rag"
	usageHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/usage"
	v1Handlers "github.com/liliang-cn/rago/v2/internal/api/handlers/v1"
	"github.com/liliang-cn/rago/v2/internal/web"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/rag/chunker"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
	pkgStore "github.com/liliang-cn/rago/v2/pkg/store"
	"github.com/liliang-cn/rago/v2/pkg/usage"
	"github.com/spf13/cobra"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

var (
	port     int
	host     string
	enableUI bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP API service",
	Long:  `Start HTTP API server to provide RESTful API endpoints.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if port == 0 {
			port = cfg.Server.Port
		}
		if host == "" {
			host = cfg.Server.Host
		}
		if !enableUI {
			enableUI = cfg.Server.EnableUI
		}

		// Initialize stores based on configuration
		var vectorStore domain.VectorStore
		var docStore *store.DocumentStore
		var err error
		
		if cfg.VectorStore != nil && cfg.VectorStore.Type != "" {
			// Use configured vector store
			storeConfig := store.StoreConfig{
				Type:       cfg.VectorStore.Type,
				Parameters: cfg.VectorStore.Parameters,
			}
			vectorStore, err = store.NewVectorStore(storeConfig)
			if err != nil {
				return fmt.Errorf("failed to create vector store: %w", err)
			}
			
			// For Qdrant, need separate document store
			if cfg.VectorStore.Type == "qdrant" {
				sqliteStore, err := store.NewSQLiteStore(cfg.Sqvect.DBPath, cfg.Sqvect.IndexType)
				if err != nil {
					return fmt.Errorf("failed to create document store: %w", err)
				}
				docStore = store.NewDocumentStore(sqliteStore.GetSqvectStore())
				defer func() {
					if err := sqliteStore.Close(); err != nil {
						fmt.Printf("Warning: failed to close document store: %v\n", err)
					}
				}()
			}
		} else {
			// Default to SQLite
			sqliteStore, err := store.NewSQLiteStore(cfg.Sqvect.DBPath, cfg.Sqvect.IndexType)
			if err != nil {
				return fmt.Errorf("failed to create vector store: %w", err)
			}
			vectorStore = sqliteStore
			docStore = store.NewDocumentStore(sqliteStore.GetSqvectStore())
			defer func() {
				if err := sqliteStore.Close(); err != nil {
					fmt.Printf("Warning: failed to close vector store: %v\n", err)
				}
			}()
		}
		
		// Close vector store when done (for non-SQLite stores)
		defer func() {
			if closer, ok := vectorStore.(interface{ Close() error }); ok {
				// Only close if it's not SQLite (already handled above)
				if _, isSQLite := vectorStore.(*store.SQLiteStore); !isSQLite {
					if err := closer.Close(); err != nil {
						fmt.Printf("Warning: failed to close vector store: %v\n", err)
					}
				}
			}
		}()
		
		// Ensure docStore is initialized for SQLite
		if docStore == nil {
			if sqliteStore, ok := vectorStore.(*store.SQLiteStore); ok {
				docStore = store.NewDocumentStore(sqliteStore.GetSqvectStore())
			}
		}

		// Initialize services using shared pool system
		ctx := context.Background()
		llmService, err := services.GetGlobalLLM()
		if err != nil {
			return fmt.Errorf("failed to get LLM service: %w", err)
		}

		embedService, err := services.GetGlobalEmbeddingService(ctx)
		if err != nil {
			return fmt.Errorf("failed to get embedder service: %w", err)
		}

		// Create metadata extractor from LLM service if it implements the interface
		var metadataExtractor domain.MetadataExtractor
		if extractor, ok := llmService.(domain.MetadataExtractor); ok {
			metadataExtractor = extractor
		}

		chunkerService := chunker.New()

		processorService := processor.New(
			embedService,
			llmService,
			chunkerService,
			vectorStore,
			docStore,
			cfg,
			metadataExtractor,
			nil, // memoryService
		)

		// 设置Gin为release模式
		gin.SetMode(gin.ReleaseMode)

		// Initialize usage service with data directory from config
		usageDataDir := ".rago/data"
		usageService, err := usage.NewServiceWithDataDir(cfg, usageDataDir)
		if err != nil {
			return fmt.Errorf("failed to initialize usage service: %w", err)
		}
		defer func() {
			if err := usageService.Close(); err != nil {
				fmt.Printf("Warning: failed to close usage service: %v\n", err)
			}
		}()

		// Wrap processor with tracking capabilities
		trackedProcessor := usage.NewTrackedRAGProcessor(processorService, usageService)

		// Initialize conversation store
		dbPath := filepath.Join(usageDataDir, "conversations.db")
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open conversation database: %w", err)
		}
		defer db.Close()
		
		conversationStore, err := pkgStore.NewConversationStore(db)
		if err != nil {
			return fmt.Errorf("failed to initialize conversation store: %w", err)
		}

		router, err := setupRouter(trackedProcessor, processorService, cfg, embedService, llmService, usageService, conversationStore)
		if err != nil {
			return fmt.Errorf("failed to setup router: %w", err)
		}

		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", host, port),
			Handler: router,
		}

		go func() {
			printServerInfo(host, port, enableUI)

			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("\nShutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server forced to shutdown: %w", err)
		}

		fmt.Println("Server stopped")
		return nil
	},
}

func setupRouter(trackedProcessor domain.RAGProcessor, processorService *processor.Service, cfg *config.Config, embedService domain.Embedder, llmService domain.Generator, usageService *usage.Service, conversationStore *pkgStore.ConversationStore) (*gin.Engine, error) {
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	api := router.Group("/api")
	{
		api.GET("/health", handlers.NewHealthHandler().Handle)

		// Platform unified API endpoints (NEW) - TEMPORARILY DISABLED
		// TODO: Fix client package import and re-enable
		/*
		platformGroup := api.Group("/platform")
		{
			platformHandler := platformHandlers.NewHandler(cfg)

			// Info
			platformGroup.GET("/info", platformHandler.Info)

			// LLM endpoints
			platformGroup.POST("/llm/generate", platformHandler.LLMGenerate)
			platformGroup.POST("/llm/chat", platformHandler.LLMChat)

			// RAG endpoints
			platformGroup.POST("/rag/ingest", platformHandler.RAGIngest)
			platformGroup.POST("/rag/query", platformHandler.RAGQuery)
			platformGroup.POST("/rag/search", platformHandler.RAGSearch)

			// Tools endpoints
			platformGroup.GET("/tools", platformHandler.ToolsList)
			platformGroup.POST("/tools/call", platformHandler.ToolCall)

			}
		*/

		// RAG endpoints
		ragGroup := api.Group("/rag")
		{
			ingestHandler := ragHandlers.NewIngestHandler(processorService)
			ragGroup.POST("/ingest", ingestHandler.Handle)

			queryHandler := ragHandlers.NewQueryHandler(trackedProcessor)
			ragGroup.POST("/query", queryHandler.Handle)
			ragGroup.POST("/query-stream", queryHandler.HandleStream)
			ragGroup.POST("/search", queryHandler.SearchOnly)

			documentsHandler := ragHandlers.NewDocumentsHandler(processorService)
			ragGroup.GET("/documents", documentsHandler.List)
			ragGroup.GET("/documents/info", documentsHandler.ListWithInfo)
			ragGroup.GET("/documents/:id", documentsHandler.GetDocumentInfo)
			ragGroup.DELETE("/documents/:id", documentsHandler.Delete)

			// Advanced search endpoints
			searchHandler := ragHandlers.NewSearchHandler(processorService)
			ragGroup.POST("/search/semantic", searchHandler.SemanticSearch)
			ragGroup.POST("/search/hybrid", searchHandler.HybridSearch)
			ragGroup.POST("/search/filtered", searchHandler.FilteredSearch)

			resetHandler := ragHandlers.NewResetHandler(processorService)
			ragGroup.POST("/reset", resetHandler.Handle)
		}

		// Keep legacy endpoints for backward compatibility
		ingestHandler := ragHandlers.NewIngestHandler(processorService)
		api.POST("/ingest", ingestHandler.Handle)

		queryHandler := ragHandlers.NewQueryHandler(trackedProcessor)
		api.POST("/query", queryHandler.Handle)
		api.POST("/query-stream", queryHandler.HandleStream)
		api.POST("/search", queryHandler.SearchOnly)

		documentsHandler := ragHandlers.NewDocumentsHandler(processorService)
		api.GET("/documents", documentsHandler.List)
		api.DELETE("/documents/:id", documentsHandler.Delete)

		// Chat endpoints
		chatGroup := api.Group("/chat")
		{
			// Create llm.Service wrapper if needed
			var llmSvc interface{}
			if svc, ok := llmService.(interface{ GetService() interface{} }); ok {
				llmSvc = svc.GetService()
			} else {
				llmSvc = llmService
			}

			// Type assert to get the llm.Service
			if llmServiceTyped, ok := llmSvc.(*llm.Service); ok {
				chatHandler := chatHandlers.NewChatHandler(trackedProcessor, llmServiceTyped, usageService)
				chatGroup.POST("/", chatHandler.Handle)
				chatGroup.POST("/complete", chatHandler.Complete)
			}
		}
		
		// Conversation endpoints
		conversationGroup := api.Group("/conversations")
		{
			convHandler := conversationHandlers.NewHandler(conversationStore)
			conversationGroup.POST("/new", convHandler.CreateNewConversation)
			conversationGroup.POST("/save", convHandler.SaveConversation)
			conversationGroup.GET("", convHandler.ListConversations)
			conversationGroup.GET("/:id", convHandler.GetConversation)
			conversationGroup.DELETE("/:id", convHandler.DeleteConversation)
			conversationGroup.GET("/search", convHandler.SearchConversations)
		}

		// LLM endpoints for direct operations
		llmGroup := api.Group("/llm")
		{
			// Create llm.Service wrapper if needed
			var llmSvc interface{}
			if svc, ok := llmService.(interface{ GetService() interface{} }); ok {
				llmSvc = svc.GetService()
			} else {
				llmSvc = llmService
			}

			// Type assert to get the llm.Service
			if llmServiceTyped, ok := llmSvc.(*llm.Service); ok {
				llmHandler := llmHandlers.NewLLMHandler(llmServiceTyped)
				llmGroup.POST("/generate", llmHandler.Generate)
				llmGroup.POST("/chat", llmHandler.Chat)
				llmGroup.POST("/structured", llmHandler.GenerateStructured)
			}
		}

		// Tools API endpoints have been removed - use MCP servers instead

		api.POST("/reset", ragHandlers.NewResetHandler(processorService).Handle)

		// MCP API endpoints (only if MCP is enabled)
		var mcpHandler *mcpHandlers.MCPHandler
		if cfg.MCP.Enabled {
			// Initialize MCP configuration
			mcpConfig := &mcp.Config{
				Enabled:  cfg.MCP.Enabled,
				Servers:  cfg.MCP.Servers,
				LogLevel: cfg.MCP.LogLevel,
			}

			// Get llm.Service for MCP handler
			var llmServiceForMCP *llm.Service
			// Try to extract llm.Service from domain.Generator
			var llmSvc interface{}
			if svc, ok := llmService.(interface{ GetService() interface{} }); ok {
				llmSvc = svc.GetService()
			} else {
				llmSvc = llmService
			}
			// Type assert to get the llm.Service
			if llmServiceTyped, ok := llmSvc.(*llm.Service); ok {
				llmServiceForMCP = llmServiceTyped
			}

			// Initialize MCP handler with LLM support
			log.Printf("Initializing MCP with config: enabled=%v, servers=%v", mcpConfig.Enabled, mcpConfig.Servers)
			var err error
			if llmServiceForMCP != nil {
				mcpHandler, err = mcpHandlers.NewMCPHandlerWithLLM(mcpConfig, conversationStore, llmServiceForMCP)
			} else {
				log.Printf("Warning: LLM service not available for MCP, using basic handler")
				mcpHandler, err = mcpHandlers.NewMCPHandler(mcpConfig, conversationStore)
			}
			if err != nil {
				log.Printf("Warning: failed to initialize MCP handler: %v", err)
				log.Printf("MCP servers will not be available. Error details: %+v", err)
			} else {
				log.Printf("MCP handler initialized successfully")
				// Create MCP service for agents
				// mcpService used to be passed to agent handlers, no longer needed

				// Setup MCP routes
				mcpGroup := api.Group("/mcp")
				{
					// Tool operations
					mcpGroup.GET("/tools", mcpHandler.ListTools)
					mcpGroup.GET("/tools/:name", mcpHandler.GetTool)
					mcpGroup.POST("/tools/call", mcpHandler.CallTool)
					mcpGroup.POST("/tools/batch", mcpHandler.BatchCallTools)

					// Enhanced MCP operations
					mcpGroup.POST("/chat", mcpHandler.ChatWithMCP)
					mcpGroup.POST("/query", mcpHandler.QueryWithMCP)

					// Server operations
					mcpGroup.GET("/servers", mcpHandler.GetServerStatus)
					mcpGroup.GET("/servers/:server/tools", mcpHandler.GetToolsByServer)
					mcpGroup.POST("/servers/start", mcpHandler.StartServer)
					mcpGroup.POST("/servers/stop", mcpHandler.StopServer)

					// LLM integration
					mcpGroup.GET("/llm/tools", mcpHandler.GetToolsForLLM)
				}

				// Register cleanup on server shutdown
				router.Use(func(c *gin.Context) {
					c.Next()
					// This will be called when server shuts down
					if c.Request.Context().Err() != nil {
						_ = mcpHandler.Close()
					}
				})
			}
		}

		// V1 API endpoints for backward compatibility and analytics
		v1Group := api.Group("/v1")
		{
			// Initialize Gin usage handler for RAG visualization and usage tracking
			usageHandler := usageHandlers.NewGinHandler(usageService)
			
			
			// Conversation routes
			v1Group.GET("/conversations", usageHandler.ListConversations)
			v1Group.POST("/conversations", usageHandler.CreateConversation)
			v1Group.GET("/conversations/:id", usageHandler.GetConversation)
			v1Group.DELETE("/conversations/:id", usageHandler.DeleteConversation)
			v1Group.GET("/conversations/:id/export", usageHandler.ExportConversation)
			
			// Usage statistics routes
			v1Group.GET("/usage/stats", usageHandler.GetUsageStats)
			v1Group.GET("/usage/stats/type", usageHandler.GetUsageStatsByType)
			v1Group.GET("/usage/stats/provider", usageHandler.GetUsageStatsByProvider)
			v1Group.GET("/usage/stats/daily", usageHandler.GetDailyUsage)
			v1Group.GET("/usage/stats/models", usageHandler.GetTopModels)
			
			// Usage records routes
			v1Group.GET("/usage/records", usageHandler.ListUsageRecords)
			v1Group.GET("/usage/records/:id", usageHandler.GetUsageRecord)
			
			// RAG visualization routes
			v1Group.GET("/rag/queries", usageHandler.ListRAGQueries)
			v1Group.GET("/rag/queries/:id", usageHandler.GetRAGQuery)
			v1Group.GET("/rag/queries/:id/visualization", usageHandler.GetRAGVisualization)
			v1Group.GET("/rag/analytics", usageHandler.GetRAGAnalytics)
			v1Group.GET("/rag/performance", usageHandler.GetRAGPerformanceReport)
			
			// Analytics handlers (keep existing ones for backward compatibility)
			analyticsHandler := v1Handlers.NewAnalyticsHandler(usageService)
			
			// Tool calls analytics
			v1Group.GET("/tool-calls/stats", analyticsHandler.GetToolCallStats)
			v1Group.GET("/tool-calls", analyticsHandler.GetToolCalls)
			v1Group.GET("/tool-calls/analytics", analyticsHandler.GetToolCallAnalytics)
			v1Group.GET("/tool-calls/:id/visualization", usageHandler.GetToolCallVisualization)
			
			// Middleware to inject usage service into context
			v1Group.Use(func(c *gin.Context) {
				c.Set("usageService", usageService)
				c.Next()
			})
		}
	}

	if enableUI {
		// Setup static file routes
		if err := web.SetupStaticRoutes(router); err != nil {
			return nil, fmt.Errorf("failed to setup static routes: %w", err)
		}
	}

	return router, nil
}

// printServerInfo 打印服务器访问信息
func printServerInfo(host string, port int, enableUI bool) {
	fmt.Printf("Starting RAGO server on %s:%d\n", host, port)

	// 显示不同的访问地址
	if host == "0.0.0.0" || host == "" {
		// 获取本机IP地址
		localIPs := getLocalIPs()

		fmt.Println("\n📡 Server accessible at:")
		fmt.Printf("   Local:    http://localhost:%d\n", port)
		fmt.Printf("   Local:    http://127.0.0.1:%d\n", port)

		for _, ip := range localIPs {
			fmt.Printf("   Network:  http://%s:%d\n", ip, port)
		}

		if enableUI {
			fmt.Println("\n🌐 Web UI accessible at:")
			fmt.Printf("   Local:    http://localhost:%d\n", port)
			for _, ip := range localIPs {
				fmt.Printf("   Network:  http://%s:%d\n", ip, port)
			}
		}

		fmt.Printf("\n🔗 API endpoints:")
		fmt.Printf("\n   Local:    http://localhost:%d/api\n", port)
		for _, ip := range localIPs {
			fmt.Printf("   Network:  http://%s:%d/api\n", ip, port)
		}
	} else {
		if enableUI {
			fmt.Printf("Web UI: http://%s:%d\n", host, port)
		}
		fmt.Printf("API: http://%s:%d/api\n", host, port)
	}

	fmt.Println("\n💡 Press Ctrl+C to stop the server")
	fmt.Println("")
}

// getLocalIPs 获取本机IP地址
func getLocalIPs() []string {
	var ips []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		// 跳过loopback和down状态的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// 只要IPv4地址，跳过loopback
			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips
}

func init() {
	serveCmd.Flags().IntVar(&port, "port", 0, "server port")
	serveCmd.Flags().StringVar(&host, "host", "", "server host address")
	serveCmd.Flags().BoolVar(&enableUI, "ui", false, "enable Web UI")
}
