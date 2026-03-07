package mcp

import (
	"testing"
)

func TestNewServer(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}

	if server.mcpServer == nil {
		t.Fatal("expected mcpServer to be non-nil")
	}

	if server.searcher == nil {
		t.Fatal("expected searcher to be non-nil")
	}
}

func TestServer_RegisterTools(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server.mcpServer == nil {
		t.Fatal("MCP server should be initialized")
	}
}
