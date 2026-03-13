package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/liliang-cn/agent-go/pkg/pool"
	toml "github.com/pelletier/go-toml/v2"
)

type SetupHandler struct {
	cfg        *config.Config
	configPath string
}

type SetupProvider struct {
	Name           string `json:"name"`
	BaseURL        string `json:"baseUrl"`
	APIKey         string `json:"apiKey,omitempty"`
	ModelName      string `json:"modelName"`
	EmbeddingModel string `json:"embeddingModel,omitempty"`
	MaxConcurrency int    `json:"maxConcurrency"`
	Capability     int    `json:"capability"`
}

type SetupState struct {
	Initialized      bool            `json:"initialized"`
	ConfigPath       string          `json:"configPath"`
	Home             string          `json:"home"`
	WorkingDirectory string          `json:"workingDirectory"`
	ServerHost       string          `json:"serverHost"`
	ServerPort       int             `json:"serverPort"`
	MCPEnabled       bool            `json:"mcpEnabled"`
	MCPAllowedDirs   []string        `json:"mcpAllowedDirs"`
	SkillsPaths      []string        `json:"skillsPaths"`
	RAGDBPath        string          `json:"ragDbPath"`
	MemoryStoreType  string          `json:"memoryStoreType"`
	MemoryPath       string          `json:"memoryPath"`
	Providers        []SetupProvider `json:"providers"`
}

type ApplySetupRequest struct {
	Home            string        `json:"home"`
	ServerHost      string        `json:"serverHost"`
	ServerPort      int           `json:"serverPort"`
	MCPEnabled      bool          `json:"mcpEnabled"`
	MemoryStoreType string        `json:"memoryStoreType"`
	Provider        SetupProvider `json:"provider"`
}

func NewSetupHandler(cfg *config.Config, configPath string) *SetupHandler {
	return &SetupHandler{cfg: cfg, configPath: configPath}
}

func (h *SetupHandler) HandleSetup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		JSONResponse(w, h.snapshot())
	case http.MethodPut:
		h.apply(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SetupHandler) snapshot() SetupState {
	providers := make([]SetupProvider, 0, len(h.cfg.LLM.Providers))
	for _, p := range h.cfg.LLM.Providers {
		providers = append(providers, SetupProvider{
			Name:           p.Name,
			BaseURL:        p.BaseURL,
			ModelName:      p.ModelName,
			MaxConcurrency: p.MaxConcurrency,
			Capability:     p.Capability,
		})
	}

	return SetupState{
		Initialized:      h.isInitialized(),
		ConfigPath:       h.configPath,
		Home:             h.cfg.Home,
		WorkingDirectory: h.cfg.WorkspaceDir(),
		ServerHost:       h.cfg.Server.Host,
		ServerPort:       h.cfg.Server.Port,
		MCPEnabled:       h.cfg.MCP.Enabled,
		MCPAllowedDirs:   append([]string{}, h.cfg.MCP.FilesystemDirs...),
		SkillsPaths:      h.cfg.SkillsPaths(),
		RAGDBPath:        h.cfg.RAG.Storage.DBPath,
		MemoryStoreType:  h.cfg.Memory.StoreType,
		MemoryPath:       h.cfg.Memory.MemoryPath,
		Providers:        providers,
	}
}

func (h *SetupHandler) isInitialized() bool {
	content, err := os.ReadFile(h.configPath)
	if err != nil || len(content) == 0 {
		return false
	}
	var data map[string]interface{}
	if err := toml.Unmarshal(content, &data); err != nil {
		return false
	}
	setup, ok := data["setup"].(map[string]interface{})
	if !ok {
		return false
	}
	completed, _ := setup["completed"].(bool)
	return completed
}

func (h *SetupHandler) apply(w http.ResponseWriter, r *http.Request) {
	var req ApplySetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Home == "" {
		JSONError(w, "home is required", http.StatusBadRequest)
		return
	}
	if req.ServerHost == "" || req.ServerPort <= 0 {
		JSONError(w, "server host and port are required", http.StatusBadRequest)
		return
	}
	if req.Provider.Name == "" || req.Provider.BaseURL == "" || req.Provider.ModelName == "" {
		JSONError(w, "provider name, baseUrl, and modelName are required", http.StatusBadRequest)
		return
	}

	h.cfg.Home = req.Home
	h.cfg.Server.Host = req.ServerHost
	h.cfg.Server.Port = req.ServerPort
	h.cfg.MCP.Enabled = req.MCPEnabled
	h.cfg.Memory.StoreType = req.MemoryStoreType
	h.cfg.ApplyHomeLayout()
	h.cfg.LLM.Enabled = true
	h.cfg.LLM.Strategy = pool.StrategyRoundRobin
	h.cfg.LLM.Providers = []pool.Provider{{
		Name:           req.Provider.Name,
		BaseURL:        req.Provider.BaseURL,
		Key:            req.Provider.APIKey,
		ModelName:      req.Provider.ModelName,
		MaxConcurrency: req.Provider.MaxConcurrency,
		Capability:     req.Provider.Capability,
	}}
	h.cfg.RAG.Embedding.Enabled = req.Provider.EmbeddingModel != ""
	if req.Provider.EmbeddingModel != "" {
		h.cfg.RAG.Embedding.Strategy = pool.StrategyRoundRobin
		h.cfg.RAG.Embedding.Providers = []pool.Provider{{
			Name:           req.Provider.Name,
			BaseURL:        req.Provider.BaseURL,
			Key:            req.Provider.APIKey,
			ModelName:      req.Provider.EmbeddingModel,
			MaxConcurrency: req.Provider.MaxConcurrency,
			Capability:     req.Provider.Capability,
		}}
	} else {
		h.cfg.RAG.Embedding.Providers = nil
	}

	if err := h.saveSetupConfig(req); err != nil {
		JSONError(w, "Failed to save setup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]interface{}{
		"success":         true,
		"requiresRestart": true,
		"setup":           h.snapshot(),
	})
}

func (h *SetupHandler) saveSetupConfig(req ApplySetupRequest) error {
	dir := filepath.Dir(h.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data := map[string]interface{}{}
	if content, err := os.ReadFile(h.configPath); err == nil && len(content) > 0 {
		if err := toml.Unmarshal(content, &data); err != nil {
			return err
		}
	}

	data["home"] = h.cfg.Home
	setNested(data, []string{"server", "host"}, h.cfg.Server.Host)
	setNested(data, []string{"server", "port"}, h.cfg.Server.Port)
	setNested(data, []string{"mcp", "enabled"}, h.cfg.MCP.Enabled)
	setNested(data, []string{"memory", "store_type"}, h.cfg.Memory.StoreType)
	deleteNested(data, []string{"mcp", "filesystem_dirs"})
	deleteNested(data, []string{"rag", "storage", "db_path"})
	deleteNested(data, []string{"memory", "memory_path"})
	deleteNested(data, []string{"cache", "path"})
	setNested(data, []string{"llm", "enabled"}, true)
	setNested(data, []string{"llm", "strategy"}, string(pool.StrategyRoundRobin))
	setNested(data, []string{"llm", "providers"}, []map[string]interface{}{{
		"name":            req.Provider.Name,
		"base_url":        req.Provider.BaseURL,
		"key":             req.Provider.APIKey,
		"model_name":      req.Provider.ModelName,
		"max_concurrency": req.Provider.MaxConcurrency,
		"capability":      req.Provider.Capability,
	}})
	setNested(data, []string{"rag", "embedding", "enabled"}, req.Provider.EmbeddingModel != "")
	if req.Provider.EmbeddingModel != "" {
		setNested(data, []string{"rag", "embedding", "strategy"}, string(pool.StrategyRoundRobin))
		setNested(data, []string{"rag", "embedding", "providers"}, []map[string]interface{}{{
			"name":            req.Provider.Name,
			"base_url":        req.Provider.BaseURL,
			"key":             req.Provider.APIKey,
			"model_name":      req.Provider.EmbeddingModel,
			"max_concurrency": req.Provider.MaxConcurrency,
			"capability":      req.Provider.Capability,
		}})
	}
	setNested(data, []string{"setup", "completed"}, true)
	setNested(data, []string{"setup", "updated_at"}, time.Now().Format(time.RFC3339))

	content, err := toml.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(h.configPath, content, 0644)
}
