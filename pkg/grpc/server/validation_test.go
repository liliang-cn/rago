package server

import (
	"context"
	"testing"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Test validation logic without requiring complex mocks
func TestRAGServiceValidation(t *testing.T) {
	// Since we can't easily mock the processor.Service, we'll test validation logic
	server := &SimpleRAGServiceServer{
		processor: nil, // This will cause method calls to panic, but validation should catch errors first
	}
	
	ctx := context.Background()
	
	t.Run("IngestDocument validation", func(t *testing.T) {
		req := &pb.IngestDocumentRequest{
			Content:  "",
			FilePath: "",
		}
		
		resp, err := server.IngestDocument(ctx, req)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "either content or file_path must be provided")
	})
	
	t.Run("Query validation", func(t *testing.T) {
		req := &pb.QueryRequest{
			Query: "",
		}
		
		resp, err := server.Query(ctx, req)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "query cannot be empty")
	})
	
	t.Run("StreamQuery validation", func(t *testing.T) {
		req := &pb.QueryRequest{
			Query: "",
		}
		
		// We can't easily mock the stream, but we can test that validation happens first
		err := server.StreamQuery(req, nil)
		
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "query cannot be empty")
	})
	
	t.Run("DeleteDocument validation", func(t *testing.T) {
		req := &pb.DeleteDocumentRequest{
			DocumentId: "",
		}
		
		resp, err := server.DeleteDocument(ctx, req)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "document_id cannot be empty")
	})
	
	t.Run("GetDocument validation", func(t *testing.T) {
		req := &pb.GetDocumentRequest{
			DocumentId: "",
		}
		
		resp, err := server.GetDocument(ctx, req)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "document_id cannot be empty")
	})
	
	t.Run("BatchIngestDocuments not implemented", func(t *testing.T) {
		req := &pb.BatchIngestDocumentsRequest{}
		
		resp, err := server.BatchIngestDocuments(ctx, req)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unimplemented, st.Code())
	})
	
	t.Run("HealthCheck always succeeds", func(t *testing.T) {
		req := &pb.HealthCheckRequest{}
		
		resp, err := server.HealthCheck(ctx, req)
		
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Healthy)
		assert.Equal(t, "2.12.0", resp.Version)
		assert.Contains(t, resp.Components, "processor")
	})
}

func TestLLMServiceValidation(t *testing.T) {
	server := &LLMServiceServer{
		llmProvider: nil, // This will cause method calls to panic, but validation should catch errors first
	}
	
	ctx := context.Background()
	
	t.Run("Generate validation", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt: "",
		}
		
		resp, err := server.Generate(ctx, req)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "prompt cannot be empty")
	})
	
	t.Run("StreamGenerate validation", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt: "",
		}
		
		err := server.StreamGenerate(req, nil)
		
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "prompt cannot be empty")
	})
	
	t.Run("GenerateWithTools validation", func(t *testing.T) {
		req := &pb.GenerateWithToolsRequest{
			Messages: []*pb.Message{},
		}
		
		resp, err := server.GenerateWithTools(ctx, req)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "messages cannot be empty")
	})
	
	t.Run("ExtractMetadata validation", func(t *testing.T) {
		req := &pb.ExtractMetadataRequest{
			Content: "",
		}
		
		resp, err := server.ExtractMetadata(ctx, req)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "content cannot be empty")
	})
}

func TestConvertGenerationOptionsValidation(t *testing.T) {
	t.Run("convert valid options", func(t *testing.T) {
		pbOpts := &pb.GenerationOptions{
			MaxTokens:   2000,
			Temperature: 0.8,
		}
		
		opts := convertGenerationOptions(pbOpts)
		
		assert.Equal(t, 2000, opts.MaxTokens)
		assert.Equal(t, 0.8, opts.Temperature)
	})
	
	t.Run("convert nil options", func(t *testing.T) {
		opts := convertGenerationOptions(nil)
		
		assert.Equal(t, 1000, opts.MaxTokens)
		assert.Equal(t, 0.7, opts.Temperature)
	})
}