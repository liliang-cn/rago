// Package ptc provides CLI commands for PTC (Programmatic Tool Calling)
package ptc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/ptc"
	ptcgrpc "github.com/liliang-cn/rago/v2/pkg/ptc/grpc"
	"github.com/liliang-cn/rago/v2/pkg/ptc/runtime/goja"
	"github.com/liliang-cn/rago/v2/pkg/ptc/runtime/wazero"
	ptcstore "github.com/liliang-cn/rago/v2/pkg/ptc/store"
	"github.com/spf13/cobra"
)

var (
	Cfg    *config.Config
	Verbose bool
)

// SetSharedVariables sets shared variables from root command
func SetSharedVariables(cfg *config.Config, verbose bool) {
	Cfg = cfg
	Verbose = verbose
}

// Cmd is the parent command for PTC operations
var Cmd = &cobra.Command{
	Use:   "ptc",
	Short: "PTC (Programmatic Tool Calling) - Execute LLM-generated code safely",
	Long: `PTC allows LLMs to generate code instead of JSON parameters for tool calls.
The code is executed in a secure sandbox environment.

Examples:
  # Execute JavaScript code
  rago ptc execute --code "return callTool('rag_query', {query: 'test'})"

  # Execute code from file
  rago ptc execute --file script.js

  # List available tools
  rago ptc tools

  # Start gRPC server
  rago ptc serve`,
}

var executeCmd = &cobra.Command{
	Use:   "execute",
	Short: "Execute code in the PTC sandbox",
	Long: `Execute JavaScript code in a secure sandbox environment.

The code can call registered tools using the callTool() function.

Examples:
  rago ptc execute --code "console.log('Hello, World!')"
  rago ptc execute --code "return callTool('rag_query', {query: 'test'})"
  rago ptc execute --file myscript.js --timeout 60s`,
	RunE: runExecute,
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available tools for PTC",
	RunE:  runTools,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the PTC gRPC server",
	Long: `Start a gRPC server for PTC execution.
This allows external services to call the PTC service.`,
	RunE: runServe,
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show execution history",
	RunE:  runHistory,
}

var (
	// Execute flags
	codeString    string
	codeFile      string
	execTimeout   string
	execLanguage  string
	execContext   string
	execTools     []string
	execMaxMemory int
	execRuntime   string
	outputJSON    bool
)

func init() {
	Cmd.AddCommand(executeCmd)
	Cmd.AddCommand(toolsCmd)
	Cmd.AddCommand(serveCmd)
	Cmd.AddCommand(historyCmd)

	// Execute command flags
	executeCmd.Flags().StringVarP(&codeString, "code", "c", "", "Code to execute")
	executeCmd.Flags().StringVarP(&codeFile, "file", "f", "", "File containing code to execute")
	executeCmd.Flags().StringVarP(&execTimeout, "timeout", "t", "30s", "Execution timeout")
	executeCmd.Flags().StringVarP(&execLanguage, "language", "l", "javascript", "Code language (javascript)")
	executeCmd.Flags().StringVarP(&execContext, "context", "x", "", "JSON context variables to inject")
	executeCmd.Flags().StringSliceVarP(&execTools, "tools", "T", []string{}, "Allowed tools (comma-separated)")
	executeCmd.Flags().IntVarP(&execMaxMemory, "memory", "m", 64, "Maximum memory in MB")
	executeCmd.Flags().StringVarP(&execRuntime, "runtime", "r", "goja", "Runtime to use (goja or wazero)")
	executeCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "Output result as JSON")

	// Tools command flags
	toolsCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "Output as JSON")

	// Serve command flags
	serveCmd.Flags().String("address", "unix:///tmp/ptc.sock", "Server address (unix://path or host:port)")
	serveCmd.Flags().String("runtime", "goja", "Runtime to use (goja or wazero)")
}

func runExecute(cmd *cobra.Command, args []string) error {
	// Get code from flag or file
	code := codeString
	if code == "" && codeFile != "" {
		data, err := os.ReadFile(codeFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		code = string(data)
	}

	if code == "" {
		// Try reading from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			code = string(data)
		}
	}

	if code == "" {
		return fmt.Errorf("no code provided. Use --code, --file, or pipe via stdin")
	}

	// Parse timeout
	timeout, err := time.ParseDuration(execTimeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}

	// Parse context
	contextVars := make(map[string]interface{})
	if execContext != "" {
		if err := json.Unmarshal([]byte(execContext), &contextVars); err != nil {
			return fmt.Errorf("invalid context JSON: %w", err)
		}
	}

	// Create PTC service
	ptcConfig := ptc.DefaultConfig()
	ptcConfig.Enabled = true
	ptcConfig.DefaultTimeout = timeout
	ptcConfig.MaxMemoryMB = execMaxMemory

	router := ptc.NewRAGORouter()
	store := ptcstore.NewMemoryStore(100)

	service, err := ptc.NewService(&ptcConfig, router, store)
	if err != nil {
		return fmt.Errorf("failed to create PTC service: %w", err)
	}

	// Create and set runtime based on selection
	var runtime ptc.SandboxRuntime
	switch execRuntime {
	case "wazero", "wasm":
		runtime = wazero.NewRuntimeWithConfig(&ptcConfig)
	case "goja", "js":
		runtime = goja.NewRuntimeWithConfig(&ptcConfig)
	default:
		runtime = goja.NewRuntimeWithConfig(&ptcConfig)
	}
	service.SetRuntime(runtime)

	// Build execution request
	req := &ptc.ExecutionRequest{
		Code:        code,
		Language:    ptc.LanguageType(execLanguage),
		Context:     contextVars,
		Tools:       execTools,
		Timeout:     timeout,
		MaxMemoryMB: execMaxMemory,
	}

	// Execute
	ctx := context.Background()
	start := time.Now()
	result, err := service.Execute(ctx, req)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	if outputJSON {
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Human-readable output
	fmt.Printf("Execution ID: %s\n", result.ID)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Success: %v\n", result.Success)

	if result.ReturnValue != nil {
		fmt.Printf("\nReturn Value:\n")
		printValue(result.ReturnValue)
	}

	if result.Output != nil {
		fmt.Printf("\nOutput:\n")
		printValue(result.Output)
	}

	if len(result.Logs) > 0 {
		fmt.Printf("\nLogs:\n")
		for _, log := range result.Logs {
			fmt.Printf("  %s\n", log)
		}
	}

	if len(result.ToolCalls) > 0 {
		fmt.Printf("\nTool Calls (%d):\n", len(result.ToolCalls))
		for _, tc := range result.ToolCalls {
			fmt.Printf("  - %s (%v)\n", tc.ToolName, tc.Duration)
			if tc.Error != "" {
				fmt.Printf("    Error: %s\n", tc.Error)
			} else if tc.Result != nil {
				fmt.Printf("    Result: %v\n", tc.Result)
			}
		}
	}

	if result.Error != "" {
		fmt.Printf("\nError: %s\n", result.Error)
	}

	_ = start // used for tracking
	return nil
}

func runTools(cmd *cobra.Command, args []string) error {
	router := ptc.NewRAGORouter()

	tools, err := router.ListAvailableTools(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	if outputJSON {
		output, err := json.MarshalIndent(tools, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal tools: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	fmt.Printf("Available Tools (%d):\n\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  %s\n", tool.Name)
		if tool.Description != "" {
			fmt.Printf("    %s\n", tool.Description)
		}
		if tool.Category != "" {
			fmt.Printf("    Category: %s\n", tool.Category)
		}
		fmt.Println()
	}

	return nil
}

func runServe(cmd *cobra.Command, args []string) error {
	address, _ := cmd.Flags().GetString("address")
	runtimeType, _ := cmd.Flags().GetString("runtime")

	// Create PTC service
	ptcConfig := ptc.DefaultConfig()
	ptcConfig.Enabled = true
	ptcConfig.GRPC.Enabled = true
	ptcConfig.GRPC.Address = address

	router := ptc.NewRAGORouter()
	store := ptcstore.NewMemoryStore(1000)

	service, err := ptc.NewService(&ptcConfig, router, store)
	if err != nil {
		return fmt.Errorf("failed to create PTC service: %w", err)
	}

	// Create and set runtime based on selection
	var runtime ptc.SandboxRuntime
	switch runtimeType {
	case "wazero", "wasm":
		runtime = wazero.NewRuntimeWithConfig(&ptcConfig)
	default:
		runtime = goja.NewRuntimeWithConfig(&ptcConfig)
	}
	service.SetRuntime(runtime)

	// Create and start gRPC server
	grpcServer := ptcgrpc.NewGRPCServer(service, &ptcConfig.GRPC)

	fmt.Printf("Starting PTC gRPC server on %s (runtime: %s)\n", address, runtimeType)
	if err := grpcServer.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	fmt.Println("Server started. Press Ctrl+C to stop.")

	// Wait for interrupt
	select {}
}

func runHistory(cmd *cobra.Command, args []string) error {
	// For now, history is not persisted between commands
	fmt.Println("History is only available during a session.")
	fmt.Println("Use --json flag with execute command to capture results.")
	return nil
}

func printValue(v interface{}) {
	switch val := v.(type) {
	case string:
		fmt.Println(val)
	default:
		b, err := json.MarshalIndent(val, "", "  ")
		if err != nil {
			fmt.Printf("%v\n", val)
		} else {
			fmt.Println(string(b))
		}
	}
}
