package settings

import (
	"time"
)

// UserProfile represents a user profile with associated settings
type UserProfile struct {
	ID                  string            `json:"id" db:"id"`
	Name                string            `json:"name" db:"name"`
	Description         string            `json:"description" db:"description"`
	DefaultSystemPrompt string            `json:"default_system_prompt" db:"default_system_prompt"`
	Metadata            map[string]string `json:"metadata" db:"metadata"`
	CreatedAt           time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at" db:"updated_at"`
	IsActive            bool              `json:"is_active" db:"is_active"`
}

// LLMSettings represents LLM-specific settings for a profile
type LLMSettings struct {
	ID           string                 `json:"id" db:"id"`
	ProfileID    string                 `json:"profile_id" db:"profile_id"`
	ProviderName string                 `json:"provider_name" db:"provider_name"`
	SystemPrompt string                 `json:"system_prompt" db:"system_prompt"`
	Temperature  *float64               `json:"temperature,omitempty" db:"temperature"`
	MaxTokens    *int                   `json:"max_tokens,omitempty" db:"max_tokens"`
	Settings     map[string]interface{} `json:"settings" db:"settings_json"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// ConversationContext represents context data for conversations
type ConversationContext struct {
	ID          string                 `json:"id" db:"id"`
	ProfileID   string                 `json:"profile_id" db:"profile_id"`
	ContextData map[string]interface{} `json:"context_data" db:"context_data"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty" db:"expires_at"`
}

// ProfileWithSettings represents a profile with its associated LLM settings
type ProfileWithSettings struct {
	Profile  UserProfile   `json:"profile"`
	Settings []LLMSettings `json:"settings"`
}

// CreateProfileRequest represents a request to create a new profile
type CreateProfileRequest struct {
	Name                string            `json:"name"`
	Description         string            `json:"description,omitempty"`
	DefaultSystemPrompt string            `json:"default_system_prompt,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
}

// UpdateProfileRequest represents a request to update a profile
type UpdateProfileRequest struct {
	Name                *string            `json:"name,omitempty"`
	Description         *string            `json:"description,omitempty"`
	DefaultSystemPrompt *string            `json:"default_system_prompt,omitempty"`
	Metadata            *map[string]string `json:"metadata,omitempty"`
}

// CreateLLMSettingsRequest represents a request to create LLM settings
type CreateLLMSettingsRequest struct {
	ProfileID    string                 `json:"profile_id"`
	ProviderName string                 `json:"provider_name"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	Temperature  *float64               `json:"temperature,omitempty"`
	MaxTokens    *int                   `json:"max_tokens,omitempty"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
}

// UpdateLLMSettingsRequest represents a request to update LLM settings
type UpdateLLMSettingsRequest struct {
	SystemPrompt *string                 `json:"system_prompt,omitempty"`
	Temperature  *float64                `json:"temperature,omitempty"`
	MaxTokens    *int                    `json:"max_tokens,omitempty"`
	Settings     *map[string]interface{} `json:"settings,omitempty"`
}