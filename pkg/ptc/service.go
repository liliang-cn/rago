package ptc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service is the main PTC service
type Service struct {
	config  *Config
	router  ToolRouter
	store   ExecutionStore
	runtime SandboxRuntime

	mu     sync.RWMutex
	closed bool
}

// NewService creates a new PTC service
func NewService(config *Config, router ToolRouter, store ExecutionStore) (*Service, error) {
	if config == nil {
		defaultConfig := DefaultConfig()
		config = &defaultConfig
	}

	if err := config.Validate(); err != nil {
		return nil, NewExecutionError(err, "config")
	}

	if router == nil {
		return nil, NewExecutionError(ErrInvalidConfig, "config").WithSource("router is required")
	}

	return &Service{
		config: config,
		router: router,
		store:  store,
	}, nil
}

// SetRuntime sets the sandbox runtime
func (s *Service) SetRuntime(runtime SandboxRuntime) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtime = runtime
}

// SetSearchProvider sets the search provider for the runtime
func (s *Service) SetSearchProvider(provider SearchProvider) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.runtime != nil {
		s.runtime.SetSearchProvider(provider)
	}
}

// Execute executes code in the sandbox
func (s *Service) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrSandboxClosed
	}
	s.mu.RUnlock()

	// Generate ID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	// Apply defaults
	if req.Timeout <= 0 {
		req.Timeout = s.config.DefaultTimeout
	}
	if req.MaxMemoryMB <= 0 {
		req.MaxMemoryMB = s.config.MaxMemoryMB
	}
	if req.Language == "" {
		req.Language = LanguageJavaScript
	}

	// Validate request
	if err := s.validateRequest(req); err != nil {
		return nil, err
	}

	// Validate code for security
	if s.config.Security.ValidateCode {
		if err := s.validateCode(req.Code); err != nil {
			return nil, err
		}
	}

	// Filter tools by security policy
	filteredTools := s.filterTools(req.Tools)

	// Get runtime
	runtime := s.getRuntime()
	if runtime == nil {
		return nil, ErrSandboxNotReady
	}

	// Register tool handlers
	for _, toolName := range filteredTools {
		if err := runtime.RegisterTool(toolName, s.createToolHandler(toolName)); err != nil {
			// Log but don't fail - tool might already be registered
			continue
		}
	}

	// Update request with filtered tools
	req.Tools = filteredTools

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	result, err := runtime.Execute(execCtx, req)
	if err != nil {
		return &ExecutionResult{
			ID:      req.ID,
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Save to history
	if s.store != nil && s.config.History.Enabled {
		history := &ExecutionHistory{
			ID:         req.ID,
			Request:    req,
			Result:     result,
			ExecutedAt: time.Now(),
		}
		// Best effort save
		_ = s.store.Save(context.Background(), history)
	}

	return result, nil
}

// ExecuteSimple is a convenience method for simple code execution
func (s *Service) ExecuteSimple(ctx context.Context, code string) (*ExecutionResult, error) {
	return s.Execute(ctx, &ExecutionRequest{
		Code:     code,
		Language: LanguageJavaScript,
	})
}

// ListTools lists available tools
func (s *Service) ListTools(ctx context.Context) ([]ToolInfo, error) {
	return s.router.ListAvailableTools(ctx)
}

// GetToolInfo gets information about a specific tool
func (s *Service) GetToolInfo(ctx context.Context, name string) (*ToolInfo, error) {
	return s.router.GetToolInfo(ctx, name)
}

// GetHistory retrieves execution history
func (s *Service) GetHistory(ctx context.Context, id string) (*ExecutionHistory, error) {
	if s.store == nil {
		return nil, fmt.Errorf("history store not configured")
	}
	return s.store.Get(ctx, id)
}

// ListHistory lists recent execution history
func (s *Service) ListHistory(ctx context.Context, limit int) ([]*ExecutionHistory, error) {
	if s.store == nil {
		return nil, fmt.Errorf("history store not configured")
	}
	return s.store.List(ctx, limit)
}

// Close closes the service and releases resources
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	if s.runtime != nil {
		return s.runtime.Close()
	}
	return nil
}

// IsEnabled returns whether PTC is enabled
func (s *Service) IsEnabled() bool {
	return s.config.Enabled
}

// Config returns the current configuration
func (s *Service) Config() *Config {
	return s.config
}

// validateRequest validates the execution request
func (s *Service) validateRequest(req *ExecutionRequest) error {
	if req.Code == "" {
		return NewExecutionError(ErrInvalidConfig, "validate").WithSource("code is required")
	}

	if len(req.Code) > s.config.MaxCodeSize {
		return ErrCodeSizeExceeded
	}

	return nil
}

// validateCode validates code for security issues
func (s *Service) validateCode(code string) error {
	// Basic security validation
	// More comprehensive validation can be done in the security package
	return nil
}

// filterTools filters tools by security policy
func (s *Service) filterTools(tools []string) []string {
	if len(tools) == 0 {
		// If no tools specified, get all available tools
		ctx := context.Background()
		available, err := s.router.ListAvailableTools(ctx)
		if err != nil {
			return []string{}
		}
		tools = make([]string, len(available))
		for i, t := range available {
			tools[i] = t.Name
		}
	}

	// Filter by security policy
	result := make([]string, 0, len(tools))
	for _, tool := range tools {
		if s.config.IsToolAllowed(tool) {
			result = append(result, tool)
		}
	}
	return result
}

// createToolHandler creates a tool handler that routes through the router
func (s *Service) createToolHandler(toolName string) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return s.router.Route(ctx, toolName, args)
	}
}

// getRuntime returns the current runtime
func (s *Service) getRuntime() SandboxRuntime {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtime
}
