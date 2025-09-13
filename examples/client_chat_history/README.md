# Interactive Chat with History Example

This example demonstrates advanced chat capabilities with conversation history management.

## Features Demonstrated

- Conversation history management
- Chat with persistent context
- RAG-enhanced chat with history
- Interactive chat mode with commands
- Streaming responses with history
- History manipulation (view, clear)
- Dynamic RAG mode toggling

## Usage

```bash
# Run the interactive chat demo
go run main.go
```

## What It Does

1. **Basic Chat with History**: Shows how conversation context is maintained across messages
2. **RAG-Enhanced Chat**: Combines knowledge base with conversation history
3. **Interactive Mode**: Provides a full interactive chat experience with commands
4. **Streaming Responses**: Demonstrates real-time streaming with history context

## Interactive Commands

When in interactive mode:
- `quit` or `exit` - Exit the program
- `history` - Show conversation history
- `clear` - Clear conversation history
- `rag` - Toggle RAG mode on/off
- Any other text - Chat with the assistant

## Conversation History Features

### History Management
```go
// Create history with system prompt and max messages
history := client.NewConversationHistory(systemPrompt, maxMessages)

// Add messages
history.AddUserMessage("user input")
history.AddAssistantMessage("response", toolCalls)

// History is automatically trimmed to maintain max size
```

### Different Chat Modes
- **Basic Chat**: Uses only conversation history
- **RAG Chat**: Combines knowledge base with history
- **MCP Chat**: Includes tool calls in history
- **Combined**: RAG + MCP + history

## Configuration Options

```go
// Customize generation options
opts := &domain.GenerationOptions{
    Temperature: 0.7,  // Creativity level
    MaxTokens:   500,  // Response length
}
```

## Use Cases

- Customer support chatbots
- Educational tutoring systems
- Technical documentation assistants
- Interactive coding helpers
- Creative writing assistants

## Prerequisites

- LLM provider must be running
- For RAG mode: Knowledge base should have content
- For best results: Use a conversational model