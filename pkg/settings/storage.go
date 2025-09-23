package settings

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Storage interface defines the contract for settings persistence
type Storage interface {
	// Profile operations
	CreateProfile(req CreateProfileRequest) (*UserProfile, error)
	GetProfile(id string) (*UserProfile, error)
	GetActiveProfile() (*UserProfile, error)
	ListProfiles() ([]*UserProfile, error)
	UpdateProfile(id string, req UpdateProfileRequest) (*UserProfile, error)
	DeleteProfile(id string) error
	SetActiveProfile(id string) error

	// LLM Settings operations
	CreateLLMSettings(req CreateLLMSettingsRequest) (*LLMSettings, error)
	GetLLMSettings(profileID, providerName string) (*LLMSettings, error)
	GetAllLLMSettings(profileID string) ([]*LLMSettings, error)
	UpdateLLMSettings(id string, req UpdateLLMSettingsRequest) (*LLMSettings, error)
	DeleteLLMSettings(id string) error

	// Conversation context operations
	SaveConversationContext(profileID string, contextData map[string]interface{}, ttl time.Duration) (*ConversationContext, error)
	GetConversationContext(profileID string) (*ConversationContext, error)
	CleanupExpiredContexts() error

	// Utility operations
	Close() error
}

// SQLiteStorage implements Storage interface using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to enable foreign keys: %w (also failed to close db: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	storage := &SQLiteStorage{db: db}
	if err := storage.initTables(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to initialize tables: %w (also failed to close db: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return storage, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// initTables creates the necessary tables
func (s *SQLiteStorage) initTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS user_profiles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		default_system_prompt TEXT,
		metadata TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		is_active BOOLEAN DEFAULT FALSE
	);

	CREATE TABLE IF NOT EXISTS llm_settings (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		provider_name TEXT NOT NULL,
		system_prompt TEXT,
		temperature REAL,
		max_tokens INTEGER,
		settings_json TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		FOREIGN KEY (profile_id) REFERENCES user_profiles (id) ON DELETE CASCADE,
		UNIQUE(profile_id, provider_name)
	);

	CREATE TABLE IF NOT EXISTS conversation_contexts (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		context_data TEXT,
		created_at DATETIME,
		expires_at DATETIME,
		FOREIGN KEY (profile_id) REFERENCES user_profiles (id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_profiles_active ON user_profiles(is_active);
	CREATE INDEX IF NOT EXISTS idx_llm_settings_profile ON llm_settings(profile_id);
	CREATE INDEX IF NOT EXISTS idx_llm_settings_provider ON llm_settings(provider_name);
	CREATE INDEX IF NOT EXISTS idx_contexts_profile ON conversation_contexts(profile_id);
	CREATE INDEX IF NOT EXISTS idx_contexts_expires ON conversation_contexts(expires_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateProfile creates a new user profile
func (s *SQLiteStorage) CreateProfile(req CreateProfileRequest) (*UserProfile, error) {
	id := uuid.New().String()
	now := time.Now()

	metadataJSON := "{}"
	if req.Metadata != nil {
		data, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(data)
	}

	query := `
	INSERT INTO user_profiles (id, name, description, default_system_prompt, metadata, created_at, updated_at, is_active)
	VALUES (?, ?, ?, ?, ?, ?, ?, FALSE)`

	_, err := s.db.Exec(query, id, req.Name, req.Description, req.DefaultSystemPrompt, metadataJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	return s.GetProfile(id)
}

// GetProfile retrieves a profile by ID
func (s *SQLiteStorage) GetProfile(id string) (*UserProfile, error) {
	query := `
	SELECT id, name, description, default_system_prompt, metadata, created_at, updated_at, is_active
	FROM user_profiles WHERE id = ?`

	row := s.db.QueryRow(query, id)
	return s.scanProfile(row)
}

// GetActiveProfile retrieves the currently active profile
func (s *SQLiteStorage) GetActiveProfile() (*UserProfile, error) {
	query := `
	SELECT id, name, description, default_system_prompt, metadata, created_at, updated_at, is_active
	FROM user_profiles WHERE is_active = TRUE LIMIT 1`

	row := s.db.QueryRow(query)
	profile, err := s.scanProfile(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active profile found")
		}
		return nil, err
	}
	return profile, nil
}

// ListProfiles retrieves all profiles
func (s *SQLiteStorage) ListProfiles() ([]*UserProfile, error) {
	query := `
	SELECT id, name, description, default_system_prompt, metadata, created_at, updated_at, is_active
	FROM user_profiles ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			// Log error but don't override the main error
			// In a production system, you might want to log this error
			_ = closeErr
		}
	}()

	var profiles []*UserProfile
	for rows.Next() {
		profile, err := s.scanProfile(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// UpdateProfile updates an existing profile
func (s *SQLiteStorage) UpdateProfile(id string, req UpdateProfileRequest) (*UserProfile, error) {
	// First get the existing profile
	existing, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.DefaultSystemPrompt != nil {
		existing.DefaultSystemPrompt = *req.DefaultSystemPrompt
	}
	if req.Metadata != nil {
		existing.Metadata = *req.Metadata
	}
	existing.UpdatedAt = time.Now()

	metadataJSON, err := json.Marshal(existing.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
	UPDATE user_profiles SET 
		name = ?, description = ?, default_system_prompt = ?, metadata = ?, updated_at = ?
	WHERE id = ?`

	_, err = s.db.Exec(query, existing.Name, existing.Description, existing.DefaultSystemPrompt, string(metadataJSON), existing.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return s.GetProfile(id)
}

// DeleteProfile deletes a profile and all associated data
func (s *SQLiteStorage) DeleteProfile(id string) error {
	result, err := s.db.Exec("DELETE FROM user_profiles WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("profile not found: %s", id)
	}

	return nil
}

// SetActiveProfile sets a profile as the active one
func (s *SQLiteStorage) SetActiveProfile(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			// Log error but don't override the main error
			// In a production system, you might want to log this error
			_ = rollbackErr
		}
	}()

	// First, deactivate all profiles
	_, err = tx.Exec("UPDATE user_profiles SET is_active = FALSE")
	if err != nil {
		return fmt.Errorf("failed to deactivate profiles: %w", err)
	}

	// Then activate the specified profile
	result, err := tx.Exec("UPDATE user_profiles SET is_active = TRUE WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to activate profile: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("profile not found: %s", id)
	}

	return tx.Commit()
}

// CreateLLMSettings creates new LLM settings for a profile
func (s *SQLiteStorage) CreateLLMSettings(req CreateLLMSettingsRequest) (*LLMSettings, error) {
	id := uuid.New().String()
	now := time.Now()

	settingsJSON := "{}"
	if req.Settings != nil {
		data, err := json.Marshal(req.Settings)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal settings: %w", err)
		}
		settingsJSON = string(data)
	}

	query := `
	INSERT INTO llm_settings (id, profile_id, provider_name, system_prompt, temperature, max_tokens, settings_json, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, id, req.ProfileID, req.ProviderName, req.SystemPrompt, req.Temperature, req.MaxTokens, settingsJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM settings: %w", err)
	}

	return s.getLLMSettingsByID(id)
}

// GetLLMSettings retrieves LLM settings for a specific profile and provider
func (s *SQLiteStorage) GetLLMSettings(profileID, providerName string) (*LLMSettings, error) {
	query := `
	SELECT id, profile_id, provider_name, system_prompt, temperature, max_tokens, settings_json, created_at, updated_at
	FROM llm_settings WHERE profile_id = ? AND provider_name = ?`

	row := s.db.QueryRow(query, profileID, providerName)
	return s.scanLLMSettings(row)
}

// GetAllLLMSettings retrieves all LLM settings for a profile
func (s *SQLiteStorage) GetAllLLMSettings(profileID string) ([]*LLMSettings, error) {
	query := `
	SELECT id, profile_id, provider_name, system_prompt, temperature, max_tokens, settings_json, created_at, updated_at
	FROM llm_settings WHERE profile_id = ? ORDER BY provider_name`

	rows, err := s.db.Query(query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query LLM settings: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			// Log error but don't override the main error
			// In a production system, you might want to log this error
			_ = closeErr
		}
	}()

	var settings []*LLMSettings
	for rows.Next() {
		setting, err := s.scanLLMSettings(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan LLM settings: %w", err)
		}
		settings = append(settings, setting)
	}

	return settings, nil
}

// UpdateLLMSettings updates existing LLM settings
func (s *SQLiteStorage) UpdateLLMSettings(id string, req UpdateLLMSettingsRequest) (*LLMSettings, error) {
	// First get the existing settings
	existing, err := s.getLLMSettingsByID(id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.SystemPrompt != nil {
		existing.SystemPrompt = *req.SystemPrompt
	}
	if req.Temperature != nil {
		existing.Temperature = req.Temperature
	}
	if req.MaxTokens != nil {
		existing.MaxTokens = req.MaxTokens
	}
	if req.Settings != nil {
		existing.Settings = *req.Settings
	}
	existing.UpdatedAt = time.Now()

	settingsJSON, err := json.Marshal(existing.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
	UPDATE llm_settings SET 
		system_prompt = ?, temperature = ?, max_tokens = ?, settings_json = ?, updated_at = ?
	WHERE id = ?`

	_, err = s.db.Exec(query, existing.SystemPrompt, existing.Temperature, existing.MaxTokens, string(settingsJSON), existing.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update LLM settings: %w", err)
	}

	return s.getLLMSettingsByID(id)
}

// DeleteLLMSettings deletes LLM settings
func (s *SQLiteStorage) DeleteLLMSettings(id string) error {
	result, err := s.db.Exec("DELETE FROM llm_settings WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete LLM settings: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("LLM settings not found: %s", id)
	}

	return nil
}

// SaveConversationContext saves conversation context with expiration
func (s *SQLiteStorage) SaveConversationContext(profileID string, contextData map[string]interface{}, ttl time.Duration) (*ConversationContext, error) {
	id := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(ttl)

	contextJSON, err := json.Marshal(contextData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context data: %w", err)
	}

	// Delete existing context for this profile first
	_, err = s.db.Exec("DELETE FROM conversation_contexts WHERE profile_id = ?", profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing context: %w", err)
	}

	query := `
	INSERT INTO conversation_contexts (id, profile_id, context_data, created_at, expires_at)
	VALUES (?, ?, ?, ?, ?)`

	_, err = s.db.Exec(query, id, profileID, string(contextJSON), now, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to save conversation context: %w", err)
	}

	return &ConversationContext{
		ID:          id,
		ProfileID:   profileID,
		ContextData: contextData,
		CreatedAt:   now,
		ExpiresAt:   &expiresAt,
	}, nil
}

// GetConversationContext retrieves the latest conversation context for a profile
func (s *SQLiteStorage) GetConversationContext(profileID string) (*ConversationContext, error) {
	query := `
	SELECT id, profile_id, context_data, created_at, expires_at
	FROM conversation_contexts 
	WHERE profile_id = ? AND (expires_at IS NULL OR expires_at > ?)
	ORDER BY created_at DESC LIMIT 1`

	row := s.db.QueryRow(query, profileID, time.Now())
	return s.scanConversationContext(row)
}

// CleanupExpiredContexts removes expired conversation contexts
func (s *SQLiteStorage) CleanupExpiredContexts() error {
	_, err := s.db.Exec("DELETE FROM conversation_contexts WHERE expires_at IS NOT NULL AND expires_at <= ?", time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired contexts: %w", err)
	}
	return nil
}

// Helper methods for scanning

type RowScanner interface {
	Scan(dest ...interface{}) error
}

func (s *SQLiteStorage) scanProfile(row RowScanner) (*UserProfile, error) {
	var profile UserProfile
	var metadataJSON string

	err := row.Scan(
		&profile.ID, &profile.Name, &profile.Description, &profile.DefaultSystemPrompt,
		&metadataJSON, &profile.CreatedAt, &profile.UpdatedAt, &profile.IsActive,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found")
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(metadataJSON), &profile.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &profile, nil
}

func (s *SQLiteStorage) scanLLMSettings(row RowScanner) (*LLMSettings, error) {
	var settings LLMSettings
	var settingsJSON string

	err := row.Scan(
		&settings.ID, &settings.ProfileID, &settings.ProviderName, &settings.SystemPrompt,
		&settings.Temperature, &settings.MaxTokens, &settingsJSON, &settings.CreatedAt, &settings.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("LLM settings not found")
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(settingsJSON), &settings.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return &settings, nil
}

func (s *SQLiteStorage) scanConversationContext(row RowScanner) (*ConversationContext, error) {
	var context ConversationContext
	var contextJSON string
	var expiresAt sql.NullTime

	err := row.Scan(
		&context.ID, &context.ProfileID, &contextJSON, &context.CreatedAt, &expiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conversation context not found")
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(contextJSON), &context.ContextData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context data: %w", err)
	}

	if expiresAt.Valid {
		context.ExpiresAt = &expiresAt.Time
	}

	return &context, nil
}

func (s *SQLiteStorage) getLLMSettingsByID(id string) (*LLMSettings, error) {
	query := `
	SELECT id, profile_id, provider_name, system_prompt, temperature, max_tokens, settings_json, created_at, updated_at
	FROM llm_settings WHERE id = ?`

	row := s.db.QueryRow(query, id)
	return s.scanLLMSettings(row)
}