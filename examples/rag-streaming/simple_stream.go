package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// SimpleStreamingExample demonstrates basic RAG streaming usage
func SimpleStreamingExample() {
	// Initialize RAGO client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Prepare a question for RAG Q&A
	question := "What are the benefits of neural enhancement technology?"

	fmt.Printf("🤖 RAG Streaming Q&A\n")
	fmt.Printf("Question: %s\n", question)
	fmt.Printf("Answer: ")

	// Create streaming request
	req := core.QARequest{
		Question:    question,
		MaxSources:  5,
		MinScore:    0.1,
		Temperature: 0.7,
		MaxTokens:   200,
		Stream:      true, // Enable streaming
	}

	// Stream the answer
	err = ragoClient.RAG().StreamAnswer(ctx, req, func(chunk core.StreamChunk) error {
		if !chunk.Finished {
			// Print only the new content (delta)
			fmt.Print(chunk.Delta)
		} else {
			// Final chunk with statistics
			fmt.Printf("\n\n✅ Completed in %v\n", chunk.Duration)
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Streaming failed: %v", err)
	}
}

// ComparisonExample shows the difference between streaming and non-streaming
func ComparisonExample() {
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()
	question := "Explain how memory enhancement works"

	// Non-streaming version
	fmt.Println("\n📄 Non-Streaming Version:")
	start := time.Now()
	
	nonStreamReq := core.QARequest{
		Question:  question,
		MaxTokens: 150,
		Stream:    false,
	}

	resp, err := ragoClient.RAG().Answer(ctx, nonStreamReq)
	if err != nil {
		log.Fatalf("Non-streaming failed: %v", err)
	}

	fmt.Printf("Answer: %s\n", resp.Answer)
	fmt.Printf("Time to first response: %v\n", time.Since(start))

	// Streaming version
	fmt.Println("\n🎯 Streaming Version:")
	start = time.Now()
	var firstToken time.Duration
	firstTokenReceived := false

	streamReq := core.QARequest{
		Question:  question,
		MaxTokens: 150,
		Stream:    true,
	}

	fmt.Print("Answer: ")
	err = ragoClient.RAG().StreamAnswer(ctx, streamReq, func(chunk core.StreamChunk) error {
		if !firstTokenReceived && chunk.Delta != "" {
			firstToken = time.Since(start)
			firstTokenReceived = true
		}

		if !chunk.Finished {
			fmt.Print(chunk.Delta)
		} else {
			fmt.Printf("\n")
			fmt.Printf("Time to first token: %v\n", firstToken)
			fmt.Printf("Total time: %v\n", chunk.Duration)
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Streaming failed: %v", err)
	}
}

// CustomCallbackExample shows advanced callback handling
func CustomCallbackExample() {
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	fmt.Println("\n🔧 Custom Callback Example:")

	// Create a custom callback that processes the stream
	var fullResponse string
	wordCount := 0

	customCallback := func(chunk core.StreamChunk) error {
		if chunk.Finished {
			fmt.Printf("\n\n📊 Stream Statistics:\n")
			fmt.Printf("  - Words generated: %d\n", wordCount)
			fmt.Printf("  - Total length: %d chars\n", len(fullResponse))
			fmt.Printf("  - Generation time: %v\n", chunk.Duration)
			if chunk.Usage.TotalTokens > 0 {
				fmt.Printf("  - Tokens used: %d\n", chunk.Usage.TotalTokens)
			}
			return nil
		}

		// Process the delta
		if chunk.Delta != "" {
			fullResponse += chunk.Delta
			wordCount += len(strings.Fields(chunk.Delta))
			
			// You could do custom processing here, like:
			// - Send to WebSocket
			// - Update UI
			// - Save to buffer
			// - Apply filters
			
			fmt.Print(chunk.Delta)
		}

		return nil
	}

	req := core.QARequest{
		Question: "What are the safety protocols for neural implants?",
		Stream:   true,
	}

	fmt.Printf("Question: %s\n", req.Question)
	fmt.Print("Answer: ")

	err = ragoClient.RAG().StreamAnswer(ctx, req, customCallback)
	if err != nil {
		log.Fatalf("Streaming with custom callback failed: %v", err)
	}
}

// ErrorHandlingExample shows how to handle streaming errors
func ErrorHandlingExample() {
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	fmt.Println("\n⚠️ Error Handling Example:")

	// Example with timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := core.QARequest{
		Question: "Provide a detailed analysis of cognitive enhancement",
		Stream:   true,
		MaxTokens: 1000, // Large response that might timeout
	}

	err = ragoClient.RAG().StreamAnswer(timeoutCtx, req, func(chunk core.StreamChunk) error {
		if !chunk.Finished {
			fmt.Print(chunk.Delta)
		}
		
		// Simulate error condition
		if strings.Contains(chunk.Delta, "error") {
			return fmt.Errorf("detected error keyword in stream")
		}
		
		return nil
	})

	if err != nil {
		fmt.Printf("\n\n❌ Stream error: %v\n", err)
	}
}

func main() {
	fmt.Println("🚀 RAGO RAG Streaming Library Examples\n")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Run examples
	SimpleStreamingExample()
	
	fmt.Println("\n" + strings.Repeat("-", 50))
	ComparisonExample()
	
	fmt.Println("\n" + strings.Repeat("-", 50))
	CustomCallbackExample()
	
	fmt.Println("\n" + strings.Repeat("-", 50))
	ErrorHandlingExample()

	fmt.Println("\n✨ All examples completed!")
}