package server

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
	"github.com/liliang-cn/rago/v2/pkg/store"
	"github.com/liliang-cn/rago/v2/pkg/usage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Config holds the gRPC server configuration
type Config struct {
	Port              int
	MaxMessageSize    int
	EnableTLS         bool
	CertFile          string
	KeyFile           string
	EnableReflection  bool
	EnableAuth        bool
	AuthToken         string
	MaxConnectionIdle time.Duration
	MaxConnectionAge  time.Duration
	KeepAliveTime     time.Duration
	KeepAliveTimeout  time.Duration
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Port:              50051,
		MaxMessageSize:    100 * 1024 * 1024, // 100MB
		EnableReflection:  true,
		EnableAuth:        false,
		MaxConnectionIdle: 15 * time.Minute,
		MaxConnectionAge:  30 * time.Minute,
		KeepAliveTime:     5 * time.Minute,
		KeepAliveTimeout:  20 * time.Second,
	}
}

// Server represents the gRPC server
type Server struct {
	config               Config
	grpcServer           *grpc.Server
	ragService           *SimpleRAGServiceServer
	llmService           *LLMServiceServer
	embeddingService     *EmbeddingServiceServer
	conversationService  *ConversationService
	usageService         *UsageService
}

// NewServer creates a new gRPC server
func NewServer(
	config Config,
	processor *processor.Service,
	llmProvider domain.LLMProvider,
	embedder domain.EmbedderProvider,
	convStore *store.ConversationStore,
	usageTracker *usage.Service,
) (*Server, error) {
	// Create server options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(config.MaxMessageSize),
		grpc.MaxSendMsgSize(config.MaxMessageSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     config.MaxConnectionIdle,
			MaxConnectionAge:      config.MaxConnectionAge,
			MaxConnectionAgeGrace: 10 * time.Second,
			Time:                  config.KeepAliveTime,
			Timeout:               config.KeepAliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	// Add interceptors
	opts = append(opts, grpc.ChainUnaryInterceptor(
		loggingInterceptor,
		recoveryInterceptor,
	))

	if config.EnableAuth {
		opts = append(opts, grpc.ChainUnaryInterceptor(authInterceptor(config.AuthToken)))
		opts = append(opts, grpc.StreamInterceptor(streamAuthInterceptor(config.AuthToken)))
	}

	// Add TLS if enabled
	if config.EnableTLS {
		creds, err := credentials.NewServerTLSFromFile(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(opts...)

	// Create service servers
	ragService := NewSimpleRAGServiceServer(processor)
	llmService := NewLLMServiceServer(llmProvider)
	embeddingService := NewEmbeddingServiceServer(embedder)
	conversationService := NewConversationService(convStore)
	usageService := NewUsageService(usageTracker)

	// Register services
	pb.RegisterRAGServiceServer(grpcServer, ragService)
	pb.RegisterLLMServiceServer(grpcServer, llmService)
	pb.RegisterEmbeddingServiceServer(grpcServer, embeddingService)
	pb.RegisterConversationServiceServer(grpcServer, conversationService)
	pb.RegisterUsageServiceServer(grpcServer, usageService)

	// Enable reflection for debugging
	if config.EnableReflection {
		reflection.Register(grpcServer)
	}

	return &Server{
		config:              config,
		grpcServer:          grpcServer,
		ragService:          ragService,
		llmService:          llmService,
		embeddingService:    embeddingService,
		conversationService: conversationService,
		usageService:        usageService,
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.config.Port, err)
	}

	fmt.Printf("gRPC server starting on port %d...\n", s.config.Port)
	if s.config.EnableTLS {
		fmt.Println("TLS enabled")
	}
	if s.config.EnableAuth {
		fmt.Println("Authentication enabled")
	}
	if s.config.EnableReflection {
		fmt.Println("Reflection enabled")
	}

	// Start serving
	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	fmt.Println("Stopping gRPC server...")
	s.grpcServer.GracefulStop()
}

// Health checks the health of the server
func (s *Server) Health(ctx context.Context) error {
	// Check if server is serving
	if s.grpcServer == nil {
		return fmt.Errorf("server not initialized")
	}

	// You could add more health checks here
	return nil
}