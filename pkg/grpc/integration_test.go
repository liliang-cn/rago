package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/liliang-cn/rago/v2/pkg/grpc/client"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// MockService implements a minimal service for integration testing
type MockRAGService struct {
	pb.UnimplementedRAGServiceServer
}

func (m *MockRAGService) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Healthy: true,
		Version: "test",
		Components: map[string]*pb.ComponentHealth{
			"test": {
				Healthy: true,
				Status:  "ok",
			},
		},
	}, nil
}

func (m *MockRAGService) IngestDocument(ctx context.Context, req *pb.IngestDocumentRequest) (*pb.IngestDocumentResponse, error) {
	if req.Content == "" && req.FilePath == "" {
		return nil, fmt.Errorf("content or file path required")
	}
	
	return &pb.IngestDocumentResponse{
		DocumentId: "test-doc-123",
		ChunkCount: 3,
		Message:    "Document ingested successfully",
		Success:    true,
	}, nil
}

func (m *MockRAGService) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	
	return &pb.QueryResponse{
		Answer: "This is a test response to: " + req.Query,
		Results: []*pb.SearchResult{
			{
				DocumentId: "test-doc-123",
				ChunkId:    "chunk-1",
				Content:    "Relevant test content",
				Score:      0.95,
				Source:     "test document",
			},
		},
		ProcessingTimeMs: 100,
	}, nil
}

type MockLLMService struct {
	pb.UnimplementedLLMServiceServer
}

func (m *MockLLMService) Generate(ctx context.Context, req *pb.GenerateRequest) (*pb.GenerateResponse, error) {
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}
	
	return &pb.GenerateResponse{
		Text:             "Generated response to: " + req.Prompt,
		Model:            "test-model",
		GenerationTimeMs: 50,
	}, nil
}

type MockEmbeddingService struct {
	pb.UnimplementedEmbeddingServiceServer
}

func (m *MockEmbeddingService) GenerateEmbedding(ctx context.Context, req *pb.GenerateEmbeddingRequest) (*pb.GenerateEmbeddingResponse, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	
	// Return a mock embedding vector
	embedding := make([]float32, 512)
	for i := range embedding {
		embedding[i] = 0.1 * float32(i%10)
	}
	
	return &pb.GenerateEmbeddingResponse{
		Embedding:  embedding,
		Dimensions: 512,
		Model:      "test-embedding-model",
	}, nil
}

func (m *MockEmbeddingService) BatchGenerateEmbeddings(ctx context.Context, req *pb.BatchGenerateEmbeddingsRequest) (*pb.BatchGenerateEmbeddingsResponse, error) {
	if len(req.Texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}
	
	var results []*pb.EmbeddingResult
	for i, text := range req.Texts {
		if text == "" {
			results = append(results, &pb.EmbeddingResult{
				Index: int32(i),
				Error: "empty text",
			})
			continue
		}
		
		// Generate mock embedding for each text
		embedding := make([]float32, 512)
		for j := range embedding {
			embedding[j] = 0.1 * float32(j%10)
		}
		
		results = append(results, &pb.EmbeddingResult{
			Embedding: embedding,
			Index:     int32(i),
		})
	}
	
	return &pb.BatchGenerateEmbeddingsResponse{
		Results: results,
	}, nil
}

func (m *MockEmbeddingService) ComputeSimilarity(ctx context.Context, req *pb.ComputeSimilarityRequest) (*pb.ComputeSimilarityResponse, error) {
	if req.Text1 == "" || req.Text2 == "" {
		return nil, fmt.Errorf("both texts must be provided")
	}
	
	// Return a mock similarity score
	return &pb.ComputeSimilarityResponse{
		Similarity: 0.85,
		Metric:     req.Metric,
	}, nil
}

// Integration test that starts a real gRPC server and client
func TestGRPCIntegration(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	
	address := fmt.Sprintf("localhost:%d", port)
	
	// Create gRPC server
	grpcServer := grpc.NewServer()
	
	// Register mock services
	pb.RegisterRAGServiceServer(grpcServer, &MockRAGService{})
	pb.RegisterLLMServiceServer(grpcServer, &MockLLMService{})
	pb.RegisterEmbeddingServiceServer(grpcServer, &MockEmbeddingService{})
	
	// Start server in background
	listener, err = net.Listen("tcp", address)
	assert.NoError(t, err)
	
	go func() {
		grpcServer.Serve(listener)
	}()
	defer grpcServer.Stop()
	
	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)
	
	// Create client
	config := client.DefaultConfig(address)
	config.Timeout = 5 * time.Second
	
	grpcClient, err := client.NewClient(config)
	assert.NoError(t, err)
	defer grpcClient.Close()
	
	ctx := context.Background()
	
	t.Run("health check", func(t *testing.T) {
		resp, err := grpcClient.HealthCheck(ctx)
		assert.NoError(t, err)
		assert.True(t, resp.Healthy)
		assert.Equal(t, "test", resp.Version)
	})
	
	t.Run("document ingestion", func(t *testing.T) {
		resp, err := grpcClient.IngestDocument(ctx, "Test document content")
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "test-doc-123", resp.DocumentId)
		assert.Equal(t, int32(3), resp.ChunkCount)
	})
	
	t.Run("document ingestion with options", func(t *testing.T) {
		resp, err := grpcClient.IngestDocument(ctx, "Test content", 
			client.WithMetadata(map[string]string{
				"title": "Test Document",
			}),
		)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
	})
	
	t.Run("query", func(t *testing.T) {
		resp, err := grpcClient.Query(ctx, "What is AI?")
		assert.NoError(t, err)
		assert.Contains(t, resp.Answer, "What is AI?")
		assert.Len(t, resp.Results, 1)
		assert.Equal(t, "test-doc-123", resp.Results[0].DocumentId)
	})
	
	t.Run("query with options", func(t *testing.T) {
		resp, err := grpcClient.Query(ctx, "What is machine learning?",
			client.WithTopK(5),
			client.WithMinScore(0.5),
		)
		assert.NoError(t, err)
		assert.Contains(t, resp.Answer, "machine learning")
	})
	
	t.Run("text generation", func(t *testing.T) {
		resp, err := grpcClient.Generate(ctx, "Write a story about AI")
		assert.NoError(t, err)
		assert.Contains(t, resp.Text, "Write a story about AI")
		assert.Greater(t, resp.GenerationTimeMs, int64(0))
	})
	
	t.Run("text generation with options", func(t *testing.T) {
		resp, err := grpcClient.Generate(ctx, "Explain quantum computing",
			client.WithModel("test-model"),
			client.WithMaxTokens(2000),
		)
		assert.NoError(t, err)
		assert.Contains(t, resp.Text, "quantum computing")
		assert.Equal(t, "test-model", resp.Model)
	})
	
	t.Run("embedding generation", func(t *testing.T) {
		resp, err := grpcClient.GenerateEmbedding(ctx, "Hello world", "test-model")
		assert.NoError(t, err)
		assert.Len(t, resp.Embedding, 512)
		assert.Equal(t, int32(512), resp.Dimensions)
		assert.Equal(t, "test-embedding-model", resp.Model)
	})
	
	t.Run("batch embedding generation", func(t *testing.T) {
		texts := []string{"Hello", "World", "AI"}
		resp, err := grpcClient.BatchGenerateEmbeddings(ctx, texts, "test-model")
		assert.NoError(t, err)
		assert.Len(t, resp.Results, 3)
		for i, result := range resp.Results {
			assert.Len(t, result.Embedding, 512)
			assert.Equal(t, int32(i), result.Index)
			assert.Empty(t, result.Error)
		}
	})
	
	t.Run("similarity computation", func(t *testing.T) {
		resp, err := grpcClient.ComputeSimilarity(ctx, "Hello world", "Hi earth", "cosine")
		assert.NoError(t, err)
		assert.Greater(t, resp.Similarity, 0.0)
		assert.LessOrEqual(t, resp.Similarity, 1.0)
	})
}

// Test error handling in integration scenarios
func TestGRPCIntegrationErrors(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	
	address := fmt.Sprintf("localhost:%d", port)
	
	// Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterRAGServiceServer(grpcServer, &MockRAGService{})
	pb.RegisterLLMServiceServer(grpcServer, &MockLLMService{})
	pb.RegisterEmbeddingServiceServer(grpcServer, &MockEmbeddingService{})
	
	// Start server
	listener, err = net.Listen("tcp", address)
	assert.NoError(t, err)
	
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()
	
	time.Sleep(100 * time.Millisecond)
	
	// Create client
	config := client.DefaultConfig(address)
	grpcClient, err := client.NewClient(config)
	assert.NoError(t, err)
	defer grpcClient.Close()
	
	ctx := context.Background()
	
	t.Run("empty query error", func(t *testing.T) {
		_, err := grpcClient.Query(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query cannot be empty")
	})
	
	t.Run("empty prompt error", func(t *testing.T) {
		_, err := grpcClient.Generate(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt cannot be empty")
	})
	
	t.Run("empty text for embedding error", func(t *testing.T) {
		_, err := grpcClient.GenerateEmbedding(ctx, "", "model")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "text cannot be empty")
	})
}

// Test connection errors
func TestGRPCConnectionErrors(t *testing.T) {
	t.Run("connection to non-existent server", func(t *testing.T) {
		config := client.DefaultConfig("localhost:99999")
		config.Timeout = 1 * time.Second
		
		_, err := client.NewClient(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect")
	})
}