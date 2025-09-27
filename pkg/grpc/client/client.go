package client

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// Config holds the gRPC client configuration
type Config struct {
	Address           string
	EnableTLS         bool
	CAFile            string
	ServerName        string
	AuthToken         string
	Timeout           time.Duration
	MaxMessageSize    int
	KeepAliveTime     time.Duration
	KeepAliveTimeout  time.Duration
	EnableCompression bool
}

// DefaultConfig returns default client configuration
func DefaultConfig(address string) Config {
	return Config{
		Address:          address,
		Timeout:          30 * time.Second,
		MaxMessageSize:   100 * 1024 * 1024, // 100MB
		KeepAliveTime:    10 * time.Second,
		KeepAliveTimeout: 5 * time.Second,
	}
}

// Client represents a gRPC client for RAGO services
type Client struct {
	config    Config
	conn      *grpc.ClientConn
	rag       pb.RAGServiceClient
	llm       pb.LLMServiceClient
	embedding pb.EmbeddingServiceClient
}

// NewClient creates a new gRPC client
func NewClient(config Config) (*Client, error) {
	// Create dial options
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxMessageSize),
			grpc.MaxCallSendMsgSize(config.MaxMessageSize),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepAliveTime,
			Timeout:             config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
	}

	// Add compression if enabled
	if config.EnableCompression {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	}

	// Add TLS or insecure transport
	if config.EnableTLS {
		tlsConfig, err := credentials.NewClientTLSFromFile(config.CAFile, config.ServerName)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(tlsConfig))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add interceptors
	opts = append(opts, grpc.WithChainUnaryInterceptor(
		timeoutInterceptor(config.Timeout),
		authInterceptor(config.AuthToken),
		retryInterceptor(3),
	))

	// Create connection
	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &Client{
		config:    config,
		conn:      conn,
		rag:       pb.NewRAGServiceClient(conn),
		llm:       pb.NewLLMServiceClient(conn),
		embedding: pb.NewEmbeddingServiceClient(conn),
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// RAG returns the RAG service client
func (c *Client) RAG() pb.RAGServiceClient {
	return c.rag
}

// LLM returns the LLM service client
func (c *Client) LLM() pb.LLMServiceClient {
	return c.llm
}

// Embedding returns the Embedding service client
func (c *Client) Embedding() pb.EmbeddingServiceClient {
	return c.embedding
}

// IngestDocument ingests a document into the RAG system
func (c *Client) IngestDocument(ctx context.Context, content string, opts ...IngestOption) (*pb.IngestDocumentResponse, error) {
	req := &pb.IngestDocumentRequest{
		Content: content,
		ChunkOptions: &pb.ChunkOptions{
			Method:  "sentence",
			Size:    500,
			Overlap: 50,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(req)
	}

	return c.rag.IngestDocument(ctx, req)
}

// Query performs a RAG query
func (c *Client) Query(ctx context.Context, query string, opts ...QueryOption) (*pb.QueryResponse, error) {
	req := &pb.QueryRequest{
		Query: query,
		TopK:  5,
		Options: &pb.QueryOptions{
			UseCache: true,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(req)
	}

	return c.rag.Query(ctx, req)
}

// StreamQuery performs a streaming RAG query
func (c *Client) StreamQuery(ctx context.Context, query string, callback StreamCallback, opts ...QueryOption) error {
	req := &pb.QueryRequest{
		Query:  query,
		TopK:   5,
		Stream: true,
		Options: &pb.QueryOptions{
			UseCache: true,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(req)
	}

	// Create stream
	stream, err := c.rag.StreamQuery(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	// Process stream
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		if err := callback(resp); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}
	}

	return nil
}

// Generate generates text using the LLM
func (c *Client) Generate(ctx context.Context, prompt string, opts ...GenerateOption) (*pb.GenerateResponse, error) {
	req := &pb.GenerateRequest{
		Prompt: prompt,
		Options: &pb.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(req)
	}

	return c.llm.Generate(ctx, req)
}

// StreamGenerate performs streaming text generation
func (c *Client) StreamGenerate(ctx context.Context, prompt string, callback GenerateCallback, opts ...GenerateOption) error {
	req := &pb.GenerateRequest{
		Prompt: prompt,
		Options: &pb.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(req)
	}

	// Create stream
	stream, err := c.llm.StreamGenerate(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	// Process stream
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		if err := callback(resp); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}
	}

	return nil
}

// GenerateEmbedding generates an embedding for text
func (c *Client) GenerateEmbedding(ctx context.Context, text string, model string) (*pb.GenerateEmbeddingResponse, error) {
	return c.embedding.GenerateEmbedding(ctx, &pb.GenerateEmbeddingRequest{
		Text:  text,
		Model: model,
	})
}

// BatchGenerateEmbeddings generates embeddings for multiple texts
func (c *Client) BatchGenerateEmbeddings(ctx context.Context, texts []string, model string) (*pb.BatchGenerateEmbeddingsResponse, error) {
	return c.embedding.BatchGenerateEmbeddings(ctx, &pb.BatchGenerateEmbeddingsRequest{
		Texts: texts,
		Model: model,
	})
}

// ComputeSimilarity computes similarity between two texts
func (c *Client) ComputeSimilarity(ctx context.Context, text1, text2 string, metric string) (*pb.ComputeSimilarityResponse, error) {
	if metric == "" {
		metric = "cosine"
	}
	return c.embedding.ComputeSimilarity(ctx, &pb.ComputeSimilarityRequest{
		Text1:  text1,
		Text2:  text2,
		Metric: metric,
	})
}

// HealthCheck performs a health check
func (c *Client) HealthCheck(ctx context.Context) (*pb.HealthCheckResponse, error) {
	return c.rag.HealthCheck(ctx, &pb.HealthCheckRequest{})
}

// Callbacks and option types

// StreamCallback is called for each streaming query response
type StreamCallback func(*pb.StreamQueryResponse) error

// GenerateCallback is called for each streaming generation response
type GenerateCallback func(*pb.StreamGenerateResponse) error

// IngestOption configures document ingestion
type IngestOption func(*pb.IngestDocumentRequest)

// WithFilePath sets the file path for ingestion
func WithFilePath(path string) IngestOption {
	return func(req *pb.IngestDocumentRequest) {
		req.FilePath = path
	}
}

// WithCollection sets the collection for ingestion
func WithCollection(collection string) IngestOption {
	return func(req *pb.IngestDocumentRequest) {
		req.Collection = collection
	}
}

// WithMetadata sets metadata for ingestion
func WithMetadata(metadata map[string]string) IngestOption {
	return func(req *pb.IngestDocumentRequest) {
		req.Metadata = metadata
	}
}

// WithChunkOptions sets chunking options
func WithChunkOptions(method string, size, overlap int32) IngestOption {
	return func(req *pb.IngestDocumentRequest) {
		req.ChunkOptions = &pb.ChunkOptions{
			Method:  method,
			Size:    size,
			Overlap: overlap,
		}
	}
}

// QueryOption configures a query
type QueryOption func(*pb.QueryRequest)

// WithTopK sets the number of results to return
func WithTopK(k int32) QueryOption {
	return func(req *pb.QueryRequest) {
		req.TopK = k
	}
}

// WithMinScore sets the minimum score for results
func WithMinScore(score float64) QueryOption {
	return func(req *pb.QueryRequest) {
		req.MinScore = score
	}
}

// WithQueryCollection sets the collection to query
func WithQueryCollection(collection string) QueryOption {
	return func(req *pb.QueryRequest) {
		req.Collection = collection
	}
}

// WithHybridSearch enables hybrid search
func WithHybridSearch(alpha float64) QueryOption {
	return func(req *pb.QueryRequest) {
		if req.Options == nil {
			req.Options = &pb.QueryOptions{}
		}
		req.Options.HybridSearch = true
		req.Options.Alpha = alpha
	}
}

// WithCache controls cache usage
func WithCache(useCache bool) QueryOption {
	return func(req *pb.QueryRequest) {
		if req.Options == nil {
			req.Options = &pb.QueryOptions{}
		}
		req.Options.UseCache = useCache
	}
}

// GenerateOption configures text generation
type GenerateOption func(*pb.GenerateRequest)

// WithModel sets the model to use
func WithModel(model string) GenerateOption {
	return func(req *pb.GenerateRequest) {
		req.Model = model
	}
}

// WithMaxTokens sets the maximum tokens to generate
func WithMaxTokens(tokens int32) GenerateOption {
	return func(req *pb.GenerateRequest) {
		if req.Options == nil {
			req.Options = &pb.GenerationOptions{}
		}
		req.Options.MaxTokens = tokens
	}
}

// WithTemperature sets the generation temperature
func WithTemperature(temp float64) GenerateOption {
	return func(req *pb.GenerateRequest) {
		if req.Options == nil {
			req.Options = &pb.GenerationOptions{}
		}
		req.Options.Temperature = temp
	}
}

// WithTopP sets the top-p value
func WithTopP(p float64) GenerateOption {
	return func(req *pb.GenerateRequest) {
		if req.Options == nil {
			req.Options = &pb.GenerationOptions{}
		}
		req.Options.TopP = p
	}
}

// Interceptors

// timeoutInterceptor adds timeout to requests
func timeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// authInterceptor adds authentication token to requests
func authInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// retryInterceptor implements retry logic
func retryInterceptor(maxRetries int) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var lastErr error
		for i := 0; i <= maxRetries; i++ {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}
			lastErr = err

			// Don't retry on certain errors
			if !isRetryable(err) {
				return err
			}

			// Exponential backoff
			if i < maxRetries {
				time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			}
		}
		return lastErr
	}
}

// isRetryable checks if an error is retryable
func isRetryable(err error) bool {
	// Add logic to determine if error is retryable
	// For now, we'll keep it simple
	return false
}