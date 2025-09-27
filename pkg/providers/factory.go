package providers

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Factory implements the ProviderFactory interface
type Factory struct{}

// NewFactory creates a new provider factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateLLMProvider creates an LLM provider based on the configuration
func (f *Factory) CreateLLMProvider(ctx context.Context, config interface{}) (domain.LLMProvider, error) {
	// Try to handle as a typed configuration first
	switch cfg := config.(type) {
	case *domain.OllamaProviderConfig:
		return NewOllamaLLMProvider(cfg)
	case *domain.OpenAIProviderConfig:
		return NewOpenAILLMProvider(cfg)
	case *domain.LMStudioProviderConfig:
		return NewLMStudioLLMProvider(cfg)
	case *domain.ClaudeProviderConfig:
		return NewClaudeProvider(cfg)
	case *domain.GeminiProviderConfig:
		return NewGeminiProvider(cfg)
	case map[string]interface{}:
		// Handle dynamic configuration with type field
		return f.CreateLLMProviderFromMap(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported LLM provider config type: %T", config)
	}
}

// CreateLLMProviderFromMap creates an LLM provider from a map configuration
func (f *Factory) CreateLLMProviderFromMap(ctx context.Context, configMap map[string]interface{}) (domain.LLMProvider, error) {
	providerType, err := DetectProviderType(configMap)
	if err != nil {
		return nil, err
	}
	
	switch providerType {
	case domain.ProviderOllama:
		cfg := &domain.OllamaProviderConfig{}
		if err := mapToStruct(configMap, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse ollama config: %w", err)
		}
		cfg.Type = domain.ProviderOllama
		return NewOllamaLLMProvider(cfg)
	case domain.ProviderOpenAI:
		cfg := &domain.OpenAIProviderConfig{}
		if err := mapToStruct(configMap, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse openai config: %w", err)
		}
		cfg.Type = domain.ProviderOpenAI
		return NewOpenAILLMProvider(cfg)
	case domain.ProviderLMStudio:
		cfg := &domain.LMStudioProviderConfig{}
		if err := mapToStruct(configMap, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse lmstudio config: %w", err)
		}
		cfg.Type = domain.ProviderLMStudio
		return NewLMStudioLLMProvider(cfg)
	case domain.ProviderClaude:
		cfg := &domain.ClaudeProviderConfig{}
		if err := mapToStruct(configMap, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse claude config: %w", err)
		}
		cfg.Type = domain.ProviderClaude
		return NewClaudeProvider(cfg)
	case domain.ProviderGemini:
		cfg := &domain.GeminiProviderConfig{}
		if err := mapToStruct(configMap, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse gemini config: %w", err)
		}
		cfg.Type = domain.ProviderGemini
		return NewGeminiProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// CreateEmbedderProvider creates an embedder provider based on the configuration
func (f *Factory) CreateEmbedderProvider(ctx context.Context, config interface{}) (domain.EmbedderProvider, error) {
	// Try to handle as a typed configuration first
	switch cfg := config.(type) {
	case *domain.OllamaProviderConfig:
		return NewOllamaEmbedderProvider(cfg)
	case *domain.OpenAIProviderConfig:
		return NewOpenAIEmbedderProvider(cfg)
	case *domain.LMStudioProviderConfig:
		return NewLMStudioEmbedderProvider(cfg)
	case map[string]interface{}:
		// Handle dynamic configuration with type field
		return f.CreateEmbedderProviderFromMap(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported embedder provider config type: %T", config)
	}
}

// CreateEmbedderProviderFromMap creates an embedder provider from a map configuration
func (f *Factory) CreateEmbedderProviderFromMap(ctx context.Context, configMap map[string]interface{}) (domain.EmbedderProvider, error) {
	providerType, err := DetectProviderType(configMap)
	if err != nil {
		return nil, err
	}
	
	switch providerType {
	case domain.ProviderOllama:
		cfg := &domain.OllamaProviderConfig{}
		if err := mapToStruct(configMap, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse ollama config: %w", err)
		}
		cfg.Type = domain.ProviderOllama
		return NewOllamaEmbedderProvider(cfg)
	case domain.ProviderOpenAI:
		cfg := &domain.OpenAIProviderConfig{}
		if err := mapToStruct(configMap, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse openai config: %w", err)
		}
		cfg.Type = domain.ProviderOpenAI
		return NewOpenAIEmbedderProvider(cfg)
	case domain.ProviderLMStudio:
		cfg := &domain.LMStudioProviderConfig{}
		if err := mapToStruct(configMap, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse lmstudio config: %w", err)
		}
		cfg.Type = domain.ProviderLMStudio
		return NewLMStudioEmbedderProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// DetermineProviderType determines the provider type from configuration
func DetermineProviderType(config *domain.ProviderConfig) (domain.ProviderType, error) {
	if config.Ollama != nil {
		return domain.ProviderOllama, nil
	}
	if config.OpenAI != nil {
		return domain.ProviderOpenAI, nil
	}
	if config.LMStudio != nil {
		return domain.ProviderLMStudio, nil
	}
	return "", fmt.Errorf("no valid provider configuration found")
}

// GetProviderConfig returns the appropriate provider configuration
func GetProviderConfig(config *domain.ProviderConfig) (interface{}, error) {
	providerType, err := DetermineProviderType(config)
	if err != nil {
		return nil, err
	}

	switch providerType {
	case domain.ProviderOllama:
		return config.Ollama, nil
	case domain.ProviderOpenAI:
		return config.OpenAI, nil
	case domain.ProviderLMStudio:
		return config.LMStudio, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetLLMProviderConfig returns the LLM provider configuration based on default settings
func GetLLMProviderConfig(config *domain.ProviderConfig, defaultProvider string) (interface{}, error) {
	switch defaultProvider {
	case "ollama":
		if config.Ollama == nil {
			return nil, fmt.Errorf("ollama provider configuration not found")
		}
		return config.Ollama, nil
	case "openai":
		if config.OpenAI == nil {
			return nil, fmt.Errorf("openai provider configuration not found")
		}
		return config.OpenAI, nil
	case "lmstudio":
		if config.LMStudio == nil {
			return nil, fmt.Errorf("lmstudio provider configuration not found")
		}
		return config.LMStudio, nil
	default:
		return nil, fmt.Errorf("unsupported default LLM provider: %s", defaultProvider)
	}
}

// CreateLLMPool creates a pool of LLM providers from configuration
func (f *Factory) CreateLLMPool(ctx context.Context, providerConfigs map[string]interface{}, poolConfig LLMPoolConfig) (*LLMPool, error) {
	providers := make(map[string]domain.LLMProvider)

	for name, config := range providerConfigs {
		provider, err := f.CreateLLMProvider(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", name, err)
		}
		providers[name] = provider
	}

	return NewLLMPool(providers, poolConfig)
}

// GetEmbedderProviderConfig returns the embedder provider configuration based on default settings
func GetEmbedderProviderConfig(config *domain.ProviderConfig, defaultProvider string) (interface{}, error) {
	switch defaultProvider {
	case "ollama":
		if config.Ollama == nil {
			return nil, fmt.Errorf("ollama provider configuration not found")
		}
		return config.Ollama, nil
	case "openai":
		if config.OpenAI == nil {
			return nil, fmt.Errorf("openai provider configuration not found")
		}
		return config.OpenAI, nil
	case "lmstudio":
		if config.LMStudio == nil {
			return nil, fmt.Errorf("lmstudio provider configuration not found")
		}
		return config.LMStudio, nil
	default:
		return nil, fmt.Errorf("unsupported default embedder provider: %s", defaultProvider)
	}
}

// GetProviderConfigByName returns a provider configuration by name from dynamic providers map
func GetProviderConfigByName(providers map[string]interface{}, name string) (interface{}, error) {
	if providers == nil {
		return nil, fmt.Errorf("no providers configured")
	}
	
	config, exists := providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	
	return config, nil
}

// DetectProviderType detects the provider type from a configuration map
func DetectProviderType(config interface{}) (domain.ProviderType, error) {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid provider configuration format")
	}
	
	typeStr, exists := configMap["type"]
	if !exists {
		return "", fmt.Errorf("provider type not specified in configuration")
	}
	
	typeString, ok := typeStr.(string)
	if !ok {
		return "", fmt.Errorf("provider type must be a string")
	}
	
	switch typeString {
	case "ollama":
		return domain.ProviderOllama, nil
	case "openai":
		return domain.ProviderOpenAI, nil
	case "lmstudio":
		return domain.ProviderLMStudio, nil
	case "claude":
		return domain.ProviderClaude, nil
	case "gemini":
		return domain.ProviderGemini, nil
	default:
		return "", fmt.Errorf("unsupported provider type: %s", typeString)
	}
}

// mapToStruct converts a map to a struct using reflection
func mapToStruct(m map[string]interface{}, s interface{}) error {
	structValue := reflect.ValueOf(s).Elem()
	structType := structValue.Type()
	
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)
		
		// Get the mapstructure tag or use field name
		tagName := field.Tag.Get("mapstructure")
		if tagName == "" || tagName == ",squash" {
			tagName = toSnakeCase(field.Name)
		} else {
			// Handle tags with options like "timeout,omitempty"
			parts := strings.Split(tagName, ",")
			tagName = parts[0]
		}
		
		// Skip if field is not settable
		if !fieldValue.CanSet() {
			continue
		}
		
		// Get value from map
		value, exists := m[tagName]
		if !exists {
			continue
		}
		
		// Handle different types
		switch fieldValue.Kind() {
		case reflect.String:
			if str, ok := value.(string); ok {
				fieldValue.SetString(str)
			}
		case reflect.Int, reflect.Int64:
			switch v := value.(type) {
			case float64:
				fieldValue.SetInt(int64(v))
			case int:
				fieldValue.SetInt(int64(v))
			case int64:
				fieldValue.SetInt(v)
			case string:
				// Handle duration strings for time.Duration fields
				if field.Type == reflect.TypeOf(time.Duration(0)) {
					if d, err := time.ParseDuration(v); err == nil {
						fieldValue.SetInt(int64(d))
					}
				}
			}
		case reflect.Bool:
			if b, ok := value.(bool); ok {
				fieldValue.SetBool(b)
			}
		case reflect.Struct:
			// Handle embedded structs
			if field.Anonymous {
				if err := mapToStruct(m, fieldValue.Addr().Interface()); err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}

// toSnakeCase converts a CamelCase string to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, rune(strings.ToLower(string(r))[0]))
	}
	return string(result)
}
