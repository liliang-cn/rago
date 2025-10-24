package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"
)

// ChatRequest represents the MCP chat request
type ChatRequest struct {
	Message        string                 `json:"message"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Options        *ChatOptions           `json:"options,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
}

// ChatOptions represents chat configuration
type ChatOptions struct {
	Temperature  float64  `json:"temperature"`
	MaxTokens    int      `json:"max_tokens"`
	ShowThinking bool     `json:"show_thinking"`
	AllowedTools []string `json:"allowed_tools"`
	MaxToolCalls int      `json:"max_tool_calls"`
}

// ChatResponse represents the response from MCP chat
type ChatResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Content        string         `json:"content"`
		FinalResponse  string         `json:"final_response"`
		ToolCalls      []ToolCall     `json:"tool_calls"`
		Thinking       string         `json:"thinking"`
		HasThinking    bool           `json:"has_thinking"`
		ConversationID string         `json:"conversation_id"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

// ToolCall represents a tool execution result
type ToolCall struct {
	ToolName string                 `json:"tool_name"`
	Args     map[string]interface{} `json:"args"`
	Result   interface{}            `json:"result"`
	Success  bool                   `json:"success"`
	Error    string                 `json:"error,omitempty"`
	Duration string                 `json:"duration"`
}

func main() {
	fmt.Println("MCP Chat with LLM Integration Demo")
	fmt.Println("===================================")
	
	// Start the server in the background
	serverCmd := exec.Command("./rago", "serve", "--port", "7127")
	if err := serverCmd.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
	defer serverCmd.Process.Kill()
	
	// Wait for server to be ready
	fmt.Println("Starting RAGO server...")
	time.Sleep(5 * time.Second)
	
	// Test cases
	testCases := []struct {
		name    string
		message string
		options *ChatOptions
	}{
		{
			name:    "Simple question without tools",
			message: "What is the capital of France?",
			options: &ChatOptions{
				Temperature:  0.7,
				MaxTokens:    500,
				ShowThinking: false,
				MaxToolCalls: 0,
			},
		},
		{
			name:    "Request with tool usage",
			message: "Can you check the current system status?",
			options: &ChatOptions{
				Temperature:  0.7,
				MaxTokens:    1000,
				ShowThinking: true,
				MaxToolCalls: 5,
			},
		},
		{
			name:    "Complex request with multiple tools",
			message: "Read the README.md file and summarize its content",
			options: &ChatOptions{
				Temperature:  0.5,
				MaxTokens:    1500,
				ShowThinking: true,
				AllowedTools: []string{"filesystem_read_file", "filesystem_list_directory"},
				MaxToolCalls: 10,
			},
		},
	}
	
	// Conversation ID for maintaining context
	conversationID := "demo-" + fmt.Sprintf("%d", time.Now().Unix())
	
	for i, tc := range testCases {
		fmt.Printf("\n%d. %s\n", i+1, tc.name)
		fmt.Println(strings.Repeat("-", 50))
		
		// Create request
		req := ChatRequest{
			Message:        tc.message,
			ConversationID: conversationID,
			Options:        tc.options,
			Context: map[string]interface{}{
				"test_case": tc.name,
			},
		}
		
		// Send request
		resp, err := sendChatRequest(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		// Display results
		displayResponse(resp)
	}
	
	fmt.Println("\n✅ MCP Chat with LLM Integration demo completed!")
}

func sendChatRequest(req ChatRequest) (*ChatResponse, error) {
	// Convert request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Send HTTP request
	resp, err := http.Post("http://localhost:7127/api/mcp/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Parse response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if !chatResp.Success && chatResp.Error != "" {
		return nil, fmt.Errorf("API error: %s", chatResp.Error)
	}
	
	return &chatResp, nil
}

func displayResponse(resp *ChatResponse) {
	fmt.Printf("📝 User Message: %s\n", resp.Data.ConversationID)
	
	if resp.Data.HasThinking && resp.Data.Thinking != "" {
		fmt.Printf("\n🤔 Thinking:\n%s\n", resp.Data.Thinking)
	}
	
	if len(resp.Data.ToolCalls) > 0 {
		fmt.Printf("\n🔧 Tool Calls:\n")
		for _, tc := range resp.Data.ToolCalls {
			status := "✅"
			if !tc.Success {
				status = "❌"
			}
			fmt.Printf("  %s %s(%v) - Duration: %s\n", status, tc.ToolName, tc.Args, tc.Duration)
			if tc.Success {
				fmt.Printf("     Result: %v\n", tc.Result)
			} else {
				fmt.Printf("     Error: %s\n", tc.Error)
			}
		}
	}
	
	fmt.Printf("\n💬 Response:\n%s\n", resp.Data.FinalResponse)
}

// Add missing import
var strings = struct {
	Repeat func(s string, count int) string
}{
	Repeat: func(s string, count int) string {
		result := ""
		for i := 0; i < count; i++ {
			result += s
		}
		return result
	},
}