package settings

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
)

// Service provides settings management functionality
type Service struct {
	storage Storage
	config  *config.Config
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
	if s.config.Providers.ProviderConfigs.Ollama != nil {
		providers = append(providers, "ollama")
	}
	if s.config.Providers.ProviderConfigs.OpenAI != nil {
		providers = append(providers, "openai")
	}
	if s.config.Providers.ProviderConfigs.LMStudio != nil {
		providers = append(providers, "lmstudio")
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