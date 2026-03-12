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
	"time"

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

func newTestManager(t *testing.T) *agent.SquadManager {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "agent.db")
	store, err := agent.NewStore(dbPath)
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := agent.NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
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
	h := &Handler{cfg: cfg, squadManager: manager, aiChatSessions: map[string]string{}, opsLogs: []OpsLogEntry{}}

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
		body := []byte(`{"name":"Writer","description":"Writes","instructions":"Write clearly","enable_mcp":true,"required_llm_capability":4}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewReader(body))
		h.HandleAgents(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		model, err := manager.GetMemberByName("Writer")
		if err != nil {
			t.Fatalf("expected persisted agent: %v", err)
		}
		if !model.EnableMCP {
			t.Fatal("expected enable_mcp to persist")
		}
		if model.RequiredLLMCapability != 4 {
			t.Fatalf("expected required_llm_capability to persist, got %d", model.RequiredLLMCapability)
		}
		if model.Kind != agent.AgentKindCaptain {
			t.Fatalf("expected captain kind, got %q", model.Kind)
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
		assistant, err := manager.GetMemberByName("Assistant")
		if err != nil {
			t.Fatalf("get seeded assistant failed: %v", err)
		}
		if assistant.Kind != agent.AgentKindCaptain {
			t.Fatalf("expected Assistant to be captain, got %q", assistant.Kind)
		}

		model, err := manager.GetMemberByName("Coder")
		if err != nil {
			t.Fatalf("get seeded agent failed: %v", err)
		}
		if model.Kind != agent.AgentKindSpecialist {
			t.Fatalf("expected Coder to be specialist, got %q", model.Kind)
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/agents/Writer/start", nil)
		req.URL.Path = "/api/agents/Writer/start"
		h.HandleAgentOperation(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected start status: %d body=%s", rec.Code, rec.Body.String())
		}

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/agents/Writer", nil)
		req.URL.Path = "/api/agents/Writer"
		h.HandleAgentOperation(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected get status: %d body=%s", rec.Code, rec.Body.String())
		}

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/api/agents/Writer/stop", nil)
		req.URL.Path = "/api/agents/Writer/stop"
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

	t.Run("ops logs", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/ops/logs?limit=5", nil)
		h.HandleOpsLogs(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected logs status: %d body=%s", rec.Code, rec.Body.String())
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"logs"`)) {
			t.Fatalf("expected logs payload, got %s", rec.Body.String())
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

func TestParseMultiAgentPrompt(t *testing.T) {
	t.Run("extracts mentions and strips prompt", func(t *testing.T) {
		names, prompt := parseMultiAgentPrompt("@Assistant   @Coder compare the API and propose fixes")
		if want := []string{"Assistant", "Coder"}; len(names) != len(want) || names[0] != want[0] || names[1] != want[1] {
			t.Fatalf("unexpected names: %#v", names)
		}
		if prompt != "compare the API and propose fixes" {
			t.Fatalf("unexpected prompt: %q", prompt)
		}
	})

	t.Run("deduplicates mentions", func(t *testing.T) {
		names, prompt := parseMultiAgentPrompt("@Coder @coder ship it")
		if len(names) != 1 || names[0] != "Coder" {
			t.Fatalf("unexpected names: %#v", names)
		}
		if prompt != "ship it" {
			t.Fatalf("unexpected prompt: %q", prompt)
		}
	})
}

func TestHandleMultiAgentChat(t *testing.T) {
	t.Run("unavailable manager", func(t *testing.T) {
		h := &Handler{}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/chat/multi", bytes.NewBufferString(`{"id":"chat-1","messages":[{"role":"user","parts":[{"type":"text","text":"@Coder test"}]}]}`))
		h.HandleMultiAgentChat(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("requires mentions", func(t *testing.T) {
		h := &Handler{squadManager: newTestManager(t)}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/chat/multi", bytes.NewBufferString(`{"id":"chat-1","messages":[{"role":"user","parts":[{"type":"text","text":"plain prompt"}]}]}`))
		h.HandleMultiAgentChat(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandleSquadTasks(t *testing.T) {
	h := &Handler{squadManager: newTestManager(t)}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/squads/tasks", bytes.NewBufferString(`{"captain_name":"Assistant","message":"@Coder say hi","agent_names":["Coder"]}`))
	h.HandleSquadTasks(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected enqueue status: %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"ack_message"`)) {
		t.Fatalf("expected ack payload, got %s", rec.Body.String())
	}

	time.Sleep(50 * time.Millisecond)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/squads/tasks?captain_name=Assistant", nil)
	h.HandleSquadTasks(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"tasks"`)) {
		t.Fatalf("expected tasks payload, got %s", rec.Body.String())
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

	created, err := manager.CreateMember(context.Background(), &agent.AgentModel{Name: "Reviewer", Description: "Reviews", Instructions: "Review", RequiredLLMCapability: 3})
	if err != nil {
		t.Fatalf("create agent failed: %v", err)
	}
	if created.Name != "Reviewer" {
		t.Fatalf("unexpected created agent: %+v", created)
	}
	if created.RequiredLLMCapability != 3 {
		t.Fatalf("unexpected required capability: %+v", created)
	}

	if err := manager.EnableCommander(context.Background(), "Reviewer"); err != nil {
		t.Fatalf("start agent failed: %v", err)
	}
	if err := manager.DisableCommander(context.Background(), "Reviewer"); err != nil {
		t.Fatalf("stop agent failed: %v", err)
	}
}

func TestHandleChatAISDKAgentUnavailable(t *testing.T) {
	h := &Handler{cfg: testConfig(t), aiChatSessions: map[string]string{}}

	body := []byte(`{
		"id":"chat-1",
		"mode":"agent",
		"agent_name":"Coder",
		"messages":[{"role":"user","parts":[{"type":"text","text":"hello"}]}]
	}`)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	h.HandleChat(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`Agent manager unavailable`)) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestDequeueToolCallID(t *testing.T) {
	queue := map[string][]string{
		"read_file": {"call-1", "call-2"},
	}

	if got := dequeueToolCallID(queue, "read_file"); got != "call-1" {
		t.Fatalf("unexpected first call id: %s", got)
	}
	if got := dequeueToolCallID(queue, "read_file"); got != "call-2" {
		t.Fatalf("unexpected second call id: %s", got)
	}
	if got := dequeueToolCallID(queue, "read_file"); got != "" {
		t.Fatalf("expected empty id, got %s", got)
	}
}
