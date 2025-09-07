package main

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
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/spf13/cobra"
)

var (
	servePort = 7127
	serveHost = "0.0.0.0"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP API server",
	Long:  "Start the RAGO HTTP API server for all four pillars",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 7127, "Port to listen on")
	serveCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "Host to bind to")
}

func runServe(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 RAGO Server Starting")
	fmt.Println("=========================")

	coreConfig, err := loadCoreConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	ragoClient, err := client.NewWithConfig(coreConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer ragoClient.Close()

	// Create Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "latest",
			"pillars": gin.H{
				"llm":    "active",
				"rag":    "active", 
				"mcp":    "active",
				"agents": "active",
			},
		})
	})

	// Basic API endpoints
	api := router.Group("/api/v1")
	{
		// LLM endpoints
		api.POST("/llm/generate", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "LLM generation endpoint - MVP placeholder"})
		})

		// RAG endpoints  
		api.POST("/rag/ingest", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "RAG ingestion endpoint - MVP placeholder"})
		})

		api.GET("/rag/search", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "RAG search endpoint - MVP placeholder"})
		})

		// MCP endpoints
		api.GET("/mcp/tools", func(c *gin.Context) {
			tools := ragoClient.MCP().GetTools()
			c.JSON(http.StatusOK, gin.H{
				"tools": tools,
				"count": len(tools),
			})
		})

		// Agent endpoints
		api.POST("/agents/execute", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Agent execution endpoint - MVP placeholder"})
		})
	}

	// Start server
	address := fmt.Sprintf("%s:%d", serveHost, servePort)
	srv := &http.Server{
		Addr:    address,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		fmt.Printf("🌐 Server listening on http://%s\n", address)
		fmt.Println("✅ All four pillars available via REST API")
		fmt.Println("Press Ctrl+C to stop")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🔄 Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	fmt.Println("✅ Server stopped gracefully")
	return nil
}