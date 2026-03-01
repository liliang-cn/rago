package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================================
// Gateway - WebSocket Control Plane (inspired by OpenClaw)
// ============================================================

// Gateway provides a WebSocket control plane for LongRun
// Similar to OpenClaw's ws://127.0.0.1:18789
type Gateway struct {
	port        int
	longRun     *LongRunService
	agent       *Service
	upgrader    websocket.Upgrader
	clients     map[*websocket.Conn]bool
	clientsMu   sync.RWMutex
	logger      *slog.Logger
	httpServer  *http.Server

	// Channels
	broadcast   chan GatewayMessage
	register    chan *websocket.Conn
	unregister  chan *websocket.Conn
}

// GatewayMessage represents a message sent over the gateway
type GatewayMessage struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// GatewayConfig configures the gateway
type GatewayConfig struct {
	Port       int    `json:"port"`
	AuthMode   string `json:"auth_mode"` // "none", "password", "token"
	Password   string `json:"password,omitempty"`
	Token      string `json:"token,omitempty"`
	StaticDir  string `json:"static_dir,omitempty"` // Web UI files
}

// DefaultGatewayConfig returns default config
func DefaultGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		Port:     18789,
		AuthMode: "none",
	}
}

// NewGateway creates a new gateway
func NewGateway(longRun *LongRunService, cfg *GatewayConfig) *Gateway {
	if cfg == nil {
		cfg = DefaultGatewayConfig()
	}

	return &Gateway{
		port:    cfg.Port,
		longRun: longRun,
		agent:   longRun.agent,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for local development
			},
		},
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan GatewayMessage, 100),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		logger:     slog.Default().With("module", "gateway"),
	}
}

// Start starts the gateway server
func (g *Gateway) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", g.handleWebSocket)

	// REST API endpoints
	mux.HandleFunc("/api/status", g.handleStatus)
	mux.HandleFunc("/api/tasks", g.handleTasks)
	mux.HandleFunc("/api/tasks/add", g.handleAddTask)
	mux.HandleFunc("/api/checklist", g.handleChecklist)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	g.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", g.port),
		Handler: mux,
	}

	// Start message broadcaster
	go g.runBroadcaster(ctx)

	g.logger.Info("Gateway started", "port", g.port, "ws", fmt.Sprintf("ws://127.0.0.1:%d/ws", g.port))

	return g.httpServer.ListenAndServe()
}

// Stop stops the gateway
func (g *Gateway) Stop(ctx context.Context) error {
	if g.httpServer != nil {
		return g.httpServer.Shutdown(ctx)
	}
	return nil
}

// runBroadcaster handles client registration and message broadcasting
func (g *Gateway) runBroadcaster(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-g.register:
			g.clientsMu.Lock()
			g.clients[client] = true
			g.clientsMu.Unlock()
			g.logger.Debug("Client connected")

		case client := <-g.unregister:
			g.clientsMu.Lock()
			delete(g.clients, client)
			g.clientsMu.Unlock()
			client.Close()
			g.logger.Debug("Client disconnected")

		case msg := <-g.broadcast:
			g.clientsMu.RLock()
			for client := range g.clients {
				err := client.WriteJSON(msg)
				if err != nil {
					client.Close()
					delete(g.clients, client)
				}
			}
			g.clientsMu.RUnlock()
		}
	}
}

// handleWebSocket handles WebSocket connections
func (g *Gateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		g.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	g.register <- conn
	defer func() {
		g.unregister <- conn
	}()

	// Send initial status
	status := g.longRun.GetStatus()
	conn.WriteJSON(GatewayMessage{
		Type:      "status",
		Timestamp: time.Now(),
		Data:      status,
	})

	// Read messages
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var req map[string]interface{}
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}

		g.handleWSMessage(conn, req)
	}
}

// handleWSMessage handles incoming WebSocket messages
func (g *Gateway) handleWSMessage(conn *websocket.Conn, req map[string]interface{}) {
	msgType, _ := req["type"].(string)

	switch msgType {
	case "ping":
		conn.WriteJSON(GatewayMessage{Type: "pong", Timestamp: time.Now()})

	case "get_status":
		status := g.longRun.GetStatus()
		conn.WriteJSON(GatewayMessage{Type: "status", Timestamp: time.Now(), Data: status})

	case "add_task":
		goal, _ := req["goal"].(string)
		if goal != "" {
			task, err := g.longRun.AddTask(context.Background(), goal, nil)
			if err != nil {
				conn.WriteJSON(GatewayMessage{Type: "error", Data: map[string]interface{}{"error": err.Error()}})
			} else {
				conn.WriteJSON(GatewayMessage{Type: "task_added", Data: map[string]interface{}{"task_id": task.ID}})
			}
		}
	}
}

// handleStatus handles GET /api/status
func (g *Gateway) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := g.longRun.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleTasks handles GET /api/tasks
func (g *Gateway) handleTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := g.longRun.queue.GetPendingTasks(r.Context(), 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// handleAddTask handles POST /api/tasks/add
func (g *Gateway) handleAddTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := g.longRun.AddTask(r.Context(), req.Goal, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// handleChecklist handles GET /api/checklist
func (g *Gateway) handleChecklist(w http.ResponseWriter, r *http.Request) {
	checklist, err := g.longRun.readChecklist()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checklist)
}

// Broadcast sends a message to all connected clients
func (g *Gateway) Broadcast(msgType string, data map[string]interface{}) {
	g.broadcast <- GatewayMessage{
		Type:      msgType,
		Timestamp: time.Now(),
		Data:      data,
	}
}
