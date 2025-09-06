# RAGO Architecture Transformation Plan
## From RAG-Focused to Four-Pillar AI Foundation

**Version**: 1.0  
**Date**: 2025-09-06  
**Vision**: Transform RAGO into a local-first, privacy-first Go AI foundation with four equal pillars

---

## Executive Summary

RAGO will evolve from a primarily RAG-focused system with optional agent capabilities into a true four-pillar AI foundation. The transformation will establish **LLM**, **RAG**, **MCP**, and **AGENT** as equal architectural pillars, each capable of independent operation while providing powerful synergies when combined.

### Core Principles
- **Library-First Design**: Each pillar must be usable as an independent Go library
- **Local-First**: Full functionality without cloud dependencies
- **Privacy-First**: All data stays local by default
- **Production-Ready**: Enterprise-grade reliability and performance
- **Extensible**: Clean interfaces for future enhancements

---

## 1. Current State Assessment

### Existing Strengths
- ✅ Solid RAG foundation with SQLite vectors and keyword search
- ✅ Multi-provider LLM support (Ollama, OpenAI, LM Studio)
- ✅ MCP protocol integration framework
- ✅ Agent workflow system foundation
- ✅ Clean domain models and interfaces
- ✅ Comprehensive configuration system

### Architecture Gaps
- ❌ **RAG-Centric**: Other pillars are secondary features
- ❌ **Monolithic Client**: Single client aggregates everything
- ❌ **Unequal Pillars**: RAG dominates, others are optional
- ❌ **Limited Independence**: Components tightly coupled
- ❌ **Provider Limitations**: No load balancing or pooling
- ❌ **Tool Integration**: MCP tools not seamlessly integrated

### Current Package Structure Issues
```
pkg/
├── processor/      # RAG-heavy, contains everything
├── providers/      # Simple factory pattern
├── mcp/           # Basic MCP client
├── agents/        # Agent system but underutilized
├── store/         # RAG storage only
└── tools/         # Overlaps with MCP
```

---

## 2. Target Architecture: Four-Pillar Foundation

### 2.1 Pillar Equality Principle

Each pillar must be:
- **Self-Contained**: Complete functionality within its package
- **Independently Testable**: Full test coverage without dependencies
- **Library-Ready**: Usable as standalone Go library
- **Well-Documented**: Clear interfaces and examples
- **Production-Grade**: Error handling, logging, metrics

### 2.2 The Four Pillars

#### **LLM Pillar** (`pkg/llm/`)
**Purpose**: Provider management, load balancing, capability detection

**Core Features**:
- Provider pools with health checking
- Round-robin and weighted load balancing  
- Automatic failover and circuit breaking
- Streaming and batch operations
- Tool calling coordination
- Performance monitoring and metrics

**Key Interfaces**:
```go
type LLMService interface {
    // Provider Management
    AddProvider(name string, config ProviderConfig) error
    RemoveProvider(name string) error
    ListProviders() []ProviderInfo
    GetProviderHealth() map[string]HealthStatus
    
    // Generation Operations  
    Generate(ctx context.Context, req GenerationRequest) (*GenerationResponse, error)
    Stream(ctx context.Context, req GenerationRequest, callback StreamCallback) error
    
    // Tool Operations
    GenerateWithTools(ctx context.Context, req ToolGenerationRequest) (*ToolGenerationResponse, error)
    StreamWithTools(ctx context.Context, req ToolGenerationRequest, callback ToolStreamCallback) error
    
    // Batch Operations
    GenerateBatch(ctx context.Context, requests []GenerationRequest) ([]GenerationResponse, error)
}
```

#### **RAG Pillar** (`pkg/rag/`)
**Purpose**: Document ingestion, chunking, storage, retrieval

**Core Features**:
- Multi-format document processing
- Smart chunking strategies
- Dual storage (vector + keyword) with RRF fusion
- Metadata extraction and enrichment
- Document lifecycle management
- Search optimization

**Key Interfaces**:
```go
type RAGService interface {
    // Document Operations
    IngestDocument(ctx context.Context, req IngestRequest) (*IngestResponse, error)
    IngestBatch(ctx context.Context, requests []IngestRequest) (*BatchIngestResponse, error)
    DeleteDocument(ctx context.Context, docID string) error
    ListDocuments(ctx context.Context, filter DocumentFilter) ([]Document, error)
    
    // Search Operations
    Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
    HybridSearch(ctx context.Context, req HybridSearchRequest) (*HybridSearchResponse, error)
    
    // Management Operations
    GetStats(ctx context.Context) (*RAGStats, error)
    Optimize(ctx context.Context) error
    Reset(ctx context.Context) error
}
```

#### **MCP Pillar** (`pkg/mcp/`)
**Purpose**: Tool integration, external service coordination

**Core Features**:
- MCP server lifecycle management
- Tool discovery and registration
- Health monitoring and recovery
- Concurrent tool execution
- Error handling and retries
- Tool performance analytics

**Key Interfaces**:
```go
type MCPService interface {
    // Server Management
    RegisterServer(config ServerConfig) error
    UnregisterServer(name string) error
    ListServers() []ServerInfo
    GetServerHealth(name string) HealthStatus
    
    // Tool Operations
    ListTools() []ToolInfo
    GetTool(name string) (*ToolInfo, error)
    CallTool(ctx context.Context, req ToolCallRequest) (*ToolCallResponse, error)
    CallToolAsync(ctx context.Context, req ToolCallRequest) (<-chan *ToolCallResponse, error)
    
    // Batch Operations
    CallToolsBatch(ctx context.Context, requests []ToolCallRequest) ([]ToolCallResponse, error)
}
```

#### **AGENT Pillar** (`pkg/agents/`)
**Purpose**: Workflow orchestration, multi-step reasoning

**Core Features**:
- Workflow definition and execution
- Agent lifecycle management
- Task scheduling and queuing
- State persistence and recovery
- Multi-step reasoning chains
- Performance monitoring

**Key Interfaces**:
```go
type AgentService interface {
    // Workflow Management
    CreateWorkflow(definition WorkflowDefinition) error
    ExecuteWorkflow(ctx context.Context, req WorkflowRequest) (*WorkflowResponse, error)
    ListWorkflows() []WorkflowInfo
    DeleteWorkflow(name string) error
    
    // Agent Management
    CreateAgent(definition AgentDefinition) error
    ExecuteAgent(ctx context.Context, req AgentRequest) (*AgentResponse, error)
    ListAgents() []AgentInfo
    DeleteAgent(name string) error
    
    // Scheduling
    ScheduleWorkflow(name string, schedule ScheduleConfig) error
    GetScheduledTasks() []ScheduledTask
}
```

---

## 3. Interface Design

### 3.1 Unified Client Interface

The primary interface for users will be a clean, unified client that provides access to all pillars:

```go
// Primary interface for all RAGO functionality
type Client interface {
    // Pillar Access
    LLM() LLMService
    RAG() RAGService  
    MCP() MCPService
    Agents() AgentService
    
    // High-Level Operations (using multiple pillars)
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    StreamChat(ctx context.Context, req ChatRequest, callback StreamCallback) error
    ProcessDocument(ctx context.Context, req DocumentRequest) (*DocumentResponse, error)
    ExecuteTask(ctx context.Context, req TaskRequest) (*TaskResponse, error)
    
    // Client Management
    Close() error
    Health() HealthReport
}

// Individual pillar clients for library usage
type LLMClient interface {
    LLMService
    Close() error
}

type RAGClient interface {
    RAGService
    Close() error
}

type MCPClient interface {
    MCPService
    Close() error
}

type AgentClient interface {
    AgentService
    Close() error
}
```

### 3.2 Configuration Design

Each pillar will have independent configuration while supporting unified setup:

```go
type Config struct {
    // Global settings
    DataDir  string `toml:"data_dir"`
    LogLevel string `toml:"log_level"`
    
    // Pillar-specific configurations
    LLM    LLMConfig    `toml:"llm"`
    RAG    RAGConfig    `toml:"rag"`
    MCP    MCPConfig    `toml:"mcp"`
    Agents AgentsConfig `toml:"agents"`
    
    // Operational modes
    Mode ModeConfig `toml:"mode"`
}

type ModeConfig struct {
    RAGOnly      bool `toml:"rag_only"`       // Only RAG pillar
    LLMOnly      bool `toml:"llm_only"`       // Only LLM pillar  
    DisableMCP   bool `toml:"disable_mcp"`    // Disable MCP pillar
    DisableAgent bool `toml:"disable_agents"` // Disable Agent pillar
}
```

---

## 4. New Package Structure

### 4.1 Target Organization

```
pkg/
├── core/                    # Core types and interfaces
│   ├── interfaces.go       # All pillar interfaces
│   ├── types.go           # Shared types
│   ├── errors.go          # Common errors
│   └── context.go         # Context utilities
│
├── llm/                    # LLM Pillar
│   ├── service.go         # Main LLM service
│   ├── pool.go            # Provider pooling
│   ├── providers/         # Provider implementations
│   │   ├── ollama.go
│   │   ├── openai.go
│   │   └── lmstudio.go
│   ├── balancer.go        # Load balancing
│   ├── health.go          # Health checking
│   └── metrics.go         # Performance metrics
│
├── rag/                    # RAG Pillar
│   ├── service.go         # Main RAG service
│   ├── ingest/            # Document ingestion
│   │   ├── processors.go  # Document processors
│   │   ├── extractors.go  # Metadata extraction
│   │   └── chunkers.go    # Text chunking
│   ├── storage/           # Storage backends
│   │   ├── vector.go      # Vector storage
│   │   ├── keyword.go     # Keyword storage
│   │   └── document.go    # Document storage
│   ├── search/            # Search algorithms
│   │   ├── vector.go      # Vector search
│   │   ├── keyword.go     # Keyword search
│   │   └── hybrid.go      # Hybrid search (RRF)
│   └── optimize.go        # Index optimization
│
├── mcp/                    # MCP Pillar
│   ├── service.go         # Main MCP service
│   ├── server/            # Server management
│   │   ├── manager.go     # Server lifecycle
│   │   ├── health.go      # Health monitoring
│   │   └── discovery.go   # Tool discovery
│   ├── tools/             # Tool management
│   │   ├── registry.go    # Tool registry
│   │   ├── executor.go    # Tool execution
│   │   └── cache.go       # Result caching
│   ├── client.go          # MCP protocol client
│   └── transport.go       # Transport implementations
│
├── agents/                 # Agent Pillar
│   ├── service.go         # Main agent service
│   ├── workflow/          # Workflow management
│   │   ├── engine.go      # Workflow execution
│   │   ├── definition.go  # Workflow definition
│   │   └── state.go       # State management
│   ├── scheduler/         # Task scheduling
│   │   ├── cron.go        # Cron-based scheduling
│   │   ├── queue.go       # Task queue
│   │   └── executor.go    # Task execution
│   ├── reasoning/         # Multi-step reasoning
│   │   ├── chain.go       # Reasoning chains
│   │   ├── memory.go      # Working memory
│   │   └── planning.go    # Task planning
│   └── agents/            # Agent implementations
│       ├── research.go    # Research agents
│       ├── workflow.go    # Workflow agents
│       └── monitor.go     # Monitoring agents
│
├── client/                 # Unified Client
│   ├── client.go          # Main client implementation
│   ├── factory.go         # Client factory
│   ├── config.go          # Configuration management
│   └── health.go          # Health reporting
│
└── examples/               # Usage examples
    ├── llm_only/          # LLM pillar examples
    ├── rag_only/          # RAG pillar examples
    ├── mcp_only/          # MCP pillar examples
    ├── agents_only/       # Agent pillar examples
    └── combined/          # Multi-pillar examples
```

### 4.2 Backward Compatibility

Maintain backward compatibility through:
- Alias types for existing domain models
- Wrapper implementations for current interfaces
- Migration utilities for existing data
- Deprecation warnings with migration guidance

---

## 5. Implementation Plan

### Phase 1: Core Infrastructure (Weeks 1-2)
**Goal**: Establish foundation for four-pillar architecture

**Tasks**:
1. **Create new package structure**
   - Set up pkg/core/ with interfaces and types
   - Create empty pillar directories with basic structure
   - Define common configuration patterns

2. **Implement unified configuration system**  
   - Design pillar-specific config structures
   - Create config loading with backward compatibility
   - Add operational mode controls

3. **Design core interfaces**
   - Define all pillar service interfaces
   - Create shared types and error definitions
   - Establish context patterns for cross-pillar communication

4. **Set up testing framework**
   - Create test utilities for each pillar
   - Set up integration test patterns
   - Establish performance benchmarking

**Deliverables**:
- New package structure with interfaces
- Configuration system supporting all pillars
- Test framework for independent pillar testing
- Documentation for core interfaces

### Phase 2: LLM Pillar Implementation (Weeks 3-4)
**Goal**: Implement production-ready LLM pillar with provider pooling

**Tasks**:
1. **Provider pool implementation**
   - Health checking and monitoring
   - Load balancing algorithms (round-robin, weighted)
   - Circuit breaker pattern for failing providers
   - Automatic failover logic

2. **Core LLM service**
   - Generation operations (sync/async/streaming)
   - Tool calling coordination
   - Batch processing capabilities
   - Performance metrics collection

3. **Provider implementations**
   - Refactor existing providers to new interfaces
   - Add connection pooling and reuse
   - Implement capability detection
   - Add retry logic with exponential backoff

4. **Testing and validation**
   - Unit tests for all components
   - Integration tests with real providers
   - Performance benchmarks
   - Error handling validation

**Deliverables**:
- Complete LLM pillar implementation
- Provider pool with health checking
- Comprehensive test suite
- Performance benchmarks and documentation

### Phase 3: RAG Pillar Enhancement (Weeks 5-6)
**Goal**: Refactor existing RAG functionality into independent pillar

**Tasks**:
1. **Modularize existing RAG components**
   - Extract document processing from processor service
   - Separate storage backends into clean interfaces
   - Implement pluggable chunking strategies
   - Create search optimization engine

2. **Enhanced document processing**
   - Support for more document formats
   - Improved metadata extraction
   - Batch processing capabilities
   - Document versioning and updates

3. **Storage optimization**
   - Implement storage backend switching
   - Add index optimization routines
   - Create backup/restore functionality
   - Add storage analytics and monitoring

4. **Search improvements**
   - Enhanced hybrid search with configurable fusion
   - Query expansion and rewriting
   - Result ranking improvements
   - Search performance analytics

**Deliverables**:
- Independent RAG pillar implementation
- Enhanced document processing pipeline
- Optimized storage and search
- Migration utilities for existing data

### Phase 4: MCP Pillar Enhancement (Weeks 7-8)  
**Goal**: Transform MCP integration into full-featured pillar

**Tasks**:
1. **Server lifecycle management**
   - Automatic server discovery and registration
   - Health monitoring with automatic recovery
   - Version compatibility checking
   - Resource usage monitoring

2. **Tool execution engine**
   - Concurrent tool execution with proper isolation
   - Result caching and invalidation
   - Error handling with retries
   - Performance analytics per tool

3. **Integration improvements**
   - Seamless integration with LLM pillar for tool calling
   - Tool result processing and formatting
   - Multi-step tool workflows
   - Tool dependency management

4. **Developer experience**
   - Tool development SDK
   - Testing utilities for custom tools
   - Documentation generator for tool APIs
   - Tool marketplace concepts

**Deliverables**:
- Production-ready MCP pillar
- Advanced tool execution engine
- Integration with LLM pillar for tool calling
- Developer SDK and documentation

### Phase 5: Agent Pillar Implementation (Weeks 9-10)
**Goal**: Implement comprehensive agent workflow system

**Tasks**:
1. **Workflow execution engine**
   - Workflow definition language (JSON/YAML)
   - State management with persistence
   - Error recovery and retry logic
   - Workflow monitoring and debugging

2. **Agent implementations**
   - Research agents for data gathering
   - Workflow agents for task automation  
   - Monitoring agents for system health
   - Custom agent framework

3. **Scheduling system**
   - Cron-based task scheduling
   - Event-driven workflow triggers
   - Task queue with priorities
   - Distributed execution support

4. **Multi-step reasoning**
   - Reasoning chain implementation
   - Working memory management
   - Goal-oriented task planning
   - Learning from execution history

**Deliverables**:
- Complete agent pillar implementation
- Workflow execution engine with persistence
- Scheduling system with multiple triggers
- Multi-step reasoning framework

### Phase 6: Unified Client Implementation (Weeks 11-12)
**Goal**: Create library-first client supporting all pillars

**Tasks**:
1. **Client factory implementation**
   - Support for individual pillar clients
   - Unified client with all pillars
   - Configuration-based client creation
   - Dependency injection patterns

2. **High-level operations**
   - Chat interface using LLM + RAG + MCP
   - Document processing using RAG + Agents
   - Task execution using Agents + MCP
   - Multi-pillar workflow coordination

3. **Client management**
   - Resource lifecycle management
   - Health monitoring across pillars
   - Graceful shutdown procedures
   - Error propagation and handling

4. **Library packaging**
   - Independent pillar libraries
   - Combined library with all pillars
   - Documentation and examples
   - Go module publishing

**Deliverables**:
- Unified client supporting all usage patterns
- Individual pillar libraries
- Comprehensive documentation and examples
- Go module releases for each pillar

### Phase 7: Integration and Testing (Weeks 13-14)
**Goal**: Comprehensive testing and optimization

**Tasks**:
1. **Integration testing**
   - Cross-pillar integration tests
   - End-to-end workflow testing
   - Performance testing under load
   - Memory usage and leak detection

2. **Performance optimization**
   - Bottleneck identification and resolution
   - Memory usage optimization
   - Concurrent operation tuning
   - Resource pooling optimization

3. **Production readiness**
   - Monitoring and observability
   - Error handling and recovery
   - Configuration validation
   - Security review and hardening

4. **Documentation and examples**
   - Complete API documentation
   - Tutorial and getting started guides
   - Architecture documentation
   - Performance tuning guides

**Deliverables**:
- Production-ready system with all pillars
- Comprehensive test suite with >90% coverage
- Performance benchmarks and optimization
- Complete documentation package

---

## 6. Migration Strategy

### 6.1 Backward Compatibility Approach

**Immediate Compatibility**: Existing code continues to work unchanged
```go
// Old way (still works)
client, err := client.New("config.toml")
result, err := client.Query(ctx, queryReq)

// New way (preferred)
ragoClient, err := client.New("config.toml")  
result, err := ragoClient.RAG().Search(ctx, searchReq)
```

**Gradual Migration Path**:
1. Phase 1: Old interfaces work through adapters
2. Phase 2: Deprecation warnings guide users to new APIs
3. Phase 3: Old APIs marked for removal in future version
4. Phase 4: Clean removal with major version bump

### 6.2 Data Migration

**Vector Store Migration**:
- Automatic detection of existing sqvect databases
- In-place migration to new schema if needed
- Backup creation before migration
- Rollback procedures for failed migrations

**Configuration Migration**:
- Automatic conversion of old config format
- Validation with helpful error messages
- Configuration upgrade utilities
- Template generation for new format

### 6.3 Timeline for Breaking Changes

**v3.0**: Current release with existing architecture
**v3.1-v3.5**: Gradual introduction of new pillars (non-breaking)
**v4.0**: New architecture as default, old APIs deprecated
**v5.0**: Clean architecture, old APIs removed

---

## 7. Success Metrics

### 7.1 Technical Metrics

**Performance**:
- Query response time < 100ms for cached results
- Document ingestion throughput > 1000 docs/minute
- Tool execution latency < 500ms average
- Memory usage < 100MB for basic operations

**Reliability**:
- 99.9% uptime for all pillar services
- Zero data loss during normal operations  
- < 5 seconds recovery time from provider failures
- 100% test coverage for core interfaces

**Usability**:
- < 5 lines of code for basic operations
- Zero-configuration operation out of the box
- Complete API documentation with examples
- Library import size < 10MB compiled

### 7.2 Adoption Metrics

**Library Usage**:
- Independent pillar usage tracking
- Common usage patterns identification
- Performance bottleneck identification
- User feedback integration

**Developer Experience**:
- Time to first working example < 5 minutes
- Clear migration path documentation
- Comprehensive error messages
- Active community support

---

## 8. Risk Mitigation

### 8.1 Technical Risks

**Performance Degradation**: 
- Risk: New architecture introduces overhead
- Mitigation: Continuous benchmarking, performance budgets
- Fallback: Performance optimization phase with profiling

**Backward Compatibility Issues**:
- Risk: Breaking changes affect existing users
- Mitigation: Comprehensive adapter layer, extensive testing
- Fallback: Maintain v3 branch for critical fixes

**Complexity Increase**:
- Risk: Four pillars increase cognitive load
- Mitigation: Clear documentation, simple default configurations
- Fallback: Operational modes to reduce complexity

### 8.2 Project Risks

**Timeline Overrun**:
- Risk: 14-week timeline is ambitious
- Mitigation: Incremental delivery, MVP approach
- Fallback: Reduce scope, focus on core pillars first

**Resource Constraints**:
- Risk: Implementation requires significant effort
- Mitigation: Phased approach, community involvement
- Fallback: Prioritize most critical pillars (LLM, RAG)

---

## 9. Conclusion

This transformation plan positions RAGO as a comprehensive, production-ready AI foundation for Go developers. The four-pillar architecture provides:

1. **Flexibility**: Use any pillar independently or in combination
2. **Scalability**: Each pillar can be optimized and scaled separately  
3. **Extensibility**: Clean interfaces for future enhancements
4. **Maintainability**: Clear separation of concerns and responsibilities

The phased implementation approach ensures continuous value delivery while maintaining backward compatibility. Upon completion, RAGO will be the premier Go library for local AI applications, serving developers from simple RAG applications to complex multi-agent workflows.

**Key Success Factors**:
- Maintain laser focus on library-first design
- Ensure each pillar provides standalone value
- Keep performance and usability as top priorities
- Build comprehensive testing and documentation
- Foster active community engagement

This architectural transformation will establish RAGO as the foundation for the next generation of privacy-first, local-first AI applications built with Go.