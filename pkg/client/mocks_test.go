// Package client - mocks_test.go
// This file provides comprehensive mock implementations for all four RAGO pillar services
// to enable thorough testing of the client package without external dependencies.

package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// ===== MOCK LLM SERVICE =====

type MockLLMService struct {
	mu                sync.RWMutex
	providers         map[string]core.ProviderConfig
	providerHealths   map[string]core.HealthStatus
	generateFunc      func(ctx context.Context, req core.GenerationRequest) (*core.GenerationResponse, error)
	streamFunc        func(ctx context.Context, req core.GenerationRequest, callback core.StreamCallback) error
	generateWithToolsFunc func(ctx context.Context, req core.ToolGenerationRequest) (*core.ToolGenerationResponse, error)
	streamWithToolsFunc   func(ctx context.Context, req core.ToolGenerationRequest, callback core.ToolStreamCallback) error
	generateBatchFunc func(ctx context.Context, requests []core.GenerationRequest) ([]core.GenerationResponse, error)
	closed            bool
}

func NewMockLLMService() *MockLLMService {
	return &MockLLMService{
		providers:       make(map[string]core.ProviderConfig),
		providerHealths: make(map[string]core.HealthStatus),
	}
}

func (m *MockLLMService) AddProvider(name string, config core.ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	m.providers[name] = config
	m.providerHealths[name] = core.HealthStatusHealthy
	return nil
}

func (m *MockLLMService) RemoveProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	delete(m.providers, name)
	delete(m.providerHealths, name)
	return nil
}

func (m *MockLLMService) ListProviders() []core.ProviderInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil
	}
	
	var providers []core.ProviderInfo
	for name, config := range m.providers {
		providers = append(providers, core.ProviderInfo{
			Name:   name,
			Type:   config.Type,
			Model:  config.Model,
			Health: m.providerHealths[name],
		})
	}
	return providers
}

func (m *MockLLMService) GetProviderHealth() map[string]core.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil
	}
	
	healthCopy := make(map[string]core.HealthStatus)
	for name, health := range m.providerHealths {
		healthCopy[name] = health
	}
	return healthCopy
}

func (m *MockLLMService) Generate(ctx context.Context, req core.GenerationRequest) (*core.GenerationResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	
	// Default mock response
	return &core.GenerationResponse{
		Content: fmt.Sprintf("Mock response to: %s", req.Prompt),
		Usage: core.TokenUsage{
			PromptTokens:  len(req.Prompt) / 4,  // Rough estimate
			CompletionTokens: 50,
			TotalTokens:  len(req.Prompt)/4 + 50,
		},
		Provider: "mock-provider",
		Model:    "mock-model",
		Duration: 100 * time.Millisecond,
	}, nil
}

func (m *MockLLMService) Stream(ctx context.Context, req core.GenerationRequest, callback core.StreamCallback) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	if m.streamFunc != nil {
		return m.streamFunc(ctx, req, callback)
	}
	
	// Default mock streaming response
	response := fmt.Sprintf("Mock response to: %s", req.Prompt)
	for i, char := range response {
		if err := callback(core.StreamChunk{
			Content:  string(char),
			Finished: i == len(response)-1,
		}); err != nil {
			return err
		}
		
		// Simulate streaming delay
		time.Sleep(10 * time.Millisecond)
	}
	
	return nil
}

func (m *MockLLMService) GenerateWithTools(ctx context.Context, req core.ToolGenerationRequest) (*core.ToolGenerationResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if m.generateWithToolsFunc != nil {
		return m.generateWithToolsFunc(ctx, req)
	}
	
	// Default mock tool response
	return &core.ToolGenerationResponse{
		GenerationResponse: core.GenerationResponse{
			Content: fmt.Sprintf("Mock tool response to: %s", req.Prompt),
			Usage: core.TokenUsage{
				PromptTokens:  len(req.Prompt) / 4,
				CompletionTokens: 75,
				TotalTokens:  len(req.Prompt)/4 + 75,
			},
			Provider: "mock-provider",
			Model:    "mock-model",
			Duration: 150 * time.Millisecond,
		},
		ToolCalls: []core.ToolCall{
			{
				ID:         "mock-tool-call-1",
				Name:       "mock_function",
				Parameters: map[string]interface{}{"query": "mock query"},
			},
		},
	}, nil
}

func (m *MockLLMService) StreamWithTools(ctx context.Context, req core.ToolGenerationRequest, callback core.ToolStreamCallback) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	if m.streamWithToolsFunc != nil {
		return m.streamWithToolsFunc(ctx, req, callback)
	}
	
	// Default mock tool streaming response
	return callback(core.ToolStreamChunk{
		StreamChunk: core.StreamChunk{
			Content:  "Mock streaming tool response",
			Finished: true,
		},
		ToolCalls: []core.ToolCall{
			{
				ID:         "stream-tool-1",
				Name:       "mock_stream_function",
				Parameters: map[string]interface{}{"stream": "true"},
			},
		},
	})
}

func (m *MockLLMService) GenerateBatch(ctx context.Context, requests []core.GenerationRequest) ([]core.GenerationResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if m.generateBatchFunc != nil {
		return m.generateBatchFunc(ctx, requests)
	}
	
	// Default mock batch response
	var responses []core.GenerationResponse
	for _, req := range requests {
		responses = append(responses, core.GenerationResponse{
			Content:  fmt.Sprintf("Batch mock response to: %s", req.Prompt),
			Provider: "mock-provider",
			Model:    "mock-model",
			Usage: core.TokenUsage{
				PromptTokens:  len(req.Prompt) / 4,
				CompletionTokens: 40,
				TotalTokens:  len(req.Prompt)/4 + 40,
			},
		})
	}
	
	return responses, nil
}

func (m *MockLLMService) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// Mock configuration methods
func (m *MockLLMService) SetGenerateFunc(fn func(ctx context.Context, req core.GenerationRequest) (*core.GenerationResponse, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateFunc = fn
}

func (m *MockLLMService) SetProviderHealth(name string, health core.HealthStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providerHealths[name] = health
}

// ===== MOCK RAG SERVICE =====

type MockRAGService struct {
	mu              sync.RWMutex
	documents       map[string]core.Document
	chunks          map[string][]DocumentChunk
	stats           core.RAGStats
	ingestFunc      func(ctx context.Context, req core.IngestRequest) (*core.IngestResponse, error)
	searchFunc      func(ctx context.Context, req core.SearchRequest) (*core.SearchResponse, error)
	hybridSearchFunc func(ctx context.Context, req core.HybridSearchRequest) (*core.HybridSearchResponse, error)
	closed          bool
}

func NewMockRAGService() *MockRAGService {
	return &MockRAGService{
		documents: make(map[string]core.Document),
		chunks:    make(map[string][]DocumentChunk),
		stats: core.RAGStats{
			TotalDocuments:  0,
			TotalChunks:     0,
			StorageSize:     0,
			LastOptimized:   time.Now(),
		},
	}
}

func (m *MockRAGService) IngestDocument(ctx context.Context, req core.IngestRequest) (*core.IngestResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if m.ingestFunc != nil {
		return m.ingestFunc(ctx, req)
	}
	
	// Default mock ingestion
	docID := req.DocumentID
	if docID == "" {
		docID = fmt.Sprintf("doc-%d", time.Now().UnixNano())
	}
	
	doc := core.Document{
		ID:          docID,
		Content:     req.Content,
		Metadata:    req.Metadata,
		ContentType: req.ContentType,
		CreatedAt:  time.Now(),
	}
	
	m.documents[docID] = doc
	
	// Create mock chunks
	chunkCount := len(req.Content) / 100 + 1
	chunks := make([]DocumentChunk, chunkCount)
	for i := 0; i < chunkCount; i++ {
		start := i * 100
		end := (i + 1) * 100
		if end > len(req.Content) {
			end = len(req.Content)
		}
		
		chunks[i] = DocumentChunk{
			ID:         fmt.Sprintf("%s-chunk-%d", docID, i),
			DocumentID: docID,
			Content:    req.Content[start:end],
			Index:      i,
			Metadata:   req.Metadata,
		}
	}
	m.chunks[docID] = chunks
	
	// Update stats
	m.stats.TotalDocuments++
	m.stats.TotalChunks += chunkCount
	m.stats.StorageSize += int64(len(req.Content))
	
	return &core.IngestResponse{
		DocumentID:  docID,
		ChunksCount: chunkCount,
		ProcessedAt: time.Now(),
		Duration:    100 * time.Millisecond,
	}, nil
}

func (m *MockRAGService) IngestBatch(ctx context.Context, requests []core.IngestRequest) (*core.BatchIngestResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	var responses []core.IngestResponse
	var totalChunks int
	
	for _, req := range requests {
		resp, err := m.IngestDocument(ctx, req)
		if err != nil {
			responses = append(responses, core.IngestResponse{
				DocumentID: req.DocumentID,
				ChunksCount: 0,
				ProcessedAt: time.Now(),
			})
		} else {
			responses = append(responses, *resp)
			totalChunks += resp.ChunksCount
		}
	}
	
	return &core.BatchIngestResponse{
		Responses:       responses,
		TotalDocuments:  len(requests),
		SuccessfulCount: len(responses),
		Duration:        100 * time.Millisecond,
	}, nil
}

func (m *MockRAGService) DeleteDocument(ctx context.Context, docID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	if _, exists := m.documents[docID]; !exists {
		return fmt.Errorf("document not found: %s", docID)
	}
	
	// Update stats
	chunks := m.chunks[docID]
	m.stats.TotalChunks -= len(chunks)
	m.stats.StorageSize -= int64(len(m.documents[docID].Content))
	m.stats.TotalDocuments--
	
	delete(m.documents, docID)
	delete(m.chunks, docID)
	
	return nil
}

func (m *MockRAGService) ListDocuments(ctx context.Context, filter core.DocumentFilter) ([]core.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	var docs []core.Document
	count := 0
	
	for _, doc := range m.documents {
		if count >= filter.Offset && len(docs) < filter.Limit {
			docs = append(docs, doc)
		}
		count++
	}
	
	return docs, nil
}

func (m *MockRAGService) Search(ctx context.Context, req core.SearchRequest) (*core.SearchResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if m.searchFunc != nil {
		return m.searchFunc(ctx, req)
	}
	
	// Default mock search
	var results []core.SearchResult
	count := 0
	
	for docID, chunks := range m.chunks {
		if count >= req.Limit {
			break
		}
		
		for _, chunk := range chunks {
			if count >= req.Limit {
				break
			}
			
			// Simple mock relevance based on query presence
			score := 0.8
			if req.Query != "" && len(chunk.Content) > 0 {
				// Simulate relevance scoring
				if chunk.Content[:minInt(len(chunk.Content), len(req.Query))] == req.Query[:minInt(len(req.Query), len(chunk.Content))] {
					score = 0.95
				}
			}
			
			results = append(results, core.SearchResult{
				DocumentID: docID,
				ChunkID:    chunk.ID,
				Content:    chunk.Content,
				Score:      float32(score),
				Metadata:   chunk.Metadata,
			})
			count++
		}
	}
	
	return &core.SearchResponse{
		Results:  results,
		Query:    req.Query,
		Duration: 50 * time.Millisecond,
	}, nil
}

func (m *MockRAGService) HybridSearch(ctx context.Context, req core.HybridSearchRequest) (*core.HybridSearchResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if m.hybridSearchFunc != nil {
		return m.hybridSearchFunc(ctx, req)
	}
	
	// Mock hybrid search by combining vector and keyword results
	vectorResp, _ := m.Search(ctx, core.SearchRequest{
		Query: req.Query,
		Limit: req.Limit,
	})
	
	return &core.HybridSearchResponse{
		SearchResponse: core.SearchResponse{
			Results:  vectorResp.Results,
			Query:    req.Query,
			Duration: 75 * time.Millisecond,
		},
		VectorResults:  vectorResp.Results[:minInt(len(vectorResp.Results), req.Limit/2)],
		KeywordResults: vectorResp.Results[minInt(len(vectorResp.Results), req.Limit/2):],
		FusionMethod:   "RRF",
	}, nil
}

func (m *MockRAGService) GetStats(ctx context.Context) (*core.RAGStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	return &m.stats, nil
}

func (m *MockRAGService) Optimize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	m.stats.LastOptimized = time.Now()
	return nil
}

func (m *MockRAGService) Reset(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	m.documents = make(map[string]core.Document)
	m.chunks = make(map[string][]DocumentChunk)
	m.stats = core.RAGStats{
		TotalDocuments: 0,
		TotalChunks:    0,
		StorageSize:    0,
		LastOptimized:  time.Now(),
	}
	
	return nil
}

func (m *MockRAGService) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// Mock configuration methods
func (m *MockRAGService) SetSearchFunc(fn func(ctx context.Context, req core.SearchRequest) (*core.SearchResponse, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchFunc = fn
}

func (m *MockRAGService) AddMockDocument(doc core.Document) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.documents[doc.ID] = doc
	m.stats.TotalDocuments++
}

// ===== MOCK MCP SERVICE =====

type MockMCPService struct {
	mu            sync.RWMutex
	servers       map[string]core.ServerInfo
	serverHealths map[string]core.HealthStatus
	tools         map[string]core.ToolInfo
	callToolFunc  func(ctx context.Context, req core.ToolCallRequest) (*core.ToolCallResponse, error)
	closed        bool
}

func NewMockMCPService() *MockMCPService {
	return &MockMCPService{
		servers:       make(map[string]core.ServerInfo),
		serverHealths: make(map[string]core.HealthStatus),
		tools:         make(map[string]core.ToolInfo),
	}
}

func (m *MockMCPService) RegisterServer(config core.ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	m.servers[config.Name] = core.ServerInfo{
		Name:        config.Name,
		Description: config.Description,
		Status:      "running",
	}
	m.serverHealths[config.Name] = core.HealthStatusHealthy
	
	return nil
}

func (m *MockMCPService) UnregisterServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	delete(m.servers, name)
	delete(m.serverHealths, name)
	return nil
}

func (m *MockMCPService) ListServers() []core.ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil
	}
	
	var servers []core.ServerInfo
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	return servers
}

func (m *MockMCPService) GetServerHealth(name string) core.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return core.HealthStatusUnknown
	}
	
	if health, exists := m.serverHealths[name]; exists {
		return health
	}
	return core.HealthStatusUnknown
}

func (m *MockMCPService) ListTools() []core.ToolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil
	}
	
	var tools []core.ToolInfo
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (m *MockMCPService) GetTool(name string) (*core.ToolInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if tool, exists := m.tools[name]; exists {
		return &tool, nil
	}
	return nil, fmt.Errorf("tool not found: %s", name)
}

func (m *MockMCPService) CallTool(ctx context.Context, req core.ToolCallRequest) (*core.ToolCallResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if m.callToolFunc != nil {
		return m.callToolFunc(ctx, req)
	}
	
	// Default mock tool call
	return &core.ToolCallResponse{
		Result: fmt.Sprintf("Mock tool result for %s with args %v", req.ToolName, req.Arguments),
		Success: true,
		Duration: 100 * time.Millisecond,
	}, nil
}

func (m *MockMCPService) CallToolAsync(ctx context.Context, req core.ToolCallRequest) (<-chan *core.ToolCallResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	respChan := make(chan *core.ToolCallResponse, 1)
	
	go func() {
		defer close(respChan)
		
		// Simulate async work
		time.Sleep(50 * time.Millisecond)
		
		respChan <- &core.ToolCallResponse{
			Result: fmt.Sprintf("Async mock result for %s", req.ToolName),
			Success: true,
			Duration: 50 * time.Millisecond,
		}
	}()
	
	return respChan, nil
}

func (m *MockMCPService) CallToolsBatch(ctx context.Context, requests []core.ToolCallRequest) ([]core.ToolCallResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	var responses []core.ToolCallResponse
	for _, req := range requests {
		resp, _ := m.CallTool(ctx, req)
		responses = append(responses, *resp)
	}
	
	return responses, nil
}

func (m *MockMCPService) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// Mock configuration methods
func (m *MockMCPService) AddMockTool(tool core.ToolInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name] = tool
}

func (m *MockMCPService) SetServerHealth(name string, health core.HealthStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serverHealths[name] = health
}

func (m *MockMCPService) SetCallToolFunc(fn func(ctx context.Context, req core.ToolCallRequest) (*core.ToolCallResponse, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callToolFunc = fn
}

// ===== MOCK AGENT SERVICE =====

type MockAgentService struct {
	mu             sync.RWMutex
	workflows      map[string]core.WorkflowInfo
	agents         map[string]core.AgentInfo
	scheduledTasks []core.ScheduledTask
	executeWorkflowFunc func(ctx context.Context, req core.WorkflowRequest) (*core.WorkflowResponse, error)
	executeAgentFunc    func(ctx context.Context, req core.AgentRequest) (*core.AgentResponse, error)
	closed         bool
}

func NewMockAgentService() *MockAgentService {
	return &MockAgentService{
		workflows:      make(map[string]core.WorkflowInfo),
		agents:         make(map[string]core.AgentInfo),
		scheduledTasks: make([]core.ScheduledTask, 0),
	}
}

func (m *MockAgentService) CreateWorkflow(definition core.WorkflowDefinition) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	m.workflows[definition.Name] = core.WorkflowInfo{
		Name:        definition.Name,
		Description: definition.Description,
		StepsCount:  0,
		CreatedAt:   time.Now(),
	}
	
	return nil
}

func (m *MockAgentService) ExecuteWorkflow(ctx context.Context, req core.WorkflowRequest) (*core.WorkflowResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if m.executeWorkflowFunc != nil {
		return m.executeWorkflowFunc(ctx, req)
	}
	
	// Default mock workflow execution
	return &core.WorkflowResponse{
		WorkflowName: req.WorkflowName,
		Status:       "completed",
		Steps: []core.StepResult{
			{
				StepID:   "step-1",
				Status:   "completed",
				Output:   "Mock step 1 completed",
				Duration: 50 * time.Millisecond,
			},
			{
				StepID:   "step-2",
				Status:   "completed",
				Output:   "Mock step 2 completed",
				Duration: 75 * time.Millisecond,
			},
		},
		Duration: 125 * time.Millisecond,
	}, nil
}

func (m *MockAgentService) ListWorkflows() []core.WorkflowInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil
	}
	
	var workflows []core.WorkflowInfo
	for _, workflow := range m.workflows {
		workflows = append(workflows, workflow)
	}
	return workflows
}

func (m *MockAgentService) DeleteWorkflow(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	delete(m.workflows, name)
	return nil
}

func (m *MockAgentService) CreateAgent(definition core.AgentDefinition) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	m.agents[definition.Name] = core.AgentInfo{
		Name:        definition.Name,
		Type:        definition.Type,
		Description: definition.Description,
		ToolsCount:  0,
		CreatedAt:   time.Now(),
	}
	
	return nil
}

func (m *MockAgentService) ExecuteAgent(ctx context.Context, req core.AgentRequest) (*core.AgentResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("service is closed")
	}
	
	if m.executeAgentFunc != nil {
		return m.executeAgentFunc(ctx, req)
	}
	
	// Default mock agent execution
	return &core.AgentResponse{
		AgentName: req.AgentName,
		Result:    fmt.Sprintf("Mock agent %s completed task: %s", req.AgentName, req.Task),
		Steps: []core.AgentStep{
			{
				StepNumber: 1,
				Action:     "analyze",
				Output:     "Task analyzed",
				Duration:   30 * time.Millisecond,
			},
			{
				StepNumber: 2,
				Action:     "execute",
				Output:     "Task executed",
				Duration:   70 * time.Millisecond,
			},
		},
		Duration: 100 * time.Millisecond,
	}, nil
}

func (m *MockAgentService) ListAgents() []core.AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil
	}
	
	var agents []core.AgentInfo
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
}

func (m *MockAgentService) DeleteAgent(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	delete(m.agents, name)
	return nil
}

func (m *MockAgentService) ScheduleWorkflow(name string, schedule core.ScheduleConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("service is closed")
	}
	
	task := core.ScheduledTask{
		ID:           fmt.Sprintf("task-%d", time.Now().UnixNano()),
		WorkflowName: name,
		Schedule:     schedule,
		Status:       "scheduled",
		NextRun:      time.Now().Add(24 * time.Hour),
	}
	
	m.scheduledTasks = append(m.scheduledTasks, task)
	return nil
}

func (m *MockAgentService) GetScheduledTasks() []core.ScheduledTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil
	}
	
	return append([]core.ScheduledTask(nil), m.scheduledTasks...)
}

func (m *MockAgentService) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// Mock configuration methods
func (m *MockAgentService) AddMockWorkflow(workflow core.WorkflowInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflows[workflow.Name] = workflow
}

func (m *MockAgentService) AddMockAgent(agent core.AgentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[agent.Name] = agent
}

func (m *MockAgentService) SetExecuteAgentFunc(fn func(ctx context.Context, req core.AgentRequest) (*core.AgentResponse, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeAgentFunc = fn
}

// ===== UTILITY FUNCTIONS =====

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DocumentChunk represents a document chunk for RAG operations
type DocumentChunk struct {
	ID         string                 `json:"id"`
	DocumentID string                 `json:"document_id"`
	Content    string                 `json:"content"`
	Index      int                    `json:"index"`
	Metadata   map[string]interface{} `json:"metadata"`
}