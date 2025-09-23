package settings

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_ProfileOperations(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "settings-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	dbPath := filepath.Join(tempDir, "test.db")
	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer func() { _ = storage.Close() }()

	t.Run("CreateProfile", func(t *testing.T) {
		req := CreateProfileRequest{
			Name:        "test-profile",
			Description: "Test profile description",
			DefaultSystemPrompt: "Test system prompt",
			Metadata:    map[string]string{"key": "value"},
		}

		profile, err := storage.CreateProfile(req)
		require.NoError(t, err)
		assert.NotEmpty(t, profile.ID)
		assert.Equal(t, req.Name, profile.Name)
		assert.Equal(t, req.Description, profile.Description)
		assert.Equal(t, req.DefaultSystemPrompt, profile.DefaultSystemPrompt)
		assert.Equal(t, req.Metadata, profile.Metadata)
		assert.False(t, profile.IsActive)
	})

	t.Run("GetProfile", func(t *testing.T) {
		// Create a profile first
		req := CreateProfileRequest{
			Name:        "get-test-profile",
			Description: "Get test description",
		}
		created, err := storage.CreateProfile(req)
		require.NoError(t, err)

		// Get the profile
		retrieved, err := storage.GetProfile(created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.Name, retrieved.Name)
		assert.Equal(t, created.Description, retrieved.Description)
	})

	t.Run("GetProfile_NotFound", func(t *testing.T) {
		_, err := storage.GetProfile("non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "profile not found")
	})

	t.Run("ListProfiles", func(t *testing.T) {
		// Create multiple profiles
		names := []string{"profile1", "profile2", "profile3"}
		for _, name := range names {
			req := CreateProfileRequest{Name: name, Description: "Test " + name}
			_, err := storage.CreateProfile(req)
			require.NoError(t, err)
		}

		profiles, err := storage.ListProfiles()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(profiles), 3)

		// Check that our profiles are in the list
		profileNames := make(map[string]bool)
		for _, p := range profiles {
			profileNames[p.Name] = true
		}
		for _, name := range names {
			assert.True(t, profileNames[name], "Profile %s not found in list", name)
		}
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		// Create a profile
		req := CreateProfileRequest{
			Name:        "update-test",
			Description: "Original description",
		}
		created, err := storage.CreateProfile(req)
		require.NoError(t, err)

		// Update the profile
		newName := "updated-name"
		newDesc := "Updated description"
		updateReq := UpdateProfileRequest{
			Name:        &newName,
			Description: &newDesc,
		}

		updated, err := storage.UpdateProfile(created.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, newName, updated.Name)
		assert.Equal(t, newDesc, updated.Description)
		assert.True(t, updated.UpdatedAt.After(created.UpdatedAt))
	})

	t.Run("SetActiveProfile", func(t *testing.T) {
		// Create two profiles
		req1 := CreateProfileRequest{Name: "profile-1"}
		profile1, err := storage.CreateProfile(req1)
		require.NoError(t, err)

		req2 := CreateProfileRequest{Name: "profile-2"}
		profile2, err := storage.CreateProfile(req2)
		require.NoError(t, err)

		// Set first profile as active
		err = storage.SetActiveProfile(profile1.ID)
		require.NoError(t, err)

		// Verify first profile is active
		active, err := storage.GetActiveProfile()
		require.NoError(t, err)
		assert.Equal(t, profile1.ID, active.ID)
		assert.True(t, active.IsActive)

		// Set second profile as active
		err = storage.SetActiveProfile(profile2.ID)
		require.NoError(t, err)

		// Verify second profile is active and first is not
		active, err = storage.GetActiveProfile()
		require.NoError(t, err)
		assert.Equal(t, profile2.ID, active.ID)

		// Check first profile is no longer active
		profile1Updated, err := storage.GetProfile(profile1.ID)
		require.NoError(t, err)
		assert.False(t, profile1Updated.IsActive)
	})

	t.Run("DeleteProfile", func(t *testing.T) {
		// Create a profile
		req := CreateProfileRequest{Name: "delete-test"}
		created, err := storage.CreateProfile(req)
		require.NoError(t, err)

		// Delete the profile
		err = storage.DeleteProfile(created.ID)
		require.NoError(t, err)

		// Verify it's deleted
		_, err = storage.GetProfile(created.ID)
		assert.Error(t, err)
	})
}

func TestSQLiteStorage_LLMSettingsOperations(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "settings-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	dbPath := filepath.Join(tempDir, "test.db")
	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer func() { _ = storage.Close() }()

	// Create a profile first
	profileReq := CreateProfileRequest{Name: "test-profile"}
	profile, err := storage.CreateProfile(profileReq)
	require.NoError(t, err)

	t.Run("CreateLLMSettings", func(t *testing.T) {
		temperature := 0.7
		maxTokens := 1000
		req := CreateLLMSettingsRequest{
			ProfileID:    profile.ID,
			ProviderName: "ollama",
			SystemPrompt: "Test system prompt",
			Temperature:  &temperature,
			MaxTokens:    &maxTokens,
			Settings:     map[string]interface{}{"custom": "value"},
		}

		settings, err := storage.CreateLLMSettings(req)
		require.NoError(t, err)
		assert.NotEmpty(t, settings.ID)
		assert.Equal(t, req.ProfileID, settings.ProfileID)
		assert.Equal(t, req.ProviderName, settings.ProviderName)
		assert.Equal(t, req.SystemPrompt, settings.SystemPrompt)
		assert.Equal(t, req.Temperature, settings.Temperature)
		assert.Equal(t, req.MaxTokens, settings.MaxTokens)
		assert.Equal(t, req.Settings, settings.Settings)
	})

	t.Run("GetLLMSettings", func(t *testing.T) {
		// Create settings first
		req := CreateLLMSettingsRequest{
			ProfileID:    profile.ID,
			ProviderName: "test-provider",
			SystemPrompt: "Get test prompt",
		}
		created, err := storage.CreateLLMSettings(req)
		require.NoError(t, err)

		// Get the settings
		retrieved, err := storage.GetLLMSettings(profile.ID, "test-provider")
		require.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.SystemPrompt, retrieved.SystemPrompt)
	})

	t.Run("GetLLMSettings_NotFound", func(t *testing.T) {
		_, err := storage.GetLLMSettings(profile.ID, "non-existent-provider")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LLM settings not found")
	})

	t.Run("GetAllLLMSettings", func(t *testing.T) {
		// Create multiple settings for the profile
		providers := []string{"provider1", "provider2", "provider3"}
		for _, provider := range providers {
			req := CreateLLMSettingsRequest{
				ProfileID:    profile.ID,
				ProviderName: provider,
				SystemPrompt: "Prompt for " + provider,
			}
			_, err := storage.CreateLLMSettings(req)
			require.NoError(t, err)
		}

		allSettings, err := storage.GetAllLLMSettings(profile.ID)
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

	t.Run("UpdateLLMSettings", func(t *testing.T) {
		// Create settings
		req := CreateLLMSettingsRequest{
			ProfileID:    profile.ID,
			ProviderName: "update-provider",
			SystemPrompt: "Original prompt",
		}
		created, err := storage.CreateLLMSettings(req)
		require.NoError(t, err)

		// Update settings
		newPrompt := "Updated prompt"
		newTemp := 0.9
		updateReq := UpdateLLMSettingsRequest{
			SystemPrompt: &newPrompt,
			Temperature:  &newTemp,
		}

		updated, err := storage.UpdateLLMSettings(created.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, newPrompt, updated.SystemPrompt)
		assert.Equal(t, &newTemp, updated.Temperature)
		assert.True(t, updated.UpdatedAt.After(created.UpdatedAt))
	})

	t.Run("DeleteLLMSettings", func(t *testing.T) {
		// Create settings
		req := CreateLLMSettingsRequest{
			ProfileID:    profile.ID,
			ProviderName: "delete-provider",
		}
		created, err := storage.CreateLLMSettings(req)
		require.NoError(t, err)

		// Delete settings
		err = storage.DeleteLLMSettings(created.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = storage.GetLLMSettings(profile.ID, "delete-provider")
		assert.Error(t, err)
	})
}

func TestSQLiteStorage_ConversationContextOperations(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "settings-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	dbPath := filepath.Join(tempDir, "test.db")
	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer func() { _ = storage.Close() }()

	// Create a profile first
	profileReq := CreateProfileRequest{Name: "test-profile"}
	profile, err := storage.CreateProfile(profileReq)
	require.NoError(t, err)

	t.Run("SaveConversationContext", func(t *testing.T) {
		contextData := map[string]interface{}{
			"conversation_id": "test-conversation",
			"topic":          "testing",
			"user_level":     "expert",
		}

		ttl := time.Hour
		context, err := storage.SaveConversationContext(profile.ID, contextData, ttl)
		require.NoError(t, err)
		assert.NotEmpty(t, context.ID)
		assert.Equal(t, profile.ID, context.ProfileID)
		assert.Equal(t, contextData, context.ContextData)
		assert.NotNil(t, context.ExpiresAt)
		assert.True(t, context.ExpiresAt.After(time.Now()))
	})

	t.Run("GetConversationContext", func(t *testing.T) {
		// Save context first
		contextData := map[string]interface{}{
			"test_key": "test_value",
		}
		ttl := time.Hour
		saved, err := storage.SaveConversationContext(profile.ID, contextData, ttl)
		require.NoError(t, err)

		// Get context
		retrieved, err := storage.GetConversationContext(profile.ID)
		require.NoError(t, err)
		assert.Equal(t, saved.ID, retrieved.ID)
		assert.Equal(t, saved.ContextData, retrieved.ContextData)
	})

	t.Run("GetConversationContext_NotFound", func(t *testing.T) {
		// Create another profile
		profileReq := CreateProfileRequest{Name: "empty-profile"}
		emptyProfile, err := storage.CreateProfile(profileReq)
		require.NoError(t, err)

		_, err = storage.GetConversationContext(emptyProfile.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversation context not found")
	})

	t.Run("CleanupExpiredContexts", func(t *testing.T) {
		// Save context with very short TTL
		contextData := map[string]interface{}{"expired": "data"}
		shortTTL := time.Millisecond
		_, err := storage.SaveConversationContext(profile.ID, contextData, shortTTL)
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(time.Millisecond * 10)

		// Cleanup expired contexts
		err = storage.CleanupExpiredContexts()
		require.NoError(t, err)

		// Verify context is gone
		_, err = storage.GetConversationContext(profile.ID)
		assert.Error(t, err)
	})
}

func TestSQLiteStorage_DatabaseConstraints(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "settings-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	dbPath := filepath.Join(tempDir, "test.db")
	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer func() { _ = storage.Close() }()

	t.Run("UniqueProfileName", func(t *testing.T) {
		// Create first profile
		req := CreateProfileRequest{Name: "unique-test"}
		_, err := storage.CreateProfile(req)
		require.NoError(t, err)

		// Try to create another profile with same name
		_, err = storage.CreateProfile(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create profile")
	})

	t.Run("CascadeDeleteProfile", func(t *testing.T) {
		// Create profile
		profileReq := CreateProfileRequest{Name: "cascade-test"}
		profile, err := storage.CreateProfile(profileReq)
		require.NoError(t, err)

		// Create LLM settings for the profile
		llmReq := CreateLLMSettingsRequest{
			ProfileID:    profile.ID,
			ProviderName: "test-provider",
		}
		_, err = storage.CreateLLMSettings(llmReq)
		require.NoError(t, err)

		// Create conversation context
		contextData := map[string]interface{}{"test": "data"}
		_, err = storage.SaveConversationContext(profile.ID, contextData, time.Hour)
		require.NoError(t, err)

		// Delete profile
		err = storage.DeleteProfile(profile.ID)
		require.NoError(t, err)

		// Verify LLM settings are also deleted
		_, err = storage.GetLLMSettings(profile.ID, "test-provider")
		assert.Error(t, err)

		// Verify conversation context is also deleted
		_, err = storage.GetConversationContext(profile.ID)
		assert.Error(t, err)
	})
}

func TestSQLiteStorage_InitializationErrors(t *testing.T) {
	t.Run("InvalidDatabasePath", func(t *testing.T) {
		// Try to create storage with invalid path
		_, err := NewSQLiteStorage("/invalid/path/that/does/not/exist/test.db")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create database directory")
	})
}