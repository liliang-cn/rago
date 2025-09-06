package marketplace

import (
	"fmt"
	"sync"
	"time"
)

// MockMarketplaceStorage implements MarketplaceStorage for testing
type MockMarketplaceStorage struct {
	mu               sync.RWMutex
	templates        map[string]*AgentTemplate
	versions         map[string][]*TemplateVersion
	saveError        error
	loadError        error
	listError        error
	deleteError      error
	versionSaveError error
	versionListError error
	downloadCounts   map[string]int
	starCounts       map[string]int
}

// NewMockMarketplaceStorage creates a new mock storage
func NewMockMarketplaceStorage() *MockMarketplaceStorage {
	return &MockMarketplaceStorage{
		templates:      make(map[string]*AgentTemplate),
		versions:       make(map[string][]*TemplateVersion),
		downloadCounts: make(map[string]int),
		starCounts:     make(map[string]int),
	}
}

// SaveTemplate saves a template to mock storage
func (m *MockMarketplaceStorage) SaveTemplate(template *AgentTemplate) error {
	if m.saveError != nil {
		return m.saveError
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.templates[template.ID] = template
	return nil
}

// LoadTemplate loads a template from mock storage
func (m *MockMarketplaceStorage) LoadTemplate(id string) (*AgentTemplate, error) {
	if m.loadError != nil {
		return nil, m.loadError
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	template, exists := m.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	
	return template, nil
}

// ListTemplates lists templates from mock storage
func (m *MockMarketplaceStorage) ListTemplates(filter *TemplateFilter) ([]*AgentTemplate, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var results []*AgentTemplate
	for _, template := range m.templates {
		results = append(results, template)
	}
	
	// Apply simple pagination if filter provided
	if filter != nil {
		start := filter.Offset
		end := filter.Offset + filter.Limit
		
		if start > len(results) {
			return []*AgentTemplate{}, nil
		}
		
		if end > len(results) || filter.Limit == 0 {
			end = len(results)
		}
		
		if start < len(results) {
			results = results[start:end]
		}
	}
	
	return results, nil
}

// DeleteTemplate deletes a template from mock storage
func (m *MockMarketplaceStorage) DeleteTemplate(id string) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.templates, id)
	return nil
}

// SaveVersion saves a template version
func (m *MockMarketplaceStorage) SaveVersion(templateID string, version *TemplateVersion) error {
	if m.versionSaveError != nil {
		return m.versionSaveError
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.versions[templateID] = append(m.versions[templateID], version)
	return nil
}

// ListVersions lists template versions
func (m *MockMarketplaceStorage) ListVersions(templateID string) ([]*TemplateVersion, error) {
	if m.versionListError != nil {
		return nil, m.versionListError
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.versions[templateID], nil
}

// IncrementDownloads increments download count
func (m *MockMarketplaceStorage) IncrementDownloads(templateID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.downloadCounts[templateID]++
	return nil
}

// IncrementStars increments star count
func (m *MockMarketplaceStorage) IncrementStars(templateID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.starCounts[templateID]++
	return nil
}

// Helper methods for testing

// SetSaveError sets the error to return on save
func (m *MockMarketplaceStorage) SetSaveError(err error) {
	m.saveError = err
}

// SetLoadError sets the error to return on load
func (m *MockMarketplaceStorage) SetLoadError(err error) {
	m.loadError = err
}

// GetDownloadCount returns the download count for a template
func (m *MockMarketplaceStorage) GetDownloadCount(templateID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.downloadCounts[templateID]
}

// GetStarCount returns the star count for a template
func (m *MockMarketplaceStorage) GetStarCount(templateID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.starCounts[templateID]
}

// MockTemplateValidator implements TemplateValidator for testing
type MockTemplateValidator struct {
	mu            sync.RWMutex
	validationMap map[string]*ValidationResult
	defaultResult *ValidationResult
	errorToReturn error
}

// NewMockTemplateValidator creates a new mock validator
func NewMockTemplateValidator() *MockTemplateValidator {
	return &MockTemplateValidator{
		validationMap: make(map[string]*ValidationResult),
		defaultResult: &ValidationResult{
			Valid:       true,
			Score:       100,
			ValidatedAt: time.Now(),
		},
	}
}

// Validate validates a template
func (v *MockTemplateValidator) Validate(template *AgentTemplate) (*ValidationResult, error) {
	if v.errorToReturn != nil {
		return nil, v.errorToReturn
	}
	
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	// Check if we have a specific result for this template
	if result, exists := v.validationMap[template.ID]; exists {
		return result, nil
	}
	
	// Return default result
	return v.defaultResult, nil
}

// SetValidationResult sets the validation result for a specific template
func (v *MockTemplateValidator) SetValidationResult(templateID string, result *ValidationResult) {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	v.validationMap[templateID] = result
}

// SetDefaultResult sets the default validation result
func (v *MockTemplateValidator) SetDefaultResult(result *ValidationResult) {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	v.defaultResult = result
}

// SetError sets the error to return
func (v *MockTemplateValidator) SetError(err error) {
	v.errorToReturn = err
}

// Helper function to create a sample template for testing
func createSampleTemplate(id, name, category string) *AgentTemplate {
	return &AgentTemplate{
		ID:          id,
		Name:        name,
		Description: "Test template description",
		Category:    category,
		Version:     "1.0.0",
		Author: Author{
			Name:     "Test Author",
			Email:    "test@example.com",
			Username: "testuser",
		},
		Config: AgentConfig{
			Type:         "test",
			Model:        "test-model",
			Temperature:  0.7,
			MaxTokens:    1000,
			Capabilities: []string{"test", "sample"},
		},
		Tags:      []string{"test", "sample"},
		License:   "MIT",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}