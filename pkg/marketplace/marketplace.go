package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Marketplace manages agent templates, sharing, and discovery
type Marketplace struct {
	mu         sync.RWMutex
	templates  map[string]*AgentTemplate
	registry   *Registry
	storage    MarketplaceStorage
	validator  TemplateValidator
	
	// Caching
	cache      *TemplateCache
	
	// Configuration
	config     *MarketplaceConfig
}

// MarketplaceConfig holds marketplace configuration
type MarketplaceConfig struct {
	EnableSharing     bool
	EnableVersioning  bool
	MaxTemplateSize   int64
	RequireValidation bool
	CacheTTL         time.Duration
}

// DefaultMarketplaceConfig returns default configuration
func DefaultMarketplaceConfig() *MarketplaceConfig {
	return &MarketplaceConfig{
		EnableSharing:     true,
		EnableVersioning:  true,
		MaxTemplateSize:   10 * 1024 * 1024, // 10MB
		RequireValidation: true,
		CacheTTL:         5 * time.Minute,
	}
}

// AgentTemplate represents a reusable agent configuration
type AgentTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Version     string                 `json:"version"`
	Author      Author                 `json:"author"`
	
	// Template content
	Config      AgentConfig            `json:"config"`
	Workflow    *WorkflowDefinition    `json:"workflow,omitempty"`
	Tools       []string               `json:"tools,omitempty"`
	
	// Metadata
	Tags        []string               `json:"tags"`
	License     string                 `json:"license"`
	Repository  string                 `json:"repository,omitempty"`
	Homepage    string                 `json:"homepage,omitempty"`
	
	// Usage stats
	Downloads   int                    `json:"downloads"`
	Stars       int                    `json:"stars"`
	LastUsed    *time.Time            `json:"last_used,omitempty"`
	
	// Timestamps
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	PublishedAt *time.Time            `json:"published_at,omitempty"`
	
	// Validation
	Validated   bool                  `json:"validated"`
	ValidationResults *ValidationResult `json:"validation_results,omitempty"`
}

// Author represents template author information
type Author struct {
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	URL      string `json:"url,omitempty"`
	Username string `json:"username,omitempty"`
}

// AgentConfig defines agent configuration
type AgentConfig struct {
	Type        string                 `json:"type"`
	Model       string                 `json:"model"`
	Temperature float64                `json:"temperature"`
	MaxTokens   int                    `json:"max_tokens"`
	SystemPrompt string                `json:"system_prompt,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	
	// Capabilities
	Capabilities []string              `json:"capabilities"`
	Limitations  []string              `json:"limitations,omitempty"`
	
	// Resource limits
	MaxExecutionTime time.Duration      `json:"max_execution_time,omitempty"`
	MaxMemory        int64             `json:"max_memory,omitempty"`
}

// WorkflowDefinition defines a workflow template
type WorkflowDefinition struct {
	Steps       []WorkflowStep         `json:"steps"`
	Variables   map[string]interface{} `json:"variables"`
	Inputs      []InputDefinition      `json:"inputs,omitempty"`
	Outputs     []OutputDefinition     `json:"outputs,omitempty"`
}

// WorkflowStep defines a workflow step
type WorkflowStep struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Conditions  []string              `json:"conditions,omitempty"`
}

// InputDefinition defines workflow input
type InputDefinition struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// OutputDefinition defines workflow output
type OutputDefinition struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ValidationResult holds template validation results
type ValidationResult struct {
	Valid       bool      `json:"valid"`
	Errors      []string  `json:"errors,omitempty"`
	Warnings    []string  `json:"warnings,omitempty"`
	Score       int       `json:"score"`
	ValidatedAt time.Time `json:"validated_at"`
}

// Registry manages template registration and discovery
type Registry struct {
	mu         sync.RWMutex
	categories map[string]*Category
	tags       map[string][]*AgentTemplate
	authors    map[string][]*AgentTemplate
}

// Category represents a template category
type Category struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"`
	Count       int    `json:"count"`
}

// MarketplaceStorage interface for persistence
type MarketplaceStorage interface {
	SaveTemplate(template *AgentTemplate) error
	LoadTemplate(id string) (*AgentTemplate, error)
	ListTemplates(filter *TemplateFilter) ([]*AgentTemplate, error)
	DeleteTemplate(id string) error
	
	SaveVersion(templateID string, version *TemplateVersion) error
	ListVersions(templateID string) ([]*TemplateVersion, error)
	
	IncrementDownloads(templateID string) error
	IncrementStars(templateID string) error
}

// TemplateValidator interface for template validation
type TemplateValidator interface {
	Validate(template *AgentTemplate) (*ValidationResult, error)
}

// TemplateFilter for querying templates
type TemplateFilter struct {
	Category    string
	Author      string
	Tags        []string
	SearchTerm  string
	MinStars    int
	SortBy      string // "downloads", "stars", "recent"
	Limit       int
	Offset      int
}

// TemplateVersion represents a template version
type TemplateVersion struct {
	Version     string         `json:"version"`
	TemplateID  string         `json:"template_id"`
	Content     *AgentTemplate `json:"content"`
	Changelog   string         `json:"changelog,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// TemplateCache provides caching for templates
type TemplateCache struct {
	mu       sync.RWMutex
	items    map[string]*cacheItem
	ttl      time.Duration
}

type cacheItem struct {
	template  *AgentTemplate
	expiry    time.Time
}

// NewMarketplace creates a new marketplace instance
func NewMarketplace(config *MarketplaceConfig, storage MarketplaceStorage) *Marketplace {
	if config == nil {
		config = DefaultMarketplaceConfig()
	}

	m := &Marketplace{
		templates: make(map[string]*AgentTemplate),
		registry: &Registry{
			categories: make(map[string]*Category),
			tags:      make(map[string][]*AgentTemplate),
			authors:   make(map[string][]*AgentTemplate),
		},
		storage: storage,
		config:  config,
		cache: &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   config.CacheTTL,
		},
	}

	// Initialize default categories
	m.initializeCategories()

	// Load templates from storage
	if storage != nil {
		if templates, err := storage.ListTemplates(&TemplateFilter{Limit: 1000}); err == nil {
			for _, tmpl := range templates {
				m.templates[tmpl.ID] = tmpl
				m.registry.index(tmpl)
			}
		}
	}

	return m
}

// initializeCategories sets up default categories
func (m *Marketplace) initializeCategories() {
	categories := []Category{
		{ID: "data-analysis", Name: "Data Analysis", Description: "Templates for data processing and analysis"},
		{ID: "automation", Name: "Automation", Description: "Task automation templates"},
		{ID: "research", Name: "Research", Description: "Research and information gathering"},
		{ID: "content", Name: "Content Generation", Description: "Content creation and editing"},
		{ID: "coding", Name: "Coding Assistant", Description: "Programming and development aids"},
		{ID: "testing", Name: "Testing", Description: "Testing and quality assurance"},
		{ID: "monitoring", Name: "Monitoring", Description: "System monitoring and alerting"},
		{ID: "integration", Name: "Integration", Description: "Third-party service integration"},
	}

	for _, cat := range categories {
		m.registry.categories[cat.ID] = &cat
	}
}

// PublishTemplate publishes a new template to the marketplace
func (m *Marketplace) PublishTemplate(ctx context.Context, template *AgentTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate ID if not provided
	if template.ID == "" {
		template.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if template.CreatedAt.IsZero() {
		template.CreatedAt = now
	}
	template.UpdatedAt = now
	template.PublishedAt = &now

	// Validate template if required
	if m.config.RequireValidation && m.validator != nil {
		result, err := m.validator.Validate(template)
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
		
		template.Validated = result.Valid
		template.ValidationResults = result
		
		if !result.Valid {
			return fmt.Errorf("template validation failed: %v", result.Errors)
		}
	}

	// Check template size
	if data, err := json.Marshal(template); err == nil {
		if int64(len(data)) > m.config.MaxTemplateSize {
			return fmt.Errorf("template exceeds maximum size limit")
		}
	}

	// Store template
	if m.storage != nil {
		if err := m.storage.SaveTemplate(template); err != nil {
			return fmt.Errorf("failed to save template: %w", err)
		}
	}

	// Add to registry
	m.templates[template.ID] = template
	m.registry.index(template)

	// Invalidate cache
	m.cache.invalidate(template.ID)

	return nil
}

// GetTemplate retrieves a template by ID
func (m *Marketplace) GetTemplate(ctx context.Context, id string) (*AgentTemplate, error) {
	// Check cache first
	if cached := m.cache.get(id); cached != nil {
		return cached, nil
	}

	m.mu.RLock()
	template, exists := m.templates[id]
	m.mu.RUnlock()

	if !exists {
		// Try loading from storage
		if m.storage != nil {
			var err error
			template, err = m.storage.LoadTemplate(id)
			if err != nil {
				return nil, fmt.Errorf("template not found: %s", id)
			}
			
			// Cache the template
			m.cache.set(id, template)
		} else {
			return nil, fmt.Errorf("template not found: %s", id)
		}
	}

	// Track usage
	now := time.Now()
	template.LastUsed = &now
	
	if m.storage != nil {
		m.storage.IncrementDownloads(id)
	}
	template.Downloads++

	return template, nil
}

// SearchTemplates searches for templates based on criteria
func (m *Marketplace) SearchTemplates(ctx context.Context, filter *TemplateFilter) ([]*AgentTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*AgentTemplate

	// Apply filters
	for _, template := range m.templates {
		if m.matchesFilter(template, filter) {
			results = append(results, template)
		}
	}

	// Sort results - always sort for consistency even if sortBy is empty
	sortBy := ""
	if filter != nil {
		sortBy = filter.SortBy
	}
	m.sortTemplates(results, sortBy)

	// Apply pagination
	if filter == nil {
		return results, nil
	}
	
	start := filter.Offset
	end := filter.Offset + filter.Limit
	
	if start >= len(results) {
		return []*AgentTemplate{}, nil
	}
	
	if end > len(results) || filter.Limit == 0 {
		end = len(results)
	}
	
	if start < 0 {
		start = 0
	}

	return results[start:end], nil
}

// matchesFilter checks if a template matches the filter criteria
func (m *Marketplace) matchesFilter(template *AgentTemplate, filter *TemplateFilter) bool {
	if filter == nil {
		return true
	}

	// Category filter
	if filter.Category != "" && template.Category != filter.Category {
		return false
	}

	// Author filter
	if filter.Author != "" && template.Author.Username != filter.Author {
		return false
	}

	// Tag filter
	if len(filter.Tags) > 0 {
		hasTag := false
		for _, filterTag := range filter.Tags {
			for _, templateTag := range template.Tags {
				if filterTag == templateTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	// Search term filter
	if filter.SearchTerm != "" {
		// Simple text search in name and description
		searchLower := filter.SearchTerm
		if !contains(template.Name, searchLower) && 
		   !contains(template.Description, searchLower) {
			return false
		}
	}

	// Star filter
	if filter.MinStars > 0 && template.Stars < filter.MinStars {
		return false
	}

	return true
}

// contains checks if text contains substring (case-insensitive)
func contains(text, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(text) < len(substr) {
		return false
	}
	
	// Simple case-insensitive contains check
	// In production, consider using strings.Contains with strings.ToLower
	for i := 0; i <= len(text)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			// Simple ASCII case-insensitive comparison
			tc := text[i+j]
			sc := substr[j]
			if tc != sc {
				// Check if they differ only by case (ASCII only)
				if tc >= 'A' && tc <= 'Z' {
					tc = tc + 32 // Convert to lowercase
				}
				if sc >= 'A' && sc <= 'Z' {
					sc = sc + 32 // Convert to lowercase
				}
				if tc != sc {
					match = false
					break
				}
			}
		}
		if match {
			return true
		}
	}
	
	return false
}

// sortTemplates sorts templates based on criteria
func (m *Marketplace) sortTemplates(templates []*AgentTemplate, sortBy string) {
	// Import sort package is needed for this to work
	// For now, we'll use a simple bubble sort for demonstration
	
	n := len(templates)
	if n <= 1 {
		return
	}
	
	// Helper function to compare templates
	less := func(i, j int) bool {
		switch sortBy {
		case "downloads":
			// Sort by downloads descending
			if templates[i].Downloads != templates[j].Downloads {
				return templates[i].Downloads > templates[j].Downloads
			}
		case "stars":
			// Sort by stars descending
			if templates[i].Stars != templates[j].Stars {
				return templates[i].Stars > templates[j].Stars
			}
		case "recent":
			// Sort by creation date descending
			if !templates[i].CreatedAt.Equal(templates[j].CreatedAt) {
				return templates[i].CreatedAt.After(templates[j].CreatedAt)
			}
		}
		// Default: sort by ID for consistency
		return templates[i].ID < templates[j].ID
	}
	
	// Simple bubble sort implementation
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if !less(j, j+1) {
				templates[j], templates[j+1] = templates[j+1], templates[j]
			}
		}
	}
}

// StarTemplate stars/unstars a template
func (m *Marketplace) StarTemplate(ctx context.Context, templateID string, star bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	template, exists := m.templates[templateID]
	if !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	if star {
		template.Stars++
		if m.storage != nil {
			m.storage.IncrementStars(templateID)
		}
	} else {
		if template.Stars > 0 {
			template.Stars--
		}
	}

	return nil
}

// InstallTemplate installs a template for local use
func (m *Marketplace) InstallTemplate(ctx context.Context, templateID string) (*AgentInstance, error) {
	template, err := m.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Create an instance from the template
	instance := &AgentInstance{
		ID:         uuid.New().String(),
		TemplateID: template.ID,
		Name:       template.Name,
		Config:     template.Config,
		Workflow:   template.Workflow,
		Tools:      template.Tools,
		CreatedAt:  time.Now(),
		Status:     "installed",
	}

	return instance, nil
}

// GetCategories returns all available categories
func (m *Marketplace) GetCategories() []*Category {
	m.registry.mu.RLock()
	defer m.registry.mu.RUnlock()

	categories := make([]*Category, 0, len(m.registry.categories))
	for _, cat := range m.registry.categories {
		categories = append(categories, cat)
	}

	return categories
}

// GetPopularTags returns popular tags
func (m *Marketplace) GetPopularTags(limit int) []string {
	m.registry.mu.RLock()
	defer m.registry.mu.RUnlock()

	// Simple implementation - return tags with most templates
	tags := make([]string, 0, limit)
	for tag := range m.registry.tags {
		tags = append(tags, tag)
		if len(tags) >= limit {
			break
		}
	}

	return tags
}

// Registry methods

// index adds a template to the registry indexes
func (r *Registry) index(template *AgentTemplate) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Index by category
	if cat, exists := r.categories[template.Category]; exists {
		cat.Count++
	}

	// Index by tags
	for _, tag := range template.Tags {
		r.tags[tag] = append(r.tags[tag], template)
	}

	// Index by author
	if template.Author.Username != "" {
		r.authors[template.Author.Username] = append(r.authors[template.Author.Username], template)
	}
}

// Cache methods

// get retrieves a template from cache
func (c *TemplateCache) get(id string) *AgentTemplate {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if item, exists := c.items[id]; exists {
		if time.Now().Before(item.expiry) {
			return item.template
		}
		// Expired, remove it
		delete(c.items, id)
	}

	return nil
}

// set adds a template to cache
func (c *TemplateCache) set(id string, template *AgentTemplate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[id] = &cacheItem{
		template: template,
		expiry:   time.Now().Add(c.ttl),
	}
}

// invalidate removes a template from cache
func (c *TemplateCache) invalidate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, id)
}

// AgentInstance represents an installed agent instance
type AgentInstance struct {
	ID         string               `json:"id"`
	TemplateID string               `json:"template_id"`
	Name       string               `json:"name"`
	Config     AgentConfig          `json:"config"`
	Workflow   *WorkflowDefinition  `json:"workflow,omitempty"`
	Tools      []string             `json:"tools,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
	Status     string               `json:"status"`
}