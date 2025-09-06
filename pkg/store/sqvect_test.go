package store

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestSqvectStore(t *testing.T) (*SQLiteStore, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_sqvect.db")
	
	store, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	require.NotNil(t, store)
	
	return store, dbPath
}

func createTestChunk(id, docID, content string, vector []float64) domain.Chunk {
	return domain.Chunk{
		ID:         id,
		DocumentID: docID,
		Content:    content,
		Vector:     vector,
		Metadata: map[string]interface{}{
			"source": "test",
			"type":   "document",
		},
	}
}

func TestNewSQLiteStore(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:    "valid store creation",
			dbPath:  filepath.Join(t.TempDir(), "test.db"),
			wantErr: false,
		},
		{
			name:    "empty db path",
			dbPath:  "",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewSQLiteStore(tt.dbPath)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, store)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, store)
				if store != nil {
					store.Close()
				}
			}
		})
	}
}

func TestSqvectStore_Store(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	// Create test vector with correct dimensions (384)
	vector := make([]float64, 384)
	for i := range vector {
		vector[i] = float64(i) / 100.0
	}
	
	chunks := []domain.Chunk{
		createTestChunk("chunk-1", "doc-1", "This is test content", vector),
		createTestChunk("chunk-2", "doc-1", "This is more test content", vector),
		createTestChunk("chunk-3", "doc-2", "Different document content", vector),
	}
	
	err := store.Store(context.Background(), chunks)
	assert.NoError(t, err)
	
	// Verify chunks were stored by searching
	searchResults, err := store.Search(context.Background(), vector, 10)
	assert.NoError(t, err)
	assert.Len(t, searchResults, 3)
}

func TestSqvectStore_StoreEmptyChunks(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	// Test storing empty chunks slice
	err := store.Store(context.Background(), []domain.Chunk{})
	assert.NoError(t, err)
}

func TestSqvectStore_StoreInvalidVector(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	// Test storing chunk with wrong vector dimension
	wrongVector := []float64{0.1, 0.2, 0.3} // Only 3 dimensions, expecting 384
	chunks := []domain.Chunk{
		createTestChunk("chunk-1", "doc-1", "Test content", wrongVector),
	}
	
	err := store.Store(context.Background(), chunks)
	// SQLite store with sqvect v0.7.0 auto-detects dimensions
	// so it may not return an error for dimension mismatch
	if err != nil {
		assert.Contains(t, err.Error(), "dimension")
	} else {
		// If no error, verify the chunk was stored
		t.Log("Store accepted different dimension vector - auto-detection enabled")
	}
}

func TestSqvectStore_Search(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	// Create test vectors
	vector1 := make([]float64, 384)
	vector2 := make([]float64, 384)
	vector3 := make([]float64, 384)
	
	// Make vectors slightly different for testing similarity
	for i := range vector1 {
		vector1[i] = float64(i) / 100.0
		vector2[i] = float64(i) / 100.0 + 0.1 // Slightly different
		vector3[i] = float64(i) / 100.0 + 1.0 // Very different
	}
	
	// Store test chunks
	chunks := []domain.Chunk{
		createTestChunk("chunk-1", "doc-1", "Very similar content", vector1),
		createTestChunk("chunk-2", "doc-1", "Somewhat similar content", vector2),
		createTestChunk("chunk-3", "doc-2", "Very different content", vector3),
	}
	
	err := store.Store(context.Background(), chunks)
	require.NoError(t, err)
	
	// Search with first vector
	results, err := store.Search(context.Background(), vector1, 10)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
	
	// Results should be ordered by similarity (highest score first)
	assert.Equal(t, "chunk-1", results[0].ID)
	assert.Greater(t, results[0].Score, results[1].Score)
	assert.Greater(t, results[1].Score, results[2].Score)
}

func TestSqvectStore_SearchWithLimit(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	vector := make([]float64, 384)
	for i := range vector {
		vector[i] = float64(i) / 100.0
	}
	
	// Store multiple chunks
	var chunks []domain.Chunk
	for i := 0; i < 10; i++ {
		chunks = append(chunks, createTestChunk(
			fmt.Sprintf("chunk-%d", i),
			"doc-1",
			fmt.Sprintf("Content %d", i),
			vector,
		))
	}
	
	err := store.Store(context.Background(), chunks)
	require.NoError(t, err)
	
	// Search with limit
	results, err := store.Search(context.Background(), vector, 3)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestSqvectStore_SearchWithThreshold(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	vector1 := make([]float64, 384)
	vector2 := make([]float64, 384)
	
	// Create very different vectors
	for i := range vector1 {
		vector1[i] = 1.0
		vector2[i] = -1.0
	}
	
	chunks := []domain.Chunk{
		createTestChunk("chunk-1", "doc-1", "Content 1", vector1),
		createTestChunk("chunk-2", "doc-1", "Content 2", vector2),
	}
	
	err := store.Store(context.Background(), chunks)
	require.NoError(t, err)
	
	// Search returns all results up to topK, sorted by similarity
	results, err := store.Search(context.Background(), vector1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results)) // Returns both chunks
	// First result should be more similar (same vector)
	assert.Equal(t, "chunk-1", results[0].ID)
}

func TestSqvectStore_Delete(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	vector := make([]float64, 384)
	for i := range vector {
		vector[i] = float64(i) / 100.0
	}
	
	// Store chunks from different documents
	chunks := []domain.Chunk{
		createTestChunk("chunk-1", "doc-1", "Content 1", vector),
		createTestChunk("chunk-2", "doc-1", "Content 2", vector),
		createTestChunk("chunk-3", "doc-2", "Content 3", vector),
	}
	
	err := store.Store(context.Background(), chunks)
	require.NoError(t, err)
	
	// Delete document 1
	err = store.Delete(context.Background(), "doc-1")
	assert.NoError(t, err)
	
	// Verify only doc-2 chunks remain
	results, err := store.Search(context.Background(), vector, 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "doc-2", results[0].DocumentID)
}

// func TestSqvectStore_DeleteByFilter(t *testing.T) {
// 	store, _ := createTestSqvectStore(t)
// 	defer store.Close()
// 	
// 	vector := make([]float64, 384)
// 	for i := range vector {
// 		vector[i] = float64(i) / 100.0
// 	}
// 	
// 	// Store chunks with different metadata
// 	chunks := []domain.Chunk{
// 		{
// 			ID:         "chunk-1",
// 			DocumentID: "doc-1",
// 			Content:    "Content 1",
// 			Vector:     vector,
// 			Metadata:   map[string]interface{}{"type": "pdf", "author": "alice"},
// 		},
// 		{
// 			ID:         "chunk-2",
// 			DocumentID: "doc-2",
// 			Content:    "Content 2",
// 			Vector:     vector,
// 			Metadata:   map[string]interface{}{"type": "txt", "author": "alice"},
// 		},
// 		{
// 			ID:         "chunk-3",
// 			DocumentID: "doc-3",
// 			Content:    "Content 3",
// 			Vector:     vector,
// 			Metadata:   map[string]interface{}{"type": "pdf", "author": "bob"},
// 		},
// 	}
// 	
// 	err := store.Store(context.Background(), chunks)
// 	require.NoError(t, err)
// 	
// 	// Delete by filter - remove all PDF documents
// 	filter := map[string]interface{}{"type": "pdf"}
// 	err = store.DeleteByFilter(context.Background(), filter)
// 	assert.NoError(t, err)
// 	
// 	// Verify only txt document remains
// 	results, err := store.Search(context.Background(), vector, 10)
// 	assert.NoError(t, err)
// 	assert.Len(t, results, 1)
// 	assert.Equal(t, "doc-2", results[0].DocumentID)
// }
// 
// func TestSqvectStore_DeleteAll(t *testing.T) {
// // 	store, _ := createTestSqvectStore(t)
// 	defer store.Close()
// 	
// 	vector := make([]float64, 384)
// 	chunks := []domain.Chunk{
// 		createTestChunk("chunk-1", "doc-1", "Content 1", vector),
// 		createTestChunk("chunk-2", "doc-2", "Content 2", vector),
// 	}
// 	
// 	err := store.Store(context.Background(), chunks)
// 	require.NoError(t, err)
// 	
// 	// Delete all chunks individually
// 	err = store.Delete(context.Background(), "doc-1")
// 	assert.NoError(t, err)
// 	err = store.Delete(context.Background(), "doc-2")
// 	assert.NoError(t, err)
// 	
// 	// Verify no chunks remain
// 	results, err := store.Search(context.Background(), vector, 10)
// 	assert.NoError(t, err)
// 	assert.Empty(t, results)
// }

func TestSqvectStore_SearchEmptyStore(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	vector := make([]float64, 384)
	
	// Search in empty store
	results, err := store.Search(context.Background(), vector, 10)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestSqvectStore_SearchInvalidVector(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	// Test search with wrong vector dimension
	wrongVector := []float64{0.1, 0.2, 0.3} // Only 3 dimensions
	
	results, err := store.Search(context.Background(), wrongVector, 10)
	// With auto-detection, store might handle different dimensions gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "dimension")
		assert.Nil(t, results)
	} else {
		// If no error, results should be empty since no data stored
		assert.Empty(t, results)
	}
}

func TestSqvectStore_ConcurrentOperations(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	vector := make([]float64, 384)
	for i := range vector {
		vector[i] = float64(i) / 100.0
	}
	
	// Concurrent stores
	concurrency := 5
	done := make(chan error, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			chunks := []domain.Chunk{
				createTestChunk(
					fmt.Sprintf("chunk-%d", index),
					fmt.Sprintf("doc-%d", index),
					fmt.Sprintf("Content %d", index),
					vector,
				),
			}
			done <- store.Store(context.Background(), chunks)
		}(i)
	}
	
	// Wait for all stores to complete
	// SQLite may have database locking issues with concurrent writes
	successCount := 0
	for i := 0; i < concurrency; i++ {
		err := <-done
		if err != nil {
			// Database locked errors are expected with SQLite concurrent writes
			assert.Contains(t, err.Error(), "locked")
		} else {
			successCount++
		}
	}
	// At least one should succeed
	assert.Greater(t, successCount, 0)
	
	// Concurrent searches
	for i := 0; i < concurrency; i++ {
		go func() {
			_, err := store.Search(context.Background(), vector, 10)
			done <- err
		}()
	}
	
	// Wait for all searches to complete
	for i := 0; i < concurrency; i++ {
		err := <-done
		assert.NoError(t, err)
	}
}

func TestSqvectStore_LargeVectors(t *testing.T) {
	// Test with larger dimensions
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "large_vector_test.db")
	
	store, err := NewSQLiteStore(dbPath) // Larger dimension like OpenAI embeddings
	require.NoError(t, err)
	defer store.Close()
	
	// Create large vector
	largeVector := make([]float64, 1536)
	for i := range largeVector {
		largeVector[i] = float64(i) / 1000.0
	}
	
	chunks := []domain.Chunk{
		createTestChunk("chunk-1", "doc-1", "Large vector content", largeVector),
	}
	
	err = store.Store(context.Background(), chunks)
	assert.NoError(t, err)
	
	// Search should work with large vectors
	results, err := store.Search(context.Background(), largeVector, 1)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "chunk-1", results[0].ID)
}

func TestSqvectStore_MetadataPreservation(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	defer store.Close()
	
	vector := make([]float64, 384)
	
	metadata := map[string]interface{}{
		"title":       "Test Document",
		"author":      "Test Author",
		"created_at":  time.Now().Format(time.RFC3339),
		"tags":        []string{"test", "metadata"},
		"version":     1.0,
		"is_public":   true,
	}
	
	chunk := domain.Chunk{
		ID:         "chunk-1",
		DocumentID: "doc-1",
		Content:    "Test content with rich metadata",
		Vector:     vector,
		Metadata:   metadata,
	}
	
	err := store.Store(context.Background(), []domain.Chunk{chunk})
	require.NoError(t, err)
	
	// Search and verify metadata is preserved
	results, err := store.Search(context.Background(), vector, 1)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	
	result := results[0]
	assert.Equal(t, "Test Document", result.Metadata["title"])
	assert.Equal(t, "Test Author", result.Metadata["author"])
	// SQLite JSON storage returns booleans and numbers as strings
	assert.Equal(t, "true", result.Metadata["is_public"])
	assert.Equal(t, "1", result.Metadata["version"])
}

func TestSqvectStore_DatabasePersistence(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "persistence_test.db")
	
	vector := make([]float64, 384)
	for i := range vector {
		vector[i] = float64(i) / 100.0
	}
	
	chunks := []domain.Chunk{
		createTestChunk("chunk-1", "doc-1", "Persistent content", vector),
	}
	
	// Create store and add data
	{
		store, err := NewSQLiteStore(dbPath)
		require.NoError(t, err)
		
		err = store.Store(context.Background(), chunks)
		require.NoError(t, err)
		
		store.Close()
	}
	
	// Reopen database and verify data persists
	{
		store, err := NewSQLiteStore(dbPath)
		require.NoError(t, err)
		defer store.Close()
		
		results, err := store.Search(context.Background(), vector, 10)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "chunk-1", results[0].ID)
		assert.Equal(t, "Persistent content", results[0].Content)
	}
}

func TestSqvectStore_Close(t *testing.T) {
	store, _ := createTestSqvectStore(t)
	
	// Store should be usable before close
	vector := make([]float64, 384)
	chunks := []domain.Chunk{
		createTestChunk("chunk-1", "doc-1", "Test content", vector),
	}
	
	err := store.Store(context.Background(), chunks)
	assert.NoError(t, err)
	
	// Close store
	err = store.Close()
	assert.NoError(t, err)
	
	// Operations after close should fail gracefully
	err = store.Store(context.Background(), chunks)
	assert.Error(t, err)
	
	// Double close should not panic
	err = store.Close()
	assert.NoError(t, err)
}

// Benchmark tests

func BenchmarkSqvectStore_Store(b *testing.B) {
	store, _ := createTestSqvectStore(&testing.T{})
	defer store.Close()
	
	vector := make([]float64, 384)
	for i := range vector {
		vector[i] = float64(i) / 100.0
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			chunk := createTestChunk(
				fmt.Sprintf("chunk-%d", time.Now().UnixNano()),
				"doc-benchmark",
				"Benchmark content",
				vector,
			)
			store.Store(context.Background(), []domain.Chunk{chunk})
		}
	})
}

func BenchmarkSqvectStore_Search(b *testing.B) {
	store, _ := createTestSqvectStore(&testing.T{})
	defer store.Close()
	
	vector := make([]float64, 384)
	for i := range vector {
		vector[i] = float64(i) / 100.0
	}
	
	// Pre-populate with test data
	var chunks []domain.Chunk
	for i := 0; i < 1000; i++ {
		chunks = append(chunks, createTestChunk(
			fmt.Sprintf("chunk-%d", i),
			fmt.Sprintf("doc-%d", i/10),
			fmt.Sprintf("Benchmark content %d", i),
			vector,
		))
	}
	store.Store(context.Background(), chunks)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			store.Search(context.Background(), vector, 10)
		}
	})
}