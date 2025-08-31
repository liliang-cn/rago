# Structured Output Guide

Rago supports structured JSON output across all major LLM providers (LMStudio, Ollama, OpenAI) using their native structured output capabilities.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    rago "github.com/liliang-cn/rago/lib"
)

type Person struct {
    Name      string   `json:"name"`
    Age       int      `json:"age"`
    City      string   `json:"city"`
    Hobbies   []string `json:"hobbies"`
    IsMarried bool     `json:"is_married"`
}

func main() {
    client, err := rago.New("config.toml")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // For LMStudio: Use struct pointer
    var person Person
    req := rago.LLMStructuredRequest{
        Prompt:      "Generate info for a 25-year-old developer named Alex from NYC who likes coding and reading",
        Schema:      &person,  // LMStudio uses struct pointer
        Temperature: 0.1,
        MaxTokens:   500,
    }

    result, err := client.LLMGenerateStructured(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Valid: %v\n", result.Valid)
    fmt.Printf("Data: %+v\n", person)  // Direct struct access
}
```

## Provider-Specific Usage

### LMStudio Provider

LMStudio uses its native `CompleteWithSchema` method and requires Go struct pointers:

```go
// Define your data structure
type WeatherReport struct {
    City        string  `json:"city"`
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
    Humidity    int     `json:"humidity"`
}

// Use struct pointer as schema
var weather WeatherReport
req := rago.LLMStructuredRequest{
    Prompt: "Generate weather report for Tokyo: sunny, 25°C, 60% humidity",
    Schema: &weather,  // Pass struct pointer
}

result, err := client.LLMGenerateStructured(ctx, req)
// weather struct is populated directly
fmt.Printf("Weather: %+v\n", weather)
```

### Ollama Provider

Ollama uses JSON Schema maps with its `Format` field:

```go
// First configure to use Ollama
// config.toml:
// [providers]
// default_llm = "ollama"
// [providers.ollama]
// llm_model = "qwen3"

// Define JSON schema
schema := map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "city":        map[string]interface{}{"type": "string"},
        "temperature": map[string]interface{}{"type": "number"},
        "condition":   map[string]interface{}{"type": "string"},
        "humidity":    map[string]interface{}{"type": "integer"},
    },
    "required": []string{"city", "temperature", "condition", "humidity"},
}

req := rago.LLMStructuredRequest{
    Prompt: "Generate weather report for Tokyo: sunny, 25°C, 60% humidity",
    Schema: schema,  // Pass JSON schema map
}

result, err := client.LLMGenerateStructured(ctx, req)
// Parse from result.Raw JSON string
var weather WeatherReport
json.Unmarshal([]byte(result.Raw), &weather)
```

### OpenAI Provider

OpenAI uses native structured output with `ResponseFormat`:

```go
// Configure OpenAI provider
// config.toml:
// [providers]
// default_llm = "openai"
// [providers.openai]
// api_key = "your-api-key"
// llm_model = "gpt-4o-2024-08-06"  # Structured output requires specific models

// Use JSON schema (similar to Ollama)
schema := map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "city":        map[string]interface{}{"type": "string"},
        "temperature": map[string]interface{}{"type": "number"},
        "condition":   map[string]interface{}{"type": "string"},
        "humidity":    map[string]interface{}{"type": "integer"},
    },
    "required": []string{"city", "temperature", "condition", "humidity"},
}

req := rago.LLMStructuredRequest{
    Prompt: "Generate weather report for Tokyo",
    Schema: schema,
}

result, err := client.LLMGenerateStructured(ctx, req)
// Parse from result.Raw JSON string
```

## Complex Examples

### Nested Structures

```go
type Company struct {
    Name      string     `json:"name"`
    Founded   int        `json:"founded"`
    Employees []Employee `json:"employees"`
    Location  Address    `json:"location"`
}

type Employee struct {
    Name       string `json:"name"`
    Title      string `json:"title"`
    Department string `json:"department"`
}

type Address struct {
    Street  string `json:"street"`
    City    string `json:"city"`
    Country string `json:"country"`
}

// For LMStudio
var company Company
req := rago.LLMStructuredRequest{
    Prompt: "Generate info for a tech company with 3 employees in San Francisco",
    Schema: &company,
}

// For Ollama/OpenAI - build corresponding JSON schema
```

### Array Responses

```go
type ProductList struct {
    Products []Product `json:"products"`
    Total    int       `json:"total"`
}

type Product struct {
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    Category    string  `json:"category"`
    InStock     bool    `json:"in_stock"`
    Description string  `json:"description"`
}
```

## Best Practices

### 1. Temperature Settings
- Use low temperature (0.0-0.2) for consistent structured output
- Higher temperatures may produce invalid JSON

```go
req := rago.LLMStructuredRequest{
    Temperature: 0.1,  // Low for consistency
    MaxTokens:   1000,
}
```

### 2. Error Handling
- Always check `result.Valid` for JSON validity
- Handle both parsing errors and generation errors

```go
result, err := client.LLMGenerateStructured(ctx, req)
if err != nil {
    log.Printf("Generation failed: %v", err)
    return
}

if !result.Valid {
    log.Printf("Invalid JSON generated: %s", result.Raw)
    // Handle invalid JSON case
}
```

### 3. Schema Design
- Use clear, descriptive field names
- Include `required` fields in JSON schemas
- Consider using JSON tags for Go structs

```go
type User struct {
    Name  string `json:"full_name"`      // Clear naming
    Email string `json:"email_address"`  // Descriptive
    Age   int    `json:"age,omitempty"`  // Optional field
}
```

### 4. Prompt Engineering
- Be specific about the data you want
- Provide examples in prompts when needed
- Keep prompts focused and clear

```go
prompt := `Generate user profile data for:
- Name: John Smith  
- Age: 28
- Location: New York
- Occupation: Software Engineer
- 3 hobbies related to technology`
```

## Provider Comparison

| Feature | LMStudio | Ollama | OpenAI |
|---------|----------|--------|--------|
| Schema Type | Go struct pointer | JSON Schema map | JSON Schema map |
| Native Support | ✅ CompleteWithSchema | ✅ Format field | ✅ ResponseFormat |
| Direct Parsing | ✅ Struct populated | ❌ Manual parsing | ❌ Manual parsing |
| Validation | ✅ Built-in | ⚠️ Basic | ✅ Strict mode |
| Performance | Fast | Fast | Fast |

## Troubleshooting

### Common Issues

1. **"missing method GenerateStructured"**
   - Update all mock objects in tests to include the new method

2. **Invalid JSON from Ollama**
   - Check that the model supports JSON format
   - Verify JSON schema is valid
   - Try lowering temperature

3. **OpenAI model compatibility**
   - Only certain models support structured output (gpt-4o-2024-08-06+)
   - Check your API key and model access

4. **LMStudio struct requirements**
   - Must pass struct pointer, not value
   - Struct must be exported (capitalized fields)
   - JSON tags are recommended

### Debug Tips

```go
// Enable debug logging
result, err := client.LLMGenerateStructured(ctx, req)
fmt.Printf("Raw response: %s\n", result.Raw)
fmt.Printf("Valid: %v\n", result.Valid)
fmt.Printf("Data: %+v\n", result.Data)
```

## Migration from Prompt-based JSON

If you were previously using prompt engineering for JSON output:

```go
// OLD: Prompt-based approach
oldPrompt := `Generate JSON for a person named John:
{"name": "...", "age": ..., "city": "..."}`

// NEW: Structured output
var person Person
req := rago.LLMStructuredRequest{
    Prompt: "Generate information for a person named John",
    Schema: &person,  // or JSON schema map
}
```

Benefits of structured output:
- ✅ Guaranteed valid JSON format
- ✅ Type safety with Go structs  
- ✅ Better error handling
- ✅ Native provider optimizations
- ✅ No prompt engineering needed