// Package api provides the HTTP API layer for RAGO's four-pillar architecture.
// It exposes REST endpoints for LLM, RAG, MCP, and Agent operations.
package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/liliang-cn/rago/v2/api/handlers/agents"
	"github.com/liliang-cn/rago/v2/api/handlers/llm"
	"github.com/liliang-cn/rago/v2/api/handlers/mcp"
	"github.com/liliang-cn/rago/v2/api/handlers/rag"
	"github.com/liliang-cn/rago/v2/api/handlers/unified"
	"github.com/liliang-cn/rago/v2/api/middleware"
	"github.com/liliang-cn/rago/v2/api/websocket"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server represents the HTTP API server for RAGO
type Server struct {
	config   *Config
	client   *client.Client
	router   *gin.Engine
	server   *http.Server
	wsHub    *websocket.Hub
	logger   zerolog.Logger
	upgrader websocket.Upgrader
}

// Config contains server configuration
type Config struct {
	// Server settings
	Host         string        `toml:"host" env:"RAGO_API_HOST" default:"0.0.0.0"`
	Port         int           `toml:"port" env:"RAGO_API_PORT" default:"7127"`
	ReadTimeout  time.Duration `toml:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `toml:"write_timeout" default:"30s"`
	IdleTimeout  time.Duration `toml:"idle_timeout" default:"120s"`

	// Security settings
	EnableAuth    bool   `toml:"enable_auth" env:"RAGO_API_AUTH_ENABLED" default:"false"`
	AuthType      string `toml:"auth_type" default:"bearer"` // bearer, basic, api_key
	AuthSecret    string `toml:"auth_secret" env:"RAGO_API_AUTH_SECRET"`
	AllowedOrigins []string `toml:"allowed_origins" default:"[\"*\"]"`

	// Rate limiting
	EnableRateLimit bool `toml:"enable_rate_limit" default:"true"`
	RateLimit       int  `toml:"rate_limit" default:"100"` // requests per minute
	RateBurst       int  `toml:"rate_burst" default:"20"`  // burst size

	// Features
	EnableSwagger bool `toml:"enable_swagger" default:"true"`
	EnableMetrics bool `toml:"enable_metrics" default:"true"`
	EnableWS      bool `toml:"enable_websocket" default:"true"`

	// Logging
	LogLevel string `toml:"log_level" default:"info"`
	LogJSON  bool   `toml:"log_json" default:"false"`

	// Client configuration
	ClientConfig string `toml:"client_config" default:"~/.rago/rago.toml"`
}

// NewServer creates a new API server instance
func NewServer(config *Config) (*Server, error) {
	// Setup logging
	zerolog.SetGlobalLevel(parseLogLevel(config.LogLevel))
	if !config.LogJSON {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	logger := log.With().Str("component", "api-server").Logger()

	// Create RAGO client
	ragoClient, err := client.New(config.ClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create RAGO client: %w", err)
	}

	// Create WebSocket hub
	wsHub := websocket.NewHub()

	// Create server instance
	s := &Server{
		config: config,
		client: ragoClient,
		wsHub:  wsHub,
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Configure origin checking based on config
				if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" {
					return true
				}
				origin := r.Header.Get("Origin")
				for _, allowed := range config.AllowedOrigins {
					if origin == allowed {
						return true
					}
				}
				return false
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	// Setup router
	s.setupRouter()

	// Create HTTP server
	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      s.router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return s, nil
}

// setupRouter configures all routes and middleware
func (s *Server) setupRouter() {
	// Set Gin mode based on log level
	if s.config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.New()

	// Global middleware
	s.router.Use(middleware.RequestID())
	s.router.Use(middleware.Logger(s.logger))
	s.router.Use(middleware.Recovery())

	// CORS
	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     s.config.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Authentication middleware (if enabled)
	var authMiddleware gin.HandlerFunc
	if s.config.EnableAuth {
		authMiddleware = middleware.Auth(s.config.AuthType, s.config.AuthSecret)
	}

	// Rate limiting middleware (if enabled)
	var rateLimitMiddleware gin.HandlerFunc
	if s.config.EnableRateLimit {
		rateLimitMiddleware = middleware.RateLimit(s.config.RateLimit, s.config.RateBurst)
	}

	// Setup routes
	s.setupRoutes(authMiddleware, rateLimitMiddleware)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes(authMiddleware, rateLimitMiddleware gin.HandlerFunc) {
	// Health and metrics endpoints (no auth)
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/ready", s.handleReady)

	if s.config.EnableMetrics {
		s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// API documentation
	if s.config.EnableSwagger {
		s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// WebSocket endpoint
	if s.config.EnableWS {
		ws := s.router.Group("/ws")
		if authMiddleware != nil {
			ws.Use(authMiddleware)
		}
		ws.GET("/stream", websocket.NewStreamHandler(s.wsHub, s.client))
		ws.GET("/events", websocket.NewEventHandler(s.wsHub, s.client))
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1")

	// Apply middleware to API routes
	if authMiddleware != nil {
		v1.Use(authMiddleware)
	}
	if rateLimitMiddleware != nil {
		v1.Use(rateLimitMiddleware)
	}

	// LLM endpoints
	llmGroup := v1.Group("/llm")
	{
		llmHandler := llm.NewHandler(s.client)
		llmGroup.POST("/generate", llmHandler.Generate)
		llmGroup.POST("/stream", llmHandler.Stream)
		llmGroup.GET("/providers", llmHandler.ListProviders)
		llmGroup.POST("/providers", llmHandler.AddProvider)
		llmGroup.DELETE("/providers/:name", llmHandler.RemoveProvider)
		llmGroup.GET("/providers/health", llmHandler.GetProviderHealth)
		llmGroup.POST("/batch", llmHandler.GenerateBatch)
		llmGroup.POST("/tools/generate", llmHandler.GenerateWithTools)
		llmGroup.POST("/tools/stream", llmHandler.StreamWithTools)
	}

	// RAG endpoints
	ragGroup := v1.Group("/rag")
	{
		ragHandler := rag.NewHandler(s.client)
		ragGroup.POST("/ingest", ragHandler.IngestDocument)
		ragGroup.POST("/ingest/batch", ragHandler.IngestBatch)
		ragGroup.GET("/documents", ragHandler.ListDocuments)
		ragGroup.DELETE("/documents/:id", ragHandler.DeleteDocument)
		ragGroup.POST("/search", ragHandler.Search)
		ragGroup.POST("/search/hybrid", ragHandler.HybridSearch)
		ragGroup.GET("/stats", ragHandler.GetStats)
		ragGroup.POST("/optimize", ragHandler.Optimize)
		ragGroup.POST("/reset", ragHandler.Reset)
	}

	// MCP endpoints
	mcpGroup := v1.Group("/mcp")
	{
		mcpHandler := mcp.NewHandler(s.client)
		mcpGroup.GET("/servers", mcpHandler.ListServers)
		mcpGroup.POST("/servers", mcpHandler.RegisterServer)
		mcpGroup.DELETE("/servers/:name", mcpHandler.UnregisterServer)
		mcpGroup.GET("/servers/:name/health", mcpHandler.GetServerHealth)
		mcpGroup.GET("/tools", mcpHandler.ListTools)
		mcpGroup.GET("/tools/:name", mcpHandler.GetTool)
		mcpGroup.POST("/tools/:name/call", mcpHandler.CallTool)
		mcpGroup.POST("/tools/:name/call-async", mcpHandler.CallToolAsync)
		mcpGroup.POST("/tools/batch", mcpHandler.CallToolsBatch)
	}

	// Agent endpoints
	agentGroup := v1.Group("/agents")
	{
		agentHandler := agents.NewHandler(s.client)
		agentGroup.GET("/workflows", agentHandler.ListWorkflows)
		agentGroup.POST("/workflows", agentHandler.CreateWorkflow)
		agentGroup.DELETE("/workflows/:name", agentHandler.DeleteWorkflow)
		agentGroup.POST("/workflows/:name/execute", agentHandler.ExecuteWorkflow)
		agentGroup.POST("/workflows/:name/schedule", agentHandler.ScheduleWorkflow)
		agentGroup.GET("/agents", agentHandler.ListAgents)
		agentGroup.POST("/agents", agentHandler.CreateAgent)
		agentGroup.DELETE("/agents/:name", agentHandler.DeleteAgent)
		agentGroup.POST("/agents/:name/execute", agentHandler.ExecuteAgent)
		agentGroup.GET("/scheduled", agentHandler.GetScheduledTasks)
	}

	// Unified endpoints (multi-pillar operations)
	unifiedGroup := v1.Group("/")
	{
		unifiedHandler := unified.NewHandler(s.client)
		unifiedGroup.POST("/chat", unifiedHandler.Chat)
		unifiedGroup.POST("/chat/stream", unifiedHandler.StreamChat)
		unifiedGroup.POST("/process", unifiedHandler.ProcessDocument)
		unifiedGroup.POST("/task", unifiedHandler.ExecuteTask)
	}

	// API v3 routes (Frontend expects these)
	v3 := s.router.Group("/api/v3")

	// Apply middleware to V3 API routes
	if authMiddleware != nil {
		v3.Use(authMiddleware)
	}
	if rateLimitMiddleware != nil {
		v3.Use(rateLimitMiddleware)
	}

	// V3 LLM endpoints - adapted for frontend expectations
	llmV3Group := v3.Group("/llm")
	{
		llmHandler := llm.NewHandler(s.client)
		llmV3Group.GET("/providers", llmHandler.ListProviders)
		llmV3Group.POST("/generate", llmHandler.Generate)
		llmV3Group.POST("/chat", llmHandler.Generate) // Frontend calls /chat but expects same as /generate
	}

	// V3 RAG endpoints - adapted for frontend expectations  
	ragV3Group := v3.Group("/rag")
	{
		ragHandler := rag.NewHandler(s.client)
		ragV3Group.GET("/documents", ragHandler.ListDocuments)
		ragV3Group.POST("/ingest", ragHandler.IngestDocument)
		ragV3Group.POST("/query", ragHandler.Search) // Frontend calls /query but expects same as /search
	}

	// V3 MCP endpoints - adapted for frontend expectations
	mcpV3Group := v3.Group("/mcp")
	{
		mcpHandler := mcp.NewHandler(s.client)
		mcpV3Group.GET("/servers", mcpHandler.ListServers)
		mcpV3Group.GET("/tools", mcpHandler.ListTools)
		mcpV3Group.POST("/tools/call", s.handleV3MCPToolCall) // Custom handler for V3 frontend expectations
	}

	// V3 Agents endpoints - adapted for frontend expectations
	agentV3Group := v3.Group("/agents")
	{
		agentHandler := agents.NewHandler(s.client)
		agentV3Group.POST("/task", agentHandler.ExecuteAgent) // Frontend expects /task for direct execution
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start WebSocket hub
	if s.config.EnableWS {
		go s.wsHub.Run()
	}

	// Setup graceful shutdown
	go s.handleShutdown()

	s.logger.Info().
		Str("host", s.config.Host).
		Int("port", s.config.Port).
		Bool("auth", s.config.EnableAuth).
		Bool("rate_limit", s.config.EnableRateLimit).
		Bool("swagger", s.config.EnableSwagger).
		Bool("metrics", s.config.EnableMetrics).
		Bool("websocket", s.config.EnableWS).
		Msg("Starting API server")

	// Start server
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.logger.Info().Msg("Shutting down API server")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop WebSocket hub
	if s.wsHub != nil {
		s.wsHub.Stop()
	}

	// Shutdown HTTP server
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// Close RAGO client
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("failed to close RAGO client: %w", err)
	}

	s.logger.Info().Msg("API server stopped")
	return nil
}

// handleShutdown handles graceful shutdown on interrupt signals
func (s *Server) handleShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := s.Stop(); err != nil {
		s.logger.Error().Err(err).Msg("Error during shutdown")
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *gin.Context) {
	health := s.client.Health()
	
	status := http.StatusOK
	if health.Status == "unhealthy" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, health)
}

// handleReady handles readiness check requests
func (s *Server) handleReady(c *gin.Context) {
	// Check if all critical services are ready
	health := s.client.Health()
	
	// For readiness, we need at least LLM and RAG to be healthy
	ready := true
	message := "ready"

	if health.LLM.Status == "unhealthy" {
		ready = false
		message = "LLM service not ready"
	}
	if health.RAG.Status == "unhealthy" {
		ready = false
		message = "RAG service not ready"
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"ready":   ready,
		"message": message,
	})
}

// handleV3MCPToolCall handles V3 MCP tool calls where tool name is in request body
func (s *Server) handleV3MCPToolCall(c *gin.Context) {
	var req struct {
		ToolName   string                 `json:"tool_name" binding:"required"`
		Parameters map[string]interface{} `json:"parameters"`
		ServerName string                 `json:"server_name"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create core.ToolCallRequest
	toolReq := core.ToolCallRequest{
		ToolName:   req.ToolName,
		Parameters: req.Parameters,
		ServerName: req.ServerName,
	}

	resp, err := s.client.MCP().CallTool(c.Request.Context(), toolReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// parseLogLevel converts string log level to zerolog level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}