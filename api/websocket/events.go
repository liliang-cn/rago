package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/client"
)

// EventHandler handles WebSocket event connections
type EventHandler struct {
	hub    *Hub
	client *client.Client
}

// NewEventHandler creates a new event handler
func NewEventHandler(hub *Hub, client *client.Client) gin.HandlerFunc {
	h := &EventHandler{
		hub:    hub,
		client: client,
	}
	return h.Handle
}

// Handle manages WebSocket connections for events
func (h *EventHandler) Handle(c *gin.Context) {
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

	// Send welcome message
	welcome := Event{
		Type:      "connected",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"client_id": clientID,
			"message":   "Connected to RAGO event stream",
		},
	}
	h.sendEvent(wsClient, welcome)

	// Start monitoring and sending events
	go h.monitorSystem(wsClient)

	// Start goroutines for reading and writing
	go wsClient.WritePump()
	go h.handleEventSubscriptions(wsClient)

	// Start read pump (blocks until connection closes)
	wsClient.ReadPump()
}

// handleEventSubscriptions handles event subscription messages
func (h *EventHandler) handleEventSubscriptions(client *Client) {
	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			break
		}

		// Parse subscription message
		var sub EventSubscription
		if err := json.Unmarshal(message, &sub); err != nil {
			h.sendErrorEvent(client, "invalid subscription format")
			continue
		}

		// Handle subscription
		switch sub.Action {
		case "subscribe":
			h.handleSubscribe(client, sub)
		case "unsubscribe":
			h.handleUnsubscribe(client, sub)
		default:
			h.sendErrorEvent(client, "unknown action")
		}
	}
}

// handleSubscribe handles event subscriptions
func (h *EventHandler) handleSubscribe(client *Client, sub EventSubscription) {
	// In production, maintain subscription state per client
	// For now, send confirmation
	event := Event{
		Type:      "subscribed",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"events": sub.Events,
		},
	}
	h.sendEvent(client, event)
}

// handleUnsubscribe handles event unsubscriptions
func (h *EventHandler) handleUnsubscribe(client *Client, sub EventSubscription) {
	event := Event{
		Type:      "unsubscribed",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"events": sub.Events,
		},
	}
	h.sendEvent(client, event)
}

// monitorSystem monitors system events and sends them to the client
func (h *EventHandler) monitorSystem(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Send health check event
			health := h.client.Health()
			event := Event{
				Type:      "health",
				Timestamp: time.Now(),
				Data:      health,
			}
			h.sendEvent(client, event)

		case <-client.conn.LocalAddr().Network():
			// Connection closed
			return
		}
	}
}

// sendEvent sends an event to the client
func (h *EventHandler) sendEvent(client *Client, event Event) {
	data, _ := json.Marshal(event)
	select {
	case client.send <- data:
	default:
		// Client send channel is full
	}
}

// sendErrorEvent sends an error event
func (h *EventHandler) sendErrorEvent(client *Client, errMsg string) {
	event := Event{
		Type:      "error",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"error": errMsg,
		},
	}
	h.sendEvent(client, event)
}

// Event represents a system event
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// EventSubscription represents an event subscription request
type EventSubscription struct {
	Action string   `json:"action"` // subscribe, unsubscribe
	Events []string `json:"events"` // event types to subscribe to
}

// SystemEvent types
const (
	EventTypeHealth       = "health"
	EventTypeProviderUp   = "provider_up"
	EventTypeProviderDown = "provider_down"
	EventTypeToolExecuted = "tool_executed"
	EventTypeWorkflowStart = "workflow_start"
	EventTypeWorkflowEnd   = "workflow_end"
	EventTypeDocumentIngested = "document_ingested"
	EventTypeSearchPerformed = "search_performed"
	EventTypeError = "error"
)