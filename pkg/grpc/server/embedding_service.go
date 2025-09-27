package server

import (
	"context"
	"math"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EmbeddingServiceServer implements the Embedding gRPC service
type EmbeddingServiceServer struct {
	pb.UnimplementedEmbeddingServiceServer
	embedder domain.EmbedderProvider
}

// NewEmbeddingServiceServer creates a new embedding service server
func NewEmbeddingServiceServer(embedder domain.EmbedderProvider) *EmbeddingServiceServer {
	return &EmbeddingServiceServer{
		embedder: embedder,
	}
}

// GenerateEmbedding generates embeddings for a single text
func (s *EmbeddingServiceServer) GenerateEmbedding(ctx context.Context, req *pb.GenerateEmbeddingRequest) (*pb.GenerateEmbeddingResponse, error) {
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text cannot be empty")
	}

	// Generate embedding
	embedding, err := s.embedder.Embed(ctx, req.Text)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "embedding generation failed: %v", err)
	}

	// Convert to float32
	embedding32 := make([]float32, len(embedding))
	for i, v := range embedding {
		embedding32[i] = float32(v)
	}

	return &pb.GenerateEmbeddingResponse{
		Embedding:  embedding32,
		Dimensions: int32(len(embedding)),
		Model:      req.Model,
	}, nil
}

// BatchGenerateEmbeddings generates embeddings for multiple texts
func (s *EmbeddingServiceServer) BatchGenerateEmbeddings(ctx context.Context, req *pb.BatchGenerateEmbeddingsRequest) (*pb.BatchGenerateEmbeddingsResponse, error) {
	if len(req.Texts) == 0 {
		return nil, status.Error(codes.InvalidArgument, "texts cannot be empty")
	}

	results := make([]*pb.EmbeddingResult, 0, len(req.Texts))

	// Generate embeddings for each text
	for i, text := range req.Texts {
		if text == "" {
			results = append(results, &pb.EmbeddingResult{
				Index: int32(i),
				Error: "text is empty",
			})
			continue
		}

		embedding, err := s.embedder.Embed(ctx, text)
		if err != nil {
			results = append(results, &pb.EmbeddingResult{
				Index: int32(i),
				Error: err.Error(),
			})
			continue
		}

		// Convert to float32
		embedding32 := make([]float32, len(embedding))
		for j, v := range embedding {
			embedding32[j] = float32(v)
		}

		results = append(results, &pb.EmbeddingResult{
			Embedding: embedding32,
			Index:     int32(i),
		})
	}

	return &pb.BatchGenerateEmbeddingsResponse{
		Results: results,
	}, nil
}

// ComputeSimilarity computes the similarity between two texts
func (s *EmbeddingServiceServer) ComputeSimilarity(ctx context.Context, req *pb.ComputeSimilarityRequest) (*pb.ComputeSimilarityResponse, error) {
	if req.Text1 == "" || req.Text2 == "" {
		return nil, status.Error(codes.InvalidArgument, "both text1 and text2 must be provided")
	}

	// Generate embeddings for both texts
	embedding1, err := s.embedder.Embed(ctx, req.Text1)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate embedding for text1: %v", err)
	}

	embedding2, err := s.embedder.Embed(ctx, req.Text2)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate embedding for text2: %v", err)
	}

	// Compute similarity based on metric
	var similarity float64
	metric := req.Metric
	if metric == "" {
		metric = "cosine"
	}

	switch metric {
	case "cosine":
		similarity = cosineSimilarity(embedding1, embedding2)
	case "euclidean":
		similarity = euclideanDistance(embedding1, embedding2)
	case "dot_product":
		similarity = dotProduct(embedding1, embedding2)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported metric: %s", metric)
	}

	return &pb.ComputeSimilarityResponse{
		Similarity: similarity,
		Metric:     metric,
	}, nil
}

// cosineSimilarity computes cosine similarity between two vectors
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

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// euclideanDistance computes Euclidean distance between two vectors
func euclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	var sum float64
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

// dotProduct computes dot product between two vectors
func dotProduct(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var product float64
	for i := range a {
		product += a[i] * b[i]
	}

	return product
}