package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	ragolog "github.com/liliang-cn/rago/v2/pkg/log"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/skills"
	"github.com/spf13/cobra"
)

//go:embed all:dist
var staticFS embed.FS

var (
	uiPort    int
	uiHost    string
	cfgFile   string
	uiVersion string = "dev"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Println("Error:", err)
	}
}

func Execute() error {
	var rootCmd = &cobra.Command{
		Use:   "rago-ui",
		Short: "RAGO Web UI Server",
		Long:  `RAGO Web UI provides a web interface for interacting with RAGO's RAG and Agent capabilities.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Initialize global pool service
			globalPoolService := services.GetGlobalPoolService()
			ctx := context.Background()
			if err := globalPoolService.Initialize(ctx, cfg); err != nil {
				return fmt.Errorf("failed to initialize global pool service: %w", err)
			}

			return nil
		},
		RunE: runServer,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file path")
	rootCmd.PersistentFlags().IntVarP(&uiPort, "port", "p", 7127, "port to run the UI server on")
	rootCmd.PersistentFlags().StringVar(&uiHost, "host", "0.0.0.0", "host to bind the UI server to")
	rootCmd.Version = uiVersion

	return rootCmd.Execute()
}

func runServer(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get pool service
	poolService := services.GetGlobalPoolService()

	// Get LLM and Embedder from pool
	llm, err := poolService.GetLLMService()
	if err != nil {
		return fmt.Errorf("failed to get LLM service: %w", err)
	}

	embedder, err := poolService.GetEmbeddingService(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get embedding service: %w", err)
	}

	// Create RAG client
	ragClient, err := rag.NewClient(cfg, embedder, llm, nil)
	if err != nil {
		return fmt.Errorf("failed to create RAG client: %w", err)
	}

	// Create Skills service
	skillsService, err := skills.NewService(&skills.Config{
		Paths:    []string{".skills"},
		Enabled:  true,
	})
	if err != nil {
		ragolog.Warn("Failed to create skills service: %v", err)
	}

	// Create MCP service
	mcpConfig := &mcp.Config{
		Enabled:           cfg.MCP.Enabled,
		Servers:           cfg.MCP.Servers,
		ServersConfigPath: cfg.MCP.ServersConfigPath,
	}
	var mcpService *mcp.Service
	if cfg.MCP.Enabled {
		mcpService, err = mcp.NewService(mcpConfig, llm)
		if err != nil {
			ragolog.Warn("Failed to create MCP service: %v", err)
		}
	}

	// Create API router
	mux := http.NewServeMux()

	// API routes
	apiHandler := &apiHandler{
		cfg:           cfg,
		ragClient:     ragClient,
		skillsService: skillsService,
		mcpService:    mcpService,
		llm:           llm,
	}

	// RAG endpoints
	mux.HandleFunc("/api/query", apiHandler.handleQuery)
	mux.HandleFunc("/api/documents", apiHandler.handleDocuments)
	mux.HandleFunc("/api/collections", apiHandler.handleCollections)
	mux.HandleFunc("/api/status", apiHandler.handleStatus)
	mux.HandleFunc("/api/chat", apiHandler.handleChat)
	mux.HandleFunc("/api/ingest", apiHandler.handleIngest)

	// Skills endpoints
	mux.HandleFunc("/api/skills", apiHandler.handleSkillsList)
	mux.HandleFunc("/api/skills/add", apiHandler.handleSkillsAdd)
	mux.HandleFunc("/api/skills/", apiHandler.handleSkillsOperation)

	// MCP endpoints
	mux.HandleFunc("/api/mcp/servers", apiHandler.handleMCPServers)
	mux.HandleFunc("/api/mcp/tools", apiHandler.handleMCPTools)
	mux.HandleFunc("/api/mcp/add", apiHandler.handleMCPAddServer)
	mux.HandleFunc("/api/mcp/call", apiHandler.handleMCPCallTool)

	// Serve static files
	distFS, err := fs.Sub(staticFS, "dist")
	if err != nil {
		return fmt.Errorf("failed to load static files: %w", err)
	}

	// SPA fallback - serve index.html for unmatched routes
	fileServer := http.FileServer(http.FS(distFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve static file first
		if r.URL.Path != "/" {
			if _, err := distFS.Open(r.URL.Path[1:]); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// Serve index.html for SPA routes
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("%s:%d", uiHost, uiPort)
	ragolog.Info("Starting RAGO UI server on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server.ListenAndServe()
}

type apiHandler struct {
	cfg           *config.Config
	ragClient     *rag.Client
	skillsService *skills.Service
	mcpService    *mcp.Service
	llm           domain.Generator
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ============================================
// RAG Handlers
// ============================================

func (h *apiHandler) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query      string `json:"query"`
		Collection string `json:"collection"`
		TopK       int    `json:"top_k"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.TopK == 0 {
		req.TopK = 5
	}

	ctx := r.Context()
	opts := &rag.QueryOptions{
		TopK:        req.TopK,
		Temperature: 0.7,
		ShowSources: true,
	}

	result, err := h.ragClient.Query(ctx, req.Query, opts)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, result)
}

func (h *apiHandler) handleDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	documents, err := h.ragClient.ListDocuments(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, documents)
}

func (h *apiHandler) handleCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.ragClient.GetStats(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	collections := []map[string]interface{}{
		{"name": "default", "count": stats.TotalDocuments},
	}

	jsonResponse(w, collections)
}

func (h *apiHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerStatuses := []map[string]interface{}{}

	poolService := services.GetGlobalPoolService()
	llmStatus := poolService.GetLLMPoolStatus()
	embeddingStatus := poolService.GetEmbeddingPoolStatus()

	for name := range llmStatus {
		providerStatuses = append(providerStatuses, map[string]interface{}{
			"name":    name,
			"status":  "online",
			"latency": 0,
			"type":    "llm",
		})
	}

	for name := range embeddingStatus {
		providerStatuses = append(providerStatuses, map[string]interface{}{
			"name":    name,
			"status":  "online",
			"latency": 0,
			"type":    "embedding",
		})
	}

	response := map[string]interface{}{
		"status":    "running",
		"version":   uiVersion,
		"providers": providerStatuses,
	}

	jsonResponse(w, response)
}

func (h *apiHandler) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	ctx := r.Context()

	session, err := h.ragClient.StartChat(ctx, map[string]interface{}{
		"session_id": req.SessionID,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := h.ragClient.Chat(ctx, session.ID, req.Message, &rag.QueryOptions{
		Temperature: 0.7,
		ShowSources: false,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"response":   response.Answer,
		"session_id": session.ID,
	})
}

func (h *apiHandler) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, _, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "No file provided", http.StatusBadRequest)
		return
	}

	jsonError(w, "File ingestion via UI is not yet implemented. Use the CLI to ingest files.", http.StatusNotImplemented)
}

// ============================================
// Skills Handlers
// ============================================

func (h *apiHandler) handleSkillsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.skillsService == nil {
		jsonResponse(w, []interface{}{})
		return
	}

	skillList, err := h.skillsService.ListSkills(r.Context(), skills.SkillFilter{})
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to serializable format
	result := make([]map[string]interface{}, len(skillList))
	for i, s := range skillList {
		result[i] = map[string]interface{}{
			"id":            s.ID,
			"name":          s.Name,
			"description":   s.Description,
			"version":       s.Version,
			"enabled":       s.Enabled,
			"user_invocable": s.UserInvocable,
			"path":          s.Path,
		}
	}

	jsonResponse(w, result)
}

func (h *apiHandler) handleSkillsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.skillsService == nil {
		jsonError(w, "Skills service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Content     string                 `json:"content"`
		Variables   map[string]interface{} `json:"variables"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Content == "" {
		jsonError(w, "Name and content are required", http.StatusBadRequest)
		return
	}

	// Return not implemented - skill creation requires file system access
	jsonError(w, "Skill creation via UI is not yet implemented. Create SKILL.md files in .skills/ directory.", http.StatusNotImplemented)
}

func (h *apiHandler) handleSkillsOperation(w http.ResponseWriter, r *http.Request) {
	// Extract skill ID from path
	path := r.URL.Path
	if len(path) <= len("/api/skills/") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	skillID := path[len("/api/skills/"):]

	if h.skillsService == nil {
		jsonError(w, "Skills service not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get skill details
		skill, err := h.skillsService.GetSkill(r.Context(), skillID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		result := map[string]interface{}{
			"id":            skill.ID,
			"name":          skill.Name,
			"description":   skill.Description,
			"version":       skill.Version,
			"enabled":       skill.Enabled,
			"user_invocable": skill.UserInvocable,
			"path":          skill.Path,
		}
		jsonResponse(w, result)

	case http.MethodDelete:
		// Delete skill - not implemented
		jsonError(w, "Skill deletion via UI is not yet implemented. Delete the SKILL.md file manually.", http.StatusNotImplemented)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ============================================
// MCP Handlers
// ============================================

func (h *apiHandler) handleMCPServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.mcpService == nil {
		jsonResponse(w, []interface{}{})
		return
	}

	servers := h.mcpService.ListServers()
	jsonResponse(w, servers)
}

func (h *apiHandler) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.mcpService == nil {
		jsonResponse(w, []interface{}{})
		return
	}

	tools := h.mcpService.GetAvailableTools(r.Context())
	jsonResponse(w, tools)
}

func (h *apiHandler) handleMCPAddServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.mcpService == nil {
		jsonError(w, "MCP service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Type    string   `json:"type"`
		URL     string   `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		jsonError(w, "Server name is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if req.Command != "" {
		// Add stdio server
		if err := h.mcpService.AddDynamicServer(ctx, req.Name, req.Command, req.Args); err != nil {
			jsonError(w, fmt.Sprintf("Failed to add server: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		jsonError(w, "Command is required for stdio servers", http.StatusBadRequest)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"name":    req.Name,
	})
}

func (h *apiHandler) handleMCPCallTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.mcpService == nil {
		jsonError(w, "MCP service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ToolName  string                 `json:"tool_name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ToolName == "" {
		jsonError(w, "Tool name is required", http.StatusBadRequest)
		return
	}

	result, err := h.mcpService.CallTool(r.Context(), req.ToolName, req.Arguments)
	if err != nil {
		jsonError(w, fmt.Sprintf("Tool call failed: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, result)
}

// Helper functions
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
