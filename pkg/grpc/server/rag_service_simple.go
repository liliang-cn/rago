package server

import (
	"context"
	"fmt"
	"time"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SimpleRAGServiceServer implements a simplified RAG gRPC service
type SimpleRAGServiceServer struct {
	pb.UnimplementedRAGServiceServer
	processor *processor.Service
}

// NewSimpleRAGServiceServer creates a new simplified RAG service server
func NewSimpleRAGServiceServer(processor *processor.Service) *SimpleRAGServiceServer {
	return &SimpleRAGServiceServer{
		processor: processor,
	}
}

// IngestDocument ingests a single document into the RAG system
func (s *SimpleRAGServiceServer) IngestDocument(ctx context.Context, req *pb.IngestDocumentRequest) (*pb.IngestDocumentResponse, error) {
	if req.Content == "" && req.FilePath == "" {
		return nil, status.Error(codes.InvalidArgument, "either content or file_path must be provided")
	}

	// Convert metadata
	metadata := make(map[string]interface{})
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	// Create ingest request
	ingestReq := domain.IngestRequest{
		Content:  req.Content,
		FilePath: req.FilePath,
		Metadata: metadata,
	}

	// Use the processor's Ingest method
	resp, err := s.processor.Ingest(ctx, ingestReq)
	if err != nil {
		return &pb.IngestDocumentResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.IngestDocumentResponse{
		DocumentId: resp.DocumentID,
		ChunkCount: int32(resp.ChunkCount),
		Message:    resp.Message,
		Success:    true,
	}, nil
}

// Query performs a RAG query
func (s *SimpleRAGServiceServer) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query cannot be empty")
	}

	startTime := time.Now()

	// Use the processor's Query method
	queryReq := domain.QueryRequest{
		Query: req.Query,
		// Note: Collection field not supported in current interface
	}

	result, err := s.processor.Query(ctx, queryReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}

	// Convert search results (simplified)
	searchResults := make([]*pb.SearchResult, 0, len(result.Sources))
	for _, chunk := range result.Sources {
		// Convert metadata
		metadata := make(map[string]string)
		for k, v := range chunk.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			}
		}
		
		searchResults = append(searchResults, &pb.SearchResult{
			DocumentId: chunk.DocumentID,
			ChunkId:    chunk.ID,
			Content:    chunk.Content,
			Score:      chunk.Score, 
			Source:     chunk.Content, // Use content as source for now
			Metadata:   metadata,
		})
	}

	return &pb.QueryResponse{
		Answer:           result.Answer,
		Results:          searchResults,
		Metadata:         make(map[string]string),
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		FromCache:        false,
	}, nil
}

// StreamQuery performs a streaming RAG query
func (s *SimpleRAGServiceServer) StreamQuery(req *pb.QueryRequest, stream pb.RAGService_StreamQueryServer) error {
	if req.Query == "" {
		return status.Error(codes.InvalidArgument, "query cannot be empty")
	}

	ctx := stream.Context()
	startTime := time.Now()

	// Use the processor's StreamQuery method
	queryReq := domain.QueryRequest{
		Query: req.Query,
	}

	// Stream callback
	callback := func(chunk string) {
		stream.Send(&pb.StreamQueryResponse{
			Content: &pb.StreamQueryResponse_TextChunk{
				TextChunk: chunk,
			},
		})
	}

	// Execute streaming query
	err := s.processor.StreamQuery(ctx, queryReq, callback)
	if err != nil {
		stream.Send(&pb.StreamQueryResponse{
			Content: &pb.StreamQueryResponse_Error{
				Error: err.Error(),
			},
		})
		return nil
	}

	// Send metadata at the end
	stream.Send(&pb.StreamQueryResponse{
		Content: &pb.StreamQueryResponse_Metadata{
			Metadata: &pb.QueryMetadata{
				ProcessingTimeMs: time.Since(startTime).Milliseconds(),
				FromCache:        false,
				ModelUsed:        "",
			},
		},
	})

	return nil
}

// ListDocuments lists documents (simplified implementation)
func (s *SimpleRAGServiceServer) ListDocuments(ctx context.Context, req *pb.ListDocumentsRequest) (*pb.ListDocumentsResponse, error) {
	// Get all documents from the processor
	docs, err := s.processor.ListDocuments(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list documents: %v", err)
	}

	// Convert to proto documents
	pbDocs := make([]*pb.Document, 0, len(docs))
	for _, doc := range docs {
		// Convert metadata
		metadata := make(map[string]string)
		for k, v := range doc.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			}
		}
		
		pbDocs = append(pbDocs, &pb.Document{
			Id:         doc.ID,
			Content:    doc.Content,
			FilePath:   doc.Path, // Use Path field
			Collection: "", // Not available in current domain
			Metadata:   metadata,
			CreatedAt:  timestamppb.New(doc.Created),
			UpdatedAt:  timestamppb.New(doc.Created), // Use same for now
		})
	}

	// Apply simple pagination
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	start := 0
	if req.PageToken != "" {
		// Simple token parsing (in production, use proper pagination tokens)
		fmt.Sscanf(req.PageToken, "%d", &start)
	}

	end := start + pageSize
	if end > len(pbDocs) {
		end = len(pbDocs)
	}

	nextPageToken := ""
	if end < len(pbDocs) {
		nextPageToken = fmt.Sprintf("%d", end)
	}

	return &pb.ListDocumentsResponse{
		Documents:     pbDocs[start:end],
		NextPageToken: nextPageToken,
		TotalCount:    int32(len(pbDocs)),
	}, nil
}

// DeleteDocument deletes a document (simplified implementation)
func (s *SimpleRAGServiceServer) DeleteDocument(ctx context.Context, req *pb.DeleteDocumentRequest) (*pb.DeleteDocumentResponse, error) {
	if req.DocumentId == "" {
		return nil, status.Error(codes.InvalidArgument, "document_id cannot be empty")
	}

	// Note: Delete not directly supported in current interface
	// This would need to be implemented in the processor
	return &pb.DeleteDocumentResponse{
		Success: false,
		Message: "Delete operation not implemented",
	}, nil
}

// GetDocument retrieves a document by ID (simplified implementation)
func (s *SimpleRAGServiceServer) GetDocument(ctx context.Context, req *pb.GetDocumentRequest) (*pb.GetDocumentResponse, error) {
	if req.DocumentId == "" {
		return nil, status.Error(codes.InvalidArgument, "document_id cannot be empty")
	}

	// List all documents and find the one with matching ID
	docs, err := s.processor.ListDocuments(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list documents: %v", err)
	}
	
	for _, doc := range docs {
		if doc.ID == req.DocumentId {
			// Convert metadata
			metadata := make(map[string]string)
			for k, v := range doc.Metadata {
				if str, ok := v.(string); ok {
					metadata[k] = str
				}
			}
			
			return &pb.GetDocumentResponse{
				Document: &pb.Document{
					Id:         doc.ID,
					Content:    doc.Content,
					FilePath:   doc.Path,
					Collection: "", // Not available
					Metadata:   metadata,
					CreatedAt:  timestamppb.New(doc.Created),
					UpdatedAt:  timestamppb.New(doc.Created),
				},
				Found: true,
			}, nil
		}
	}

	return &pb.GetDocumentResponse{
		Found: false,
	}, nil
}

// GetStatistics returns system statistics (simplified implementation)
func (s *SimpleRAGServiceServer) GetStatistics(ctx context.Context, req *pb.GetStatisticsRequest) (*pb.GetStatisticsResponse, error) {
	// Get document count
	docs, _ := s.processor.ListDocuments(ctx)
	
	return &pb.GetStatisticsResponse{
		TotalDocuments:           int64(len(docs)),
		TotalChunks:              0, // Not available
		TotalEmbeddings:          0, // Not available
		DocumentsByCollection:    make(map[string]int64),
		AverageChunksPerDocument: 0.0,
		StorageSizeBytes:         0,
		LastIngestion:            nil,
	}, nil
}

// HealthCheck performs a health check
func (s *SimpleRAGServiceServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	components := make(map[string]*pb.ComponentHealth)

	// Basic health check
	components["processor"] = &pb.ComponentHealth{
		Healthy:   true,
		Status:    "healthy",
		Error:     "",
		LastCheck: timestamppb.Now(),
	}

	return &pb.HealthCheckResponse{
		Healthy:    true,
		Components: components,
		Version:    "2.12.0",
	}, nil
}

// BatchIngestDocuments is not implemented in this simplified version
func (s *SimpleRAGServiceServer) BatchIngestDocuments(ctx context.Context, req *pb.BatchIngestDocumentsRequest) (*pb.BatchIngestDocumentsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "batch ingest not implemented")
}