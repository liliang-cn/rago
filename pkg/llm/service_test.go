package llm

import (
	"context"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

func TestLLMService_New(t *testing.T) {
	config := core.TestLLMConfig()
	
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	if service == nil {
		t.Error("Expected service to be created, got nil")
	}
}

func TestLLMService_AddProvider(t *testing.T) {
	config := core.TestLLMConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	providerConfig := core.ProviderConfig{
		Type:    "test",
		BaseURL: "http://localhost:8080",
		Model:   "test-model",
		Weight:  1,
	}
	
	// This will return an error since we haven't implemented it yet
	err = service.AddProvider("test-provider", providerConfig)
	core.AssertError(t, err) // Expecting error until implementation
}

func TestLLMService_Generate(t *testing.T) {
	config := core.TestLLMConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	req := core.GenerationRequest{
		Prompt: "Hello, world!",
	}
	
	// This will return an error since we haven't implemented it yet
	_, err = service.Generate(ctx, req)
	core.AssertError(t, err) // Expecting error until implementation
}

func TestLLMService_ListProviders(t *testing.T) {
	config := core.TestLLMConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	providers := service.ListProviders()
	// Should be empty since we haven't implemented provider management yet
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
}

func TestLLMService_GetProviderHealth(t *testing.T) {
	config := core.TestLLMConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	health := service.GetProviderHealth()
	// Should be nil/empty since we haven't implemented health checking yet
	if health != nil && len(health) > 0 {
		t.Error("Expected empty health map, got populated map")
	}
}

func TestLLMService_Close(t *testing.T) {
	config := core.TestLLMConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	err = service.Close()
	core.AssertNoError(t, err)
}