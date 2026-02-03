package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMemoryNotFound = errors.New("memory not found")
)

// Memory represents a single long-term memory
type Memory struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id,omitempty"`
	Type        string                 `json:"type"`
	Content     string                 `json:"content"`
	Vector      []float64              `json:"vector,omitempty"`
	Importance  float64                `json:"importance"`
	AccessCount int                    `json:"access_count"`
	LastAccessed time.Time             `json:"last_accessed"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// MemoryWithScore represents a memory with its similarity score
type MemoryWithScore struct {
	*Memory
	Score float64
}

// MemoryStore handles memory persistence
type MemoryStore struct {
	db *sql.DB
}

// NewMemoryStore creates a new memory store
func NewMemoryStore(db *sql.DB) (*MemoryStore, error) {
	store := &MemoryStore{db: db}
	if db != nil {
		if err := store.InitSchema(context.Background()); err != nil {
			return nil, err
		}
	}
	return store, nil
}

// InitSchema creates necessary tables for memories
func (s *MemoryStore) InitSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS agent_memory (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		memory_type TEXT NOT NULL,
		content TEXT NOT NULL,
		vector BLOB,
		importance REAL DEFAULT 0.5,
		access_count INTEGER DEFAULT 0,
		last_accessed DATETIME,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_agent_memory_session ON agent_memory(session_id);
	CREATE INDEX IF NOT EXISTS idx_agent_memory_type ON agent_memory(memory_type);
	CREATE INDEX IF NOT EXISTS idx_agent_memory_importance ON agent_memory(importance DESC);
	CREATE INDEX IF NOT EXISTS idx_agent_memory_last_accessed ON agent_memory(last_accessed DESC);

	-- FTS5 for keyword search
	CREATE VIRTUAL TABLE IF NOT EXISTS agent_memory_fts USING fts5(
		content,
		content='agent_memory',
		content_rowid='rowid'
	);

	-- Trigger to keep FTS in sync
	CREATE TRIGGER IF NOT EXISTS agent_memory_fts_insert AFTER INSERT ON agent_memory BEGIN
		INSERT INTO agent_memory_fts(rowid, content) VALUES (new.rowid, new.content);
	END;

	CREATE TRIGGER IF NOT EXISTS agent_memory_fts_delete AFTER DELETE ON agent_memory BEGIN
		DELETE FROM agent_memory_fts WHERE rowid = old.rowid;
	END;

	CREATE TRIGGER IF NOT EXISTS agent_memory_fts_update AFTER UPDATE OF content ON agent_memory BEGIN
		UPDATE agent_memory_fts SET content = new.content WHERE rowid = new.rowid;
	END;
	`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// Store saves a new memory
func (s *MemoryStore) Store(ctx context.Context, memory *Memory) error {
	if memory.ID == "" {
		memory.ID = uuid.New().String()
	}

	now := time.Now()
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = now
	}
	memory.UpdatedAt = now

	// Serialize vector
	var vectorData []byte
	if len(memory.Vector) > 0 {
		vectorData = serializeVector(memory.Vector)
	}

	// Serialize metadata
	var metadataJSON []byte
	if memory.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(memory.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
	INSERT INTO agent_memory (
		id, session_id, memory_type, content, vector, importance,
		access_count, last_accessed, metadata, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		session_id = excluded.session_id,
		memory_type = excluded.memory_type,
		content = excluded.content,
		vector = excluded.vector,
		importance = excluded.importance,
		access_count = excluded.access_count,
		last_accessed = excluded.last_accessed,
		metadata = excluded.metadata,
		updated_at = excluded.updated_at
	`

	_, err := s.db.ExecContext(ctx, query,
		memory.ID,
		memory.SessionID,
		memory.Type,
		memory.Content,
		vectorData,
		memory.Importance,
		memory.AccessCount,
		formatTime(memory.LastAccessed),
		metadataJSON,
		formatTime(memory.CreatedAt),
		formatTime(memory.UpdatedAt),
	)

	return err
}

// Search performs vector search for related memories
func (s *MemoryStore) Search(ctx context.Context, vector []float64, topK int, minScore float64) ([]*MemoryWithScore, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("empty query vector")
	}

	if topK <= 0 {
		topK = 5
	}

	// Get all memories with vectors
	query := `
	SELECT id, session_id, memory_type, content, vector, importance,
	       access_count, last_accessed, metadata, created_at, updated_at
	FROM agent_memory
	WHERE vector IS NOT NULL
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		mem, err := s.scanMemory(rows)
		if err != nil {
			continue
		}
		memories = append(memories, mem)
	}

	// Calculate similarity scores
	results := make([]*MemoryWithScore, 0)
	for _, mem := range memories {
		score := cosineSimilarity(vector, mem.Vector)
		if score >= minScore {
			results = append(results, &MemoryWithScore{
				Memory: mem,
				Score:  score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return topK
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// SearchBySession searches memories within a specific session
func (s *MemoryStore) SearchBySession(ctx context.Context, sessionID string, vector []float64, topK int) ([]*MemoryWithScore, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("empty query vector")
	}

	if topK <= 0 {
		topK = 5
	}

	query := `
	SELECT id, session_id, memory_type, content, vector, importance,
	       access_count, last_accessed, metadata, created_at, updated_at
	FROM agent_memory
	WHERE session_id = ? AND vector IS NOT NULL
	`

	rows, err := s.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		mem, err := s.scanMemory(rows)
		if err != nil {
			continue
		}
		memories = append(memories, mem)
	}

	// Calculate similarity scores
	results := make([]*MemoryWithScore, 0)
	for _, mem := range memories {
		score := cosineSimilarity(vector, mem.Vector)
		results = append(results, &MemoryWithScore{
			Memory: mem,
			Score:  score,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return topK
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// Get retrieves a memory by ID
func (s *MemoryStore) Get(ctx context.Context, id string) (*Memory, error) {
	query := `
	SELECT id, session_id, memory_type, content, vector, importance,
	       access_count, last_accessed, metadata, created_at, updated_at
	FROM agent_memory
	WHERE id = ?
	`

	var mem Memory
	var vectorData []byte
	var metadataJSON sql.NullString
	var lastAccessed sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&mem.ID,
		&mem.SessionID,
		&mem.Type,
		&mem.Content,
		&vectorData,
		&mem.Importance,
		&mem.AccessCount,
		&lastAccessed,
		&metadataJSON,
		&mem.CreatedAt,
		&mem.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemoryNotFound
		}
		return nil, err
	}

	if lastAccessed.Valid {
		mem.LastAccessed, _ = time.Parse(time.RFC3339, lastAccessed.String)
	}

	// Deserialize vector
	if len(vectorData) > 0 {
		mem.Vector = deserializeVector(vectorData)
	}

	// Deserialize metadata
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &mem.Metadata)
	}

	if mem.Metadata == nil {
		mem.Metadata = make(map[string]interface{})
	}

	return &mem, nil
}

// Update updates an existing memory
func (s *MemoryStore) Update(ctx context.Context, memory *Memory) error {
	memory.UpdatedAt = time.Now()

	// Serialize vector
	var vectorData []byte
	if len(memory.Vector) > 0 {
		vectorData = serializeVector(memory.Vector)
	}

	// Serialize metadata
	var metadataJSON []byte
	if memory.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(memory.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
	UPDATE agent_memory
	SET session_id = ?, memory_type = ?, content = ?, vector = ?, importance = ?,
	    access_count = ?, last_accessed = ?, metadata = ?, updated_at = ?
	WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		memory.SessionID,
		memory.Type,
		memory.Content,
		vectorData,
		memory.Importance,
		memory.AccessCount,
		formatTime(memory.LastAccessed),
		metadataJSON,
		formatTime(memory.UpdatedAt),
		memory.ID,
	)

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrMemoryNotFound
	}

	return nil
}

// IncrementAccess increments the access count and updates last_accessed
func (s *MemoryStore) IncrementAccess(ctx context.Context, id string) error {
	query := `
	UPDATE agent_memory
	SET access_count = access_count + 1,
	    last_accessed = ?
	WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, time.Now().Format(time.RFC3339), id)
	return err
}

// GetByType retrieves memories by type
func (s *MemoryStore) GetByType(ctx context.Context, memoryType string, limit int) ([]*Memory, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
	SELECT id, session_id, memory_type, content, vector, importance,
	       access_count, last_accessed, metadata, created_at, updated_at
	FROM agent_memory
	WHERE memory_type = ?
	ORDER BY importance DESC, last_accessed DESC
	LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, memoryType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		mem, err := s.scanMemory(rows)
		if err != nil {
			continue
		}
		memories = append(memories, mem)
	}

	return memories, nil
}

// List lists all memories with pagination
func (s *MemoryStore) List(ctx context.Context, limit, offset int) ([]*Memory, int, error) {
	if limit <= 0 {
		limit = 10
	}

	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM agent_memory").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get memories
	query := `
	SELECT id, session_id, memory_type, content, vector, importance,
	       access_count, last_accessed, metadata, created_at, updated_at
	FROM agent_memory
	ORDER BY created_at DESC
	LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		mem, err := s.scanMemory(rows)
		if err != nil {
			continue
		}
		memories = append(memories, mem)
	}

	return memories, total, nil
}

// Delete removes a memory
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM agent_memory WHERE id = ?", id)
	return err
}

// DeleteBySession removes all memories for a session
func (s *MemoryStore) DeleteBySession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM agent_memory WHERE session_id = ?", sessionID)
	return err
}

// scanMemory scans a row into a Memory struct
func (s *MemoryStore) scanMemory(rows *sql.Rows) (*Memory, error) {
	var mem Memory
	var vectorData []byte
	var metadataJSON sql.NullString
	var lastAccessed sql.NullString

	err := rows.Scan(
		&mem.ID,
		&mem.SessionID,
		&mem.Type,
		&mem.Content,
		&vectorData,
		&mem.Importance,
		&mem.AccessCount,
		&lastAccessed,
		&metadataJSON,
		&mem.CreatedAt,
		&mem.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if lastAccessed.Valid {
		mem.LastAccessed, _ = time.Parse(time.RFC3339, lastAccessed.String)
	}

	// Deserialize vector
	if len(vectorData) > 0 {
		mem.Vector = deserializeVector(vectorData)
	}

	// Deserialize metadata
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &mem.Metadata)
	}

	if mem.Metadata == nil {
		mem.Metadata = make(map[string]interface{})
	}

	return &mem, nil
}

// Helper functions

// formatTime formats a time for SQLite, returning empty string for zero time
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// serializeVector converts []float64 to []byte
func serializeVector(v []float64) []byte {
	data := make([]byte, len(v)*8)
	for i, f := range v {
		bits := math.Float64bits(f)
		data[i*8] = byte(bits)
		data[i*8+1] = byte(bits >> 8)
		data[i*8+2] = byte(bits >> 16)
		data[i*8+3] = byte(bits >> 24)
		data[i*8+4] = byte(bits >> 32)
		data[i*8+5] = byte(bits >> 40)
		data[i*8+6] = byte(bits >> 48)
		data[i*8+7] = byte(bits >> 56)
	}
	return data
}

// deserializeVector converts []byte to []float64
func deserializeVector(data []byte) []float64 {
	v := make([]float64, len(data)/8)
	for i := range v {
		bits := uint64(data[i*8]) |
			uint64(data[i*8+1])<<8 |
			uint64(data[i*8+2])<<16 |
			uint64(data[i*8+3])<<24 |
			uint64(data[i*8+4])<<32 |
			uint64(data[i*8+5])<<40 |
			uint64(data[i*8+6])<<48 |
			uint64(data[i*8+7])<<56
		v[i] = math.Float64frombits(bits)
	}
	return v
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float64) float64 {
	// Simple Newton's method for square root
	if x == 0 {
		return 0
	}
	z := 1.0
	for i := 0; i < 20; i++ {
		z = (z + x/z) / 2
	}
	return z
}
