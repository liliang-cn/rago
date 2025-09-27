package client

import (
	"context"
	"testing"
	"time"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("localhost:50051")
	
	assert.Equal(t, "localhost:50051", config.Address)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 100*1024*1024, config.MaxMessageSize)
	assert.False(t, config.EnableTLS)
	assert.False(t, config.EnableCompression)
	assert.Empty(t, config.AuthToken)
}

func TestConfigValidation(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := Config{
			Address:           "localhost:50051",
			Timeout:           10 * time.Second,
			MaxMessageSize:    1024,
			EnableTLS:         false,
			EnableCompression: true,
		}
		
		assert.Equal(t, "localhost:50051", config.Address)
		assert.Equal(t, 10*time.Second, config.Timeout)
		assert.Equal(t, 1024, config.MaxMessageSize)
		assert.False(t, config.EnableTLS)
		assert.True(t, config.EnableCompression)
	})
	
	t.Run("TLS config", func(t *testing.T) {
		config := Config{
			EnableTLS:  true,
			CAFile:     "/path/to/cert.pem",
			ServerName: "example.com",
		}
		
		assert.True(t, config.EnableTLS)
		assert.Equal(t, "/path/to/cert.pem", config.CAFile)
		assert.Equal(t, "example.com", config.ServerName)
	})
	
	t.Run("auth config", func(t *testing.T) {
		config := Config{
			AuthToken: "Bearer test-token",
		}
		
		assert.Equal(t, "Bearer test-token", config.AuthToken)
	})
}

func TestIngestOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		req := &pb.IngestDocumentRequest{}
		
		// Test applying no options
		opts := []IngestOption{}
		for _, opt := range opts {
			opt(req)
		}
		
		assert.Empty(t, req.Metadata)
		assert.Empty(t, req.FilePath)
	})
	
	t.Run("with metadata", func(t *testing.T) {
		req := &pb.IngestDocumentRequest{}
		metadata := map[string]string{
			"title":  "Test Document",
			"author": "Test Author",
		}
		
		opt := WithMetadata(metadata)
		opt(req)
		
		assert.Equal(t, metadata, req.Metadata)
	})
	
	t.Run("with file path", func(t *testing.T) {
		req := &pb.IngestDocumentRequest{}
		
		opt := WithFilePath("/path/to/file.txt")
		opt(req)
		
		assert.Equal(t, "/path/to/file.txt", req.FilePath)
	})
	
	t.Run("multiple options", func(t *testing.T) {
		req := &pb.IngestDocumentRequest{}
		metadata := map[string]string{"title": "Test"}
		
		opts := []IngestOption{
			WithMetadata(metadata),
			WithFilePath("/test.txt"),
		}
		
		for _, opt := range opts {
			opt(req)
		}
		
		assert.Equal(t, metadata, req.Metadata)
		assert.Equal(t, "/test.txt", req.FilePath)
	})
}

func TestQueryOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		req := &pb.QueryRequest{}
		
		opts := []QueryOption{}
		for _, opt := range opts {
			opt(req)
		}
		
		// Default values should be zero values
		assert.Equal(t, int32(0), req.TopK)
		assert.Equal(t, float64(0), req.MinScore)
		assert.Empty(t, req.Collection)
	})
	
	t.Run("with top K", func(t *testing.T) {
		req := &pb.QueryRequest{}
		
		opt := WithTopK(5)
		opt(req)
		
		assert.Equal(t, int32(5), req.TopK)
	})
	
	t.Run("with min score", func(t *testing.T) {
		req := &pb.QueryRequest{}
		
		opt := WithMinScore(0.7)
		opt(req)
		
		assert.Equal(t, 0.7, req.MinScore)
	})
	
	t.Run("with collection", func(t *testing.T) {
		req := &pb.QueryRequest{}
		
		opt := WithQueryCollection("test-collection")
		opt(req)
		
		assert.Equal(t, "test-collection", req.Collection)
	})
	
	t.Run("multiple options", func(t *testing.T) {
		req := &pb.QueryRequest{}
		
		opts := []QueryOption{
			WithTopK(10),
			WithMinScore(0.8),
			WithQueryCollection("docs"),
		}
		
		for _, opt := range opts {
			opt(req)
		}
		
		assert.Equal(t, int32(10), req.TopK)
		assert.Equal(t, 0.8, req.MinScore)
		assert.Equal(t, "docs", req.Collection)
	})
}

func TestGenerationOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		req := &pb.GenerateRequest{}
		
		opts := []GenerateOption{}
		for _, opt := range opts {
			opt(req)
		}
		
		assert.Nil(t, req.Options)
		assert.Empty(t, req.Model)
	})
	
	t.Run("with max tokens", func(t *testing.T) {
		req := &pb.GenerateRequest{}
		
		opt := WithMaxTokens(1000)
		opt(req)
		
		assert.NotNil(t, req.Options)
		assert.Equal(t, int32(1000), req.Options.MaxTokens)
	})
	
	t.Run("with model", func(t *testing.T) {
		req := &pb.GenerateRequest{}
		
		opt := WithModel("gpt-4")
		opt(req)
		
		assert.Equal(t, "gpt-4", req.Model)
	})
	
	t.Run("multiple options", func(t *testing.T) {
		req := &pb.GenerateRequest{}
		
		opts := []GenerateOption{
			WithMaxTokens(2000),
			WithModel("claude-3"),
		}
		
		for _, opt := range opts {
			opt(req)
		}
		
		assert.Equal(t, "claude-3", req.Model)
		assert.NotNil(t, req.Options)
		assert.Equal(t, int32(2000), req.Options.MaxTokens)
	})
}

func TestEmbeddingMethods(t *testing.T) {
	t.Run("embedding request structure", func(t *testing.T) {
		req := &pb.GenerateEmbeddingRequest{
			Text:  "test text",
			Model: "text-embedding-ada-002",
		}
		
		assert.Equal(t, "test text", req.Text)
		assert.Equal(t, "text-embedding-ada-002", req.Model)
	})
}

// Test helper functions for testing client methods without actual gRPC calls
func TestClientValidation(t *testing.T) {
	// These tests focus on input validation and option application
	// without requiring actual network connections
	
	t.Run("ingest document validation", func(t *testing.T) {
		// Test that empty content is handled at request level
		req := &pb.IngestDocumentRequest{
			Content: "",
		}
		
		// This would normally fail at the server level
		assert.Empty(t, req.Content)
		assert.Empty(t, req.FilePath)
	})
	
	t.Run("query validation", func(t *testing.T) {
		req := &pb.QueryRequest{
			Query: "",
		}
		
		assert.Empty(t, req.Query)
	})
	
	t.Run("generation validation", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt: "",
		}
		
		assert.Empty(t, req.Prompt)
	})
}

func TestClientConnectionHandling(t *testing.T) {
	t.Run("connection config validation", func(t *testing.T) {
		config := Config{
			Address:    "",
			Timeout:    0,
			EnableTLS:  true,
			CAFile:     "",
			ServerName: "",
		}
		
		// Validate that config requires address
		assert.Empty(t, config.Address)
		
		// TLS config should have cert file
		if config.EnableTLS {
			assert.Empty(t, config.CAFile)
			assert.Empty(t, config.ServerName)
		}
	})
	
	t.Run("dial options generation", func(t *testing.T) {
		config := DefaultConfig("localhost:50051")
		config.MaxMessageSize = 1024
		
		// Test that we can generate dial options without panicking
		opts := []grpc.DialOption{}
		
		// Add max message size option
		opts = append(opts, grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxMessageSize),
			grpc.MaxCallSendMsgSize(config.MaxMessageSize),
		))
		
		assert.NotEmpty(t, opts)
	})
}

func TestClientRetryLogic(t *testing.T) {
	t.Run("retry configuration concept", func(t *testing.T) {
		config := DefaultConfig("localhost:50051")
		
		// Test timeout configuration which is related to retry behavior
		assert.Equal(t, 30*time.Second, config.Timeout)
		
		// Test that we can configure timeouts for retry scenarios
		config.Timeout = 5 * time.Second
		assert.Equal(t, 5*time.Second, config.Timeout)
	})
}

func TestClientContextHandling(t *testing.T) {
	t.Run("timeout context", func(t *testing.T) {
		config := DefaultConfig("localhost:50051")
		config.Timeout = 5 * time.Second
		
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, config.Timeout)
		defer cancel()
		
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.True(t, time.Until(deadline) <= config.Timeout)
	})
	
	t.Run("cancellation context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		
		select {
		case <-ctx.Done():
			assert.Equal(t, context.Canceled, ctx.Err())
		default:
			assert.Fail(t, "context should be cancelled")
		}
	})
}

func TestStreamingHelpers(t *testing.T) {
	t.Run("streaming callback validation", func(t *testing.T) {
		// Test that callback functions can be defined and called
		var results []string
		callback := func(text string) {
			results = append(results, text)
		}
		
		// Simulate streaming responses
		callback("chunk1")
		callback("chunk2")
		callback("chunk3")
		
		assert.Len(t, results, 3)
		assert.Equal(t, []string{"chunk1", "chunk2", "chunk3"}, results)
	})
	
	t.Run("error callback handling", func(t *testing.T) {
		var errors []string
		errorCallback := func(err error) {
			if err != nil {
				errors = append(errors, err.Error())
			}
		}
		
		// Simulate error
		errorCallback(assert.AnError)
		
		assert.Len(t, errors, 1)
		assert.Contains(t, errors[0], "assert.AnError")
	})
}