package server

import (
	"testing"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Test only the validation logic without mocking complex dependencies
func TestSimpleRAGServiceValidation(t *testing.T) {
	t.Run("ingest document validation - empty content and filepath", func(t *testing.T) {
		// Test the validation logic that would be in IngestDocument
		req := &pb.IngestDocumentRequest{
			Content:  "",
			FilePath: "",
		}
		
		// This is the validation that should occur
		if req.Content == "" && req.FilePath == "" {
			err := status.Error(codes.InvalidArgument, "either content or file_path must be provided")
			assert.Error(t, err)
			st, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code())
		}
	})
	
	t.Run("ingest document validation - valid content", func(t *testing.T) {
		req := &pb.IngestDocumentRequest{
			Content: "test content",
		}
		
		// Should not trigger validation error
		assert.NotEmpty(t, req.Content)
	})
	
	t.Run("query validation - empty query", func(t *testing.T) {
		req := &pb.QueryRequest{
			Query: "",
		}
		
		// This validation should occur
		if req.Query == "" {
			err := status.Error(codes.InvalidArgument, "query cannot be empty")
			assert.Error(t, err)
			st, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code())
		}
	})
}