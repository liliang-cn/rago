# Complete LLM Operations Example

This example demonstrates all LLM operations available through the RAGO client, including direct generation, chat, streaming, and structured output.

## Features Demonstrated

- Simple text generation
- Multi-turn chat conversations
- Streaming responses
- Structured JSON generation
- Temperature control
- Code generation
- Creative vs factual generation
- Chat streaming

## Usage

```bash
go run main.go
```

## What It Does

1. **Simple Generation**: Basic prompt-response generation
2. **Multi-turn Chat**: Maintains conversation context
3. **Streaming**: Real-time token-by-token output
4. **Structured Output**: Generate JSON matching a schema
5. **Temperature Effects**: Compare outputs at different temperatures
6. **Code Generation**: Generate code with documentation
7. **Style Control**: Factual vs creative outputs

## LLM Operations

### Simple Generation
```go
req := client.LLMGenerateRequest{
    Prompt:      "Your prompt",
    Temperature: 0.7,
    MaxTokens:   500,
}
resp, err := client.LLMGenerate(ctx, req)
```

### Chat with History
```go
req := client.LLMChatRequest{
    Messages: []client.ChatMessage{
        {Role: "user", Content: "Hello"},
        {Role: "assistant", Content: "Hi!"},
        {Role: "user", Content: "How are you?"},
    },
    Temperature: 0.7,
    MaxTokens:   500,
}
resp, err := client.LLMChat(ctx, req)
```

### Structured Generation
```go
type MyStruct struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}

req := client.LLMStructuredRequest{
    Prompt:      "Generate data",
    Schema:      MyStruct{},
    Temperature: 0.3,
    MaxTokens:   500,
}
resp, err := client.LLMGenerateStructured(ctx, req)
```

### Streaming
```go
err := client.LLMGenerateStream(ctx, req, func(chunk string) {
    fmt.Print(chunk) // Handle each token
})
```

## Temperature Guide

- **0.0-0.3**: Deterministic, factual, consistent
- **0.4-0.6**: Balanced creativity and accuracy
- **0.7-0.8**: Creative but coherent
- **0.9-1.0**: Very creative, may be inconsistent

## Use Cases

### Low Temperature (0.1-0.3)
- Code generation
- Technical documentation
- Factual Q&A
- Data extraction
- API responses

### Medium Temperature (0.4-0.7)
- General chatbots
- Educational content
- Product descriptions
- Email drafts
- Summary generation

### High Temperature (0.8-1.0)
- Creative writing
- Brainstorming
- Story generation
- Marketing copy
- Poetry and art

## Structured Generation Use Cases

- **Code Review**: Analyze code and return structured feedback
- **Data Extraction**: Extract entities from text
- **Form Generation**: Create structured forms from descriptions
- **API Responses**: Generate consistent API response formats
- **Report Generation**: Create structured reports

## Best Practices

1. **Use streaming** for long responses to improve UX
2. **Lower temperature** for factual/technical content
3. **Higher temperature** for creative tasks
4. **Structured generation** for consistent output formats
5. **Chat mode** for maintaining context across turns
6. **Set appropriate MaxTokens** to control response length
7. **Include system prompts** in chat for role definition

## Prerequisites

- LLM provider must be running (Ollama, OpenAI, etc.)
- Provider must support all operations (some may not support structured generation)
- Sufficient model context length for chat history