package skills

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Service orchestrates all skill functionality
type Service struct {
	config     *Config
	store      SkillStore
	loader     *Loader
	registry   *Registry

	// Integration services (optional)
	ragService interface{} // domain.RAGProcessor
	mcpService interface{} // mcp.Service
	agentSvc   interface{} // agent.Service

	mu         sync.RWMutex
	loaded     bool
}

// NewService creates a new skills service
func NewService(cfg *Config) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	svc := &Service{
		config: cfg,
		loader: NewLoader(cfg.Paths),
	}

	// Create registry
	svc.registry = NewRegistry()

	return svc, nil
}

// SetStore sets the skill store
func (s *Service) SetStore(store SkillStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store = store
}

// SetRAGService sets the RAG service for integration
func (s *Service) SetRAGService(ragService interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ragService = ragService
}

// SetMCPService sets the MCP service for integration
func (s *Service) SetMCPService(mcpService interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpService = mcpService
}

// SetAgentService sets the agent service for integration
func (s *Service) SetAgentService(agentSvc interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentSvc = agentSvc
}

// LoadAll loads skills from all configured paths
func (s *Service) LoadAll(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	skills, err := s.loader.LoadAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	// Register skills
	for _, skill := range skills {
		s.registry.Register(skill)

		// Save to store if available
		if s.store != nil {
			if err := s.store.SaveSkill(ctx, skill); err != nil {
				return fmt.Errorf("failed to save skill %s: %w", skill.ID, err)
			}
		}
	}

	s.loaded = true
	return nil
}

// IsLoaded returns whether skills have been loaded
func (s *Service) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}

// ListSkills lists available skills
func (s *Service) ListSkills(ctx context.Context, filter SkillFilter) ([]*Skill, error) {
	if s.store != nil {
		return s.store.ListSkills(ctx, filter)
	}

	// Return from registry
	s.mu.RLock()
	defer s.mu.RUnlock()

	allSkills := s.registry.List()
	var result []*Skill

	for _, skill := range allSkills {
		if s.matchesFilter(skill, filter) {
			result = append(result, skill)
		}
	}

	return result, nil
}

// matchesFilter checks if a skill matches the filter
func (s *Service) matchesFilter(skill *Skill, filter SkillFilter) bool {
	if filter.Category != "" && skill.Category != filter.Category {
		return false
	}
	if filter.Tag != "" {
		found := false
		for _, tag := range skill.Tags {
			if tag == filter.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if filter.Enabled != nil && *filter.Enabled != skill.Enabled {
		return false
	}
	if filter.SearchTerm != "" {
		searchTerm := strings.ToLower(filter.SearchTerm)
		combined := strings.ToLower(skill.Name + " " + skill.Description)
		if !strings.Contains(combined, searchTerm) {
			return false
		}
	}
	return true
}

// GetSkill retrieves a skill by ID
func (s *Service) GetSkill(ctx context.Context, id string) (*Skill, error) {
	if s.store != nil {
		return s.store.GetSkill(ctx, id)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	skill := s.registry.Get(id)
	if skill == nil {
		return nil, fmt.Errorf("skill not found: %s", id)
	}

	return skill, nil
}

// Resolve resolves skills for a given query
func (s *Service) Resolve(ctx context.Context, query string) ([]*Skill, error) {
	skills, err := s.ListSkills(ctx, SkillFilter{})
	if err != nil {
		return nil, err
	}

	// Simple keyword matching for now
	// Can be enhanced with embeddings/semantic search
	queryLower := strings.ToLower(query)
	var matched []*Skill

	for _, skill := range skills {
		if !skill.Enabled || !skill.UserInvocable {
			continue
		}

		// Check name and description
		if strings.Contains(strings.ToLower(skill.Name), queryLower) ||
			strings.Contains(strings.ToLower(skill.Description), queryLower) {
			matched = append(matched, skill)
			continue
		}

		// Check tags
		for _, tag := range skill.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				matched = append(matched, skill)
				break
			}
		}
	}

	return matched, nil
}

// Execute executes a skill
func (s *Service) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error) {
	start := time.Now()

	// Get skill
	skill, err := s.GetSkill(ctx, req.SkillID)
	if err != nil {
		return &ExecutionResult{
			Success:    false,
			SkillID:    req.SkillID,
			Error:      err.Error(),
			ExecutedAt: start,
			Duration:   time.Since(start),
		}, err
	}

	if !skill.Enabled {
		return &ExecutionResult{
			Success:    false,
			SkillID:    req.SkillID,
			Error:      fmt.Sprintf("skill %s is disabled", skill.ID),
			ExecutedAt: start,
			Duration:   time.Since(start),
		}, fmt.Errorf("skill %s is disabled", skill.ID)
	}

	var output string
	var execErr error

	// Handle different skill types
	switch skill.ID {
	case "rag-query", "rag":
		output, execErr = s.executeRAGQuery(ctx, req)
	default:
		// Generic skill execution
		if skill.Command != "" {
			output = fmt.Sprintf("Command: %s\nVariables: %v", skill.Command, req.Variables)
		} else {
			output = fmt.Sprintf("Skill '%s' executed. Variables: %v", skill.Name, req.Variables)
		}
	}

	result := &ExecutionResult{
		Success:    execErr == nil,
		SkillID:    req.SkillID,
		Output:     output,
		Variables:  req.Variables,
		ExecutedAt: start,
		Duration:   time.Since(start),
	}

	if execErr != nil {
		result.Error = execErr.Error()
	}

	// Save execution result if store is available
	if s.store != nil {
		_ = s.store.SaveExecution(ctx, result)
	}

	return result, nil
}

// executeRAGQuery executes a RAG query
func (s *Service) executeRAGQuery(ctx context.Context, req *ExecutionRequest) (string, error) {
	if s.ragService == nil {
		return "", fmt.Errorf("RAG service not configured")
	}

	// Get query from variables
	query, ok := req.Variables["query"].(string)
	if !ok {
		return "", fmt.Errorf("query variable is required")
	}

	// Get top_k from variables (default 5)
	topK := 5
	if tk, ok := req.Variables["top_k"].(float64); ok {
		topK = int(tk)
	} else if tk, ok := req.Variables["top_k"].(int); ok {
		topK = tk
	}

	// Get temperature from variables (default 0.7)
	temperature := 0.7
	if temp, ok := req.Variables["temperature"].(float64); ok {
		temperature = temp
	}

	// Call RAG service - need to use the processor interface
	// The ragService should be a processor.Service or similar
	if ragProc, ok := s.ragService.(interface {
		Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error)
	}); ok {
		ragReq := domain.QueryRequest{
			Query:        query,
			TopK:         topK,
			Temperature:  temperature,
			ShowSources:  true,
			ShowThinking: false,
		}

		resp, err := ragProc.Query(ctx, ragReq)
		if err != nil {
			return "", fmt.Errorf("RAG query failed: %w", err)
		}

		// Format output
		output := fmt.Sprintf("Question: %s\n\nAnswer:\n%s\n", query, resp.Answer)
		if len(resp.Sources) > 0 {
			output += fmt.Sprintf("\nSources: %d documents found", len(resp.Sources))
		}
		return output, nil
	}

	return "", fmt.Errorf("RAG service does not support Query interface")
}

// BuildSystemPrompt builds a system prompt for LLM with skills context
func (s *Service) BuildSystemPrompt(ctx context.Context, skills []*Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var prompt string
	prompt += "# Available Skills\n\n"

	for _, skill := range skills {
		prompt += fmt.Sprintf("## /%s", skill.Name)
		if skill.Description != "" {
			prompt += fmt.Sprintf(": %s", skill.Description)
		}
		prompt += "\n"

		if skill.Command != "" {
			prompt += fmt.Sprintf("Command: `%s`\n", skill.Command)
		}

		if len(skill.Variables) > 0 {
			prompt += "Variables:\n"
			for _, v := range skill.Variables {
				required := ""
				if v.Required {
					required = " (required)"
				}
				prompt += fmt.Sprintf("  - %s%s: %s\n", v.Name, required, v.Description)
			}
		}

		prompt += "\n"
	}

	return prompt
}

// RegisterAsMCPTools registers skills as MCP tools
func (s *Service) RegisterAsMCPTools() ([]domain.ToolDefinition, error) {
	skills, err := s.ListSkills(context.Background(), SkillFilter{})
	if err != nil {
		return nil, err
	}

	var tools []domain.ToolDefinition

	for _, skill := range skills {
		if !skill.UserInvocable || !skill.Enabled {
			continue
		}

		// Build parameter schema from skill variables
		properties := make(map[string]interface{})
		required := make([]string, 0)

		for _, v := range skill.Variables {
			prop := map[string]interface{}{
				"type":        getTypeString(v.Type),
				"description": v.Description,
			}
			if v.Default != nil {
				prop["default"] = v.Default
			}
			properties[v.Name] = prop

			if v.Required {
				required = append(required, v.Name)
			}
		}

		tool := domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        fmt.Sprintf("skill_%s", skill.ID),
				Description: skill.Description,
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": properties,
					"required":   required,
				},
			},
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

// GetExecutionHistory retrieves execution history
func (s *Service) GetExecutionHistory(ctx context.Context, skillID string, limit int) ([]*ExecutionResult, error) {
	if s.store != nil {
		return s.store.GetExecutions(ctx, skillID, limit)
	}
	return nil, fmt.Errorf("no store configured")
}

// Close closes the skills service
func (s *Service) Close() error {
	// Close store if it has a Close method
	if s.store != nil {
		if closer, ok := s.store.(interface{ Close() error }); ok {
			return closer.Close()
		}
	}
	return nil
}

// Helper functions

// getTypeString returns the JSON schema type string for a variable type
func getTypeString(typ string) string {
	switch typ {
	case "string", "file":
		return "string"
	case "number", "integer":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "array"
	case "object":
		return "object"
	default:
		return "string"
	}
}
