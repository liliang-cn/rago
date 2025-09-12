package store

import (
	"testing"
	"time"
)

func TestDocument(t *testing.T) {
	doc := &Document{
		ID:         "test-id",
		Content:    "test content",
		Embedding:  []float32{0.1, 0.2, 0.3},
		Source:     "test source",
		Metadata:   map[string]interface{}{"key": "value"},
		ChunkIndex: 1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if doc.ID != "test-id" {
		t.Error("Document ID not set correctly")
	}

	if doc.Content != "test content" {
		t.Error("Document content not set correctly")
	}

	if len(doc.Embedding) != 3 {
		t.Error("Document embedding not set correctly")
	}

	if doc.Source != "test source" {
		t.Error("Document source not set correctly")
	}

	if doc.ChunkIndex != 1 {
		t.Error("Document chunk index not set correctly")
	}
}

func TestSearchQuery(t *testing.T) {
	query := SearchQuery{
		Embedding:       []float32{0.1, 0.2, 0.3},
		TopK:            5,
		Threshold:       0.7,
		Filter:          map[string]interface{}{"source": "test"},
		IncludeMetadata: true,
		IncludeVector:   false,
	}

	if len(query.Embedding) != 3 {
		t.Error("Query embedding not set correctly")
	}

	if query.TopK != 5 {
		t.Error("Query TopK not set correctly")
	}

	if query.Threshold != 0.7 {
		t.Error("Query threshold not set correctly")
	}

	if !query.IncludeMetadata {
		t.Error("Query IncludeMetadata not set correctly")
	}

	if query.IncludeVector {
		t.Error("Query IncludeVector should be false")
	}
}

func TestHybridSearchQuery(t *testing.T) {
	query := HybridSearchQuery{
		Embedding:       []float32{0.1, 0.2, 0.3},
		Keywords:        "test keywords",
		TopK:            10,
		Threshold:       0.8,
		Filter:          map[string]interface{}{"type": "document"},
		VectorWeight:    0.6,
		KeywordWeight:   0.4,
		IncludeMetadata: true,
		IncludeVector:   false,
	}

	if query.Keywords != "test keywords" {
		t.Error("Hybrid query keywords not set correctly")
	}

	if query.VectorWeight != 0.6 {
		t.Error("Hybrid query vector weight not set correctly")
	}

	if query.KeywordWeight != 0.4 {
		t.Error("Hybrid query keyword weight not set correctly")
	}
}

func TestSearchResult(t *testing.T) {
	doc := &ScoredDocument{
		Document: Document{
			ID:      "result-id",
			Content: "result content",
		},
		Score:           0.95,
		VectorScore:     0.9,
		KeywordScore:    0.8,
		HighlightedText: "highlighted result content",
	}

	result := &SearchResult{
		Documents:  []*ScoredDocument{doc},
		TotalCount: 1,
		QueryTime:  time.Millisecond * 100,
	}

	if len(result.Documents) != 1 {
		t.Error("Search result documents count incorrect")
	}

	if result.TotalCount != 1 {
		t.Error("Search result total count incorrect")
	}

	if result.Documents[0].Score != 0.95 {
		t.Error("Search result document score incorrect")
	}
}

func TestListOptions(t *testing.T) {
	opts := ListOptions{
		Offset: 10,
		Limit:  50,
		Filter: map[string]interface{}{"active": true},
		SortBy: "created_at",
		Order:  "desc",
	}

	if opts.Offset != 10 {
		t.Error("List options offset not set correctly")
	}

	if opts.Limit != 50 {
		t.Error("List options limit not set correctly")
	}

	if opts.SortBy != "created_at" {
		t.Error("List options sort by not set correctly")
	}

	if opts.Order != "desc" {
		t.Error("List options order not set correctly")
	}
}

func TestIndexConfig(t *testing.T) {
	config := IndexConfig{
		Dimensions: 768,
		Metric:     DistanceCosine,
		IndexType:  "hnsw",
		Parameters: map[string]interface{}{"m": 16, "ef_construction": 200},
	}

	if config.Dimensions != 768 {
		t.Error("Index config dimensions not set correctly")
	}

	if config.Metric != DistanceCosine {
		t.Error("Index config metric not set correctly")
	}

	if config.IndexType != "hnsw" {
		t.Error("Index config index type not set correctly")
	}
}

func TestIndexInfo(t *testing.T) {
	info := IndexInfo{
		Name: "test_index",
		Config: IndexConfig{
			Dimensions: 384,
			Metric:     DistanceEuclidean,
		},
		DocCount:  1000,
		CreatedAt: time.Now(),
	}

	if info.Name != "test_index" {
		t.Error("Index info name not set correctly")
	}

	if info.DocCount != 1000 {
		t.Error("Index info doc count not set correctly")
	}

	if info.Config.Dimensions != 384 {
		t.Error("Index info config dimensions not set correctly")
	}
}

func TestDistanceMetrics(t *testing.T) {
	metrics := []DistanceMetric{
		DistanceCosine,
		DistanceEuclidean,
		DistanceDotProduct,
	}

	expectedValues := []string{"cosine", "euclidean", "dot_product"}

	for i, metric := range metrics {
		if string(metric) != expectedValues[i] {
			t.Errorf("Distance metric %d: expected %s, got %s", i, expectedValues[i], string(metric))
		}
	}
}

func TestBasicStoreConfig(t *testing.T) {
	config := StoreConfig{
		Type: "sqvect",
		Parameters: map[string]interface{}{
			"db_path":    "./test.db",
			"dimensions": 768,
		},
	}

	if config.Type != "sqvect" {
		t.Error("Store config type not set correctly")
	}

	if config.Parameters["db_path"] != "./test.db" {
		t.Error("Store config db_path parameter not set correctly")
	}

	if config.Parameters["dimensions"] != 768 {
		t.Error("Store config dimensions parameter not set correctly")
	}
}
