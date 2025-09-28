package usage

import (
	"strings"
	"unicode/utf8"
)

// TokenCounter provides methods for counting tokens
type TokenCounter struct {
	// Model-specific token counting configurations
	modelConfig map[string]float64 // tokens per character ratio
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		modelConfig: map[string]float64{
			// Approximate tokens per character for different models
			"gpt-3.5-turbo":    0.25,
			"gpt-4":            0.25,
			"gpt-4-turbo":      0.25,
			"claude-3-opus":    0.25,
			"claude-3-sonnet":  0.25,
			"claude-3-haiku":   0.25,
			"llama2":           0.3,
			"llama3":           0.28,
			"mixtral":          0.27,
			"qwen":             0.3,
			"qwen2":            0.28,
			"qwen3":            0.28,
			"default":          0.25, // Default ratio
		},
	}
}

// EstimateTokens estimates the number of tokens in a text
// This is a simple implementation that can be replaced with more accurate methods
func (tc *TokenCounter) EstimateTokens(text string, model string) int {
	if text == "" {
		return 0
	}

	// For more accurate counting, you could integrate tiktoken or similar libraries
	// For now, we use a simple heuristic based on character and word count
	
	// Get model-specific ratio
	ratio, exists := tc.modelConfig[strings.ToLower(model)]
	if !exists {
		// Try to match partial model names
		modelLower := strings.ToLower(model)
		for key, val := range tc.modelConfig {
			if strings.Contains(modelLower, key) {
				ratio = val
				break
			}
		}
		if ratio == 0 {
			ratio = tc.modelConfig["default"]
		}
	}

	// Count characters
	charCount := utf8.RuneCountInString(text)
	
	// Estimate tokens based on character count
	estimatedTokens := int(float64(charCount) * ratio)
	
	// Ensure at least 1 token for non-empty text
	if estimatedTokens == 0 && text != "" {
		estimatedTokens = 1
	}
	
	return estimatedTokens
}

// EstimateMessagesTokens estimates tokens for a list of messages
func (tc *TokenCounter) EstimateMessagesTokens(messages []Message, model string) int {
	totalTokens := 0
	for _, msg := range messages {
		// Add tokens for role (usually 1-2 tokens)
		totalTokens += 2
		// Add tokens for content
		totalTokens += tc.EstimateTokens(msg.Content, model)
		// Add separator tokens (usually 3-4 tokens)
		totalTokens += 3
	}
	return totalTokens
}

// EstimateConversationTokens estimates tokens for a conversation
func (tc *TokenCounter) EstimateConversationTokens(messages []Message, model string) int {
	return tc.EstimateMessagesTokens(messages, model)
}

// ExtractTokensFromResponse extracts token counts from provider responses
// Different providers return token counts in different formats
func ExtractTokensFromResponse(provider string, response interface{}) (input int, output int, total int) {
	// This function should be implemented based on the actual response structure
	// from different providers. For now, returning placeholder values
	
	switch provider {
	case "openai":
		// Extract from response.usage.prompt_tokens, completion_tokens, total_tokens
		if resp, ok := response.(map[string]interface{}); ok {
			if usage, ok := resp["usage"].(map[string]interface{}); ok {
				if val, ok := usage["prompt_tokens"].(float64); ok {
					input = int(val)
				}
				if val, ok := usage["completion_tokens"].(float64); ok {
					output = int(val)
				}
				if val, ok := usage["total_tokens"].(float64); ok {
					total = int(val)
				}
			}
		}
	case "anthropic":
		// Extract from response.usage.input_tokens, output_tokens
		if resp, ok := response.(map[string]interface{}); ok {
			if usage, ok := resp["usage"].(map[string]interface{}); ok {
				if val, ok := usage["input_tokens"].(float64); ok {
					input = int(val)
				}
				if val, ok := usage["output_tokens"].(float64); ok {
					output = int(val)
				}
				total = input + output
			}
		}
	case "ollama":
		// Ollama may not return token counts, estimate them
		// This would need to be implemented based on actual Ollama response structure
	default:
		// For unknown providers, return zeros
	}
	
	return input, output, total
}

// CalculateCost calculates the cost based on token usage and model pricing
func CalculateCost(model string, inputTokens, outputTokens int) float64 {
	// Model pricing in USD per 1K tokens
	// These are example prices and should be updated with actual pricing
	inputPricing := map[string]float64{
		"gpt-3.5-turbo":     0.0005,
		"gpt-4":             0.03,
		"gpt-4-turbo":       0.01,
		"gpt-4o":            0.005,
		"claude-3-opus":     0.015,
		"claude-3-sonnet":   0.003,
		"claude-3-haiku":    0.00025,
		"claude-3.5-sonnet": 0.003,
	}
	
	outputPricing := map[string]float64{
		"gpt-3.5-turbo":     0.0015,
		"gpt-4":             0.06,
		"gpt-4-turbo":       0.03,
		"gpt-4o":            0.015,
		"claude-3-opus":     0.075,
		"claude-3-sonnet":   0.015,
		"claude-3-haiku":    0.00125,
		"claude-3.5-sonnet": 0.015,
	}
	
	modelLower := strings.ToLower(model)
	var inputPrice, outputPrice float64
	
	// Find matching pricing
	for key, price := range inputPricing {
		if strings.Contains(modelLower, key) {
			inputPrice = price
			break
		}
	}
	
	for key, price := range outputPricing {
		if strings.Contains(modelLower, key) {
			outputPrice = price
			break
		}
	}
	
	// Calculate cost
	inputCost := float64(inputTokens) / 1000.0 * inputPrice
	outputCost := float64(outputTokens) / 1000.0 * outputPrice
	
	return inputCost + outputCost
}