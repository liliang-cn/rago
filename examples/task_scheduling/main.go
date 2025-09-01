package main

import (
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	fmt.Println("🚀 RAGO Task Scheduling Example")
	
	// Create client
	c, err := client.New("")
	if err != nil {
		log.Fatal("Failed to create RAGO client:", err)
	}
	defer c.Close()

	fmt.Println("✅ Client created successfully!")

	// Example 1: Query about task management capabilities
	fmt.Println("\n📋 Testing query about task management...")
	
	response, err := c.Query("How can I schedule and manage automated tasks?")
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	fmt.Println("🤖 Response:", response.Answer)
	
	// Show tool usage if any
	if len(response.ToolsUsed) > 0 {
		fmt.Printf("🛠️  Tools used: %v\n", response.ToolsUsed)
	}

	// Example 2: Demonstrate workflow-style queries
	fmt.Println("\n⚡ Testing workflow-style processing...")
	
	// First, ingest some sample data
	sampleData := `
# Daily Tasks
- Review project progress
- Update documentation  
- Schedule team meetings
- Process user feedback
`
	
	err = c.IngestText(sampleData, "daily_tasks")
	if err != nil {
		log.Printf("Failed to ingest sample data: %v", err)
	} else {
		fmt.Println("✅ Sample task data ingested")
	}
	
	// Then query it
	response, err = c.Query("What are my daily tasks and how should I prioritize them?")
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Println("🤖 Task Analysis:", response.Answer)
	}

	fmt.Println("\n💡 Note: RAGO can help you manage and organize tasks through")
	fmt.Println("   intelligent document processing and contextual queries.")
	fmt.Println("   For advanced task scheduling features, check the RAGO")
	fmt.Println("   server's task management capabilities.")
	fmt.Println("\n✨ Task scheduling example completed!")
}