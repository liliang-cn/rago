package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/spf13/cobra"
)


var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of LLM providers",
	Long:  `Display LLM provider connectivity and health status.`,
	RunE:  runStatusV3,
}

func runStatusV3(cmd *cobra.Command, args []string) error {
	fmt.Println("RAGO LLM Status")
	fmt.Println(strings.Repeat("=", 60))
	
	// Suppress logs for status command (always quiet for cleaner output)
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(originalOutput)
	
	// Also redirect stdout logs
	if !verbose {
		// Save original stderr
		originalStderr := os.Stderr
		// Create a pipe to discard output
		r, w, _ := os.Pipe()
		os.Stderr = w
		defer func() {
			w.Close()
			os.Stderr = originalStderr
			// Drain the pipe
			io.Copy(io.Discard, r)
			r.Close()
		}()
	}
	
	// Load configuration
	coreConfig, err := loadCoreConfig()
	if err != nil {
		fmt.Printf("\n⚠️  Configuration: Error loading config: %v\n", err)
		return nil
	}
	
	fmt.Println("\n✅ Configuration: Loaded successfully")

	// Create client
	clientInstance, err := client.NewWithConfig(coreConfig)
	if err != nil {
		fmt.Printf("\n❌ Client initialization failed: %v\n", err)
		return nil
	}
	defer clientInstance.Close()

	// Get initial health report
	health := clientInstance.Health()
	
	// Check if LLM status is unknown
	llmStatus, hasLLM := health.Pillars["LLM"]
	if hasLLM && llmStatus == core.HealthStatusUnknown {
		// Trigger LLM health check
		fmt.Println("\n⏳ Checking LLM providers...")
		
		healthCheckDone := make(chan struct{})
		go func() {
			clientInstance.TriggerHealthCheck() // Only check LLM
			close(healthCheckDone)
		}()
		
		// Wait for health check with timeout
		select {
		case <-healthCheckDone:
			// Health check completed
		case <-time.After(2 * time.Second):
			// Timeout - continue anyway
		}
		
		// Get updated health report
		health = clientInstance.Health()
	}
	
	// Display LLM status
	fmt.Println("\n🧠 LLM Pillar:")
	
	if llmHealth, ok := health.Pillars["LLM"]; ok {
		fmt.Printf("   Status: %s\n", getStatusIcon(string(llmHealth)))
		
		if clientInstance.LLM() != nil {
			fmt.Printf("   LLM Service: Available\n")
		}
	} else {
		fmt.Println("   Status: Not configured")
	}
	
	return nil
}


func getStatusIcon(status string) string {
	switch status {
	case "healthy", "running", "active":
		return "✅ " + status
	case "degraded", "warning":
		return "⚠️  " + status
	case "unhealthy", "error", "failed":
		return "❌ " + status
	default:
		return "❓ " + status
	}
}

func init() {
	// Status command can be enhanced with flags in the future if needed
}