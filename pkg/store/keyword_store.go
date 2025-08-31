package store

import (
	"context"
	"os"

	"github.com/blevesearch/bleve/v2"
	"github.com/liliang-cn/rago/pkg/domain"
)

// KeywordStore provides an interface for full-text search on document chunks.
type KeywordStore struct {
	path  string
	index bleve.Index
}

// NewKeywordStore creates or opens a keyword store at the given path.
func NewKeywordStore(path string) (*KeywordStore, error) {
	index, err := openOrCreateBleveIndex(path)
	if err != nil {
		return nil, err
	}

	return &KeywordStore{
		path:  path,
		index: index,
	}, nil
}

// openOrCreateBleveIndex handles the logic of opening an existing index or creating a new one.
func openOrCreateBleveIndex(path string) (bleve.Index, error) {
	// Check if the index already exists.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create a new index with a standard mapping.
		mapping := bleve.NewIndexMapping()
		// TODO: Customize mapping if necessary (e.g., for different languages).
		index, err := bleve.New(path, mapping)
		if err != nil {
			return nil, err
		}
		return index, nil
	}

	// Open the existing index.
	index, err := bleve.Open(path)
	if err != nil {
		return nil, err
	}
	return index, nil
}

// Index adds or updates a chunk in the keyword store.
func (s *KeywordStore) Index(ctx context.Context, chunk domain.Chunk) error {
	return s.index.Index(chunk.ID, chunk)
}

// Search performs a full-text search against the indexed chunks.
func (s *KeywordStore) Search(ctx context.Context, query string, topK int) ([]domain.Chunk, error) {
	// Create a new match query for the given text.
	matchQuery := bleve.NewMatchQuery(query)

	// Create a search request.
	searchRequest := bleve.NewSearchRequest(matchQuery)
	searchRequest.Size = topK
	searchRequest.Fields = []string{"*"} // Request all stored fields

	// Execute the search.
	searchResult, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	// Process the results.
	var chunks []domain.Chunk
	for _, hit := range searchResult.Hits {
		// Safely extract fields with type checking
		var content, documentID string

		// Try different possible field names that Bleve might use
		if contentField, ok := hit.Fields["content"]; ok && contentField != nil {
			if contentStr, ok := contentField.(string); ok {
				content = contentStr
			}
		} else if contentField, ok := hit.Fields["Content"]; ok && contentField != nil {
			if contentStr, ok := contentField.(string); ok {
				content = contentStr
			}
		}

		if docIDField, ok := hit.Fields["document_id"]; ok && docIDField != nil {
			if docIDStr, ok := docIDField.(string); ok {
				documentID = docIDStr
			}
		} else if docIDField, ok := hit.Fields["DocumentID"]; ok && docIDField != nil {
			if docIDStr, ok := docIDField.(string); ok {
				documentID = docIDStr
			}
		}

		// The ID of the hit is the chunk ID we assigned during indexing.
		chunk := domain.Chunk{
			ID:         hit.ID,
			Score:      hit.Score,
			Content:    content,
			DocumentID: documentID,
			// Metadata might need more careful reconstruction if it's complex.
			// For now, we assume it's not needed for the search result context.
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// Delete removes all chunks associated with a given documentID from the store.
func (s *KeywordStore) Delete(ctx context.Context, documentID string) error {
	query := bleve.NewTermQuery(documentID)
	// Try both possible field names
	query.SetField("document_id")

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = 1000 // Adjust size as needed, or implement pagination
	searchResult, err := s.index.Search(searchRequest)
	if err != nil {
		return err
	}

	// If no results with lowercase field name, try uppercase
	if len(searchResult.Hits) == 0 {
		query.SetField("DocumentID")
		searchResult, err = s.index.Search(searchRequest)
		if err != nil {
			return err
		}
	}

	batch := s.index.NewBatch()
	for _, hit := range searchResult.Hits {
		batch.Delete(hit.ID)
	}

	return s.index.Batch(batch)
}

// Reset deletes the entire index and creates a new, empty one.
func (s *KeywordStore) Reset(ctx context.Context) error {
	if err := s.index.Close(); err != nil {
		// Log error but continue, as we are about to delete the directory anyway
	}

	if err := os.RemoveAll(s.path); err != nil {
		return err
	}

	newIndex, err := openOrCreateBleveIndex(s.path)
	if err != nil {
		return err
	}
	s.index = newIndex
	return nil
}

// Close closes the underlying Bleve index.
func (s *KeywordStore) Close() error {
	if s.index != nil {
		err := s.index.Close()
		s.index = nil // Prevent double close
		return err
	}
	return nil
}
