package client

import "errors"

// Common errors
var (
	// ErrMCPNotEnabled indicates MCP is not enabled
	ErrMCPNotEnabled = errors.New("MCP is not enabled")

	// ErrRAGNotInitialized indicates RAG client is not initialized
	ErrRAGNotInitialized = errors.New("RAG client not initialized")

	// ErrLLMNotInitialized indicates LLM is not initialized
	ErrLLMNotInitialized = errors.New("LLM not initialized")

	// ErrAgentNotInitialized indicates agent is not initialized
	ErrAgentNotInitialized = errors.New("agent not initialized")
)
