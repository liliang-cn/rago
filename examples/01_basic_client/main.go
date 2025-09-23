// Example: Basic Client Initialization
// This example shows different ways to initialize the RAGO client

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/config"
)

func main() {
	// Method 1: Initialize with default config file (~/.rago/rago.toml)
	fmt.Println("=== Method 1: Using Default Config ===")
	defaultConfigPath := filepath.Join(os.Getenv("HOME"), ".rago", "rago.toml")
	if _, err := os.Stat(defaultConfigPath); err == nil {
		c1, err := client.New(defaultConfigPath)
		if err != nil {
			log.Printf("Error with default config: %v\n", err)
		} else {
			fmt.Println("✓ Client initialized with default config")
			defer c1.Close()

			// Get config info
			cfg := c1.GetConfig()
			fmt.Printf("  Default LLM: %s\n", cfg.Providers.DefaultLLM)
			fmt.Printf("  Default Embedder: %s\n", cfg.Providers.DefaultEmbedder)
		}
	} else {
		fmt.Println("  Default config not found, skipping...")
	}

	// Method 2: Initialize with custom config file
	fmt.Println("\n=== Method 2: Using Custom Config File ===")
	customConfig := "custom.toml"
	if _, err := os.Stat(customConfig); err == nil {
		c2, err := client.New(customConfig)
		if err != nil {
			log.Printf("Error with custom config: %v\n", err)
		} else {
			fmt.Println("✓ Client initialized with custom config")
			defer c2.Close()
		}
	} else {
		fmt.Println("  Custom config not found, skipping...")
	}

	// Method 3: Initialize with programmatic config
	fmt.Println("\n=== Method 3: Using Programmatic Config ===")
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			DefaultLLM:      "ollama",
			DefaultEmbedder: "ollama",
		},
	}

	c3, err := client.NewWithConfig(cfg)
	if err != nil {
		log.Printf("Error with programmatic config: %v\n", err)
	} else {
		fmt.Println("✓ Client initialized with programmatic config")
		defer c3.Close()

		// Test basic operations
		fmt.Println("\n  Testing client capabilities:")

		// Check if LLM is available
		if c3.LLM != nil {
			fmt.Println("  ✓ LLM wrapper available")
		}

		// Check if RAG is available
		if c3.RAG != nil {
			fmt.Println("  ✓ RAG wrapper available")
		}

		// Check if Tools are available
		if c3.Tools != nil {
			fmt.Println("  ✓ Tools wrapper available")
		}

		// Check if Agent is available
		if c3.Agent != nil {
			fmt.Println("  ✓ Agent wrapper available")
		}
	}

	// Method 4: Error handling example
	fmt.Println("\n=== Method 4: Error Handling ===")
	_, err = client.New("/non/existent/path.toml")
	if err != nil {
		fmt.Printf("✓ Expected error caught: %v\n", err)
	}

	// Method 5: Using client for simple task
	fmt.Println("\n=== Method 5: Simple Task Execution ===")
	if c3 != nil {
		ctx := context.Background()
		req := client.TaskRequest{
			Task:    "Hello World",
			Verbose: false,
		}

		resp, err := c3.RunTask(ctx, req)
		if err != nil {
			log.Printf("Task error: %v\n", err)
		} else {
			fmt.Printf("✓ Task completed: %v\n", resp.Success)
			if resp.Output != nil {
				fmt.Printf("  Output: %v\n", resp.Output)
			}
		}
	}

	fmt.Println("\n=== Example Complete ===")
}
