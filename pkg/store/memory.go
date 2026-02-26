package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/sqvect/v2/pkg/hindsight"
	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrMemoryNotFound = errors.New("memory not found")
)

// Memory is a local internal structure, but we prefer using domain.Memory for interface methods.
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

// Store saves a new memory using Hindsight
func (s *MemoryStore) Store(ctx context.Context, memory *domain.Memory) error {
	bankID := memory.SessionID
	if bankID == "" {
		bankID = "default"
	}

	hMem := &hindsight.Memory{
		ID:         memory.ID,
		BankID:     bankID,
		Type:       hindsight.MemoryType(memory.Type),
		Content:    memory.Content,
		Vector:     toFloat32(memory.Vector),
		Confidence: memory.Importance,
		Metadata:   memory.Metadata,
		CreatedAt:  memory.CreatedAt,
	}

	return s.sys.Retain(ctx, hMem)
}

// Search performs vector search across all banks
func (s *MemoryStore) Search(ctx context.Context, vector []float64, topK int, minScore float64) ([]*domain.MemoryWithScore, error) {
	banks := s.sys.ListBanks()
	if len(banks) == 0 {
		bankIDs, _ := s.getBankIDsFromDB()
		for _, id := range bankIDs {
			banks = append(banks, &hindsight.Bank{ID: id, Name: id})
		}
	}

	var allMemories []*domain.MemoryWithScore
	for _, bank := range banks {
		req := &hindsight.RecallRequest{
			BankID:      bank.ID,
			QueryVector: toFloat32(vector),
			TopK:        topK,
			Strategy:    hindsight.DefaultStrategy(),
		}

		results, err := s.sys.Recall(ctx, req)
		if err != nil {
			continue
		}

		for _, res := range results {
			if float64(res.Score) < minScore {
				continue
			}

			allMemories = append(allMemories, &domain.MemoryWithScore{
				Memory: toDomainMemory(toInternalMemory(res.Memory)),
				Score:  float64(res.Score),
			})
		}
	}

	// Simple sort
	for i := 0; i < len(allMemories)-1; i++ {
		for j := i + 1; j < len(allMemories); j++ {
			if allMemories[i].Score < allMemories[j].Score {
				allMemories[i], allMemories[j] = allMemories[j], allMemories[i]
			}
		}
	}

	if len(allMemories) > topK {
		allMemories = allMemories[:topK]
	}

	return allMemories, nil
}

func (s *MemoryStore) SearchBySession(ctx context.Context, sessionID string, vector []float64, topK int) ([]*domain.MemoryWithScore, error) {
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

	var memories []*domain.MemoryWithScore
	for _, res := range results {
		memories = append(memories, &domain.MemoryWithScore{
			Memory: toDomainMemory(toInternalMemory(res.Memory)),
			Score:  float64(res.Score),
		})
	}

	return memories, nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (*domain.Memory, error) {
	// Fallback scan
	mems, _, err := s.List(ctx, 1000, 0)
	if err != nil {
		return nil, err
	}
	for _, m := range mems {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, ErrMemoryNotFound
}

func (s *MemoryStore) Update(ctx context.Context, memory *domain.Memory) error {
	return s.Store(ctx, memory)
}

func (s *MemoryStore) IncrementAccess(ctx context.Context, id string) error {
	m, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	m.AccessCount++
	m.LastAccessed = time.Now()
	return s.Update(ctx, m)
}

func (s *MemoryStore) GetByType(ctx context.Context, memoryType domain.MemoryType, limit int) ([]*domain.Memory, error) {
	all, _, _ := s.List(ctx, 1000, 0)
	var filtered []*domain.Memory
	for _, m := range all {
		if m.Type == memoryType {
			filtered = append(filtered, m)
		}
		if len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}

func (s *MemoryStore) List(ctx context.Context, limit, offset int) ([]*domain.Memory, int, error) {
	banks := s.sys.ListBanks()
	var all []*domain.Memory

	for _, bank := range banks {
		req := &hindsight.RecallRequest{
			BankID:   bank.ID,
			TopK:     1000,
			Strategy: hindsight.DefaultStrategy(),
		}
		results, err := s.sys.Recall(ctx, req)
		if err == nil {
			for _, res := range results {
				all = append(all, toDomainMemory(toInternalMemory(res.Memory)))
			}
		}
	}

	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	return s.sys.Store().Delete(ctx, id)
}

func (s *MemoryStore) DeleteBySession(ctx context.Context, sessionID string) error {
	return s.sys.DeleteBank(sessionID)
}

func (s *MemoryStore) InitSchema(ctx context.Context) error {
	return nil
}

func (s *MemoryStore) ConfigureBank(ctx context.Context, bankID string, config *domain.MemoryBankConfig) error {
	bank := hindsight.NewBank(bankID, bankID)
	bank.Description = config.Mission
	bank.Disposition.Skepticism = config.Skepticism
	bank.Disposition.Literalism = config.Literalism
	bank.Disposition.Empathy = config.Empathy
	return s.sys.CreateBank(ctx, bank)
}

func (s *MemoryStore) Reflect(ctx context.Context, bankID string) (string, error) {
	req := &hindsight.ContextRequest{
		BankID:   bankID,
		Strategy: hindsight.DefaultStrategy(),
		TopK:     10,
	}
	resp, err := s.sys.Reflect(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Context, nil
}

func (s *MemoryStore) AddMentalModel(ctx context.Context, model *domain.MentalModel) error {
	hMem := &hindsight.Memory{
		ID:         model.ID,
		BankID:     "global",
		Type:       hindsight.ObservationMemory,
		Content:    fmt.Sprintf("Mental Model: %s\n%s", model.Name, model.Content),
		Confidence: 1.0,
		Metadata: map[string]any{
			"name":        model.Name,
			"description": model.Description,
			"tags":        model.Tags,
		},
	}
	return s.sys.Retain(ctx, hMem)
}

func (s *MemoryStore) Close() error {
	return s.sys.Close()
}

// Helpers

func (s *MemoryStore) getBankIDsFromDB() ([]string, error) {
	db, err := sql.Open("sqlite3", s.dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT DISTINCT json_extract(metadata, '$.bank_id') FROM embeddings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err == nil && id.Valid {
			ids = append(ids, id.String)
		}
	}
	return ids, nil
}

func toFloat32(v []float64) []float32 {
	res := make([]float32, len(v))
	for i, f := range v {
		res[i] = float32(f)
	}
	return res
}

func toInternalMemory(hm *hindsight.Memory) *Memory {
	if hm == nil {
		return nil
	}
	vec := make([]float64, len(hm.Vector))
	for i, v := range hm.Vector {
		vec[i] = float64(v)
	}
	return &Memory{
		ID:         hm.ID,
		SessionID:  hm.BankID,
		Type:       string(hm.Type),
		Content:    hm.Content,
		Vector:     vec,
		Importance: hm.Confidence,
		Metadata:   hm.Metadata,
		CreatedAt:  hm.CreatedAt,
	}
}

func toDomainMemory(im *Memory) *domain.Memory {
	if im == nil {
		return nil
	}
	return &domain.Memory{
		ID:           im.ID,
		SessionID:    im.SessionID,
		Type:         domain.MemoryType(im.Type),
		Content:      im.Content,
		Vector:       im.Vector,
		Importance:   im.Importance,
		AccessCount:  im.AccessCount,
		LastAccessed: im.LastAccessed,
		Metadata:     im.Metadata,
		CreatedAt:    im.CreatedAt,
		UpdatedAt:    im.UpdatedAt,
	}
}
