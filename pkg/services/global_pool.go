package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/liliang-cn/agent-go/pkg/domain"
	"github.com/liliang-cn/agent-go/pkg/pool"
)

var (
	globalPoolService *GlobalPoolService
	globalPoolMu      sync.RWMutex
)

// GlobalPoolService 管理全局LLM和Embedding Pools
type GlobalPoolService struct {
	config        *config.Config
	llmPool       *pool.Pool
	embeddingPool *pool.Pool
	initialized   bool
	mu            sync.RWMutex
}

// GetGlobalPoolService 获取全局pool服务
func GetGlobalPoolService() *GlobalPoolService {
	globalPoolMu.RLock()
	if globalPoolService != nil {
		globalPoolMu.RUnlock()
		return globalPoolService
	}
	globalPoolMu.RUnlock()

	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()

	if globalPoolService != nil {
		return globalPoolService
	}

	globalPoolService = &GlobalPoolService{}
	return globalPoolService
}

// Initialize 初始化pool
func (s *GlobalPoolService) Initialize(ctx context.Context, cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	s.config = cfg

	// 2. LLM Pool
	llmPool, err := pool.NewPool(pool.PoolConfig{
		Enabled:   cfg.LLM.Enabled,
		Strategy:  cfg.LLM.Strategy,
		Providers: cfg.LLM.Providers,
	})
	if err != nil {
		return fmt.Errorf("failed to create LLM pool: %w", err)
	}
	s.llmPool = llmPool

	// 3. Embedding Pool
	embeddingPool, err := pool.NewPool(pool.PoolConfig{
		Enabled:   cfg.RAG.Embedding.Enabled,
		Strategy:  cfg.RAG.Embedding.Strategy,
		Providers: cfg.RAG.Embedding.Providers,
	})
	if err != nil {
		return fmt.Errorf("failed to create embedding pool: %w", err)
	}
	s.embeddingPool = embeddingPool

	s.initialized = true
	return nil
}

// GetLLM 获取LLM client（自动选择）
func (s *GlobalPoolService) GetLLM() (*pool.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.llmPool.Get()
}

// GetLLMByName 按名称获取LLM client
func (s *GlobalPoolService) GetLLMByName(name string) (*pool.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.llmPool.GetByName(name)
}

// GetLLMByCapability 按能力等级获取LLM client
func (s *GlobalPoolService) GetLLMByCapability(minCapability int) (*pool.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.llmPool.GetByCapability(minCapability)
}

// ReleaseLLM 释放LLM client
func (s *GlobalPoolService) ReleaseLLM(client *pool.Client) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.initialized {
		s.llmPool.Release(client)
	}
}

// GetEmbedding 获取Embedding client（自动选择）
func (s *GlobalPoolService) GetEmbedding() (*pool.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.embeddingPool.Get()
}

// GetEmbeddingByName 按名称获取Embedding client
func (s *GlobalPoolService) GetEmbeddingByName(name string) (*pool.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.embeddingPool.GetByName(name)
}

// GetEmbeddingByCapability 按能力等级获取Embedding client
func (s *GlobalPoolService) GetEmbeddingByCapability(minCapability int) (*pool.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.embeddingPool.GetByCapability(minCapability)
}

// ReleaseEmbedding 释放Embedding client
func (s *GlobalPoolService) ReleaseEmbedding(client *pool.Client) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.initialized {
		s.embeddingPool.Release(client)
	}
}

// Generate 使用pool生成文本（自动获取和释放）
func (s *GlobalPoolService) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return "", fmt.Errorf("pool service not initialized")
	}

	return s.llmPool.Generate(ctx, prompt, opts)
}

// GenerateWithTools 使用pool和工具生成
func (s *GlobalPoolService) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.llmPool.GenerateWithTools(ctx, messages, tools, opts)
}

// GenerateStructured 使用pool生成结构化输出
func (s *GlobalPoolService) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.llmPool.GenerateStructured(ctx, prompt, schema, opts)
}

// RecognizeIntent 使用pool识别意图
func (s *GlobalPoolService) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.llmPool.RecognizeIntent(ctx, request)
}

// Stream 使用pool流式生成
func (s *GlobalPoolService) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return fmt.Errorf("pool service not initialized")
	}

	return s.llmPool.Stream(ctx, prompt, opts, callback)
}

// StreamWithTools 使用pool和工具流式生成
func (s *GlobalPoolService) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return fmt.Errorf("pool service not initialized")
	}

	return s.llmPool.StreamWithTools(ctx, messages, tools, opts, callback)
}

// Embed 使用pool向量化
func (s *GlobalPoolService) Embed(ctx context.Context, text string) ([]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.embeddingPool.Embed(ctx, text)
}

// EmbedMultiple 使用pool向量化多个文本
func (s *GlobalPoolService) EmbedMultiple(ctx context.Context, texts []string) ([][]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}

	return s.embeddingPool.EmbedMultiple(ctx, texts)
}

// EmbedBatch 使用pool批量向量化（实现 domain.Embedder 接口）
func (s *GlobalPoolService) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	return s.EmbedMultiple(ctx, texts)
}

// GetLLMStatus 获取LLM pool状态
func (s *GlobalPoolService) GetLLMStatus() map[string]pool.ClientStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil
	}

	return s.llmPool.GetStatus()
}

// GetEmbeddingStatus 获取Embedding pool状态
func (s *GlobalPoolService) GetEmbeddingStatus() map[string]pool.ClientStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil
	}

	return s.embeddingPool.GetStatus()
}

// IsInitialized 是否已初始化
func (s *GlobalPoolService) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// Close 关闭pool
func (s *GlobalPoolService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil
	}

	if s.llmPool != nil {
		s.llmPool.Close()
	}
	if s.embeddingPool != nil {
		s.embeddingPool.Close()
	}

	s.initialized = false
	return nil
}

// Shutdown 关闭并清理全局pool
func (s *GlobalPoolService) Shutdown() error {
	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()

	if err := s.Close(); err != nil {
		return err
	}

	globalPoolService = nil
	return nil
}

// ===== 兼容层 - 让旧代码继续工作 =====

// llmServiceWrapper 包装Pool为domain.Generator
type llmServiceWrapper struct {
	pool *pool.Pool
	hint pool.SelectionHint
}

func (w *llmServiceWrapper) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	client, err := w.pool.GetWithHint(w.hint)
	if err != nil {
		return "", err
	}
	defer w.pool.Release(client)
	return client.Generate(ctx, prompt, opts)
}

func (w *llmServiceWrapper) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	client, err := w.pool.GetWithHint(w.hint)
	if err != nil {
		return err
	}
	defer w.pool.Release(client)
	return client.Stream(ctx, prompt, opts, callback)
}

func (w *llmServiceWrapper) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	client, err := w.pool.GetWithHint(w.hint)
	if err != nil {
		return nil, err
	}
	defer w.pool.Release(client)
	return client.GenerateWithTools(ctx, messages, tools, opts)
}

func (w *llmServiceWrapper) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	client, err := w.pool.GetWithHint(w.hint)
	if err != nil {
		return err
	}
	defer w.pool.Release(client)
	return client.StreamWithTools(ctx, messages, tools, opts, callback)
}

func (w *llmServiceWrapper) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	client, err := w.pool.GetWithHint(w.hint)
	if err != nil {
		return nil, err
	}
	defer w.pool.Release(client)
	return client.GenerateStructured(ctx, prompt, schema, opts)
}

func (w *llmServiceWrapper) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	client, err := w.pool.GetWithHint(w.hint)
	if err != nil {
		return nil, err
	}
	defer w.pool.Release(client)
	return client.RecognizeIntent(ctx, request)
}

func (w *llmServiceWrapper) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	return w.pool.ExtractMetadataWithHint(ctx, w.hint, content, model)
}

// embeddingServiceWrapper 包装Pool为domain.Embedder
type embeddingServiceWrapper struct {
	pool *pool.Pool
}

func (w *embeddingServiceWrapper) Embed(ctx context.Context, text string) ([]float64, error) {
	return w.pool.Embed(ctx, text)
}

func (w *embeddingServiceWrapper) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	return w.pool.EmbedMultiple(ctx, texts)
}

// GetGlobalLLM 获取全局LLM服务（兼容旧代码）
func GetGlobalLLM() (domain.Generator, error) {
	service := GetGlobalPoolService()
	if !service.IsInitialized() {
		return nil, fmt.Errorf("pool service not initialized")
	}
	return &llmServiceWrapper{pool: service.llmPool}, nil
}

// GetGlobalEmbeddingService 获取全局Embedding服务（兼容旧代码）
func GetGlobalEmbeddingService(ctx context.Context) (domain.Embedder, error) {
	service := GetGlobalPoolService()
	if !service.IsInitialized() {
		return nil, fmt.Errorf("pool service not initialized")
	}
	return &embeddingServiceWrapper{pool: service.embeddingPool}, nil
}

// GetGlobalLLMService 获取全局LLM Service（兼容旧代码）
// 这个函数返回GlobalPoolService，兼容旧的GetGlobalLLMService()调用
func GetGlobalLLMService() *GlobalPoolService {
	return GetGlobalPoolService()
}

// GetLLMService 获取LLM服务（兼容旧代码）
func (s *GlobalPoolService) GetLLMService() (domain.Generator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}
	return &llmServiceWrapper{pool: s.llmPool}, nil
}

func (s *GlobalPoolService) GetLLMServiceWithHint(hint pool.SelectionHint) (domain.Generator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}
	return &llmServiceWrapper{pool: s.llmPool, hint: hint}, nil
}

// GetEmbeddingService 获取Embedding服务（兼容旧代码）
func (s *GlobalPoolService) GetEmbeddingService(ctx context.Context) (domain.Embedder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, fmt.Errorf("pool service not initialized")
	}
	return &embeddingServiceWrapper{pool: s.embeddingPool}, nil
}
