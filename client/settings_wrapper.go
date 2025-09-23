package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/settings"
)

// SettingsWrapper provides easy access to settings functionality
type SettingsWrapper struct {
	service *settings.Service
}

// NewSettingsWrapper creates a new settings wrapper
func NewSettingsWrapper(service *settings.Service) *SettingsWrapper {
	return &SettingsWrapper{service: service}
}

// Profile operations

// CreateProfile creates a new user profile
func (s *SettingsWrapper) CreateProfile(name, description, systemPrompt string) (*settings.UserProfile, error) {
	req := settings.CreateProfileRequest{
		Name:                name,
		Description:         description,
		DefaultSystemPrompt: systemPrompt,
		Metadata:            make(map[string]string),
	}
	return s.service.CreateProfile(req)
}

// GetActiveProfile returns the currently active profile
func (s *SettingsWrapper) GetActiveProfile() (*settings.UserProfile, error) {
	return s.service.GetActiveProfile()
}

// ListProfiles returns all profiles
func (s *SettingsWrapper) ListProfiles() ([]*settings.UserProfile, error) {
	return s.service.ListProfiles()
}

// GetProfile gets a profile by ID
func (s *SettingsWrapper) GetProfile(profileID string) (*settings.UserProfile, error) {
	return s.service.GetProfile(profileID)
}

// SwitchProfile switches to a different profile by ID
func (s *SettingsWrapper) SwitchProfile(profileID string) error {
	return s.service.SetActiveProfile(profileID)
}

// UpdateProfile updates profile information
func (s *SettingsWrapper) UpdateProfile(profileID, name, description, systemPrompt string) (*settings.UserProfile, error) {
	req := settings.UpdateProfileRequest{}
	if name != "" {
		req.Name = &name
	}
	if description != "" {
		req.Description = &description
	}
	if systemPrompt != "" {
		req.DefaultSystemPrompt = &systemPrompt
	}
	return s.service.UpdateProfile(profileID, req)
}

// DeleteProfile deletes a profile
func (s *SettingsWrapper) DeleteProfile(profileID string) error {
	return s.service.DeleteProfile(profileID)
}

// LLM Settings operations

// SetLLMSettings sets LLM settings for a provider
func (s *SettingsWrapper) SetLLMSettings(providerName, systemPrompt string, temperature *float64, maxTokens *int) (*settings.LLMSettings, error) {
	activeProfile, err := s.service.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	req := settings.CreateLLMSettingsRequest{
		ProfileID:    activeProfile.ID,
		ProviderName: providerName,
		SystemPrompt: systemPrompt,
		Temperature:  temperature,
		MaxTokens:    maxTokens,
		Settings:     make(map[string]interface{}),
	}

	return s.service.CreateOrUpdateLLMSettings(req)
}

// GetLLMSettings gets LLM settings for a provider
func (s *SettingsWrapper) GetLLMSettings(providerName string) (*settings.LLMSettings, error) {
	activeProfile, err := s.service.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	return s.service.GetLLMSettings(activeProfile.ID, providerName)
}

// GetAllLLMSettings gets all LLM settings for the active profile
func (s *SettingsWrapper) GetAllLLMSettings() ([]*settings.LLMSettings, error) {
	activeProfile, err := s.service.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	return s.service.GetAllLLMSettings(activeProfile.ID)
}

// GetSystemPrompt gets the effective system prompt for a provider
func (s *SettingsWrapper) GetSystemPrompt(providerName string) (string, error) {
	return s.service.GetSystemPromptForProvider(providerName)
}

// SetSystemPrompt sets the system prompt for a provider
func (s *SettingsWrapper) SetSystemPrompt(providerName, systemPrompt string) error {
	activeProfile, err := s.service.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("failed to get active profile: %w", err)
	}

	req := settings.CreateLLMSettingsRequest{
		ProfileID:    activeProfile.ID,
		ProviderName: providerName,
		SystemPrompt: systemPrompt,
		Settings:     make(map[string]interface{}),
	}

	_, err = s.service.CreateOrUpdateLLMSettings(req)
	return err
}

// Conversation Context operations

// SaveConversationContext saves conversation context for the active profile
func (s *SettingsWrapper) SaveConversationContext(contextData map[string]interface{}) (*settings.ConversationContext, error) {
	return s.service.SaveConversationContext(contextData)
}

// GetConversationContext gets conversation context for the active profile
func (s *SettingsWrapper) GetConversationContext() (*settings.ConversationContext, error) {
	return s.service.GetConversationContext()
}

// Utility methods

// GetProfileWithSettings gets the active profile with all its settings
func (s *SettingsWrapper) GetProfileWithSettings() (*settings.UserProfile, []*settings.LLMSettings, error) {
	activeProfile, err := s.service.GetActiveProfile()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	llmSettings, err := s.service.GetAllLLMSettings(activeProfile.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get LLM settings: %w", err)
	}

	return activeProfile, llmSettings, nil
}

// InitializeProfileForProviders initializes LLM settings for all configured providers
func (s *SettingsWrapper) InitializeProfileForProviders() error {
	activeProfile, err := s.service.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("failed to get active profile: %w", err)
	}

	return s.service.InitializeProviderSettings(activeProfile.ID)
}

// Enhanced LLM methods with settings integration

// GenerateWithProfile generates text using profile settings
func (c *BaseClient) GenerateWithProfile(ctx context.Context, prompt string, opts *GenerateOptions) (string, error) {
	if c.configWithSettings == nil {
		return "", fmt.Errorf("settings not initialized")
	}

	domainOpts := &domain.GenerationOptions{}
	if opts != nil {
		domainOpts.Temperature = opts.Temperature
		domainOpts.MaxTokens = opts.MaxTokens
	}

	return c.configWithSettings.QuickGenerate(ctx, prompt, domainOpts)
}

// ChatWithProfile performs chat using profile settings
func (c *BaseClient) ChatWithProfile(ctx context.Context, messages []ChatMessage, opts *GenerateOptions) (string, error) {
	if c.configWithSettings == nil {
		return "", fmt.Errorf("settings not initialized")
	}

	// Convert ChatMessage to domain.Message
	domainMessages := make([]domain.Message, len(messages))
	for i, msg := range messages {
		domainMessages[i] = domain.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	domainOpts := &domain.GenerationOptions{}
	if opts != nil {
		domainOpts.Temperature = opts.Temperature
		domainOpts.MaxTokens = opts.MaxTokens
	}

	result, err := c.configWithSettings.QuickGenerateWithTools(ctx, domainMessages, nil, domainOpts)
	if err != nil {
		return "", err
	}

	return result.Content, nil
}

// GetProviderForProfile gets a provider with profile-specific settings
func (c *BaseClient) GetProviderForProfile(providerName string) (domain.LLMProvider, error) {
	if c.configWithSettings == nil {
		return nil, fmt.Errorf("settings not initialized")
	}

	ctx := context.Background()
	return c.configWithSettings.CreateLLMProvider(ctx, providerName)
}

// GetSystemPromptForProvider gets the system prompt for a provider
func (c *BaseClient) GetSystemPromptForProvider(providerName string) (string, error) {
	if c.configWithSettings == nil {
		return "", fmt.Errorf("settings not initialized")
	}

	return c.configWithSettings.GetSystemPromptForProvider(providerName)
}

// SetSystemPromptForProvider sets the system prompt for a provider
func (c *BaseClient) SetSystemPromptForProvider(providerName, systemPrompt string) error {
	if c.configWithSettings == nil {
		return fmt.Errorf("settings not initialized")
	}

	return c.configWithSettings.SetSystemPromptForProvider(providerName, systemPrompt)
}