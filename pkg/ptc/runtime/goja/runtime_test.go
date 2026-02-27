package goja

import (
	"context"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/ptc"
)

func TestRuntime_Type(t *testing.T) {
	runtime := NewRuntime()
	defer runtime.Close()

	if runtime.Type() != ptc.RuntimeGoja {
		t.Errorf("expected runtime type %s, got %s", ptc.RuntimeGoja, runtime.Type())
	}
}

func TestRuntime_ExecuteSimple(t *testing.T) {
	runtime := NewRuntime()
	defer runtime.Close()

	ctx := context.Background()
	req := &ptc.ExecutionRequest{
		Code:     "1 + 1",
		Language: ptc.LanguageJavaScript,
		Timeout:  10 * time.Second,
	}

	result, err := runtime.Execute(ctx, req)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	if result.ReturnValue == nil {
		t.Error("expected return value")
	}

	// The return value should be 2
	if rv, ok := result.ReturnValue.(int64); ok {
		if rv != 2 {
			t.Errorf("expected return value 2, got %d", rv)
		}
	} else if rv, ok := result.ReturnValue.(float64); ok {
		if rv != 2 {
			t.Errorf("expected return value 2, got %f", rv)
		}
	}
}

func TestRuntime_ConsoleLog(t *testing.T) {
	runtime := NewRuntime()
	defer runtime.Close()

	ctx := context.Background()
	req := &ptc.ExecutionRequest{
		Code:     "console.log('hello'); console.log('world');",
		Language: ptc.LanguageJavaScript,
		Timeout:  10 * time.Second,
	}

	result, err := runtime.Execute(ctx, req)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	if len(result.Logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(result.Logs))
	}

	if result.Logs[0] != "hello" {
		t.Errorf("expected first log 'hello', got %s", result.Logs[0])
	}
}

func TestRuntime_ContextVariables(t *testing.T) {
	runtime := NewRuntime()
	defer runtime.Close()

	ctx := context.Background()
	req := &ptc.ExecutionRequest{
		Code:     "x + y",
		Language: ptc.LanguageJavaScript,
		Context: map[string]interface{}{
			"x": 10,
			"y": 20,
		},
		Timeout: 10 * time.Second,
	}

	result, err := runtime.Execute(ctx, req)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

func TestRuntime_ToolRegistration(t *testing.T) {
	runtime := NewRuntime()
	defer runtime.Close()

	// Register a test tool
	err := runtime.RegisterTool("test_tool", func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"result": "ok"}, nil
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	tools := runtime.ListTools()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	if tools[0] != "test_tool" {
		t.Errorf("expected tool 'test_tool', got %s", tools[0])
	}
}

func TestRuntime_CallTool(t *testing.T) {
	runtime := NewRuntime()
	defer runtime.Close()

	// Register a test tool
	err := runtime.RegisterTool("add", func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return a + b, nil
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	ctx := context.Background()
	req := &ptc.ExecutionRequest{
		Code:     "callTool('add', {a: 5, b: 3})",
		Language: ptc.LanguageJavaScript,
		Timeout:  10 * time.Second,
	}

	result, err := runtime.Execute(ctx, req)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	if len(result.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(result.ToolCalls))
	}

	if result.ToolCalls[0].ToolName != "add" {
		t.Errorf("expected tool name 'add', got %s", result.ToolCalls[0].ToolName)
	}
}

func TestRuntime_Timeout(t *testing.T) {
	runtime := NewRuntime()
	defer runtime.Close()

	ctx := context.Background()
	req := &ptc.ExecutionRequest{
		Code:     "while(true) {}",
		Language: ptc.LanguageJavaScript,
		Timeout:  100 * time.Millisecond,
	}

	_, err := runtime.Execute(ctx, req)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestRuntime_SyntaxError(t *testing.T) {
	runtime := NewRuntime()
	defer runtime.Close()

	ctx := context.Background()
	req := &ptc.ExecutionRequest{
		Code:     "invalid javascript syntax {{{",
		Language: ptc.LanguageJavaScript,
		Timeout:  10 * time.Second,
	}

	result, err := runtime.Execute(ctx, req)
	if err == nil {
		t.Error("expected syntax error")
	}

	if result.Success {
		t.Error("expected failure")
	}
}

func TestRuntime_Closed(t *testing.T) {
	runtime := NewRuntime()

	// Close the runtime
	if err := runtime.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	ctx := context.Background()
	req := &ptc.ExecutionRequest{
		Code:     "1 + 1",
		Language: ptc.LanguageJavaScript,
		Timeout:  10 * time.Second,
	}

	_, err := runtime.Execute(ctx, req)
	if err != ptc.ErrSandboxClosed {
		t.Errorf("expected ErrSandboxClosed, got %v", err)
	}
}
