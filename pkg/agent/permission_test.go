package agent

import (
	"context"
	"testing"
)

func TestDefaultPermissionPolicy(t *testing.T) {
	t.Parallel()

	if !DefaultPermissionPolicy(PermissionRequest{ToolName: "write_file"}) {
		t.Fatal("expected write_file to require permission")
	}
	if DefaultPermissionPolicy(PermissionRequest{ToolName: "rag_query"}) {
		t.Fatal("expected rag_query to not require permission")
	}
}

func TestAuthorizeTool(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	if err := svc.authorizeTool(context.Background(), PermissionRequest{ToolName: "write_file"}); err != nil {
		t.Fatalf("expected nil handler to allow, got %v", err)
	}

	called := false
	svc.SetPermissionPolicy(DefaultPermissionPolicy)
	svc.SetPermissionHandler(func(ctx context.Context, req PermissionRequest) (*PermissionResponse, error) {
		called = true
		return &PermissionResponse{Allowed: false, Reason: "denied"}, nil
	})

	err := svc.authorizeTool(context.Background(), PermissionRequest{ToolName: "write_file"})
	if err == nil {
		t.Fatal("expected denied permission to fail")
	}
	if !called {
		t.Fatal("expected permission handler to be called")
	}

	called = false
	err = svc.authorizeTool(context.Background(), PermissionRequest{ToolName: "rag_query"})
	if err != nil {
		t.Fatalf("expected low-risk tool to bypass permission handler, got %v", err)
	}
	if called {
		t.Fatal("expected permission handler to be skipped for low-risk tool")
	}
}
