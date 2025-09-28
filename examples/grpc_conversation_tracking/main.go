package main

import (
	"fmt"

	"github.com/google/uuid"
)

func main() {
	fmt.Println("gRPC Conversation Tracking Example")
	fmt.Println("===================================")

	// Demonstrate conversation ID generation
	conversationID := uuid.New().String()
	fmt.Printf("Generated Conversation ID: %s\n", conversationID)

	// Show the updated proto message structures
	fmt.Println("\nUpdated Proto Message Structures:")
	fmt.Println("---------------------------------")
	
	fmt.Println("1. QueryRequest now includes:")
	fmt.Println("   - conversation_id field for tracking conversations")
	
	fmt.Println("\n2. GenerateRequest now includes:")
	fmt.Println("   - conversation_id field for LLM tracking")
	
	fmt.Println("\n3. New ConversationService with:")
	fmt.Println("   - SaveConversation: Save conversation with UUID")
	fmt.Println("   - GetConversation: Retrieve by UUID")
	fmt.Println("   - ListConversations: List with pagination")
	fmt.Println("   - DeleteConversation: Delete by UUID")
	
	fmt.Println("\n4. New UsageService with:")
	fmt.Println("   - RecordUsage: Track token usage per conversation")
	fmt.Println("   - GetUsageStats: Get aggregated statistics")
	fmt.Println("   - GetUsageHistory: Get detailed history")
	
	fmt.Println("\nKey Features:")
	fmt.Println("-------------")
	fmt.Println("✓ All conversation IDs use UUID format")
	fmt.Println("✓ Timestamps use Unix format (int64)")
	fmt.Println("✓ Usage tracking includes token counts and costs")
	fmt.Println("✓ Full pagination support for listings")
	fmt.Println("✓ Metadata support using protobuf Value types")
	
	// Example conversation message structure
	fmt.Println("\nExample Conversation Message:")
	fmt.Printf(`{
  "id": "%s",
  "title": "RAG System Discussion",
  "messages": [
    {
      "role": "user",
      "content": "What is RAG?",
      "timestamp": 1735517520
    },
    {
      "role": "assistant", 
      "content": "RAG stands for Retrieval-Augmented Generation...",
      "timestamp": 1735517525
    }
  ],
  "created_at": 1735517520,
  "updated_at": 1735517525
}`, conversationID)

	fmt.Println("\n\nImplementation Files:")
	fmt.Println("--------------------")
	fmt.Println("• proto/rago/rago.proto - Updated proto definitions")
	fmt.Println("• pkg/grpc/server/conversation_service.go - Conversation tracking")
	fmt.Println("• pkg/grpc/server/usage_service.go - Usage tracking")
	fmt.Println("• pkg/grpc/server/server.go - Updated server registration")
}