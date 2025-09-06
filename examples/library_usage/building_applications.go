// This example demonstrates how to build complete applications using RAGO as a library.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	// Example 1: Build a Chatbot Application
	fmt.Println("=== Example 1: Chatbot Application ===")
	chatbotApp := NewChatbotApp()
	if err := chatbotApp.Initialize(); err != nil {
		log.Printf("Failed to initialize chatbot: %v", err)
	} else {
		chatbotApp.Demo()
		chatbotApp.Shutdown()
	}
	
	// Example 2: Build a Document Q&A System
	fmt.Println("\n=== Example 2: Document Q&A System ===")
	docQA := NewDocumentQASystem()
	if err := docQA.Initialize(); err != nil {
		log.Printf("Failed to initialize Doc Q&A: %v", err)
	} else {
		docQA.Demo()
		docQA.Shutdown()
	}
	
	// Example 3: Build an Automation Platform
	fmt.Println("\n=== Example 3: Automation Platform ===")
	automation := NewAutomationPlatform()
	if err := automation.Initialize(); err != nil {
		log.Printf("Failed to initialize automation: %v", err)
	} else {
		automation.Demo()
		automation.Shutdown()
	}
	
	// Example 4: Build a Knowledge Management System
	fmt.Println("\n=== Example 4: Knowledge Management System ===")
	kms := NewKnowledgeManagementSystem()
	if err := kms.Initialize(); err != nil {
		log.Printf("Failed to initialize KMS: %v", err)
	} else {
		kms.Demo()
		kms.Shutdown()
	}
}

// ============ CHATBOT APPLICATION ============

// ChatbotApp represents a complete chatbot application
type ChatbotApp struct {
	client       core.Client
	sessions     map[string]*ChatSession
	sessionMutex sync.RWMutex
}

// ChatSession represents a user chat session
type ChatSession struct {
	ID           string
	UserID       string
	History      []ChatMessage
	Context      map[string]interface{}
	LastActivity time.Time
}

// ChatMessage represents a single message in the chat
type ChatMessage struct {
	Role      string    // "user" or "assistant"
	Content   string
	Timestamp time.Time
}

// NewChatbotApp creates a new chatbot application
func NewChatbotApp() *ChatbotApp {
	return &ChatbotApp{
		sessions: make(map[string]*ChatSession),
	}
}

// Initialize sets up the chatbot with RAGO
func (app *ChatbotApp) Initialize() error {
	// Create a chat-optimized client
	ragoClient, err := client.NewBuilder().
		WithLLM(core.LLMConfig{
			DefaultProvider: "ollama",
			Providers: map[string]core.ProviderConfig{
				"ollama": {
					Type:    "ollama",
					BaseURL: "http://localhost:11434",
					Model:   "llama3.2",
					Weight:  1,
					Timeout: 30 * time.Second,
				},
			},
		}).
		WithRAG(core.RAGConfig{
			StorageBackend: "dual",
			ChunkingStrategy: core.ChunkingConfig{
				Strategy:  "recursive",
				ChunkSize: 400,
			},
		}).
		WithoutMCP().    // MCP not needed for basic chat
		WithoutAgents(). // Agents not needed for basic chat
		Build()
	
	if err != nil {
		return fmt.Errorf("failed to create RAGO client: %w", err)
	}
	
	app.client = ragoClient
	
	// Load initial knowledge base
	app.loadKnowledgeBase()
	
	return nil
}

// loadKnowledgeBase loads initial knowledge into RAG
func (app *ChatbotApp) loadKnowledgeBase() {
	ctx := context.Background()
	
	// Sample knowledge base
	knowledge := []struct {
		id      string
		content string
	}{
		{
			id:      "company-info",
			content: "Our company specializes in AI solutions. We offer chatbots, document processing, and automation services.",
		},
		{
			id:      "product-features",
			content: "Our chatbot features include: multi-language support, context awareness, integration with various platforms, and 24/7 availability.",
		},
		{
			id:      "pricing",
			content: "We offer three pricing tiers: Starter ($99/month), Professional ($299/month), and Enterprise (custom pricing).",
		},
	}
	
	for _, kb := range knowledge {
		req := core.IngestRequest{
			DocumentID: kb.id,
			Content:    kb.content,
			Metadata: map[string]interface{}{
				"type": "knowledge_base",
			},
		}
		
		if _, err := app.client.RAG().Ingest(ctx, req); err != nil {
			log.Printf("Failed to ingest %s: %v", kb.id, err)
		}
	}
}

// CreateSession creates a new chat session
func (app *ChatbotApp) CreateSession(userID string) string {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	
	app.sessionMutex.Lock()
	defer app.sessionMutex.Unlock()
	
	app.sessions[sessionID] = &ChatSession{
		ID:           sessionID,
		UserID:       userID,
		History:      make([]ChatMessage, 0),
		Context:      make(map[string]interface{}),
		LastActivity: time.Now(),
	}
	
	return sessionID
}

// Chat handles a chat message
func (app *ChatbotApp) Chat(sessionID, message string) (string, error) {
	app.sessionMutex.RLock()
	session, exists := app.sessions[sessionID]
	app.sessionMutex.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("session not found")
	}
	
	// Add user message to history
	session.History = append(session.History, ChatMessage{
		Role:      "user",
		Content:   message,
		Timestamp: time.Now(),
	})
	
	// Build context from history
	var contextPrompt string
	for _, msg := range session.History[len(session.History)-10:] { // Last 10 messages
		contextPrompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}
	
	// Use RAGO to generate response
	ctx := context.Background()
	chatReq := core.ChatRequest{
		Message:      message,
		SystemPrompt: "You are a helpful customer service chatbot. Be friendly and professional. Use the knowledge base to answer questions accurately.",
		UseRAG:       true,
		RAGLimit:     3,
		RAGThreshold: 0.6,
		Temperature:  0.7,
		MaxTokens:    300,
	}
	
	resp, err := app.client.Chat(ctx, chatReq)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}
	
	// Add assistant response to history
	session.History = append(session.History, ChatMessage{
		Role:      "assistant",
		Content:   resp.Response,
		Timestamp: time.Now(),
	})
	
	session.LastActivity = time.Now()
	
	return resp.Response, nil
}

// Demo demonstrates the chatbot
func (app *ChatbotApp) Demo() {
	sessionID := app.CreateSession("demo_user")
	
	messages := []string{
		"Hello! What services do you offer?",
		"Tell me more about the chatbot features",
		"What are your pricing options?",
	}
	
	for _, msg := range messages {
		fmt.Printf("\nUser: %s\n", msg)
		
		response, err := app.Chat(sessionID, msg)
		if err != nil {
			log.Printf("Chat error: %v", err)
			continue
		}
		
		fmt.Printf("Bot: %s\n", response)
		time.Sleep(500 * time.Millisecond) // Simulate conversation pace
	}
}

// Shutdown closes the chatbot application
func (app *ChatbotApp) Shutdown() {
	if app.client != nil {
		app.client.Close()
	}
}

// ============ DOCUMENT Q&A SYSTEM ============

// DocumentQASystem represents a document question-answering system
type DocumentQASystem struct {
	client    core.Client
	documents map[string]DocumentMetadata
	mutex     sync.RWMutex
}

// DocumentMetadata stores metadata about ingested documents
type DocumentMetadata struct {
	ID          string
	Title       string
	Source      string
	ChunkCount  int
	IngestedAt  time.Time
}

// NewDocumentQASystem creates a new document Q&A system
func NewDocumentQASystem() *DocumentQASystem {
	return &DocumentQASystem{
		documents: make(map[string]DocumentMetadata),
	}
}

// Initialize sets up the document Q&A system
func (qa *DocumentQASystem) Initialize() error {
	// Create a knowledge-focused client
	ragoClient, err := client.NewKnowledgeClient(core.Config{
		DataDir:  "~/.rago/doc-qa",
		LogLevel: "info",
		LLM: core.LLMConfig{
			DefaultProvider: "ollama",
			Providers: map[string]core.ProviderConfig{
				"ollama": {
					Type:    "ollama",
					BaseURL: "http://localhost:11434",
					Model:   "llama3.2",
				},
			},
		},
		RAG: core.RAGConfig{
			StorageBackend: "dual",
			ChunkingStrategy: core.ChunkingConfig{
				Strategy:     "recursive",
				ChunkSize:    500,
				ChunkOverlap: 100,
			},
			Search: core.SearchConfig{
				DefaultLimit:     10,
				DefaultThreshold: 0.7,
			},
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to create RAGO client: %w", err)
	}
	
	qa.client = ragoClient
	return nil
}

// IngestDocument ingests a document into the system
func (qa *DocumentQASystem) IngestDocument(title, content, source string) error {
	ctx := context.Background()
	docID := fmt.Sprintf("doc_%d", time.Now().UnixNano())
	
	req := core.IngestRequest{
		DocumentID: docID,
		Content:    content,
		Metadata: map[string]interface{}{
			"title":  title,
			"source": source,
		},
	}
	
	resp, err := qa.client.RAG().Ingest(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to ingest document: %w", err)
	}
	
	qa.mutex.Lock()
	qa.documents[docID] = DocumentMetadata{
		ID:         docID,
		Title:      title,
		Source:     source,
		ChunkCount: resp.ChunkCount,
		IngestedAt: time.Now(),
	}
	qa.mutex.Unlock()
	
	return nil
}

// AskQuestion asks a question about the ingested documents
func (qa *DocumentQASystem) AskQuestion(question string) (string, []string, error) {
	ctx := context.Background()
	
	// First, search for relevant content
	searchReq := core.SearchRequest{
		Query: question,
		Limit: 5,
	}
	
	searchResp, err := qa.client.RAG().Search(ctx, searchReq)
	if err != nil {
		return "", nil, fmt.Errorf("search failed: %w", err)
	}
	
	// Extract sources
	sources := make([]string, 0)
	seenSources := make(map[string]bool)
	
	for _, result := range searchResp.Results {
		if meta, ok := result.Metadata["source"].(string); ok {
			if !seenSources[meta] {
				sources = append(sources, meta)
				seenSources[meta] = true
			}
		}
	}
	
	// Generate answer using retrieved context
	var contextContent string
	for _, result := range searchResp.Results {
		contextContent += result.Content + "\n\n"
	}
	
	genReq := core.GenerateRequest{
		Prompt: fmt.Sprintf(`Based on the following context, answer the question.
		
Context:
%s

Question: %s

Answer:`, contextContent, question),
		Temperature: 0.3, // Lower temperature for factual answers
		MaxTokens:   500,
	}
	
	genResp, err := qa.client.LLM().Generate(ctx, genReq)
	if err != nil {
		return "", sources, fmt.Errorf("generation failed: %w", err)
	}
	
	return genResp.Text, sources, nil
}

// Demo demonstrates the document Q&A system
func (qa *DocumentQASystem) Demo() {
	// Ingest sample documents
	documents := []struct {
		title   string
		content string
		source  string
	}{
		{
			title: "Introduction to Go",
			content: `Go is a statically typed, compiled programming language designed at Google. 
			It provides excellent support for concurrent programming through goroutines and channels. 
			Go's simplicity and efficiency make it ideal for building scalable web services and cloud applications.`,
			source: "go_tutorial.pdf",
		},
		{
			title: "Go Best Practices",
			content: `When writing Go code, follow these best practices: 
			Use clear and descriptive names, handle errors explicitly, 
			keep functions small and focused, use interfaces to define behavior, 
			and leverage Go's built-in testing framework for comprehensive tests.`,
			source: "go_best_practices.md",
		},
	}
	
	fmt.Println("Ingesting documents...")
	for _, doc := range documents {
		if err := qa.IngestDocument(doc.title, doc.content, doc.source); err != nil {
			log.Printf("Failed to ingest %s: %v", doc.title, err)
		} else {
			fmt.Printf("  Ingested: %s\n", doc.title)
		}
	}
	
	// Ask questions
	questions := []string{
		"What is Go programming language?",
		"What are Go's best practices for error handling?",
		"How does Go support concurrent programming?",
	}
	
	for _, question := range questions {
		fmt.Printf("\nQ: %s\n", question)
		
		answer, sources, err := qa.AskQuestion(question)
		if err != nil {
			log.Printf("Failed to answer: %v", err)
			continue
		}
		
		fmt.Printf("A: %s\n", answer)
		if len(sources) > 0 {
			fmt.Printf("Sources: %v\n", sources)
		}
	}
}

// Shutdown closes the document Q&A system
func (qa *DocumentQASystem) Shutdown() {
	if qa.client != nil {
		qa.client.Close()
	}
}

// ============ AUTOMATION PLATFORM ============

// AutomationPlatform represents an automation platform
type AutomationPlatform struct {
	client    core.Client
	workflows map[string]*Workflow
	tasks     map[string]*Task
	mutex     sync.RWMutex
}

// Workflow represents an automation workflow
type Workflow struct {
	ID          string
	Name        string
	Description string
	Steps       []WorkflowStep
	Schedule    string // cron expression
}

// WorkflowStep represents a step in a workflow
type WorkflowStep struct {
	Name       string
	Type       string // "tool", "llm", "condition", "loop"
	Parameters map[string]interface{}
}

// Task represents an automation task
type Task struct {
	ID        string
	Workflow  string
	Status    string
	StartedAt time.Time
	Output    interface{}
}

// NewAutomationPlatform creates a new automation platform
func NewAutomationPlatform() *AutomationPlatform {
	return &AutomationPlatform{
		workflows: make(map[string]*Workflow),
		tasks:     make(map[string]*Task),
	}
}

// Initialize sets up the automation platform
func (ap *AutomationPlatform) Initialize() error {
	// Create an automation-focused client
	ragoClient, err := client.NewAutomationClient(core.Config{
		DataDir:  "~/.rago/automation",
		LogLevel: "info",
		LLM: core.LLMConfig{
			DefaultProvider: "ollama",
			Providers: map[string]core.ProviderConfig{
				"ollama": {
					Type:    "ollama",
					BaseURL: "http://localhost:11434",
					Model:   "llama3.2",
				},
			},
		},
		MCP: core.MCPConfig{
			ServersPath: "~/.rago/mcpServers.json",
			ToolExecution: core.ToolExecutionConfig{
				MaxConcurrent:  5,
				DefaultTimeout: 30 * time.Second,
			},
		},
		Agents: core.AgentsConfig{
			WorkflowEngine: core.WorkflowEngineConfig{
				MaxSteps:    100,
				StepTimeout: 5 * time.Minute,
			},
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to create RAGO client: %w", err)
	}
	
	ap.client = ragoClient
	
	// Create sample workflows
	ap.createSampleWorkflows()
	
	return nil
}

// createSampleWorkflows creates sample automation workflows
func (ap *AutomationPlatform) createSampleWorkflows() {
	ap.workflows["data_processing"] = &Workflow{
		ID:          "data_processing",
		Name:        "Data Processing Pipeline",
		Description: "Process and analyze incoming data",
		Steps: []WorkflowStep{
			{
				Name: "fetch_data",
				Type: "tool",
				Parameters: map[string]interface{}{
					"tool": "web_fetch",
					"url":  "https://api.example.com/data",
				},
			},
			{
				Name: "analyze_data",
				Type: "llm",
				Parameters: map[string]interface{}{
					"prompt": "Analyze the following data and provide insights",
				},
			},
			{
				Name: "save_results",
				Type: "tool",
				Parameters: map[string]interface{}{
					"tool": "filesystem_write",
					"path": "/tmp/analysis_results.json",
				},
			},
		},
		Schedule: "0 */6 * * *", // Every 6 hours
	}
	
	ap.workflows["report_generation"] = &Workflow{
		ID:          "report_generation",
		Name:        "Report Generation",
		Description: "Generate daily reports",
		Steps: []WorkflowStep{
			{
				Name: "collect_metrics",
				Type: "tool",
				Parameters: map[string]interface{}{
					"tool": "database_query",
					"query": "SELECT * FROM metrics WHERE date = CURRENT_DATE",
				},
			},
			{
				Name: "generate_summary",
				Type: "llm",
				Parameters: map[string]interface{}{
					"prompt": "Create an executive summary of the metrics",
				},
			},
		},
		Schedule: "0 9 * * *", // Daily at 9 AM
	}
}

// ExecuteWorkflow executes a workflow
func (ap *AutomationPlatform) ExecuteWorkflow(workflowID string) (*Task, error) {
	ap.mutex.RLock()
	workflow, exists := ap.workflows[workflowID]
	ap.mutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("workflow not found")
	}
	
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	task := &Task{
		ID:        taskID,
		Workflow:  workflowID,
		Status:    "executing",
		StartedAt: time.Now(),
	}
	
	ap.mutex.Lock()
	ap.tasks[taskID] = task
	ap.mutex.Unlock()
	
	ctx := context.Background()
	
	// Execute using RAGO's task execution
	taskReq := core.TaskRequest{
		TaskID:      taskID,
		Description: workflow.Description,
		Context: map[string]interface{}{
			"workflow": workflow,
		},
	}
	
	resp, err := ap.client.ExecuteTask(ctx, taskReq)
	if err != nil {
		task.Status = "failed"
		return task, err
	}
	
	task.Status = resp.Status
	task.Output = resp.Result
	
	return task, nil
}

// Demo demonstrates the automation platform
func (ap *AutomationPlatform) Demo() {
	fmt.Println("Available Workflows:")
	for id, workflow := range ap.workflows {
		fmt.Printf("  - %s: %s (Schedule: %s)\n", id, workflow.Name, workflow.Schedule)
	}
	
	// Execute a workflow
	fmt.Println("\nExecuting 'report_generation' workflow...")
	task, err := ap.ExecuteWorkflow("report_generation")
	if err != nil {
		log.Printf("Workflow execution failed: %v", err)
		return
	}
	
	fmt.Printf("Task %s completed with status: %s\n", task.ID, task.Status)
	if task.Output != nil {
		fmt.Printf("Output: %v\n", task.Output)
	}
}

// Shutdown closes the automation platform
func (ap *AutomationPlatform) Shutdown() {
	if ap.client != nil {
		ap.client.Close()
	}
}

// ============ KNOWLEDGE MANAGEMENT SYSTEM ============

// KnowledgeManagementSystem represents a knowledge management system
type KnowledgeManagementSystem struct {
	client      core.Client
	collections map[string]*Collection
	mutex       sync.RWMutex
}

// Collection represents a knowledge collection
type Collection struct {
	ID          string
	Name        string
	Description string
	Documents   []string
	Tags        []string
	CreatedAt   time.Time
}

// NewKnowledgeManagementSystem creates a new knowledge management system
func NewKnowledgeManagementSystem() *KnowledgeManagementSystem {
	return &KnowledgeManagementSystem{
		collections: make(map[string]*Collection),
	}
}

// Initialize sets up the knowledge management system
func (kms *KnowledgeManagementSystem) Initialize() error {
	// Create a full-featured client for knowledge management
	ragoClient, err := client.NewFullClient(core.Config{
		DataDir:  "~/.rago/kms",
		LogLevel: "info",
		LLM: core.LLMConfig{
			DefaultProvider: "ollama",
			Providers: map[string]core.ProviderConfig{
				"ollama": {
					Type:    "ollama",
					BaseURL: "http://localhost:11434",
					Model:   "llama3.2",
				},
			},
		},
		RAG: core.RAGConfig{
			StorageBackend: "dual",
			ChunkingStrategy: core.ChunkingConfig{
				Strategy:     "semantic",
				ChunkSize:    600,
				ChunkOverlap: 150,
			},
		},
		MCP: core.MCPConfig{
			ServersPath: "~/.rago/mcpServers.json",
		},
		Agents: core.AgentsConfig{
			WorkflowEngine: core.WorkflowEngineConfig{
				MaxSteps: 50,
			},
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to create RAGO client: %w", err)
	}
	
	kms.client = ragoClient
	return nil
}

// CreateCollection creates a new knowledge collection
func (kms *KnowledgeManagementSystem) CreateCollection(name, description string, tags []string) string {
	collectionID := fmt.Sprintf("col_%d", time.Now().UnixNano())
	
	kms.mutex.Lock()
	defer kms.mutex.Unlock()
	
	kms.collections[collectionID] = &Collection{
		ID:          collectionID,
		Name:        name,
		Description: description,
		Documents:   make([]string, 0),
		Tags:        tags,
		CreatedAt:   time.Now(),
	}
	
	return collectionID
}

// AddDocument adds a document to a collection
func (kms *KnowledgeManagementSystem) AddDocument(collectionID, title, content string) error {
	kms.mutex.RLock()
	collection, exists := kms.collections[collectionID]
	kms.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("collection not found")
	}
	
	ctx := context.Background()
	
	// Process document using RAGO
	docReq := core.DocumentRequest{
		DocumentID:  fmt.Sprintf("%s_doc_%d", collectionID, time.Now().UnixNano()),
		Content:     content,
		IngestToRAG: true,
		Metadata: map[string]interface{}{
			"collection": collectionID,
			"title":      title,
			"tags":       collection.Tags,
		},
		AnalysisPrompt: "Extract key concepts and create a summary",
	}
	
	resp, err := kms.client.ProcessDocument(ctx, docReq)
	if err != nil {
		return fmt.Errorf("failed to process document: %w", err)
	}
	
	kms.mutex.Lock()
	collection.Documents = append(collection.Documents, resp.DocumentID)
	kms.mutex.Unlock()
	
	return nil
}

// SearchKnowledge searches across all collections
func (kms *KnowledgeManagementSystem) SearchKnowledge(query string) ([]SearchResult, error) {
	ctx := context.Background()
	
	// Use hybrid search for better results
	hybridReq := core.HybridSearchRequest{
		Query:         query,
		VectorWeight:  0.7,
		KeywordWeight: 0.3,
		Limit:         10,
	}
	
	resp, err := kms.client.RAG().HybridSearch(ctx, hybridReq)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	results := make([]SearchResult, 0)
	for _, result := range resp.Results {
		sr := SearchResult{
			Content: result.Content,
			Score:   result.Score,
		}
		
		// Extract metadata
		if col, ok := result.Metadata["collection"].(string); ok {
			sr.Collection = col
		}
		if title, ok := result.Metadata["title"].(string); ok {
			sr.Title = title
		}
		
		results = append(results, sr)
	}
	
	return results, nil
}

// SearchResult represents a knowledge search result
type SearchResult struct {
	Collection string
	Title      string
	Content    string
	Score      float32
}

// Demo demonstrates the knowledge management system
func (kms *KnowledgeManagementSystem) Demo() {
	// Create collections
	techCol := kms.CreateCollection(
		"Technology",
		"Technology and programming knowledge",
		[]string{"tech", "programming", "software"},
	)
	
	businessCol := kms.CreateCollection(
		"Business",
		"Business and management knowledge",
		[]string{"business", "management", "strategy"},
	)
	
	fmt.Printf("Created collections: Technology (%s), Business (%s)\n", techCol, businessCol)
	
	// Add documents
	techDocs := []struct {
		title   string
		content string
	}{
		{
			title: "Microservices Architecture",
			content: `Microservices architecture is a design approach where applications are built as a collection 
			of small, independent services. Each service runs in its own process and communicates via APIs. 
			This approach provides better scalability, flexibility, and resilience.`,
		},
		{
			title: "DevOps Practices",
			content: `DevOps combines software development and IT operations to shorten the development lifecycle. 
			Key practices include continuous integration, continuous deployment, infrastructure as code, 
			and monitoring and logging.`,
		},
	}
	
	for _, doc := range techDocs {
		if err := kms.AddDocument(techCol, doc.title, doc.content); err != nil {
			log.Printf("Failed to add document: %v", err)
		} else {
			fmt.Printf("  Added: %s\n", doc.title)
		}
	}
	
	// Search knowledge
	queries := []string{
		"What is microservices architecture?",
		"How does DevOps improve development?",
	}
	
	for _, query := range queries {
		fmt.Printf("\nSearching: %s\n", query)
		
		results, err := kms.SearchKnowledge(query)
		if err != nil {
			log.Printf("Search failed: %v", err)
			continue
		}
		
		for i, result := range results[:3] { // Top 3 results
			fmt.Printf("  %d. [%.2f] %s - %s...\n", 
				i+1, result.Score, result.Title, 
				result.Content[:min(80, len(result.Content))])
		}
	}
}

// Shutdown closes the knowledge management system
func (kms *KnowledgeManagementSystem) Shutdown() {
	if kms.client != nil {
		kms.client.Close()
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}