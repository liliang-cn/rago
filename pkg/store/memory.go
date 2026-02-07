package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/liliang-cn/sqvect/v2/pkg/hindsight"
	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrMemoryNotFound = errors.New("memory not found")
)

// Memory represents a single long-term memory
type Memory struct {
	ID           string                 `json:"id"`
	SessionID    string                 `json:"session_id,omitempty"`
	Type         string                 `json:"type"`
	Content      string                 `json:"content"`
	Vector       []float64              `json:"vector,omitempty"`
	Importance   float64                `json:"importance"`
	AccessCount  int                    `json:"access_count"`
	LastAccessed time.Time              `json:"last_accessed"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// MemoryWithScore represents a memory with its similarity score
type MemoryWithScore struct {
	*Memory
	Score float64
}

// MemoryStore handles memory persistence using Hindsight
type MemoryStore struct {
	sys    *hindsight.System
	dbPath string
}

// NewMemoryStore creates a new memory store backed by Hindsight/sqvect
func NewMemoryStore(dbPath string) (*MemoryStore, error) {
	if dbPath == "" {
		return nil, errors.New("dbPath is required")
	}

	cfg := hindsight.DefaultConfig(dbPath)
	sys, err := hindsight.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize hindsight system: %w", err)
	}

	return &MemoryStore{sys: sys, dbPath: dbPath}, nil
}

// getBankIDsFromDB retrieves all unique bank_ids from the embeddings metadata
// This is a workaround because hindsight.ListBanks() might not return all banks
func (s *MemoryStore) getBankIDsFromDB() ([]string, error) {
	db, err := sql.Open("sqlite3", s.dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Get all unique bank_ids from metadata
	rows, err := db.Query(`
		SELECT DISTINCT json_extract(metadata, '$.bank_id') as bank_id
		FROM embeddings
		WHERE json_extract(metadata, '$.bank_id') IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bankIDs []string
	for rows.Next() {
		var bankID sql.NullString
		if err := rows.Scan(&bankID); err == nil && bankID.Valid {
			bankIDs = append(bankIDs, bankID.String)
		}
	}

	return bankIDs, nil
}

// InitSchema is handled by Hindsight internally, keeping for interface compatibility
func (s *MemoryStore) InitSchema(ctx context.Context) error {
	return nil
}

// Store saves a new memory using Hindsight
func (s *MemoryStore) Store(ctx context.Context, memory *Memory) error {
	// Determine bank ID
	bankID := memory.SessionID
	if bankID == "" {
		bankID = "default"
	}

	// Ensure bank exists before storing memories
	banks := s.sys.ListBanks()
	bankExists := false
	for _, b := range banks {
		if b.ID == bankID {
			bankExists = true
			break
		}
	}

	// Create bank if it doesn't exist
	if !bankExists {
		defaultBank := hindsight.NewBank(bankID, "User Memory Bank")
		defaultBank.Skepticism = 3
		defaultBank.Literalism = 3
		defaultBank.Empathy = 3
		if err := s.sys.CreateBank(ctx, defaultBank); err != nil {
			return fmt.Errorf("failed to create bank: %w", err)
		}
	}

	// Map to hindsight.Memory
	hMem := &hindsight.Memory{
		ID:         memory.ID,
		BankID:     bankID,
		Type:       hindsight.MemoryType(memory.Type),
		Content:    memory.Content,
		Vector:     toFloat32(memory.Vector),
		Confidence: memory.Importance, // Mapping Importance to Confidence
		Metadata:   memory.Metadata,
		CreatedAt:  memory.CreatedAt,
	}

	return s.sys.Retain(ctx, hMem)
}

// Search performs vector search using Hindsight Recall across all banks
func (s *MemoryStore) Search(ctx context.Context, vector []float64, topK int, minScore float64) ([]*MemoryWithScore, error) {
	// Get all available banks
	banks := s.sys.ListBanks()

	// Fallback: if ListBanks returns empty, get bank IDs from database metadata
	if len(banks) == 0 {
		bankIDs, err := s.getBankIDsFromDB()
		if err == nil && len(bankIDs) > 0 {
			// Convert bank IDs to Bank structs
			for _, bankID := range bankIDs {
				banks = append(banks, &hindsight.Bank{ID: bankID, Name: bankID})
			}
		}
	}

	// Collect results from all banks
	var allMemories []*MemoryWithScore
	for _, bank := range banks {
		req := &hindsight.RecallRequest{
			BankID:      bank.ID,
			QueryVector: toFloat32(vector),
			TopK:        topK,
			Strategy:    hindsight.DefaultStrategy(), // Uses default TEMPR fusion
		}

		results, err := s.sys.Recall(ctx, req)
		if err != nil {
			continue // Skip banks that fail
		}

		for _, res := range results {
			// Filter by minScore
			if float64(res.Score) < minScore {
				continue
			}

			mem := toStoreMemory(res.Memory)
			allMemories = append(allMemories, &MemoryWithScore{
				Memory: mem,
				Score:  float64(res.Score),
			})
		}
	}

	// Sort by score (highest first)
	for i := 0; i < len(allMemories)-1; i++ {
		for j := i + 1; j < len(allMemories); j++ {
			if allMemories[i].Score < allMemories[j].Score {
				allMemories[i], allMemories[j] = allMemories[j], allMemories[i]
			}
		}
	}

	// Return topK results
	if len(allMemories) > topK {
		allMemories = allMemories[:topK]
	}

	return allMemories, nil
}

// SearchBySession searches memories within a specific session (Bank)
func (s *MemoryStore) SearchBySession(ctx context.Context, sessionID string, vector []float64, topK int) ([]*MemoryWithScore, error) {
	req := &hindsight.RecallRequest{
		BankID:      sessionID,
		QueryVector: toFloat32(vector),
		TopK:        topK,
		Strategy:    hindsight.DefaultStrategy(),
	}

	results, err := s.sys.Recall(ctx, req)
	if err != nil {
		return nil, err
	}

	var memories []*MemoryWithScore
	for _, res := range results {
		mem := toStoreMemory(res.Memory)
		memories = append(memories, &MemoryWithScore{
			Memory: mem,
			Score:  float64(res.Score),
		})
	}

	return memories, nil
}

// Get retrieves a memory by ID
// Note: Hindsight doesn't have a direct Get(id). We use the underlying store.
func (s *MemoryStore) Get(ctx context.Context, id string) (*Memory, error) {
	// Hindsight does not expose a direct way to retrieve a raw memory by ID.
	// We return ErrMemoryNotFound as this operation is not supported in this backend implementation.
	return nil, ErrMemoryNotFound
}

// Update updates an existing memory
func (s *MemoryStore) Update(ctx context.Context, memory *Memory) error {
	// Hindsight Retain works as Upsert usually.
	return s.Store(ctx, memory)
}

// IncrementAccess increments the access count
func (s *MemoryStore) IncrementAccess(ctx context.Context, id string) error {
	// Not supported natively by Hindsight yet?
	// We could update metadata.
	mem, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	mem.AccessCount++
	mem.LastAccessed = time.Now()
	return s.Store(ctx, mem)
}

// GetByType retrieves memories by type
func (s *MemoryStore) GetByType(ctx context.Context, memoryType string, limit int) ([]*Memory, error) {
	// Hindsight doesn't support filter by type in Recall directly without vector.
	// But we can scan banks?
	// This is inefficient. For now, return empty or implement basic scan if needed.
	return nil, nil
}

// List lists all memories
func (s *MemoryStore) List(ctx context.Context, limit, offset int) ([]*Memory, int, error) {
	// List all banks and retrieve their memories
	banks := s.sys.ListBanks()
	var allMems []*Memory

	for _, bank := range banks {
		// Use Recall to get all memories for this bank
		req := &hindsight.RecallRequest{
			BankID:   bank.ID,
			TopK:     1000, // Get all memories
			Strategy: hindsight.DefaultStrategy(),
		}
		results, err := s.sys.Recall(ctx, req)
		if err == nil {
			for _, res := range results {
				allMems = append(allMems, toStoreMemory(res.Memory))
			}
		}
	}

	// Pagination
	total := len(allMems)
	if offset >= total {
		return []*Memory{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return allMems[offset:end], total, nil
}

// Delete removes a memory
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	return s.sys.Store().Delete(ctx, id)
}

// DeleteBySession removes all memories for a session
func (s *MemoryStore) DeleteBySession(ctx context.Context, sessionID string) error {
	return s.sys.DeleteBank(sessionID)
}

// Close closes the store
func (s *MemoryStore) Close() error {
	return s.sys.Close()
}

// Helpers

func toFloat32(v []float64) []float32 {
	res := make([]float32, len(v))
	for i, f := range v {
		res[i] = float32(f)
	}
	return res
}

func toFloat64(v []float32) []float64 {
	res := make([]float64, len(v))
	for i, f := range v {
		res[i] = float64(f)
	}
	return res
}

func toStoreMemory(hm *hindsight.Memory) *Memory {
	if hm == nil {
		return nil
	}
	return &Memory{
		ID:         hm.ID,
		SessionID:  hm.BankID,
		Type:       string(hm.Type),
		Content:    hm.Content,
		Vector:     toFloat64(hm.Vector),
		Importance: hm.Confidence,
		Metadata:   hm.Metadata,
		CreatedAt:  hm.CreatedAt,
		// AccessCount and UpdatedAt might not be tracked by Hindsight directly
	}
}

func observationToMemory(obs *hindsight.Observation, bankID string) *Memory {
	meta := map[string]interface{}{
		"reasoning": obs.Reasoning,
		"sources":   obs.SourceMemoryIDs,
	}
	return &Memory{
		ID:         obs.ID,
		SessionID:  bankID,
		Type:       string(obs.ObservationType),
		Content:    obs.Content,
		Importance: obs.Confidence,
		Metadata:   meta,
		CreatedAt:  obs.CreatedAt,
	}
}