package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// MCP JSON-RPC message structures
type Request struct {
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP specific structures
type InitializeResult struct {
	ProtocolVersion string   `json:"protocolVersion"`
	ServerInfo      ServerInfo `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Capabilities struct {
	Tools ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for scanner.Scan() {
		line := scanner.Text()
		
		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue // Ignore malformed requests
		}

		var resp Response
		resp.Jsonrpc = "2.0"
		resp.ID = req.ID

		switch req.Method {
		case "initialize":
			resp.Result = InitializeResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo: ServerInfo{
					Name:    "mock-time-server",
					Version: "1.0.0",
				},
				Capabilities: Capabilities{
					Tools: ToolsCapability{
						ListChanged: false,
					},
				},
			}
			
		case "tools/list":
			resp.Result = map[string]interface{}{
				"tools": []Tool{
					{
						Name:        "get_current_time",
						Description: "Get the current time in various formats",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"format": map[string]interface{}{
									"type":        "string",
									"description": "Time format (unix, iso, rfc3339)",
									"default":     "iso",
								},
							},
						},
					},
				},
			}
			
		case "tools/call":
			var params map[string]interface{}
			json.Unmarshal(req.Params, &params)
			
			toolName, _ := params["name"].(string)
			if toolName == "get_current_time" {
				resp.Result = map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC3339)),
						},
					},
				}
			} else {
				resp.Error = &Error{
					Code:    -32601,
					Message: "Tool not found",
				}
			}
			
		default:
			resp.Error = &Error{
				Code:    -32601,
				Message: "Method not found",
			}
		}

		output, _ := json.Marshal(resp)
		writer.Write(output)
		writer.WriteByte('\n')
		writer.Flush()
	}
}