package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// Service provides settings management functionality
type Service struct {
	storage       Storage
	config        *config.Config
	convStore     *store.ConversationStore
	llmService    domain.LLMProvider
}

// NewService creates a new settings service
func NewService(cfg *config.Config) (*Service, error) {
	// Create settings database path in the same directory as main database
	dbDir := filepath.Dir(cfg.Sqvect.DBPath)
	settingsDBPath := filepath.Join(dbDir, "settings.db")

	storage, err := NewSQLiteStorage(settingsDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create settings storage: %w", err)
	}

	service := &Service{
		storage: storage,
		config:  cfg,
	}

	// Initialize default profile if none exists
	if err := service.initializeDefaultProfile(); err != nil {
		return nil, fmt.Errorf("failed to initialize default profile: %w", err)
	}

	return service, nil
}

// NewServiceWithComponents creates a new settings service with additional components
func NewServiceWithComponents(cfg *config.Config, db *sql.DB, llmService domain.LLMProvider) (*Service, error) {
	// Create settings database path in the same directory as main database
	dbDir := filepath.Dir(cfg.Sqvect.DBPath)
	settingsDBPath := filepath.Join(dbDir, "settings.db")

	storage, err := NewSQLiteStorage(settingsDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create settings storage: %w", err)
	}

	// Create conversation store
	convStore, err := store.NewConversationStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation store: %w", err)
	}

	service := &Service{
		storage:    storage,
		config:     cfg,
		convStore:  convStore,
		llmService: llmService,
	}

	// Initialize default profile if none exists
	if err := service.initializeDefaultProfile(); err != nil {
		return nil, fmt.Errorf("failed to initialize default profile: %w", err)
	}

	return service, nil
}

// SetComponents sets additional components for auto-profile generation
func (s *Service) SetComponents(db *sql.DB, llmService domain.LLMProvider) error {
	convStore, err := store.NewConversationStore(db)
	if err != nil {
		return fmt.Errorf("failed to create conversation store: %w", err)
	}

	s.convStore = convStore
	s.llmService = llmService
	return nil
}

// Close closes the service and its resources
func (s *Service) Close() error {
	return s.storage.Close()
}

// initializeDefaultProfile creates a default profile if none exists
func (s *Service) initializeDefaultProfile() error {
	profiles, err := s.storage.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	// If no profiles exist, create a default one
	if len(profiles) == 0 {
		req := CreateProfileRequest{
			Name:        "default",
			Description: "Default user profile",
			Metadata:    map[string]string{"created_by": "system"},
		}

		profile, err := s.storage.CreateProfile(req)
		if err != nil {
			return fmt.Errorf("failed to create default profile: %w", err)
		}

		// Set it as active
		if err := s.storage.SetActiveProfile(profile.ID); err != nil {
			return fmt.Errorf("failed to set default profile as active: %w", err)
		}
	}

	return nil
}

// Profile Management

// CreateProfile creates a new user profile
func (s *Service) CreateProfile(req CreateProfileRequest) (*UserProfile, error) {
	return s.storage.CreateProfile(req)
}

// GetProfile retrieves a profile by ID
func (s *Service) GetProfile(id string) (*UserProfile, error) {
	return s.storage.GetProfile(id)
}

// GetActiveProfile retrieves the currently active profile
func (s *Service) GetActiveProfile() (*UserProfile, error) {
	return s.storage.GetActiveProfile()
}

// ListProfiles retrieves all profiles
func (s *Service) ListProfiles() ([]*UserProfile, error) {
	return s.storage.ListProfiles()
}

// UpdateProfile updates an existing profile
func (s *Service) UpdateProfile(id string, req UpdateProfileRequest) (*UserProfile, error) {
	return s.storage.UpdateProfile(id, req)
}

// DeleteProfile deletes a profile
func (s *Service) DeleteProfile(id string) error {
	// Check if this is the active profile
	activeProfile, err := s.storage.GetActiveProfile()
	if err == nil && activeProfile.ID == id {
		// Find another profile to make active
		profiles, err := s.storage.ListProfiles()
		if err != nil {
			return fmt.Errorf("failed to list profiles: %w", err)
		}

		var newActiveID string
		for _, p := range profiles {
			if p.ID != id {
				newActiveID = p.ID
				break
			}
		}

		if newActiveID != "" {
			if err := s.storage.SetActiveProfile(newActiveID); err != nil {
				return fmt.Errorf("failed to set new active profile: %w", err)
			}
		}
	}

	return s.storage.DeleteProfile(id)
}

// SetActiveProfile sets a profile as the active one
func (s *Service) SetActiveProfile(id string) error {
	return s.storage.SetActiveProfile(id)
}

// LLM Settings Management

// CreateOrUpdateLLMSettings creates or updates LLM settings for a profile and provider
func (s *Service) CreateOrUpdateLLMSettings(req CreateLLMSettingsRequest) (*LLMSettings, error) {
	// Check if settings already exist for this profile/provider combination
	existing, err := s.storage.GetLLMSettings(req.ProfileID, req.ProviderName)
	if err != nil {
		// Settings don't exist, create new ones
		return s.storage.CreateLLMSettings(req)
	}

	// Settings exist, update them
	updateReq := UpdateLLMSettingsRequest{
		SystemPrompt: &req.SystemPrompt,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		Settings:     &req.Settings,
	}

	return s.storage.UpdateLLMSettings(existing.ID, updateReq)
}

// GetLLMSettings retrieves LLM settings for a specific profile and provider
func (s *Service) GetLLMSettings(profileID, providerName string) (*LLMSettings, error) {
	return s.storage.GetLLMSettings(profileID, providerName)
}

// GetLLMSettingsForActiveProfile retrieves LLM settings for the active profile and provider
func (s *Service) GetLLMSettingsForActiveProfile(providerName string) (*LLMSettings, error) {
	activeProfile, err := s.storage.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	return s.storage.GetLLMSettings(activeProfile.ID, providerName)
}

// GetAllLLMSettings retrieves all LLM settings for a profile
func (s *Service) GetAllLLMSettings(profileID string) ([]*LLMSettings, error) {
	return s.storage.GetAllLLMSettings(profileID)
}

// UpdateLLMSettings updates existing LLM settings
func (s *Service) UpdateLLMSettings(id string, req UpdateLLMSettingsRequest) (*LLMSettings, error) {
	return s.storage.UpdateLLMSettings(id, req)
}

// DeleteLLMSettings deletes LLM settings
func (s *Service) DeleteLLMSettings(id string) error {
	return s.storage.DeleteLLMSettings(id)
}

// GetSystemPromptForProvider returns the system prompt for a provider, using profile settings if available
func (s *Service) GetSystemPromptForProvider(providerName string) (string, error) {
	activeProfile, err := s.storage.GetActiveProfile()
	if err != nil {
		return "", fmt.Errorf("failed to get active profile: %w", err)
	}

	// Try to get LLM-specific settings first
	llmSettings, err := s.storage.GetLLMSettings(activeProfile.ID, providerName)
	if err == nil && llmSettings.SystemPrompt != "" {
		return llmSettings.SystemPrompt, nil
	}

	// Fall back to profile default system prompt
	if activeProfile.DefaultSystemPrompt != "" {
		return activeProfile.DefaultSystemPrompt, nil
	}

	// No custom system prompt found
	return "", nil
}

// GetLLMParametersForProvider returns LLM parameters for a provider from profile settings
func (s *Service) GetLLMParametersForProvider(providerName string) (temperature *float64, maxTokens *int, settings map[string]interface{}, err error) {
	activeProfile, err := s.storage.GetActiveProfile()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	// Try to get LLM-specific settings
	llmSettings, err := s.storage.GetLLMSettings(activeProfile.ID, providerName)
	if err != nil {
		// No specific settings found, return defaults
		return nil, nil, nil, nil
	}

	return llmSettings.Temperature, llmSettings.MaxTokens, llmSettings.Settings, nil
}

// Conversation Context Management

// SaveConversationContext saves conversation context with default TTL
func (s *Service) SaveConversationContext(contextData map[string]interface{}) (*ConversationContext, error) {
	activeProfile, err := s.storage.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	// Default TTL of 24 hours
	ttl := 24 * time.Hour
	return s.storage.SaveConversationContext(activeProfile.ID, contextData, ttl)
}

// SaveConversationContextWithTTL saves conversation context with custom TTL
func (s *Service) SaveConversationContextWithTTL(contextData map[string]interface{}, ttl time.Duration) (*ConversationContext, error) {
	activeProfile, err := s.storage.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	return s.storage.SaveConversationContext(activeProfile.ID, contextData, ttl)
}

// GetConversationContext retrieves conversation context for the active profile
func (s *Service) GetConversationContext() (*ConversationContext, error) {
	activeProfile, err := s.storage.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	return s.storage.GetConversationContext(activeProfile.ID)
}

// CleanupExpiredContexts removes expired conversation contexts
func (s *Service) CleanupExpiredContexts() error {
	return s.storage.CleanupExpiredContexts()
}

// Utility Methods

// GetProfileWithSettings retrieves a profile with all its LLM settings
func (s *Service) GetProfileWithSettings(profileID string) (*ProfileWithSettings, error) {
	profile, err := s.storage.GetProfile(profileID)
	if err != nil {
		return nil, err
	}

	settings, err := s.storage.GetAllLLMSettings(profileID)
	if err != nil {
		return nil, err
	}

	return &ProfileWithSettings{
		Profile:  *profile,
		Settings: make([]LLMSettings, len(settings)),
	}, nil
}

// InitializeProviderSettings creates default LLM settings for all configured providers
func (s *Service) InitializeProviderSettings(profileID string) error {
	// Get available providers from config
	providers := []string{}
	
	// Add providers based on config
	if s.config.Providers.ProviderConfigs.OpenAI != nil {
		providers = append(providers, "openai")
	}

	// Create default settings for each provider if they don't exist
	for _, providerName := range providers {
		_, err := s.storage.GetLLMSettings(profileID, providerName)
		if err != nil {
			// Settings don't exist, create defaults
			req := CreateLLMSettingsRequest{
				ProfileID:    profileID,
				ProviderName: providerName,
				SystemPrompt: "", // Empty by default
				Settings:     make(map[string]interface{}),
			}

			if _, err := s.storage.CreateLLMSettings(req); err != nil {
				return fmt.Errorf("failed to create default settings for %s: %w", providerName, err)
			}
		}
	}

	return nil
}

// AutoGenerateProfileFromHistory generates a profile based on conversation history
func (s *Service) AutoGenerateProfileFromHistory(conversationIDs []string, profileName string) (*UserProfile, error) {
	if s.convStore == nil || s.llmService == nil {
		return nil, fmt.Errorf("conversation store or LLM service not initialized")
	}

	// Collect conversation data
	var conversationData []ConversationAnalysisData
	for _, convID := range conversationIDs {
		conv, err := s.convStore.GetConversation(convID)
		if err != nil {
			continue // Skip conversations that can't be loaded
		}

		data := s.analyzeConversation(conv)
		conversationData = append(conversationData, data)
	}

	if len(conversationData) == 0 {
		return nil, fmt.Errorf("no valid conversations found for analysis")
	}

	// Generate profile using LLM
	profile, err := s.generateProfileFromAnalysis(conversationData, profileName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate profile: %w", err)
	}

	// Create the profile
	req := CreateProfileRequest{
		Name:                profile.Name,
		Description:         profile.Description,
		DefaultSystemPrompt: profile.SystemPrompt,
		Metadata:            profile.Metadata,
	}

	createdProfile, err := s.storage.CreateProfile(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Create LLM settings for the profile if provided
	if profile.LLMSettings != nil {
		for provider, settings := range profile.LLMSettings {
			llmReq := CreateLLMSettingsRequest{
				ProfileID:    createdProfile.ID,
				ProviderName: provider,
				SystemPrompt: settings.SystemPrompt,
				Temperature:  settings.Temperature,
				MaxTokens:    settings.MaxTokens,
				Settings:     settings.AdditionalSettings,
			}

			if _, err := s.storage.CreateLLMSettings(llmReq); err != nil {
				// Log error but continue with profile creation
				continue
			}
		}
	}

	return createdProfile, nil
}

// ConversationAnalysisData represents analyzed conversation data
type ConversationAnalysisData struct {
	ID           string
	Title        string
	MessageCount int
	Topics       []string
	Style        string
	Tone         string
	Domain       string
	LLMProviders []string
	ToolsUsed    []string
}

// GeneratedProfile represents a profile generated from conversation analysis
type GeneratedProfile struct {
	Name                string
	Description         string
	SystemPrompt        string
	Metadata            map[string]string
	LLMSettings         map[string]GeneratedLLMSettings
}

// GeneratedLLMSettings represents LLM settings for a generated profile
type GeneratedLLMSettings struct {
	SystemPrompt       string                 `json:"system_prompt"`
	Temperature        *float64               `json:"temperature,omitempty"`
	MaxTokens          *int                   `json:"max_tokens,omitempty"`
	AdditionalSettings map[string]interface{} `json:"additional_settings,omitempty"`
}

// analyzeConversation analyzes a conversation to extract key information
func (s *Service) analyzeConversation(conv *store.Conversation) ConversationAnalysisData {
	data := ConversationAnalysisData{
		ID:           conv.ID,
		Title:        conv.Title,
		MessageCount: len(conv.Messages),
		Topics:       []string{},
		LLMProviders: []string{},
		ToolsUsed:    []string{},
	}

	// Analyze messages to extract patterns
	var userMessages, assistantMessages []string
	for _, msg := range conv.Messages {
		switch msg.Role {
		case "user":
			userMessages = append(userMessages, msg.Content)
		case "assistant":
			assistantMessages = append(assistantMessages, msg.Content)

			// Extract tools used from sources
			for _, source := range msg.Sources {
				if strings.Contains(source.Source, "MCP Tool:") {
					toolName := strings.TrimPrefix(source.Source, "MCP Tool: ")
					data.ToolsUsed = append(data.ToolsUsed, toolName)
				}
			}
		}
	}

	// Extract domain/topic keywords
	domainKeywords := map[string][]string{
		"programming":    {"code", "function", "programming", "debug", "algorithm", "api", "database"},
		"writing":        {"write", "article", "essay", "content", "blog", "story"},
		"analysis":       {"analyze", "research", "data", "report", "statistics"},
		"business":       {"business", "strategy", "market", "revenue", "customer"},
		"creative":       {"creative", "design", "art", "imagine", "innovate"},
		"technical":      {"technical", "system", "architecture", "infrastructure", "deployment"},
	}

	// Simple topic detection
	for domain, keywords := range domainKeywords {
		for _, keyword := range keywords {
			if strings.Contains(conv.Title+" "+strings.Join(userMessages, " "), keyword) {
				if !contains(data.Topics, domain) {
					data.Topics = append(data.Topics, domain)
				}
			}
		}
	}

	// Determine style and tone
	if len(assistantMessages) > 0 {
		allAssistantText := strings.Join(assistantMessages, " ")

		// Style detection
		if strings.Contains(allAssistantText, "```") || strings.Contains(allAssistantText, "code") {
			data.Style = "technical"
		} else if strings.Contains(allAssistantText, "step-by-step") || strings.Contains(allAssistantText, "follow") {
			data.Style = "instructional"
		} else if strings.Contains(allAssistantText, "imagine") || strings.Contains(allAssistantText, "creative") {
			data.Style = "creative"
		} else {
			data.Style = "conversational"
		}

		// Tone detection
		if strings.Contains(allAssistantText, "formal") || strings.Contains(allAssistantText, "professional") {
			data.Tone = "formal"
		} else if strings.Contains(allAssistantText, "friendly") || strings.Contains(allAssistantText, "casual") {
			data.Tone = "casual"
		} else {
			data.Tone = "neutral"
		}
	}

	// Set primary domain
	if len(data.Topics) > 0 {
		data.Domain = data.Topics[0]
	}

	return data
}

// generateProfileFromAnalysis uses LLM to generate a profile from analyzed data
func (s *Service) generateProfileFromAnalysis(data []ConversationAnalysisData, profileName string) (*GeneratedProfile, error) {
	// Get global LLM service
	llmService, err := services.GetGlobalLLM()
	if err != nil {
		return nil, fmt.Errorf("failed to get global LLM service: %w", err)
	}

	// Prepare analysis summary
	analysisText := s.prepareAnalysisSummary(data, profileName)

	// System prompt for profile generation
	systemPrompt := `You are an expert at analyzing conversation patterns and creating user profiles.
Based on the provided conversation analysis, generate a profile that captures the user's communication style, preferences, and typical use cases.
Respond with a JSON object containing:
- name: profile name (use provided name or suggest a better one)
- description: brief description of the profile's purpose
- system_prompt: appropriate system prompt for this user
- metadata: key characteristics as key-value pairs
- llm_settings: provider-specific settings (if applicable)

Focus on creating a profile that would work well for similar future conversations.`

	// Generate profile using LLM
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	genOpts := &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   1000,
	}

	response, err := llmService.Generate(ctx, systemPrompt+"\n\n"+analysisText, genOpts)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse JSON response
	var profile GeneratedProfile
	if err := json.Unmarshal([]byte(s.extractJSON(response)), &profile); err != nil {
		// Fallback: create a basic profile
		profile = s.createFallbackProfile(data, profileName)
	}

	return &profile, nil
}

// prepareAnalysisSummary creates a summary of conversation analysis for LLM
func (s *Service) prepareAnalysisSummary(data []ConversationAnalysisData, profileName string) string {
	var summary strings.Builder

	summary.WriteString("CONVERSATION ANALYSIS FOR PROFILE GENERATION\n")
	summary.WriteString("=============================================\n\n")

	summary.WriteString(fmt.Sprintf("Desired Profile Name: %s\n\n", profileName))

	summary.WriteString(fmt.Sprintf("Total Conversations Analyzed: %d\n\n", len(data)))

	// Aggregate data
	totalMessages := 0
	domainCounts := make(map[string]int)
	styleCounts := make(map[string]int)
	toneCounts := make(map[string]int)
	allTools := make(map[string]bool)

	for _, d := range data {
		totalMessages += d.MessageCount
		domainCounts[d.Domain]++
		styleCounts[d.Style]++
		toneCounts[d.Tone]++

		for _, tool := range d.ToolsUsed {
			allTools[tool] = true
		}
	}

	summary.WriteString("CONVERSATION PATTERNS:\n")
	summary.WriteString(fmt.Sprintf("- Total Messages: %d\n", totalMessages))
	summary.WriteString(fmt.Sprintf("- Average Messages per Conversation: %.1f\n", float64(totalMessages)/float64(len(data))))

	summary.WriteString("\nDOMAIN EXPERTISE:\n")
	for domain, count := range domainCounts {
		summary.WriteString(fmt.Sprintf("- %s: %d conversations (%.1f%%)\n",
			domain, count, float64(count*100)/float64(len(data))))
	}

	summary.WriteString("\nCOMMUNICATION STYLE:\n")
	for style, count := range styleCounts {
		summary.WriteString(fmt.Sprintf("- %s: %d conversations (%.1f%%)\n",
			style, count, float64(count*100)/float64(len(data))))
	}

	summary.WriteString("\nTONE PREFERENCES:\n")
	for tone, count := range toneCounts {
		summary.WriteString(fmt.Sprintf("- %s: %d conversations (%.1f%%)\n",
			tone, count, float64(count*100)/float64(len(data))))
	}

	if len(allTools) > 0 {
		summary.WriteString("\nTOOLS USED:\n")
		for tool := range allTools {
			summary.WriteString(fmt.Sprintf("- %s\n", tool))
		}
	}

	summary.WriteString("\nINDIVIDUAL CONVERSATION DETAILS:\n")
	for i, d := range data {
		summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, d.Title))
		summary.WriteString(fmt.Sprintf("   Messages: %d, Domain: %s, Style: %s, Tone: %s\n",
			d.MessageCount, d.Domain, d.Style, d.Tone))
		if len(d.Topics) > 0 {
			summary.WriteString(fmt.Sprintf("   Topics: %s\n", strings.Join(d.Topics, ", ")))
		}
		summary.WriteString("\n")
	}

	return summary.String()
}

// extractJSON extracts JSON from LLM response
func (s *Service) extractJSON(response string) string {
	// Look for JSON object in the response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start >= 0 && end > start {
		return response[start : end+1]
	}

	return "{}"
}

// createFallbackProfile creates a basic profile when LLM generation fails
func (s *Service) createFallbackProfile(data []ConversationAnalysisData, profileName string) GeneratedProfile {
	// Determine primary characteristics
	domainCounts := make(map[string]int)
	styleCounts := make(map[string]int)

	for _, d := range data {
		domainCounts[d.Domain]++
		styleCounts[d.Style]++
	}

	// Find most common domain and style
	primaryDomain := "general"
	maxCount := 0
	for domain, count := range domainCounts {
		if count > maxCount {
			maxCount = count
			primaryDomain = domain
		}
	}

	primaryStyle := "conversational"
	maxCount = 0
	for style, count := range styleCounts {
		if count > maxCount {
			maxCount = count
			primaryStyle = style
		}
	}

	profile := GeneratedProfile{
		Name:        profileName,
		Description: fmt.Sprintf("Auto-generated profile based on %d conversations", len(data)),
		Metadata: map[string]string{
			"generated_from": "conversation_history",
			"conversation_count": fmt.Sprintf("%d", len(data)),
			"primary_domain": primaryDomain,
			"communication_style": primaryStyle,
			"auto_generated": "true",
		},
	}

	// Set system prompt based on characteristics
	switch primaryDomain {
	case "programming":
		profile.SystemPrompt = "You are helpful with programming and technical questions. Provide clear, accurate code examples and explanations."
	case "writing":
		profile.SystemPrompt = "You are a writing assistant. Help with content creation, editing, and improving written materials."
	case "analysis":
		profile.SystemPrompt = "You are an analytical assistant. Help with data analysis, research, and generating insights."
	case "business":
		profile.SystemPrompt = "You are a business advisor. Provide strategic insights and business-related guidance."
	case "creative":
		profile.SystemPrompt = "You are a creative partner. Help with brainstorming, creative writing, and innovative ideas."
	default:
		profile.SystemPrompt = "You are a helpful assistant. Adapt your responses to be clear, accurate, and tailored to the user's needs."
	}

	return profile
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}