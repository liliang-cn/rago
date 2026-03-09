package agent

import "time"

// AgentStatus represents the lifecycle state of a persistent agent
type AgentStatus string

const (
	AgentStatusStopped AgentStatus = "stopped"
	AgentStatusRunning AgentStatus = "running"
	AgentStatusError   AgentStatus = "error"
)

// AgentModel represents the configuration of a dynamic agent in the database.
type AgentModel struct {
	ID                    string      `json:"id"`
	Name                  string      `json:"name"`
	Description           string      `json:"description"`
	Instructions          string      `json:"instructions"`
	Model                 string      `json:"model"`
	RequiredLLMCapability int         `json:"required_llm_capability"`
	Status                AgentStatus `json:"status"` // "running", "stopped", "error"
	MCPTools              []string    `json:"mcp_tools"`
	Skills                []string    `json:"skills"`
	EnableRAG             bool        `json:"enable_rag"`
	EnableMemory          bool        `json:"enable_memory"`
	EnablePTC             bool        `json:"enable_ptc"`
	EnableMCP             bool        `json:"enable_mcp"`
	CreatedAt             time.Time   `json:"created_at"`
	UpdatedAt             time.Time   `json:"updated_at"`
}
