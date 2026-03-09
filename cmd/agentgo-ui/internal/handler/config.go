package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/liliang-cn/agent-go/pkg/config"
	toml "github.com/pelletier/go-toml/v2"
)

type ConfigHandler struct {
	cfg        *config.Config
	configPath string
}

type Config struct {
	ConfigPath      string   `json:"configPath"`
	Home            string   `json:"home"`
	Debug           bool     `json:"debug"`
	ServerHost      string   `json:"serverHost"`
	ServerPort      int      `json:"serverPort"`
	MCPEnabled      bool     `json:"mcpEnabled"`
	MCPAllowedDirs  []string `json:"mcpAllowedDirs"`
	MCPServersPath  string   `json:"mcpServersPath"`
	SkillsPaths     []string `json:"skillsPaths"`
	RAGDBPath       string   `json:"ragDbPath"`
	MemoryStoreType string   `json:"memoryStoreType"`
	MemoryPath      string   `json:"memoryPath"`
	DataDir         string   `json:"dataDir"`
	WorkspaceDir    string   `json:"workspaceDir"`
}

type UpdateConfigRequest struct {
	Home            *string  `json:"home,omitempty"`
	Debug           *bool    `json:"debug,omitempty"`
	ServerHost      *string  `json:"serverHost,omitempty"`
	ServerPort      *int     `json:"serverPort,omitempty"`
	MCPEnabled      *bool    `json:"mcpEnabled,omitempty"`
	MCPAllowedDirs  []string `json:"mcpAllowedDirs,omitempty"`
	SkillsPaths     []string `json:"skillsPaths,omitempty"`
	RAGDBPath       *string  `json:"ragDbPath,omitempty"`
	MemoryStoreType *string  `json:"memoryStoreType,omitempty"`
	MemoryPath      *string  `json:"memoryPath,omitempty"`
}

func NewConfigHandler(cfg *config.Config, configPath string) *ConfigHandler {
	return &ConfigHandler{cfg: cfg, configPath: configPath}
}

func (h *ConfigHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getConfig(w, r)
	case http.MethodPut:
		h.updateConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ConfigHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	JSONResponse(w, h.snapshot())
}

func (h *ConfigHandler) updateConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Home != nil {
		h.cfg.Home = *req.Home
	}
	if req.Debug != nil {
		h.cfg.Debug = *req.Debug
	}
	if req.ServerHost != nil {
		h.cfg.Server.Host = *req.ServerHost
	}
	if req.ServerPort != nil {
		h.cfg.Server.Port = *req.ServerPort
	}
	if req.MCPEnabled != nil {
		h.cfg.MCP.Enabled = *req.MCPEnabled
	}
	if req.MCPAllowedDirs != nil {
		h.cfg.MCP.FilesystemDirs = req.MCPAllowedDirs
	}
	if req.SkillsPaths != nil {
		h.cfg.Skills.Paths = req.SkillsPaths
	}
	if req.RAGDBPath != nil {
		h.cfg.RAG.Storage.DBPath = *req.RAGDBPath
	}
	if req.MemoryStoreType != nil {
		h.cfg.Memory.StoreType = *req.MemoryStoreType
	}
	if req.MemoryPath != nil {
		h.cfg.Memory.MemoryPath = *req.MemoryPath
	}

	if err := h.saveConfig(); err != nil {
		JSONError(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]interface{}{
		"success": true,
		"config":  h.snapshot(),
	})
}

func (h *ConfigHandler) saveConfig() error {
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
	data["debug"] = h.cfg.Debug
	setNested(data, []string{"server", "host"}, h.cfg.Server.Host)
	setNested(data, []string{"server", "port"}, h.cfg.Server.Port)
	setNested(data, []string{"mcp", "enabled"}, h.cfg.MCP.Enabled)
	setNested(data, []string{"mcp", "filesystem_dirs"}, h.cfg.MCP.FilesystemDirs)
	setNested(data, []string{"skills", "paths"}, h.cfg.Skills.Paths)
	setNested(data, []string{"rag", "storage", "db_path"}, h.cfg.RAG.Storage.DBPath)
	setNested(data, []string{"memory", "store_type"}, h.cfg.Memory.StoreType)
	setNested(data, []string{"memory", "memory_path"}, h.cfg.Memory.MemoryPath)

	content, err := toml.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(h.configPath, content, 0644)
}

func (h *ConfigHandler) snapshot() Config {
	return Config{
		ConfigPath:      h.configPath,
		Home:            h.cfg.Home,
		Debug:           h.cfg.Debug,
		ServerHost:      h.cfg.Server.Host,
		ServerPort:      h.cfg.Server.Port,
		MCPEnabled:      h.cfg.MCP.Enabled,
		MCPAllowedDirs:  append([]string{}, h.cfg.MCP.FilesystemDirs...),
		MCPServersPath:  h.cfg.MCPServersPath(),
		SkillsPaths:     append([]string{}, h.cfg.Skills.Paths...),
		RAGDBPath:       h.cfg.RAG.Storage.DBPath,
		MemoryStoreType: h.cfg.Memory.StoreType,
		MemoryPath:      h.cfg.Memory.MemoryPath,
		DataDir:         h.cfg.DataDir(),
		WorkspaceDir:    h.cfg.WorkspaceDir(),
	}
}

func setNested(root map[string]interface{}, path []string, value interface{}) {
	current := root
	for i, key := range path {
		if i == len(path)-1 {
			current[key] = value
			return
		}

		next, ok := current[key].(map[string]interface{})
		if !ok {
			next = map[string]interface{}{}
			current[key] = next
		}
		current = next
	}
}
