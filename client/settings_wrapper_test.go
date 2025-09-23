package client

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

func createTestClient(t *testing.T) (*BaseClient, string) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "client-settings-test")
	require.NoError(t, err)

	// Create test config
	cfg := &config.Config{
		Sqvect: config.SqvectConfig{
			DBPath: filepath.Join(tempDir, "rag.db"),
		},
		Providers: config.ProvidersConfig{
			DefaultLLM:      "ollama",
			DefaultEmbedder: "ollama",
			ProviderConfigs: domain.ProviderConfig{
				Ollama: &domain.OllamaProviderConfig{
					BaseURL:        "http://localhost:11434",
					LLMModel:       "qwen3",
					EmbeddingModel: "nomic-embed-text",
				},
			},
		},
	}

	// Create client
	client, err := NewWithConfig(cfg)
	require.NoError(t, err)

	return client, tempDir
}

func TestSettingsWrapper_ProfileOperations(t *testing.T) {
	client, tempDir := createTestClient(t)
	defer func() { _ = os.RemoveAll(tempDir) }()
	defer func() { _ = client.Close() }()

	t.Run("GetActiveProfile", func(t *testing.T) {
		profile, err := client.Settings.GetActiveProfile()
		require.NoError(t, err)
		assert.NotEmpty(t, profile.ID)
		assert.Equal(t, "default", profile.Name)
		assert.True(t, profile.IsActive)
	})

	t.Run("CreateProfile", func(t *testing.T) {
		profile, err := client.Settings.CreateProfile(
			"test-profile",
			"Test profile description",
			"Test system prompt",
		)
		require.NoError(t, err)
		assert.NotEmpty(t, profile.ID)
		assert.Equal(t, "test-profile", profile.Name)
		assert.Equal(t, "Test profile description", profile.Description)
		assert.Equal(t, "Test system prompt", profile.DefaultSystemPrompt)
		assert.False(t, profile.IsActive)
	})

	t.Run("ListProfiles", func(t *testing.T) {
		// Create additional profile
		_, err := client.Settings.CreateProfile("list-test", "List test", "")
		require.NoError(t, err)

		profiles, err := client.Settings.ListProfiles()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(profiles), 2)

		// Check for default and created profiles
		profileNames := make(map[string]bool)
		for _, p := range profiles {
			profileNames[p.Name] = true
		}
		assert.True(t, profileNames["default"])
		assert.True(t, profileNames["list-test"])
	})

	t.Run("SwitchProfile", func(t *testing.T) {
		// Create profile to switch to
		newProfile, err := client.Settings.CreateProfile("switch-test", "Switch test", "")
		require.NoError(t, err)

		// Switch to new profile
		err = client.Settings.SwitchProfile(newProfile.ID)
		require.NoError(t, err)

		// Verify it's active
		activeProfile, err := client.Settings.GetActiveProfile()
		require.NoError(t, err)
		assert.Equal(t, newProfile.ID, activeProfile.ID)
		assert.True(t, activeProfile.IsActive)
	})

	t.Run("GetProfile", func(t *testing.T) {
		// Create profile
		created, err := client.Settings.CreateProfile("get-test", "Get test", "Get prompt")
		require.NoError(t, err)

		// Get profile
		retrieved, err := client.Settings.GetProfile(created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.Name, retrieved.Name)
		assert.Equal(t, created.Description, retrieved.Description)
		assert.Equal(t, created.DefaultSystemPrompt, retrieved.DefaultSystemPrompt)
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		// Create profile
		created, err := client.Settings.CreateProfile("update-test", "Original", "Original prompt")
		require.NoError(t, err)

		// Update profile
		updated, err := client.Settings.UpdateProfile(created.ID, "Updated name", "Updated description", "Updated prompt")
		require.NoError(t, err)
		assert.Equal(t, "Updated name", updated.Name)
		assert.Equal(t, "Updated description", updated.Description)
		assert.Equal(t, "Updated prompt", updated.DefaultSystemPrompt)
		assert.True(t, updated.UpdatedAt.After(created.UpdatedAt))
	})

	t.Run("DeleteProfile", func(t *testing.T) {
		// Create profile
		created, err := client.Settings.CreateProfile("delete-test", "Delete test", "")
		require.NoError(t, err)

		// Delete profile
		err = client.Settings.DeleteProfile(created.ID)
		require.NoError(t, err)

		// Verify it's deleted
		_, err = client.Settings.GetProfile(created.ID)
		assert.Error(t, err)
	})
}

func TestSettingsWrapper_LLMSettingsOperations(t *testing.T) {
	client, tempDir := createTestClient(t)
	defer func() { _ = os.RemoveAll(tempDir) }()
	defer func() { _ = client.Close() }()

	// Get active profile
	profile, err := client.Settings.GetActiveProfile()
	require.NoError(t, err)

	t.Run("SetLLMSettings", func(t *testing.T) {
		temperature := 0.8
		maxTokens := 1500

		settings, err := client.Settings.SetLLMSettings(
			"ollama",
			"Test system prompt",
			&temperature,
			&maxTokens,
		)
		require.NoError(t, err)
		assert.Equal(t, profile.ID, settings.ProfileID)
		assert.Equal(t, "ollama", settings.ProviderName)
		assert.Equal(t, "Test system prompt", settings.SystemPrompt)
		assert.Equal(t, &temperature, settings.Temperature)
		assert.Equal(t, &maxTokens, settings.MaxTokens)
	})

	t.Run("GetLLMSettings", func(t *testing.T) {
		// Set settings first
		temperature := 0.7
		_, err := client.Settings.SetLLMSettings("test-provider", "Test prompt", &temperature, nil)
		require.NoError(t, err)

		// Get settings
		settings, err := client.Settings.GetLLMSettings("test-provider")
		require.NoError(t, err)
		assert.Equal(t, "test-provider", settings.ProviderName)
		assert.Equal(t, "Test prompt", settings.SystemPrompt)
		assert.Equal(t, &temperature, settings.Temperature)
	})

	t.Run("GetAllLLMSettings", func(t *testing.T) {
		// Set multiple settings
		providers := []string{"provider1", "provider2", "provider3"}
		for _, provider := range providers {
			_, err := client.Settings.SetLLMSettings(provider, "Prompt for "+provider, nil, nil)
			require.NoError(t, err)
		}

		// Get all settings
		allSettings, err := client.Settings.GetAllLLMSettings()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allSettings), 3)

		// Check providers are present
		providerMap := make(map[string]bool)
		for _, s := range allSettings {
			providerMap[s.ProviderName] = true
		}
		for _, provider := range providers {
			assert.True(t, providerMap[provider], "Provider %s not found", provider)
		}
	})

	t.Run("GetProfileWithSettings", func(t *testing.T) {
		// Set some settings first
		temperature := 0.9
		_, err := client.Settings.SetLLMSettings("combined-test", "Combined prompt", &temperature, nil)
		require.NoError(t, err)

		// Get profile with settings
		profile, settings, err := client.Settings.GetProfileWithSettings()
		require.NoError(t, err)
		assert.NotEmpty(t, profile.ID)
		assert.GreaterOrEqual(t, len(settings), 1)

		// Find our setting
		found := false
		for _, s := range settings {
			if s.ProviderName == "combined-test" {
				found = true
				assert.Equal(t, "Combined prompt", s.SystemPrompt)
				assert.Equal(t, &temperature, s.Temperature)
				break
			}
		}
		assert.True(t, found, "combined-test provider not found in settings")
	})
}

func TestSettingsWrapper_ConversationContext(t *testing.T) {
	client, tempDir := createTestClient(t)
	defer func() { _ = os.RemoveAll(tempDir) }()
	defer func() { _ = client.Close() }()

	t.Run("SaveConversationContext", func(t *testing.T) {
		contextData := map[string]interface{}{
			"conversation_id": "test-conversation",
			"topic":          "testing",
			"user_level":     "expert",
		}

		context, err := client.Settings.SaveConversationContext(contextData)
		require.NoError(t, err)
		assert.NotEmpty(t, context.ID)
		assert.Equal(t, contextData, context.ContextData)
		assert.NotNil(t, context.ExpiresAt)
		assert.True(t, context.ExpiresAt.After(time.Now()))
	})

	t.Run("GetConversationContext", func(t *testing.T) {
		// Save context first
		contextData := map[string]interface{}{
			"test_key": "test_value",
			"number":   42.0, // Use float64 to match JSON unmarshaling behavior
		}
		saved, err := client.Settings.SaveConversationContext(contextData)
		require.NoError(t, err)

		// Get context
		retrieved, err := client.Settings.GetConversationContext()
		require.NoError(t, err)
		assert.Equal(t, saved.ID, retrieved.ID)
		assert.Equal(t, saved.ContextData, retrieved.ContextData)
	})

	t.Run("GetConversationContext_NotFound", func(t *testing.T) {
		// Create new profile (should have no context)
		newProfile, err := client.Settings.CreateProfile("no-context", "No context", "")
		require.NoError(t, err)

		// Switch to new profile
		err = client.Settings.SwitchProfile(newProfile.ID)
		require.NoError(t, err)

		// Try to get context
		_, err = client.Settings.GetConversationContext()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversation context not found")
	})
}

func TestClientIntegration_GenerationWithSettings(t *testing.T) {
	client, tempDir := createTestClient(t)
	defer func() { _ = os.RemoveAll(tempDir) }()
	defer func() { _ = client.Close() }()

	// Skip if no Ollama available (this would be an integration test)
	_ = context.Background() // Unused in unit tests

	t.Run("GetSystemPromptForProvider", func(t *testing.T) {
		// Set a system prompt
		_, err := client.Settings.SetLLMSettings("ollama", "Test system prompt for ollama", nil, nil)
		require.NoError(t, err)

		// Get system prompt
		prompt, err := client.GetSystemPromptForProvider("ollama")
		require.NoError(t, err)
		assert.Equal(t, "Test system prompt for ollama", prompt)
	})

	t.Run("SetSystemPromptForProvider", func(t *testing.T) {
		err := client.SetSystemPromptForProvider("ollama", "New system prompt")
		require.NoError(t, err)

		// Verify it was set
		prompt, err := client.GetSystemPromptForProvider("ollama")
		require.NoError(t, err)
		assert.Equal(t, "New system prompt", prompt)
	})

	t.Run("GenerateWithProfile_SettingsApplied", func(t *testing.T) {
		// This is more of an integration test that would require actual provider
		// For unit testing, we verify the settings are properly configured
		
		// Set specific settings
		temperature := 0.5
		maxTokens := 100
		_, err := client.Settings.SetLLMSettings("ollama", "You are a helpful assistant.", &temperature, &maxTokens)
		require.NoError(t, err)

		// Verify settings exist and would be applied
		settings, err := client.Settings.GetLLMSettings("ollama")
		require.NoError(t, err)
		assert.Equal(t, &temperature, settings.Temperature)
		assert.Equal(t, &maxTokens, settings.MaxTokens)
		assert.Equal(t, "You are a helpful assistant.", settings.SystemPrompt)

		// Note: Actual generation test would require a running provider
		// For unit testing, we verify the configuration is correct
	})

	t.Run("GetProviderForProfile", func(t *testing.T) {
		// Set settings for the provider
		_, err := client.Settings.SetLLMSettings("ollama", "Provider test prompt", nil, nil)
		require.NoError(t, err)

		// Get provider (this returns the wrapped provider with settings)
		provider, err := client.GetProviderForProfile("ollama")
		require.NoError(t, err)
		assert.NotNil(t, provider)

		// Verify it's a settings-aware provider (would be LLMProviderWrapper)
		// The actual type assertion would depend on the implementation
		assert.Implements(t, (*domain.LLMProvider)(nil), provider)
	})
}

func TestSettingsWrapper_ErrorHandling(t *testing.T) {
	client, tempDir := createTestClient(t)
	defer func() { _ = os.RemoveAll(tempDir) }()
	defer func() { _ = client.Close() }()

	t.Run("GetProfile_NotFound", func(t *testing.T) {
		_, err := client.Settings.GetProfile("non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "profile not found")
	})

	t.Run("SwitchProfile_NotFound", func(t *testing.T) {
		err := client.Settings.SwitchProfile("non-existent-id")
		assert.Error(t, err)
	})

	t.Run("GetLLMSettings_NotFound", func(t *testing.T) {
		_, err := client.Settings.GetLLMSettings("non-existent-provider")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LLM settings not found")
	})

	t.Run("DeleteProfile_NotFound", func(t *testing.T) {
		err := client.Settings.DeleteProfile("non-existent-id")
		assert.Error(t, err)
	})
}

func TestSettingsWrapper_BackwardCompatibility(t *testing.T) {
	client, tempDir := createTestClient(t)
	defer func() { _ = os.RemoveAll(tempDir) }()
	defer func() { _ = client.Close() }()

	t.Run("ExistingAPIStillWorks", func(t *testing.T) {
		// Verify that existing client methods still work without settings
		_ = context.Background() // Unused in unit tests

		// These methods should work regardless of settings
		config := client.GetConfig()
		assert.NotNil(t, config)

		// LLM wrapper should be available
		assert.NotNil(t, client.LLM)

		// RAG wrapper should be available
		assert.NotNil(t, client.RAG)

		// Settings wrapper should be available (new feature)
		assert.NotNil(t, client.Settings)

		// Note: Actual generation calls would require running providers
		// For unit testing, we verify the structures are properly initialized
	})
}