package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liliang-cn/agent-go/pkg/agent"
	"github.com/liliang-cn/agent-go/pkg/config"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	home := t.TempDir()
	cfg := &config.Config{
		Home:  home,
		Debug: true,
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
		RAG: config.RAGConfig{
			Storage: config.CortexdbConfig{
				DBPath: filepath.Join(home, "data", "agentgo.db"),
				TopK:   5,
			},
		},
		Skills: config.SkillsConfig{
			Paths: []string{"skills"},
		},
		Memory: config.MemoryConfig{
			StoreType:  "file",
			MemoryPath: filepath.Join(home, "data", "memories"),
		},
	}
	cfg.MCP.Enabled = true
	cfg.MCP.FilesystemDirs = []string{home}
	return cfg
}

func newTestManager(t *testing.T) *agent.AgentManager {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "agent.db")
	store, err := agent.NewStore(dbPath)
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := agent.NewAgentManager(store)
	if err := manager.SeedDefaultAgents(); err != nil {
		t.Fatalf("seed agents failed: %v", err)
	}
	return manager
}

func TestResolveConfigPath(t *testing.T) {
	cfg := testConfig(t)

	localPath := filepath.Join(".", "agentgo.toml")
	content := []byte("debug = true\n")
	if err := os.WriteFile(localPath, content, 0644); err != nil {
		t.Fatalf("write local config failed: %v", err)
	}
	defer os.Remove(localPath)

	got := resolveConfigPath(cfg)
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute config path, got %s", got)
	}
	if filepath.Base(got) != "agentgo.toml" {
		t.Fatalf("unexpected config path %s", got)
	}
}

func TestConfigHandlerGetAndPut(t *testing.T) {
	cfg := testConfig(t)
	configPath := filepath.Join(t.TempDir(), "config", "agentgo.toml")
	handler := NewConfigHandler(cfg, configPath)

	getReq := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	getRec := httptest.NewRecorder()
	handler.HandleConfig(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d", getRec.Code)
	}

	var getResp Config
	if err := json.Unmarshal(getRec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get response failed: %v", err)
	}
	if getResp.Home != cfg.Home || getResp.ServerPort != cfg.Server.Port {
		t.Fatalf("unexpected get snapshot: %+v", getResp)
	}

	reqBody := UpdateConfigRequest{
		Debug:           boolPtr(false),
		ServerHost:      stringPtr("0.0.0.0"),
		ServerPort:      intPtr(9000),
		MCPAllowedDirs:  []string{"/tmp"},
		SkillsPaths:     []string{"custom"},
		RAGDBPath:       stringPtr("/tmp/agentgo.db"),
		MemoryStoreType: stringPtr("vector"),
		MemoryPath:      stringPtr("/tmp/memory.db"),
	}
	body, _ := json.Marshal(reqBody)
	putReq := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewReader(body))
	putRec := httptest.NewRecorder()
	handler.HandleConfig(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("unexpected put status: %d body=%s", putRec.Code, putRec.Body.String())
	}
	if cfg.Server.Port != 9000 || cfg.Memory.StoreType != "vector" {
		t.Fatalf("expected config mutation, got %+v", cfg)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be written: %v", err)
	}
}

func TestConfigHandlerInvalidMethodAndBody(t *testing.T) {
	cfg := testConfig(t)
	handler := NewConfigHandler(cfg, filepath.Join(t.TempDir(), "agentgo.toml"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/config", nil)
	handler.HandleConfig(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString("{"))
	handler.HandleConfig(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected invalid body status: %d", rec.Code)
	}
}

func TestHandleAgentsAndOperations(t *testing.T) {
	cfg := testConfig(t)
	manager := newTestManager(t)
	h := &Handler{cfg: cfg, agentManager: manager}

	t.Run("list agents", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
		h.HandleAgents(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"agents"`)) {
			t.Fatalf("expected agents payload, got %s", rec.Body.String())
		}
	})

	t.Run("create agent", func(t *testing.T) {
		body := []byte(`{"name":"Writer","description":"Writes","instructions":"Write clearly","enable_mcp":true}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewReader(body))
		h.HandleAgents(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		model, err := manager.GetAgentByName("Writer")
		if err != nil {
			t.Fatalf("expected persisted agent: %v", err)
		}
		if !model.EnableMCP {
			t.Fatal("expected enable_mcp to persist")
		}
	})

	t.Run("create invalid agent", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewReader([]byte(`{"name":"   "}`)))
		h.HandleAgents(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
	})

	t.Run("start stop get agent", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/agents/Coder/start", nil)
		req.URL.Path = "/api/agents/Coder/start"
		h.HandleAgentOperation(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected start status: %d body=%s", rec.Code, rec.Body.String())
		}

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/agents/Coder", nil)
		req.URL.Path = "/api/agents/Coder"
		h.HandleAgentOperation(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected get status: %d body=%s", rec.Code, rec.Body.String())
		}

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/api/agents/Coder/stop", nil)
		req.URL.Path = "/api/agents/Coder/stop"
		h.HandleAgentOperation(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected stop status: %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid operation", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/agents/Coder", nil)
		req.URL.Path = "/api/agents/Coder"
		h.HandleAgentOperation(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("unexpected delete status: %d", rec.Code)
		}
	})
}

func TestHandleAgentsUnavailable(t *testing.T) {
	h := &Handler{}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	h.HandleAgents(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected unavailable status: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/agents/Coder", nil)
	req.URL.Path = "/api/agents/Coder"
	h.HandleAgentOperation(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected operation unavailable status: %d", rec.Code)
	}
}

func TestJSONHelpersAndServeHTTP(t *testing.T) {
	rec := httptest.NewRecorder()
	JSONResponse(rec, map[string]string{"ok": "yes"})
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected json response metadata: status=%d headers=%v", rec.Code, rec.Header())
	}

	rec = httptest.NewRecorder()
	JSONError(rec, "boom", http.StatusTeapot)
	if rec.Code != http.StatusTeapot {
		t.Fatalf("unexpected json error status: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	(&Handler{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unexpected servehttp status: %d", rec.Code)
	}
}

func boolPtr(v bool) *bool       { return &v }
func intPtr(v int) *int          { return &v }
func stringPtr(v string) *string { return &v }

func TestHandleStatusMethodGuard(t *testing.T) {
	h := &Handler{cfg: testConfig(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/status", nil)
	h.HandleStatus(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestHandleStatusMinimal(t *testing.T) {
	h := &Handler{cfg: testConfig(t)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	h.HandleStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"running"`)) {
		t.Fatalf("unexpected status body: %s", rec.Body.String())
	}
}

func TestAgentManagerOperationsPersist(t *testing.T) {
	manager := newTestManager(t)

	created, err := manager.CreateAgent(context.Background(), &agent.AgentModel{Name: "Reviewer", Description: "Reviews", Instructions: "Review"})
	if err != nil {
		t.Fatalf("create agent failed: %v", err)
	}
	if created.Name != "Reviewer" {
		t.Fatalf("unexpected created agent: %+v", created)
	}

	if err := manager.StartAgent(context.Background(), "Reviewer"); err != nil {
		t.Fatalf("start agent failed: %v", err)
	}
	if err := manager.StopAgent(context.Background(), "Reviewer"); err != nil {
		t.Fatalf("stop agent failed: %v", err)
	}
}
