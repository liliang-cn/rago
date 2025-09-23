package settings

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// LLMProviderWrapper wraps an LLM provider to apply profile settings
type LLMProviderWrapper struct {
	provider  domain.LLMProvider
	service   *Service
	override  *LLMSettings
}

// NewLLMProviderWrapper creates a new provider wrapper
func NewLLMProviderWrapper(provider domain.LLMProvider, service *Service) *LLMProviderWrapper {
	return &LLMProviderWrapper{
		provider: provider,
		service:  service,
	}
}

// NewLLMProviderWrapperWithOverride creates a wrapper with specific settings override
func NewLLMProviderWrapperWithOverride(provider domain.LLMProvider, settings *LLMSettings) *LLMProviderWrapper {
	return &LLMProviderWrapper{
		provider: provider,
		override: settings,
	}
}

// ProviderType returns the underlying provider type
func (w *LLMProviderWrapper) ProviderType() domain.ProviderType {
	return w.provider.ProviderType()
}

// getEffectiveSystemPrompt returns the effective system prompt for the current context
func (w *LLMProviderWrapper) getEffectiveSystemPrompt() (string, error) {
	if w.override != nil {
		return w.override.SystemPrompt, nil
	}

	if w.service != nil {
		prompt, err := w.service.GetSystemPromptForProvider(string(w.provider.ProviderType()))
		if err != nil {
			// Log but don't fail - return empty string to use provider defaults
			return "", nil
		}
		return prompt, nil
	}

	return "", nil
}

// getEffectiveOptions returns generation options with profile settings applied
func (w *LLMProviderWrapper) getEffectiveOptions(opts *domain.GenerationOptions) (*domain.GenerationOptions, error) {
	if opts == nil {
		opts = &domain.GenerationOptions{}
	}

	// Create a copy to avoid modifying the original
	effectiveOpts := *opts

	// Apply settings based on override or service
	if w.override != nil {
		if w.override.Temperature != nil && effectiveOpts.Temperature == 0 {
			effectiveOpts.Temperature = *w.override.Temperature
		}
		if w.override.MaxTokens != nil && effectiveOpts.MaxTokens == 0 {
			effectiveOpts.MaxTokens = *w.override.MaxTokens
		}
	} else if w.service != nil {
		temperature, maxTokens, _, err := w.service.GetLLMParametersForProvider(string(w.provider.ProviderType()))
		if err == nil {
			if temperature != nil && effectiveOpts.Temperature == 0 {
				effectiveOpts.Temperature = *temperature
			}
			if maxTokens != nil && effectiveOpts.MaxTokens == 0 {
				effectiveOpts.MaxTokens = *maxTokens
			}
		}
	}

	return &effectiveOpts, nil
}

// Generate applies profile settings and delegates to the wrapped provider
func (w *LLMProviderWrapper) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	// Get effective system prompt
	systemPrompt, err := w.getEffectiveSystemPrompt()
	if err != nil {
		return "", fmt.Errorf("failed to get system prompt: %w", err)
	}

	// Apply system prompt to the beginning of the prompt if it exists
	effectivePrompt := prompt
	if systemPrompt != "" {
		effectivePrompt = systemPrompt + "\n\n" + prompt
	}

	// Get effective options
	effectiveOpts, err := w.getEffectiveOptions(opts)
	if err != nil {
		return "", fmt.Errorf("failed to get effective options: %w", err)
	}

	return w.provider.Generate(ctx, effectivePrompt, effectiveOpts)
}

// Stream applies profile settings and delegates to the wrapped provider
func (w *LLMProviderWrapper) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	// Get effective system prompt
	systemPrompt, err := w.getEffectiveSystemPrompt()
	if err != nil {
		return fmt.Errorf("failed to get system prompt: %w", err)
	}

	// Apply system prompt to the beginning of the prompt if it exists
	effectivePrompt := prompt
	if systemPrompt != "" {
		effectivePrompt = systemPrompt + "\n\n" + prompt
	}

	// Get effective options
	effectiveOpts, err := w.getEffectiveOptions(opts)
	if err != nil {
		return fmt.Errorf("failed to get effective options: %w", err)
	}

	return w.provider.Stream(ctx, effectivePrompt, effectiveOpts, callback)
}

// GenerateWithTools applies profile settings and delegates to the wrapped provider
func (w *LLMProviderWrapper) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	// Get effective system prompt
	systemPrompt, err := w.getEffectiveSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to get system prompt: %w", err)
	}

	// Apply system prompt by prepending it as a system message if it exists
	effectiveMessages := make([]domain.Message, len(messages))
	copy(effectiveMessages, messages)

	if systemPrompt != "" {
		// Check if first message is already a system message
		if len(effectiveMessages) > 0 && effectiveMessages[0].Role == "system" {
			// Prepend to existing system message
			effectiveMessages[0].Content = systemPrompt + "\n\n" + effectiveMessages[0].Content
		} else {
			// Add new system message at the beginning
			systemMessage := domain.Message{
				Role:    "system",
				Content: systemPrompt,
			}
			effectiveMessages = append([]domain.Message{systemMessage}, effectiveMessages...)
		}
	}

	// Get effective options
	effectiveOpts, err := w.getEffectiveOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective options: %w", err)
	}

	return w.provider.GenerateWithTools(ctx, effectiveMessages, tools, effectiveOpts)
}

// StreamWithTools applies profile settings and delegates to the wrapped provider
func (w *LLMProviderWrapper) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	// Get effective system prompt
	systemPrompt, err := w.getEffectiveSystemPrompt()
	if err != nil {
		return fmt.Errorf("failed to get system prompt: %w", err)
	}

	// Apply system prompt by prepending it as a system message if it exists
	effectiveMessages := make([]domain.Message, len(messages))
	copy(effectiveMessages, messages)

	if systemPrompt != "" {
		// Check if first message is already a system message
		if len(effectiveMessages) > 0 && effectiveMessages[0].Role == "system" {
			// Prepend to existing system message
			effectiveMessages[0].Content = systemPrompt + "\n\n" + effectiveMessages[0].Content
		} else {
			// Add new system message at the beginning
			systemMessage := domain.Message{
				Role:    "system",
				Content: systemPrompt,
			}
			effectiveMessages = append([]domain.Message{systemMessage}, effectiveMessages...)
		}
	}

	// Get effective options
	effectiveOpts, err := w.getEffectiveOptions(opts)
	if err != nil {
		return fmt.Errorf("failed to get effective options: %w", err)
	}

	return w.provider.StreamWithTools(ctx, effectiveMessages, tools, effectiveOpts, callback)
}

// GenerateStructured applies profile settings and delegates to the wrapped provider
func (w *LLMProviderWrapper) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	// Get effective system prompt
	systemPrompt, err := w.getEffectiveSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to get system prompt: %w", err)
	}

	// Apply system prompt to the beginning of the prompt if it exists
	effectivePrompt := prompt
	if systemPrompt != "" {
		effectivePrompt = systemPrompt + "\n\n" + prompt
	}

	// Get effective options
	effectiveOpts, err := w.getEffectiveOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective options: %w", err)
	}

	return w.provider.GenerateStructured(ctx, effectivePrompt, schema, effectiveOpts)
}

// RecognizeIntent delegates to the wrapped provider
func (w *LLMProviderWrapper) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	return w.provider.RecognizeIntent(ctx, request)
}

// Health delegates to the wrapped provider
func (w *LLMProviderWrapper) Health(ctx context.Context) error {
	return w.provider.Health(ctx)
}

// ExtractMetadata delegates to the wrapped provider
func (w *LLMProviderWrapper) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	return w.provider.ExtractMetadata(ctx, content, model)
}

// ProviderFactory wrapper to create providers with settings integration
type ProviderFactoryWrapper struct {
	factory domain.ProviderFactory
	service *Service
}

// NewProviderFactoryWrapper creates a new factory wrapper
func NewProviderFactoryWrapper(factory domain.ProviderFactory, service *Service) *ProviderFactoryWrapper {
	return &ProviderFactoryWrapper{
		factory: factory,
		service: service,
	}
}

// CreateLLMProvider creates an LLM provider with settings integration
func (fw *ProviderFactoryWrapper) CreateLLMProvider(ctx context.Context, config interface{}) (domain.LLMProvider, error) {
	provider, err := fw.factory.CreateLLMProvider(ctx, config)
	if err != nil {
		return nil, err
	}

	// Wrap with settings service
	return NewLLMProviderWrapper(provider, fw.service), nil
}

// CreateEmbedderProvider delegates to the underlying factory
func (fw *ProviderFactoryWrapper) CreateEmbedderProvider(ctx context.Context, config interface{}) (domain.EmbedderProvider, error) {
	return fw.factory.CreateEmbedderProvider(ctx, config)
}

// ProviderClient provides a high-level interface for working with providers and settings
type ProviderClient struct {
	service   *Service
	providers map[string]domain.LLMProvider
}

// NewProviderClient creates a new provider client
func NewProviderClient(service *Service) *ProviderClient {
	return &ProviderClient{
		service:   service,
		providers: make(map[string]domain.LLMProvider),
	}
}

// RegisterProvider registers a provider with a name
func (pc *ProviderClient) RegisterProvider(name string, provider domain.LLMProvider) {
	pc.providers[name] = NewLLMProviderWrapper(provider, pc.service)
}

// GetProvider returns a provider by name, wrapped with settings
func (pc *ProviderClient) GetProvider(name string) (domain.LLMProvider, error) {
	provider, exists := pc.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// GetProviderForProfile returns a provider with specific profile settings
func (pc *ProviderClient) GetProviderForProfile(name, profileID string) (domain.LLMProvider, error) {
	baseProvider, exists := pc.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", name)
	}

	// Get the unwrapped provider
	var unwrapped domain.LLMProvider
	if wrapper, ok := baseProvider.(*LLMProviderWrapper); ok {
		unwrapped = wrapper.provider
	} else {
		unwrapped = baseProvider
	}

	// Get settings for this profile and provider
	settings, err := pc.service.GetLLMSettings(profileID, name)
	if err != nil {
		// No specific settings found, use service wrapper
		return NewLLMProviderWrapper(unwrapped, pc.service), nil
	}

	// Return wrapper with specific settings override
	return NewLLMProviderWrapperWithOverride(unwrapped, settings), nil
}

// GenerateWithToolsWithActiveProfile performs tool-based generation using the active profile settings
func (pc *ProviderClient) GenerateWithToolsWithActiveProfile(ctx context.Context, providerName string, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	activeProfile, err := pc.service.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get active profile: %w", err)
	}

	provider, err := pc.GetProviderForProfile(providerName, activeProfile.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return provider.GenerateWithTools(ctx, messages, tools, opts)
}

// GenerateWithActiveProfile performs generation using the active profile settings
func (pc *ProviderClient) GenerateWithActiveProfile(ctx context.Context, providerName string, prompt string, opts *domain.GenerationOptions) (string, error) {
	activeProfile, err := pc.service.GetActiveProfile()
	if err != nil {
		return "", fmt.Errorf("failed to get active profile: %w", err)
	}

	provider, err := pc.GetProviderForProfile(providerName, activeProfile.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get provider: %w", err)
	}

	return provider.Generate(ctx, prompt, opts)
}