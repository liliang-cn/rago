package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/liliang-cn/rago/v2/pkg/grpc/client"
)

func main() {
	fmt.Println("=== RAGO gRPC Client Example ===")

	// Create client configuration
	config := client.DefaultConfig("localhost:50051")
	config.Timeout = 30 * time.Second

	// Create gRPC client
	grpcClient, err := client.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create gRPC client: %v", err)
	}
	defer grpcClient.Close()

	ctx := context.Background()

	// Example 1: Health Check
	fmt.Println("1. Health Check")
	health, err := grpcClient.HealthCheck(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Printf("   Server healthy: %v\n", health.Healthy)
		fmt.Printf("   Version: %s\n", health.Version)
		for component, status := range health.Components {
			fmt.Printf("   - %s: %s\n", component, status.Status)
		}
	}
	fmt.Println()

	// Example 2: Ingest Document
	fmt.Println("2. Document Ingestion")
	content := `Retrieval-Augmented Generation (RAG) is an AI framework that combines the power of 
	large language models with external knowledge retrieval. RAG systems first retrieve relevant 
	information from a knowledge base, then use this context to generate more accurate and 
	grounded responses. This approach reduces hallucinations and allows models to access 
	up-to-date information beyond their training data.`

	ingestResp, err := grpcClient.IngestDocument(ctx, content,
		client.WithCollection("demo"),
		client.WithChunkOptions("sentence", 100, 20),
		client.WithMetadata(map[string]string{
			"source": "example",
			"type":   "educational",
		}),
	)
	if err != nil {
		log.Printf("Document ingestion failed: %v", err)
	} else {
		fmt.Printf("   Document ID: %s\n", ingestResp.DocumentId)
		fmt.Printf("   Chunks created: %d\n", ingestResp.ChunkCount)
		fmt.Printf("   Success: %v\n", ingestResp.Success)
	}
	fmt.Println()

	// Example 3: Query
	fmt.Println("3. RAG Query")
	queryResp, err := grpcClient.Query(ctx, "What is RAG and how does it work?",
		client.WithTopK(3),
		client.WithQueryCollection("demo"),
	)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("   Answer: %s\n", queryResp.Answer)
		fmt.Printf("   Processing time: %dms\n", queryResp.ProcessingTimeMs)
		fmt.Printf("   Results found: %d\n", len(queryResp.Results))
		for i, result := range queryResp.Results {
			fmt.Printf("   [%d] Score: %.3f, Source: %s\n", i+1, result.Score, result.Source)
		}
	}
	fmt.Println()

	// Example 4: Text Generation
	fmt.Println("4. Text Generation")
	genResp, err := grpcClient.Generate(ctx, "Explain quantum computing in simple terms",
		client.WithMaxTokens(200),
		client.WithTemperature(0.7),
	)
	if err != nil {
		log.Printf("Generation failed: %v", err)
	} else {
		fmt.Printf("   Response: %s\n", genResp.Text)
		fmt.Printf("   Model: %s\n", genResp.Model)
		fmt.Printf("   Generation time: %dms\n", genResp.GenerationTimeMs)
	}
	fmt.Println()

	// Example 5: Streaming Query
	fmt.Println("5. Streaming Query")
	fmt.Print("   Streaming response: ")
	err = grpcClient.StreamQuery(ctx, "Tell me about machine learning",
		func(resp *pb.StreamQueryResponse) error {
			switch content := resp.Content.(type) {
			case *pb.StreamQueryResponse_TextChunk:
				fmt.Print(content.TextChunk)
			case *pb.StreamQueryResponse_Metadata:
				fmt.Printf("\n   Processing time: %dms\n", content.Metadata.ProcessingTimeMs)
			case *pb.StreamQueryResponse_Error:
				return fmt.Errorf("stream error: %s", content.Error)
			}
			return nil
		},
		client.WithTopK(2),
	)
	if err != nil && err != io.EOF {
		log.Printf("Streaming query failed: %v", err)
	}
	fmt.Println()

	// Example 6: Streaming Generation
	fmt.Println("6. Streaming Text Generation")
	fmt.Print("   Generated text: ")
	err = grpcClient.StreamGenerate(ctx, "Write a haiku about programming",
		func(resp *pb.StreamGenerateResponse) error {
			switch content := resp.Content.(type) {
			case *pb.StreamGenerateResponse_TextChunk:
				fmt.Print(content.TextChunk)
			case *pb.StreamGenerateResponse_Metadata:
				fmt.Printf("\n   Tokens used: %d\n", content.Metadata.TokensUsed)
				fmt.Printf("   Finish reason: %s\n", content.Metadata.FinishReason)
			case *pb.StreamGenerateResponse_Error:
				return fmt.Errorf("stream error: %s", content.Error)
			}
			return nil
		},
		client.WithMaxTokens(50),
		client.WithTemperature(0.9),
	)
	if err != nil && err != io.EOF {
		log.Printf("Streaming generation failed: %v", err)
	}
	fmt.Println()

	// Example 7: Embeddings
	fmt.Println("7. Embeddings")
	embResp, err := grpcClient.GenerateEmbedding(ctx, "machine learning algorithms", "")
	if err != nil {
		log.Printf("Embedding generation failed: %v", err)
	} else {
		fmt.Printf("   Embedding dimensions: %d\n", embResp.Dimensions)
		fmt.Printf("   First 5 values: %.4f, %.4f, %.4f, %.4f, %.4f\n",
			embResp.Embedding[0], embResp.Embedding[1], embResp.Embedding[2],
			embResp.Embedding[3], embResp.Embedding[4])
	}
	fmt.Println()

	// Example 8: Batch Embeddings
	fmt.Println("8. Batch Embeddings")
	texts := []string{
		"artificial intelligence",
		"machine learning",
		"deep learning",
	}
	batchResp, err := grpcClient.BatchGenerateEmbeddings(ctx, texts, "")
	if err != nil {
		log.Printf("Batch embedding generation failed: %v", err)
	} else {
		fmt.Printf("   Generated %d embeddings\n", len(batchResp.Results))
		for i, result := range batchResp.Results {
			if result.Error != "" {
				fmt.Printf("   [%d] Error: %s\n", i, result.Error)
			} else {
				fmt.Printf("   [%d] Dimensions: %d\n", i, len(result.Embedding))
			}
		}
	}
	fmt.Println()

	// Example 9: Similarity Computation
	fmt.Println("9. Similarity Computation")
	simResp, err := grpcClient.ComputeSimilarity(ctx,
		"machine learning",
		"artificial intelligence",
		"cosine",
	)
	if err != nil {
		log.Printf("Similarity computation failed: %v", err)
	} else {
		fmt.Printf("   Similarity score: %.4f\n", simResp.Similarity)
		fmt.Printf("   Metric used: %s\n", simResp.Metric)
	}

	fmt.Println("\n✅ All gRPC examples completed!")
	fmt.Println("\nTo run this example:")
	fmt.Println("1. Start the gRPC server: rago grpc --port 50051")
	fmt.Println("2. Run this client: go run examples/grpc_usage/main.go")
}