// Package goja provides a JavaScript runtime using the Goja interpreter
package goja

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/liliang-cn/rago/v2/pkg/ptc"
)

// Runtime implements SandboxRuntime using Goja
type Runtime struct {
	mu       sync.RWMutex
	tools    map[string]ptc.ToolHandler
	closed   bool
	maxCalls int
}

// NewRuntime creates a new Goja runtime
func NewRuntime() *Runtime {
	return &Runtime{
		tools:    make(map[string]ptc.ToolHandler),
		maxCalls: 20,
	}
}

// NewRuntimeWithConfig creates a new Goja runtime with configuration
func NewRuntimeWithConfig(config *ptc.Config) *Runtime {
	r := NewRuntime()
	if config != nil && config.MaxToolCalls > 0 {
		r.maxCalls = config.MaxToolCalls
	}
	return r
}

// Type returns the runtime type
func (r *Runtime) Type() ptc.RuntimeType {
	return ptc.RuntimeGoja
}

// Execute runs JavaScript code in the sandbox
func (r *Runtime) Execute(ctx context.Context, req *ptc.ExecutionRequest) (*ptc.ExecutionResult, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, ptc.ErrSandboxClosed
	}
	r.mu.RUnlock()

	start := time.Now()
	result := &ptc.ExecutionResult{
		ID:        req.ID,
		ToolCalls: make([]ptc.ToolCallRecord, 0),
		Logs:      make([]string, 0),
	}

	// Create a new VM for this execution
	vm := goja.New()

	// Set up execution state
	state := &executionState{
		tools:     r.tools,
		toolCalls: &result.ToolCalls,
		logs:      &result.Logs,
		callCount: 0,
		maxCalls:  r.maxCalls,
		ctx:       ctx,
	}

	// Register built-in functions
	r.registerBuiltins(vm, state)

	// Inject context variables
	for name, value := range req.Context {
		if err := vm.Set(name, value); err != nil {
			result.Error = fmt.Sprintf("failed to inject context variable '%s': %v", name, err)
			result.Duration = time.Since(start)
			return result, ptc.NewExecutionError(err, "execute")
		}
	}

	// Execute the code
	done := make(chan struct{})
	var execErr error
	var returnValue goja.Value

	go func() {
		defer close(done)
		defer func() {
			if recovered := recover(); recovered != nil {
				execErr = fmt.Errorf("panic: %v", recovered)
			}
		}()

		returnValue, execErr = vm.RunString(req.Code)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Execution completed
	case <-ctx.Done():
		result.Error = ptc.ErrExecutionTimeout.Error()
		result.Duration = time.Since(start)
		return result, ptc.ErrExecutionTimeout
	case <-time.After(req.Timeout):
		result.Error = ptc.ErrExecutionTimeout.Error()
		result.Duration = time.Since(start)
		return result, ptc.ErrExecutionTimeout
	}

	if execErr != nil {
		result.Error = execErr.Error()
		result.Duration = time.Since(start)
		return result, ptc.NewExecutionError(execErr, "execute")
	}

	// Extract return value
	if returnValue != nil && !goja.IsUndefined(returnValue) {
		result.ReturnValue = returnValue.Export()
	}

	// Get console output as main output
	if len(*state.logs) > 0 {
		result.Output = (*state.logs)[len(*state.logs)-1]
	}

	result.Success = true
	result.Duration = time.Since(start)
	return result, nil
}

// RegisterTool registers a tool handler
func (r *Runtime) RegisterTool(name string, handler ptc.ToolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = handler
	return nil
}

// UnregisterTool removes a tool handler
func (r *Runtime) UnregisterTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
	return nil
}

// ListTools returns all registered tool names
func (r *Runtime) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Close releases resources
func (r *Runtime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	r.tools = nil
	return nil
}

// executionState holds state during execution
type executionState struct {
	tools     map[string]ptc.ToolHandler
	toolCalls *[]ptc.ToolCallRecord
	logs      *[]string
	callCount int
	maxCalls  int
	ctx       context.Context
}

// registerBuiltins registers built-in functions
func (r *Runtime) registerBuiltins(vm *goja.Runtime, state *executionState) {
	// Console object
	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		*state.logs = append(*state.logs, fmt.Sprint(args...))
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		*state.logs = append(*state.logs, "ERROR: "+fmt.Sprint(args...))
		return goja.Undefined()
	})
	_ = vm.Set("console", console)

	// callTool function
	_ = vm.Set("callTool", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("callTool requires at least 1 argument"))
		}

		toolName := call.Arguments[0].String()

		// Check call limit
		state.callCount++
		if state.callCount > state.maxCalls {
			panic(vm.NewTypeError("maximum tool calls exceeded"))
		}

		// Get tool handler
		handler, ok := state.tools[toolName]
		if !ok {
			panic(vm.NewTypeError(fmt.Sprintf("tool '%s' not found", toolName)))
		}

		// Parse arguments
		args := make(map[string]interface{})
		if len(call.Arguments) > 1 {
			argVal := call.Arguments[1].Export()
			switch v := argVal.(type) {
			case map[string]interface{}:
				args = v
			case string:
				// Try to parse as JSON
				if err := json.Unmarshal([]byte(v), &args); err != nil {
					args = map[string]interface{}{"value": v}
				}
			default:
				// Try to convert via JSON
				jsonBytes, err := json.Marshal(v)
				if err == nil {
					_ = json.Unmarshal(jsonBytes, &args)
				}
			}
		}

		// Call tool
		start := time.Now()
		toolRecord := ptc.ToolCallRecord{
			ToolName:  toolName,
			Arguments: args,
		}

		result, err := handler(state.ctx, args)
		toolRecord.Duration = time.Since(start)

		if err != nil {
			toolRecord.Error = err.Error()
		} else {
			toolRecord.Result = result
		}

		*state.toolCalls = append(*state.toolCalls, toolRecord)

		if err != nil {
			panic(vm.NewTypeError(fmt.Sprintf("tool '%s' failed: %v", toolName, err)))
		}

		return vm.ToValue(result)
	})

	// sleep function (for convenience)
	_ = vm.Set("sleep", func(ms int64) {
		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
		case <-state.ctx.Done():
		}
	})

	// JSON helper
	jsonObj := vm.NewObject()
	_ = jsonObj.Set("parse", func(s string) interface{} {
		var result interface{}
		if err := json.Unmarshal([]byte(s), &result); err != nil {
			panic(vm.NewTypeError(err.Error()))
		}
		return result
	})
	_ = jsonObj.Set("stringify", func(v interface{}) string {
		b, err := json.Marshal(v)
		if err != nil {
			panic(vm.NewTypeError(err.Error()))
		}
		return string(b)
	})
	_ = vm.Set("JSON", jsonObj)
}
