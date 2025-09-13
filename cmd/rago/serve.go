package rago

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/internal/api/handlers"
	chatHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/chat"
	llmHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/llm"
	mcpHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/mcp"
	ragHandlers "github.com/liliang-cn/rago/v2/internal/api/handlers/rag"
	"github.com/liliang-cn/rago/v2/pkg/rag/chunker"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
	"github.com/liliang-cn/rago/v2/pkg/web"
	"github.com/spf13/cobra"
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

		// Initialize stores
		vectorStore, err := store.NewSQLiteStore(
			cfg.Sqvect.DBPath,
		)
		if err != nil {
			return fmt.Errorf("failed to create vector store: %w", err)
		}
		defer func() {
			if err := vectorStore.Close(); err != nil {
				fmt.Printf("Warning: failed to close vector store: %v\n", err)
			}
		}()

		docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

		// Initialize services using shared provider system
		ctx := context.Background()
		embedService, llmService, metadataExtractor, err := initializeProviders(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize providers: %w", err)
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
		)

		// 设置Gin为release模式
		gin.SetMode(gin.ReleaseMode)

		router, err := setupRouter(processorService, cfg, embedService, llmService)
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

func setupRouter(processor *processor.Service, cfg *config.Config, embedService domain.Embedder, llmService domain.Generator) (*gin.Engine, error) {
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

		// RAG endpoints
		ragGroup := api.Group("/rag")
		{
			ingestHandler := ragHandlers.NewIngestHandler(processor)
			ragGroup.POST("/ingest", ingestHandler.Handle)

			queryHandler := ragHandlers.NewQueryHandler(processor)
			ragGroup.POST("/query", queryHandler.Handle)
			ragGroup.POST("/query-stream", queryHandler.HandleStream)
			ragGroup.POST("/search", queryHandler.SearchOnly)

			documentsHandler := ragHandlers.NewDocumentsHandler(processor)
			ragGroup.GET("/documents", documentsHandler.List)
			ragGroup.GET("/documents/info", documentsHandler.ListWithInfo)
			ragGroup.GET("/documents/:id", documentsHandler.GetDocumentInfo)
			ragGroup.DELETE("/documents/:id", documentsHandler.Delete)
			
			// Advanced search endpoints
			searchHandler := ragHandlers.NewSearchHandler(processor)
			ragGroup.POST("/search/semantic", searchHandler.SemanticSearch)
			ragGroup.POST("/search/hybrid", searchHandler.HybridSearch)
			ragGroup.POST("/search/filtered", searchHandler.FilteredSearch)

			resetHandler := ragHandlers.NewResetHandler(processor)
			ragGroup.POST("/reset", resetHandler.Handle)
		}

		// Keep legacy endpoints for backward compatibility
		ingestHandler := ragHandlers.NewIngestHandler(processor)
		api.POST("/ingest", ingestHandler.Handle)

		queryHandler := ragHandlers.NewQueryHandler(processor)
		api.POST("/query", queryHandler.Handle)
		api.POST("/query-stream", queryHandler.HandleStream)
		api.POST("/search", queryHandler.SearchOnly)

		documentsHandler := ragHandlers.NewDocumentsHandler(processor)
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
				chatHandler := chatHandlers.NewChatHandler(processor, llmServiceTyped)
				chatGroup.POST("/", chatHandler.Handle)
				chatGroup.POST("/complete", chatHandler.Complete)
			}
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

		// Tools API endpoints (only if tools are enabled)
		if cfg.Tools.Enabled {
			// Initialize tools handler
			toolsHandler := mcpHandlers.NewToolsHandler(processor.GetToolRegistry(), processor.GetToolExecutor())

			tools := api.Group("/tools")
			{
				tools.GET("", toolsHandler.ListTools)
				tools.GET("/:name", toolsHandler.GetTool)
				tools.POST("/:name/execute", toolsHandler.ExecuteTool)
				tools.GET("/stats", toolsHandler.GetToolStats)
				tools.GET("/registry/stats", toolsHandler.GetRegistryStats)
				tools.GET("/executions", toolsHandler.ListExecutions)
				tools.GET("/executions/:id", toolsHandler.GetExecution)
				tools.DELETE("/executions/:id", toolsHandler.CancelExecution)
			}
		}

		api.POST("/reset", ragHandlers.NewResetHandler(processor).Handle)

		// MCP API endpoints (only if MCP is enabled)
		var mcpHandler *mcpHandlers.MCPHandler
		if cfg.MCP.Enabled {
			// Initialize MCP configuration
			mcpConfig := &mcp.Config{
				Enabled:  cfg.MCP.Enabled,
				Servers:  cfg.MCP.Servers,
				LogLevel: cfg.MCP.LogLevel,
			}

			// Initialize MCP handler
			var err error
			mcpHandler, err = mcpHandlers.NewMCPHandler(mcpConfig)
			if err != nil {
				log.Printf("Warning: failed to initialize MCP handler: %v", err)
			} else {
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

		// Agent functionality is available via CLI: rago agent run
		// Web API for agents has been simplified and moved to CLI-only
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
