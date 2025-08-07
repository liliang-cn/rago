package rago

import (
	"context"
	"fmt"
	"log"
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

		llmService, err := llm.NewOllamaService(
			cfg.Ollama.BaseURL,
			cfg.Ollama.LLMModel,
			cfg.Ollama.Timeout,
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
			docStore,
		)

		if quiet {
			gin.SetMode(gin.ReleaseMode)
		}

		router := setupRouter(processorService, cfg)

		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", host, port),
			Handler: router,
		}

		go func() {
			fmt.Printf("Starting RAGO server on %s:%d\n", host, port)
			if enableUI {
				fmt.Printf("Web UI: http://%s:%d\n", host, port)
			}
			fmt.Printf("API: http://%s:%d/api\n", host, port)

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

func setupRouter(processor *processor.Service, cfg *config.Config) *gin.Engine {
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
		api.POST("/search", queryHandler.SearchOnly)

		documentsHandler := handlers.NewDocumentsHandler(processor)
		api.GET("/documents", documentsHandler.List)
		api.DELETE("/documents/:id", documentsHandler.Delete)

		api.POST("/reset", handlers.NewResetHandler(processor).Handle)
	}

	if enableUI {
		router.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"title": cfg.UI.Title,
				"theme": cfg.UI.Theme,
			})
		})
	}

	return router
}

func init() {
	serveCmd.Flags().IntVar(&port, "port", 0, "server port")
	serveCmd.Flags().StringVar(&host, "host", "", "server host address")
	serveCmd.Flags().BoolVar(&enableUI, "ui", false, "enable Web UI")
}
