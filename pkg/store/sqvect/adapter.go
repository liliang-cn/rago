package sqvect

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SqvectStore implements the VectorStore interface using SQLite with vector extensions
type SqvectStore struct {
	db         *sql.DB
	dbPath     string
	dimensions int
	tableName  string
}

// NewSqvectStore creates a new SQLite vector store
func NewSqvectStore(dbPath string, dimensions int) *SqvectStore {
	return &SqvectStore{
		dbPath:     dbPath,
		dimensions: dimensions,
		tableName:  "documents",
	}
}

// Initialize the store
func (s *SqvectStore) Initialize(ctx context.Context) error {
	var err error
	s.db, err = sql.Open("sqlite3", s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables if they don't exist
	if err := s.createTables(ctx); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// Close the store
func (s *SqvectStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Store a single document
func (s *SqvectStore) Store(ctx context.Context, doc *Document) error {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	doc.UpdatedAt = time.Now()

	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	embeddingJSON, err := json.Marshal(doc.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO documents 
		(id, content, embedding, source, metadata, chunk_index, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		doc.ID,
		doc.Content,
		embeddingJSON,
		doc.Source,
		string(metadataJSON),
		doc.ChunkIndex,
		doc.CreatedAt,
		doc.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}

	return nil
}

// StoreBatch stores multiple documents
func (s *SqvectStore) StoreBatch(ctx context.Context, docs []*Document) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO documents 
		(id, content, embedding, source, metadata, chunk_index, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, doc := range docs {
		if doc.ID == "" {
			doc.ID = uuid.New().String()
		}
		
		if doc.CreatedAt.IsZero() {
			doc.CreatedAt = time.Now()
		}
		doc.UpdatedAt = time.Now()

		metadataJSON, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		embeddingJSON, err := json.Marshal(doc.Embedding)
		if err != nil {
			return fmt.Errorf("failed to marshal embedding: %w", err)
		}

		_, err = stmt.ExecContext(ctx,
			doc.ID,
			doc.Content,
			embeddingJSON,
			doc.Source,
			string(metadataJSON),
			doc.ChunkIndex,
			doc.CreatedAt,
			doc.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to store document %s: %w", doc.ID, err)
		}
	}

	return tx.Commit()
}

// Search performs vector similarity search
func (s *SqvectStore) Search(ctx context.Context, query SearchQuery) (*SearchResult, error) {
	startTime := time.Now()

	embeddingJSON, err := json.Marshal(query.Embedding)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query embedding: %w", err)
	}

	// Build SQL query with cosine similarity
	sqlQuery := `
		SELECT 
			id, content, embedding, source, metadata, chunk_index, created_at, updated_at,
			(1 - vec_distance_cosine(embedding, ?)) as score
		FROM documents
		WHERE score >= ?
		ORDER BY score DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery, embeddingJSON, query.Threshold, query.TopK)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var documents []*ScoredDocument
	for rows.Next() {
		doc := &ScoredDocument{}
		var embeddingJSON, metadataJSON string

		err := rows.Scan(
			&doc.ID,
			&doc.Content,
			&embeddingJSON,
			&doc.Source,
			&metadataJSON,
			&doc.ChunkIndex,
			&doc.CreatedAt,
			&doc.UpdatedAt,
			&doc.Score,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Unmarshal embedding if requested
		if query.IncludeVector {
			if err := json.Unmarshal([]byte(embeddingJSON), &doc.Embedding); err != nil {
				return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
			}
		}

		// Unmarshal metadata if requested
		if query.IncludeMetadata && metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &doc.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		documents = append(documents, doc)
	}

	return &SearchResult{
		Documents:  documents,
		TotalCount: len(documents),
		QueryTime:  time.Since(startTime),
	}, nil
}

// HybridSearch performs combined vector and keyword search
func (s *SqvectStore) HybridSearch(ctx context.Context, query HybridSearchQuery) (*SearchResult, error) {
	// For now, just use vector search
	// In a full implementation, this would combine with FTS5 or similar
	return s.Search(ctx, SearchQuery{
		Embedding:       query.Embedding,
		TopK:            query.TopK,
		Threshold:       query.Threshold,
		Filter:          query.Filter,
		IncludeMetadata: query.IncludeMetadata,
		IncludeVector:   query.IncludeVector,
	})
}

// Delete removes a document by ID
func (s *SqvectStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM documents WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrDocumentNotFound{ID: id}
	}

	return nil
}

// DeleteBySource removes all documents from a source
func (s *SqvectStore) DeleteBySource(ctx context.Context, source string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM documents WHERE source = ?", source)
	if err != nil {
		return fmt.Errorf("failed to delete documents by source: %w", err)
	}
	return nil
}

// Get retrieves a document by ID
func (s *SqvectStore) Get(ctx context.Context, id string) (*Document, error) {
	doc := &Document{}
	var embeddingJSON, metadataJSON string

	err := s.db.QueryRowContext(ctx,
		"SELECT id, content, embedding, source, metadata, chunk_index, created_at, updated_at FROM documents WHERE id = ?",
		id,
	).Scan(
		&doc.ID,
		&doc.Content,
		&embeddingJSON,
		&doc.Source,
		&metadataJSON,
		&doc.ChunkIndex,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrDocumentNotFound{ID: id}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Unmarshal embedding
	if err := json.Unmarshal([]byte(embeddingJSON), &doc.Embedding); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
	}

	// Unmarshal metadata
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &doc.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return doc, nil
}

// List retrieves documents with pagination
func (s *SqvectStore) List(ctx context.Context, opts ListOptions) ([]*Document, error) {
	query := "SELECT id, content, embedding, source, metadata, chunk_index, created_at, updated_at FROM documents"
	
	// Add ordering
	if opts.SortBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", opts.SortBy)
		if opts.Order == "desc" {
			query += " DESC"
		} else {
			query += " ASC"
		}
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list query failed: %w", err)
	}
	defer rows.Close()

	var documents []*Document
	for rows.Next() {
		doc := &Document{}
		var embeddingJSON, metadataJSON string

		err := rows.Scan(
			&doc.ID,
			&doc.Content,
			&embeddingJSON,
			&doc.Source,
			&metadataJSON,
			&doc.ChunkIndex,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Unmarshal embedding
		if err := json.Unmarshal([]byte(embeddingJSON), &doc.Embedding); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
		}

		// Unmarshal metadata
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &doc.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

// Count returns the total number of documents
func (s *SqvectStore) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}
	return count, nil
}

// CreateIndex creates a new vector index
func (s *SqvectStore) CreateIndex(ctx context.Context, name string, config IndexConfig) error {
	// SQLite with vec extension handles indexing automatically
	// This is a no-op for now
	return nil
}

// DropIndex removes an index
func (s *SqvectStore) DropIndex(ctx context.Context, name string) error {
	// SQLite with vec extension handles indexing automatically
	// This is a no-op for now
	return nil
}

// ListIndexes returns information about indexes
func (s *SqvectStore) ListIndexes(ctx context.Context) ([]IndexInfo, error) {
	// Return default index info
	count, err := s.Count(ctx)
	if err != nil {
		return nil, err
	}

	return []IndexInfo{
		{
			Name: "default",
			Config: IndexConfig{
				Dimensions: s.dimensions,
				Metric:     DistanceCosine,
				IndexType:  "automatic",
			},
			DocCount:  count,
			CreatedAt: time.Now(),
		},
	}, nil
}

// createTables creates the necessary database tables
func (s *SqvectStore) createTables(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			embedding TEXT NOT NULL CHECK(json_valid(embedding)),
			source TEXT,
			metadata TEXT CHECK(metadata IS NULL OR json_valid(metadata)),
			chunk_index INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source);
		CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents(created_at);
		
		-- Create virtual table for vector similarity search if vec extension is loaded
		-- This would be created by the vec extension
	`)

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}