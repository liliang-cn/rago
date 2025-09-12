package store

import (
	"context"

	"github.com/liliang-cn/rago/v2/pkg/store/sqvect"
)

// SqvectWrapper wraps the sqvect.SqvectStore to implement the VectorStore interface
type SqvectWrapper struct {
	*sqvect.SqvectStore
}

// NewSqvectWrapper creates a new wrapper around sqvect store
func NewSqvectWrapper(dbPath string, dimensions int) *SqvectWrapper {
	return &SqvectWrapper{
		SqvectStore: sqvect.NewSqvectStore(dbPath, dimensions),
	}
}

// Store wraps the underlying Store method
func (w *SqvectWrapper) Store(ctx context.Context, doc *Document) error {
	sqvDoc := &sqvect.Document{
		ID:         doc.ID,
		Content:    doc.Content,
		Embedding:  doc.Embedding,
		Source:     doc.Source,
		Metadata:   doc.Metadata,
		ChunkIndex: doc.ChunkIndex,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
	}
	return w.SqvectStore.Store(ctx, sqvDoc)
}

// StoreBatch wraps the underlying StoreBatch method
func (w *SqvectWrapper) StoreBatch(ctx context.Context, docs []*Document) error {
	sqvDocs := make([]*sqvect.Document, len(docs))
	for i, doc := range docs {
		sqvDocs[i] = &sqvect.Document{
			ID:         doc.ID,
			Content:    doc.Content,
			Embedding:  doc.Embedding,
			Source:     doc.Source,
			Metadata:   doc.Metadata,
			ChunkIndex: doc.ChunkIndex,
			CreatedAt:  doc.CreatedAt,
			UpdatedAt:  doc.UpdatedAt,
		}
	}
	return w.SqvectStore.StoreBatch(ctx, sqvDocs)
}

// Search wraps the underlying Search method
func (w *SqvectWrapper) Search(ctx context.Context, query SearchQuery) (*SearchResult, error) {
	sqvQuery := sqvect.SearchQuery{
		Embedding:       query.Embedding,
		TopK:            query.TopK,
		Threshold:       query.Threshold,
		Filter:          query.Filter,
		IncludeMetadata: query.IncludeMetadata,
		IncludeVector:   query.IncludeVector,
	}

	sqvResult, err := w.SqvectStore.Search(ctx, sqvQuery)
	if err != nil {
		return nil, err
	}

	// Convert result
	result := &SearchResult{
		TotalCount: sqvResult.TotalCount,
		QueryTime:  sqvResult.QueryTime,
		Documents:  make([]*ScoredDocument, len(sqvResult.Documents)),
	}

	for i, doc := range sqvResult.Documents {
		result.Documents[i] = &ScoredDocument{
			Document: Document{
				ID:         doc.ID,
				Content:    doc.Content,
				Embedding:  doc.Embedding,
				Source:     doc.Source,
				Metadata:   doc.Metadata,
				ChunkIndex: doc.ChunkIndex,
				CreatedAt:  doc.CreatedAt,
				UpdatedAt:  doc.UpdatedAt,
			},
			Score:           doc.Score,
			VectorScore:     doc.VectorScore,
			KeywordScore:    doc.KeywordScore,
			HighlightedText: doc.HighlightedText,
		}
	}

	return result, nil
}

// HybridSearch wraps the underlying HybridSearch method
func (w *SqvectWrapper) HybridSearch(ctx context.Context, query HybridSearchQuery) (*SearchResult, error) {
	sqvQuery := sqvect.HybridSearchQuery{
		Embedding:       query.Embedding,
		Keywords:        query.Keywords,
		TopK:            query.TopK,
		Threshold:       query.Threshold,
		Filter:          query.Filter,
		VectorWeight:    query.VectorWeight,
		KeywordWeight:   query.KeywordWeight,
		IncludeMetadata: query.IncludeMetadata,
		IncludeVector:   query.IncludeVector,
	}

	sqvResult, err := w.SqvectStore.HybridSearch(ctx, sqvQuery)
	if err != nil {
		return nil, err
	}

	// Convert result
	result := &SearchResult{
		TotalCount: sqvResult.TotalCount,
		QueryTime:  sqvResult.QueryTime,
		Documents:  make([]*ScoredDocument, len(sqvResult.Documents)),
	}

	for i, doc := range sqvResult.Documents {
		result.Documents[i] = &ScoredDocument{
			Document: Document{
				ID:         doc.ID,
				Content:    doc.Content,
				Embedding:  doc.Embedding,
				Source:     doc.Source,
				Metadata:   doc.Metadata,
				ChunkIndex: doc.ChunkIndex,
				CreatedAt:  doc.CreatedAt,
				UpdatedAt:  doc.UpdatedAt,
			},
			Score:           doc.Score,
			VectorScore:     doc.VectorScore,
			KeywordScore:    doc.KeywordScore,
			HighlightedText: doc.HighlightedText,
		}
	}

	return result, nil
}

// Get wraps the underlying Get method
func (w *SqvectWrapper) Get(ctx context.Context, id string) (*Document, error) {
	sqvDoc, err := w.SqvectStore.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return &Document{
		ID:         sqvDoc.ID,
		Content:    sqvDoc.Content,
		Embedding:  sqvDoc.Embedding,
		Source:     sqvDoc.Source,
		Metadata:   sqvDoc.Metadata,
		ChunkIndex: sqvDoc.ChunkIndex,
		CreatedAt:  sqvDoc.CreatedAt,
		UpdatedAt:  sqvDoc.UpdatedAt,
	}, nil
}

// List wraps the underlying List method
func (w *SqvectWrapper) List(ctx context.Context, opts ListOptions) ([]*Document, error) {
	sqvOpts := sqvect.ListOptions{
		Offset: opts.Offset,
		Limit:  opts.Limit,
		Filter: opts.Filter,
		SortBy: opts.SortBy,
		Order:  opts.Order,
	}

	sqvDocs, err := w.SqvectStore.List(ctx, sqvOpts)
	if err != nil {
		return nil, err
	}

	docs := make([]*Document, len(sqvDocs))
	for i, sqvDoc := range sqvDocs {
		docs[i] = &Document{
			ID:         sqvDoc.ID,
			Content:    sqvDoc.Content,
			Embedding:  sqvDoc.Embedding,
			Source:     sqvDoc.Source,
			Metadata:   sqvDoc.Metadata,
			ChunkIndex: sqvDoc.ChunkIndex,
			CreatedAt:  sqvDoc.CreatedAt,
			UpdatedAt:  sqvDoc.UpdatedAt,
		}
	}

	return docs, nil
}

// CreateIndex wraps the underlying CreateIndex method
func (w *SqvectWrapper) CreateIndex(ctx context.Context, name string, config IndexConfig) error {
	sqvConfig := sqvect.IndexConfig{
		Dimensions: config.Dimensions,
		Metric:     sqvect.DistanceMetric(config.Metric),
		IndexType:  config.IndexType,
		Parameters: config.Parameters,
	}
	return w.SqvectStore.CreateIndex(ctx, name, sqvConfig)
}

// ListIndexes wraps the underlying ListIndexes method
func (w *SqvectWrapper) ListIndexes(ctx context.Context) ([]IndexInfo, error) {
	sqvIndexes, err := w.SqvectStore.ListIndexes(ctx)
	if err != nil {
		return nil, err
	}

	indexes := make([]IndexInfo, len(sqvIndexes))
	for i, sqvIdx := range sqvIndexes {
		indexes[i] = IndexInfo{
			Name: sqvIdx.Name,
			Config: IndexConfig{
				Dimensions: sqvIdx.Config.Dimensions,
				Metric:     DistanceMetric(sqvIdx.Config.Metric),
				IndexType:  sqvIdx.Config.IndexType,
				Parameters: sqvIdx.Config.Parameters,
			},
			DocCount:  sqvIdx.DocCount,
			CreatedAt: sqvIdx.CreatedAt,
		}
	}

	return indexes, nil
}
