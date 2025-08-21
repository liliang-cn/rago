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
	"github.com/liliang-cn/rago/api/handlers"
	"github.com/liliang-cn/rago/internal/chunker"
	"github.com/liliang-cn/rago/internal/config"
	"github.com/liliang-cn/rago/internal/embedder"
	"github.com/liliang-cn/rago/internal/llm"
	"github.com/liliang-cn/rago/internal/processor"
	"github.com/liliang-cn/rago/internal/store"
	"github.com/liliang-cn/rago/internal/web"
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

		processorService := processor.New(
			embedService,
			llmService,
			chunkerService,
			vectorStore,
			keywordStore,
			docStore,
			cfg,
			llmService,
		)

		// 设置Gin为release模式
		gin.SetMode(gin.ReleaseMode)

		router, err := setupRouter(processorService, cfg)
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

func setupRouter(processor *processor.Service, cfg *config.Config) (*gin.Engine, error) {
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

		ingestHandler := handlers.NewIngestHandler(processor)
		api.POST("/ingest", ingestHandler.Handle)

		queryHandler := handlers.NewQueryHandler(processor)
		api.POST("/query", queryHandler.Handle)
		api.POST("/query-stream", queryHandler.HandleStream)
		api.POST("/search", queryHandler.SearchOnly)

		documentsHandler := handlers.NewDocumentsHandler(processor)
		api.GET("/documents", documentsHandler.List)
		api.DELETE("/documents/:id", documentsHandler.Delete)

		api.POST("/reset", handlers.NewResetHandler(processor).Handle)
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
