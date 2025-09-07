package websocket

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// StreamHandler handles WebSocket streaming connections
type StreamHandler struct {
	hub    *Hub
	client *client.Client
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(hub *Hub, client *client.Client) gin.HandlerFunc {
	h := &StreamHandler{
		hub:    hub,
		client: client,
	}
	return h.Handle
}

// Handle manages WebSocket connections for streaming
func (h *StreamHandler) Handle(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to upgrade connection"})
		return
	}

	// Create client
	clientID := uuid.New().String()
	wsClient := NewClient(h.hub, conn, clientID)

	// Register client
	h.hub.RegisterClient(wsClient)

	// Start goroutines for reading and writing
	go wsClient.WritePump()
	go h.handleStreamMessages(wsClient)

	// Start read pump (blocks until connection closes)
	wsClient.ReadPump()
}

// handleStreamMessages handles incoming stream messages
func (h *StreamHandler) handleStreamMessages(client *Client) {
	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			break
		}

		// Parse message
		var msg StreamMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			h.sendError(client, "invalid message format")
			continue
		}

		// Handle different message types
		switch msg.Type {
		case "generate":
			h.handleGenerate(client, msg)
		case "chat":
			h.handleChat(client, msg)
		case "search":
			h.handleSearch(client, msg)
		case "tool":
			h.handleTool(client, msg)
		default:
			h.sendError(client, "unknown message type")
		}
	}
}

// handleGenerate handles generation requests
func (h *StreamHandler) handleGenerate(client *Client, msg StreamMessage) {
	var req core.GenerationRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.sendError(client, "invalid generation request")
		return
	}

	// Stream generation
	err := h.client.LLM().Stream(msg.Context, req, func(chunk core.StreamChunk) error {
		response := StreamResponse{
			ID:   msg.ID,
			Type: "chunk",
			Data: chunk,
		}
		data, _ := json.Marshal(response)
		client.send <- data
		return nil
	})

	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	// Send completion
	h.sendComplete(client, msg.ID)
}

// handleChat handles chat requests
func (h *StreamHandler) handleChat(client *Client, msg StreamMessage) {
	var req core.ChatRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.sendError(client, "invalid chat request")
		return
	}

	// Stream chat
	err := h.client.StreamChat(msg.Context, req, func(chunk core.StreamChunk) error {
		response := StreamResponse{
			ID:   msg.ID,
			Type: "chunk",
			Data: chunk,
		}
		data, _ := json.Marshal(response)
		client.send <- data
		return nil
	})

	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	h.sendComplete(client, msg.ID)
}

// handleSearch handles search requests
func (h *StreamHandler) handleSearch(client *Client, msg StreamMessage) {
	var req core.SearchRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.sendError(client, "invalid search request")
		return
	}

	// Execute search
	resp, err := h.client.RAG().Search(msg.Context, req)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	// Send results
	response := StreamResponse{
		ID:   msg.ID,
		Type: "result",
		Data: resp,
	}
	data, _ := json.Marshal(response)
	client.send <- data

	h.sendComplete(client, msg.ID)
}

// handleTool handles tool execution requests
func (h *StreamHandler) handleTool(client *Client, msg StreamMessage) {
	var req core.ToolCallRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.sendError(client, "invalid tool request")
		return
	}

	// Execute tool asynchronously
	respChan, err := h.client.MCP().CallToolAsync(msg.Context, req)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	// Wait for response
	select {
	case resp := <-respChan:
		response := StreamResponse{
			ID:   msg.ID,
			Type: "result",
			Data: resp,
		}
		data, _ := json.Marshal(response)
		client.send <- data
		h.sendComplete(client, msg.ID)

	case <-msg.Context.Done():
		h.sendError(client, "request cancelled")
	}
}

// sendError sends an error response
func (h *StreamHandler) sendError(client *Client, errMsg string) {
	response := StreamResponse{
		Type:  "error",
		Error: errMsg,
	}
	data, _ := json.Marshal(response)
	client.send <- data
}

// sendComplete sends a completion message
func (h *StreamHandler) sendComplete(client *Client, id string) {
	response := StreamResponse{
		ID:   id,
		Type: "complete",
	}
	data, _ := json.Marshal(response)
	client.send <- data
}

// StreamMessage represents an incoming WebSocket message
type StreamMessage struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Context context.Context `json:"-"`
}

// StreamResponse represents an outgoing WebSocket message
type StreamResponse struct {
	ID    string      `json:"id,omitempty"`
	Type  string      `json:"type"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}