// This example demonstrates streaming and real-time operations with RAGO.
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	// Create a full RAGO client
	ragoClient, err := client.NewWithDefaults()
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()
	
	ctx := context.Background()
	
	// Example 1: Basic streaming with LLM
	fmt.Println("=== Example 1: Basic LLM Streaming ===")
	basicStreamingExample(ctx, ragoClient)
	
	// Example 2: Streaming chat with RAG context
	fmt.Println("\n=== Example 2: Streaming Chat with RAG ===")
	streamingChatExample(ctx, ragoClient)
	
	// Example 3: Real-time agent updates
	fmt.Println("\n=== Example 3: Real-time Agent Updates ===")
	realtimeAgentExample(ctx, ragoClient)
	
	// Example 4: Concurrent streaming operations
	fmt.Println("\n=== Example 4: Concurrent Streaming ===")
	concurrentStreamingExample(ctx, ragoClient)
	
	// Example 5: Event-driven processing
	fmt.Println("\n=== Example 5: Event-Driven Processing ===")
	eventDrivenExample(ctx, ragoClient)
}

func basicStreamingExample(ctx context.Context, client core.Client) {
	req := core.StreamRequest{
		Prompt: `Write a short story about a robot learning to paint. 
		Include emotional elements and make it inspirational.`,
		Temperature: 0.8,
		MaxTokens:   500,
	}
	
	fmt.Println("Streaming story generation:")
	fmt.Println("---")
	
	var totalChars int
	startTime := time.Now()
	
	err := client.LLM().Stream(ctx, req, func(chunk string) error {
		fmt.Print(chunk)
		totalChars += len(chunk)
		return nil
	})
	
	duration := time.Since(startTime)
	fmt.Println("\n---")
	
	if err != nil {
		log.Printf("Streaming failed: %v", err)
		return
	}
	
	fmt.Printf("Streamed %d characters in %v\n", totalChars, duration)
	fmt.Printf("Average throughput: %.2f chars/sec\n", float64(totalChars)/duration.Seconds())
}

func streamingChatExample(ctx context.Context, client core.Client) {
	// First, add some context to RAG
	contexts := []struct {
		id      string
		content string
	}{
		{
			id: "painting-techniques",
			content: `Oil painting involves using pigments mixed with oil. 
			Common techniques include impasto (thick paint), glazing (thin layers), 
			and alla prima (wet-on-wet). Artists often use brushes, palette knives, 
			and various mediums to achieve different effects.`,
		},
		{
			id: "color-theory",
			content: `Color theory encompasses the principles of how colors interact. 
			Primary colors (red, blue, yellow) can be mixed to create secondary colors. 
			Complementary colors sit opposite on the color wheel and create contrast. 
			Understanding color temperature (warm vs cool) helps create mood.`,
		},
	}
	
	// Ingest context documents
	for _, doc := range contexts {
		req := core.IngestRequest{
			DocumentID: doc.id,
			Content:    doc.content,
			Metadata: map[string]interface{}{
				"topic": "art",
			},
		}
		
		_, err := client.RAG().Ingest(ctx, req)
		if err != nil {
			log.Printf("Failed to ingest %s: %v", doc.id, err)
		}
	}
	
	// Stream chat with RAG context
	chatReq := core.ChatRequest{
		Message:      "Explain the impasto technique and how colors can affect mood in painting",
		UseRAG:       true,
		RAGLimit:     3,
		RAGThreshold: 0.6,
		Temperature:  0.7,
		MaxTokens:    400,
	}
	
	fmt.Println("Streaming chat response with RAG context:")
	fmt.Println("---")
	
	var buffer string
	err := client.StreamChat(ctx, chatReq, func(chunk string) error {
		fmt.Print(chunk)
		buffer += chunk
		
		// Demonstrate processing streaming chunks
		// Could trigger actions based on content
		if len(buffer) > 100 && len(buffer) < 150 {
			// Example: Could save partial results, update UI, etc.
		}
		
		return nil
	})
	
	fmt.Println("\n---")
	
	if err != nil {
		log.Printf("Streaming chat failed: %v", err)
	}
}

func realtimeAgentExample(ctx context.Context, client core.Client) {
	if client.Agents() == nil {
		fmt.Println("Agent pillar not available, skipping example")
		return
	}
	
	// Create an agent for real-time task execution
	createReq := core.CreateAgentRequest{
		Name:         "realtime_monitor",
		Instructions: "Monitor and process data in real-time",
		Capabilities: []string{"monitoring", "alerting", "processing"},
	}
	
	agent, err := client.Agents().CreateAgent(ctx, createReq)
	if err != nil {
		log.Printf("Failed to create agent: %v", err)
		return
	}
	defer client.Agents().DeleteAgent(ctx, agent.ID)
	
	// Simulate real-time monitoring with status updates
	statusChan := make(chan core.AgentStatus, 10)
	
	// Start monitoring in a goroutine
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(500 * time.Millisecond)
			
			status := core.AgentStatus{
				AgentID:   agent.ID,
				State:     "processing",
				Progress:  float32((i + 1) * 20),
				Message:   fmt.Sprintf("Processing batch %d of 5", i+1),
				Timestamp: time.Now(),
			}
			
			statusChan <- status
		}
		close(statusChan)
	}()
	
	// Process status updates in real-time
	fmt.Println("Real-time agent status updates:")
	for status := range statusChan {
		fmt.Printf("[%.0f%%] %s - %s\n", 
			status.Progress, 
			status.Timestamp.Format("15:04:05"), 
			status.Message)
	}
	
	fmt.Println("Agent monitoring completed")
}

func concurrentStreamingExample(ctx context.Context, client core.Client) {
	// Stream multiple LLM responses concurrently
	prompts := []string{
		"List 3 benefits of renewable energy",
		"Describe 3 types of machine learning",
		"Name 3 famous scientists and their contributions",
	}
	
	var wg sync.WaitGroup
	results := make([]string, len(prompts))
	errors := make([]error, len(prompts))
	
	fmt.Println("Starting concurrent streaming operations:")
	
	for i, prompt := range prompts {
		wg.Add(1)
		go func(index int, p string) {
			defer wg.Done()
			
			var buffer string
			req := core.StreamRequest{
				Prompt:      p,
				Temperature: 0.7,
				MaxTokens:   150,
			}
			
			err := client.LLM().Stream(ctx, req, func(chunk string) error {
				buffer += chunk
				return nil
			})
			
			if err != nil {
				errors[index] = err
			} else {
				results[index] = buffer
			}
		}(i, prompt)
	}
	
	wg.Wait()
	
	// Display results
	for i, result := range results {
		if errors[i] != nil {
			fmt.Printf("\n[Error] Prompt %d: %v\n", i+1, errors[i])
		} else {
			fmt.Printf("\n[Result %d]:\n%s\n", i+1, result)
		}
	}
}

func eventDrivenExample(ctx context.Context, client core.Client) {
	// Simulate an event-driven system using RAGO
	
	// Event channel
	events := make(chan Event, 10)
	
	// Event processor
	processor := &EventProcessor{
		client: client,
		events: events,
	}
	
	// Start processing events
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		processor.ProcessEvents(ctx)
	}()
	
	// Generate sample events
	fmt.Println("Generating and processing events:")
	
	sampleEvents := []Event{
		{
			Type:    "user_query",
			Payload: "What is machine learning?",
		},
		{
			Type:    "document_upload",
			Payload: "Machine learning is a subset of AI that enables systems to learn from data.",
		},
		{
			Type:    "task_request",
			Payload: "Analyze the uploaded document",
		},
	}
	
	for _, event := range sampleEvents {
		events <- event
		time.Sleep(1 * time.Second) // Simulate time between events
	}
	
	close(events)
	wg.Wait()
	
	fmt.Println("Event processing completed")
}

// Event represents a system event
type Event struct {
	Type      string
	Payload   string
	Timestamp time.Time
}

// EventProcessor processes events using RAGO
type EventProcessor struct {
	client core.Client
	events chan Event
}

// ProcessEvents processes events from the channel
func (p *EventProcessor) ProcessEvents(ctx context.Context) {
	for event := range p.events {
		p.processEvent(ctx, event)
	}
}

func (p *EventProcessor) processEvent(ctx context.Context, event Event) {
	fmt.Printf("\n[Event: %s] Processing...\n", event.Type)
	
	switch event.Type {
	case "user_query":
		// Process user query with streaming response
		req := core.StreamRequest{
			Prompt:      event.Payload,
			Temperature: 0.7,
			MaxTokens:   100,
		}
		
		fmt.Print("Response: ")
		err := p.client.LLM().Stream(ctx, req, func(chunk string) error {
			fmt.Print(chunk)
			return nil
		})
		fmt.Println()
		
		if err != nil {
			log.Printf("Failed to process query: %v", err)
		}
		
	case "document_upload":
		// Ingest document into RAG
		req := core.IngestRequest{
			DocumentID: fmt.Sprintf("doc_%d", time.Now().Unix()),
			Content:    event.Payload,
			Metadata: map[string]interface{}{
				"source": "event_driven",
				"timestamp": time.Now(),
			},
		}
		
		resp, err := p.client.RAG().Ingest(ctx, req)
		if err != nil {
			log.Printf("Failed to ingest document: %v", err)
		} else {
			fmt.Printf("Document ingested: %d chunks created\n", resp.ChunkCount)
		}
		
	case "task_request":
		// Execute task if agents are available
		if p.client.Agents() != nil {
			taskReq := core.TaskRequest{
				TaskID:      fmt.Sprintf("task_%d", time.Now().Unix()),
				Description: event.Payload,
			}
			
			resp, err := p.client.ExecuteTask(ctx, taskReq)
			if err != nil {
				log.Printf("Failed to execute task: %v", err)
			} else {
				fmt.Printf("Task completed: %s\n", resp.Status)
			}
		} else {
			// Fallback to LLM
			req := core.GenerateRequest{
				Prompt:      event.Payload,
				Temperature: 0.5,
				MaxTokens:   200,
			}
			
			resp, err := p.client.LLM().Generate(ctx, req)
			if err != nil {
				log.Printf("Failed to process task: %v", err)
			} else {
				fmt.Printf("Task result: %s\n", resp.Text)
			}
		}
		
	default:
		fmt.Printf("Unknown event type: %s\n", event.Type)
	}
}