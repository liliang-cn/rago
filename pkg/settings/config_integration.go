package settings

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

// ConfigWithSettings provides configuration integration with settings service
type ConfigWithSettings struct {
	*config.Config
	SettingsService *Service
}

// NewConfigWithSettings creates a new config with settings integration
func NewConfigWithSettings(cfg *config.Config) (*ConfigWithSettings, error) {
	settingsService, err := NewService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize settings service: %w", err)
	}

	return &ConfigWithSettings{
		Config:          cfg,
		SettingsService: settingsService,
	}, nil
}

// Close closes the settings service
func (cws *ConfigWithSettings) Close() error {
	if cws.SettingsService != nil {
		return cws.SettingsService.Close()
	}
	return nil
}

// CreateProviderFactory creates a provider factory with settings integration
func (cws *ConfigWithSettings) CreateProviderFactory() domain.ProviderFactory {
	baseFactory := providers.NewFactory()
	return NewProviderFactoryWrapper(baseFactory, cws.SettingsService)
}

// CreateLLMProvider creates an LLM provider with settings integration
func (cws *ConfigWithSettings) CreateLLMProvider(ctx context.Context, providerName string) (domain.LLMProvider, error) {
	factory := cws.CreateProviderFactory()

	// Get provider config based on the provider name
	var config interface{}
	var err error

	switch providerName {
	case "ollama":
		config, err = providers.GetLLMProviderConfig(&cws.Providers.ProviderConfigs, providerName)
	case "openai":
		config, err = providers.GetLLMProviderConfig(&cws.Providers.ProviderConfigs, providerName)
	case "lmstudio":
		config, err = providers.GetLLMProviderConfig(&cws.Providers.ProviderConfigs, providerName)
	default:
		// Try dynamic providers
		config, err = providers.GetProviderConfigByName(cws.Providers.Providers, providerName)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get provider config for %s: %w", providerName, err)
	}

	return factory.CreateLLMProvider(ctx, config)
}

// CreateEmbedderProvider creates an embedder provider (no settings integration needed)
func (cws *ConfigWithSettings) CreateEmbedderProvider(ctx context.Context, providerName string) (domain.EmbedderProvider, error) {
	factory := providers.NewFactory() // Use base factory for embedders

	// Get provider config based on the provider name
	var config interface{}
	var err error

	switch providerName {
	case "ollama":
		config, err = providers.GetEmbedderProviderConfig(&cws.Providers.ProviderConfigs, providerName)
	case "openai":
		config, err = providers.GetEmbedderProviderConfig(&cws.Providers.ProviderConfigs, providerName)
	case "lmstudio":
		config, err = providers.GetEmbedderProviderConfig(&cws.Providers.ProviderConfigs, providerName)
	default:
		// Try dynamic providers
		config, err = providers.GetProviderConfigByName(cws.Providers.Providers, providerName)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get provider config for %s: %w", providerName, err)
	}

	return factory.CreateEmbedderProvider(ctx, config)
}

// CreateDefaultLLMProvider creates the default LLM provider with settings
func (cws *ConfigWithSettings) CreateDefaultLLMProvider(ctx context.Context) (domain.LLMProvider, error) {
	return cws.CreateLLMProvider(ctx, cws.Providers.DefaultLLM)
}

// CreateDefaultEmbedderProvider creates the default embedder provider
func (cws *ConfigWithSettings) CreateDefaultEmbedderProvider(ctx context.Context) (domain.EmbedderProvider, error) {
	embedderProvider := cws.Providers.DefaultEmbedder
	if embedderProvider == "" {
		embedderProvider = cws.Providers.DefaultLLM
	}
	return cws.CreateEmbedderProvider(ctx, embedderProvider)
}

// CreateProviderClient creates a provider client with all configured providers
func (cws *ConfigWithSettings) CreateProviderClient(ctx context.Context) (*ProviderClient, error) {
	client := NewProviderClient(cws.SettingsService)

	// Register all configured providers
	providerNames := []string{}

	// Add static providers
	if cws.Providers.ProviderConfigs.OpenAI != nil {
		providerNames = append(providerNames, "openai")
	}

	// Add dynamic providers
	for name := range cws.Providers.Providers {
		providerNames = append(providerNames, name)
	}

	factory := providers.NewFactory() // Use base factory to get unwrapped providers

	for _, name := range providerNames {
		var config interface{}
		var err error

		switch name {
		case "ollama":
			config, err = providers.GetLLMProviderConfig(&cws.Providers.ProviderConfigs, name)
		case "openai":
			config, err = providers.GetLLMProviderConfig(&cws.Providers.ProviderConfigs, name)
		case "lmstudio":
			config, err = providers.GetLLMProviderConfig(&cws.Providers.ProviderConfigs, name)
		default:
			config, err = providers.GetProviderConfigByName(cws.Providers.Providers, name)
		}

		if err != nil {
			// Log warning but continue with other providers
			fmt.Printf("Warning: failed to get config for provider %s: %v\n", name, err)
			continue
		}

		provider, err := factory.CreateLLMProvider(ctx, config)
		if err != nil {
			// Log warning but continue with other providers
			fmt.Printf("Warning: failed to create provider %s: %v\n", name, err)
			continue
		}

		client.RegisterProvider(name, provider)
	}

	return client, nil
}

// InitializeProfileForProviders creates LLM settings for all configured providers in the active profile
func (cws *ConfigWithSettings) InitializeProfileForProviders() error {
	activeProfile, err := cws.SettingsService.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("failed to get active profile: %w", err)
	}

	return cws.SettingsService.InitializeProviderSettings(activeProfile.ID)
}

// QuickGenerateWithTools performs tool-based generation with the default provider using active profile settings
func (cws *ConfigWithSettings) QuickGenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	provider, err := cws.CreateDefaultLLMProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create default LLM provider: %w", err)
	}

	return provider.GenerateWithTools(ctx, messages, tools, opts)
}

// QuickGenerate performs generation with the default provider using active profile settings
func (cws *ConfigWithSettings) QuickGenerate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	provider, err := cws.CreateDefaultLLMProvider(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create default LLM provider: %w", err)
	}

	return provider.Generate(ctx, prompt, opts)
}

// GetActiveProfileSettings returns the current active profile and its settings
func (cws *ConfigWithSettings) GetActiveProfileSettings() (*UserProfile, []*LLMSettings, error) {
	profile, err := cws.SettingsService.GetActiveProfile()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	settings, err := cws.SettingsService.GetAllLLMSettings(profile.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get LLM settings: %w", err)
	}

	return profile, settings, nil
}

// SetSystemPromptForProvider sets the system prompt for a specific provider in the active profile
func (cws *ConfigWithSettings) SetSystemPromptForProvider(providerName, systemPrompt string) error {
	activeProfile, err := cws.SettingsService.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("failed to get active profile: %w", err)
	}

	req := CreateLLMSettingsRequest{
		ProfileID:    activeProfile.ID,
		ProviderName: providerName,
		SystemPrompt: systemPrompt,
		Settings:     make(map[string]interface{}),
	}

	_, err = cws.SettingsService.CreateOrUpdateLLMSettings(req)
	if err != nil {
		return fmt.Errorf("failed to set system prompt: %w", err)
	}

	return nil
}

// GetSystemPromptForProvider gets the system prompt for a specific provider in the active profile
func (cws *ConfigWithSettings) GetSystemPromptForProvider(providerName string) (string, error) {
	return cws.SettingsService.GetSystemPromptForProvider(providerName)
}