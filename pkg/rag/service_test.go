package rag

import (
	"context"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

func TestRAGService_New(t *testing.T) {
	config := core.TestRAGConfig()
	
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	if service == nil {
		t.Error("Expected service to be created, got nil")
	}
}

func TestRAGService_IngestDocument(t *testing.T) {
	config := core.TestRAGConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	req := core.IngestRequest{
		DocumentID:  "test-doc-1",
		Content:     "This is test content for ingestion.",
		ContentType: "text/plain",
		Metadata:    map[string]interface{}{"source": "test"},
	}
	
	// This will return an error since we haven't implemented it yet
	_, err = service.IngestDocument(ctx, req)
	core.AssertError(t, err) // Expecting error until implementation
}

func TestRAGService_Search(t *testing.T) {
	config := core.TestRAGConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	req := core.SearchRequest{
		Query: "test query",
		Limit: 10,
	}
	
	// This will return an error since we haven't implemented it yet
	_, err = service.Search(ctx, req)
	core.AssertError(t, err) // Expecting error until implementation
}

func TestRAGService_HybridSearch(t *testing.T) {
	config := core.TestRAGConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	req := core.HybridSearchRequest{
		SearchRequest: core.SearchRequest{
			Query: "test query",
			Limit: 10,
		},
		VectorWeight:  0.7,
		KeywordWeight: 0.3,
	}
	
	// This will return an error since we haven't implemented it yet
	_, err = service.HybridSearch(ctx, req)
	core.AssertError(t, err) // Expecting error until implementation
}

func TestRAGService_ListDocuments(t *testing.T) {
	config := core.TestRAGConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	filter := core.DocumentFilter{
		ContentType: "text/plain",
		Limit:       10,
	}
	
	docs, err := service.ListDocuments(ctx, filter)
	core.AssertNoError(t, err)
	
	// Should be empty since no documents are ingested yet
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}
}

func TestRAGService_GetStats(t *testing.T) {
	config := core.TestRAGConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	
	// This will return an error since we haven't implemented it yet
	_, err = service.GetStats(ctx)
	core.AssertError(t, err) // Expecting error until implementation
}

func TestRAGService_DeleteDocument(t *testing.T) {
	config := core.TestRAGConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	
	// This will return an error since we haven't implemented it yet
	err = service.DeleteDocument(ctx, "non-existent-doc")
	core.AssertError(t, err) // Expecting error until implementation
}

func TestRAGService_Close(t *testing.T) {
	config := core.TestRAGConfig()
	service, err := NewService(config)
	core.AssertNoError(t, err)
	
	err = service.Close()
	core.AssertNoError(t, err)
}