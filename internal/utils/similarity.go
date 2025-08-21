package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/liliang-cn/ollama-go"
)

// SimilarityChecker provides methods to check text similarity using LLM
type SimilarityChecker struct {
	client *ollama.Client
	model  string
}

// NewSimilarityChecker creates a new SimilarityChecker instance
func NewSimilarityChecker(model string) (*SimilarityChecker, error) {
	client, err := ollama.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama client: %w", err)
	}

	if model == "" {
		model = "qwen3" // default model
	}

	return &SimilarityChecker{
		client: client,
		model:  model,
	}, nil
}

const isAlmostSamePromptTemplate = `You are an expert judge evaluating whether two pieces of text represent the same information or concept. 
Please compare the two texts and determine if they convey the same core meaning, topic, or refer to the same thing.

Consider the following:
- Synonyms and paraphrasing should be considered as the same
- Different wording but same meaning should be considered as the same
- Minor differences in details are acceptable if the core concept is the same
- Different topics or concepts should be considered as different

Respond with ONLY "true" if they refer to the same thing/concept, or "false" if they are different.

Text 1: "%s"
Text 2: "%s"

Are these essentially about the same thing? Respond with only "true" or "false":`

// IsAlmostSame determines if two texts refer to the same thing or concept using LLM
// Returns true if the texts are essentially about the same thing, false otherwise
func (sc *SimilarityChecker) IsAlmostSame(ctx context.Context, text1, text2 string) (bool, error) {
	if text1 == "" && text2 == "" {
		return true, nil
	}
	if text1 == "" || text2 == "" {
		return false, nil
	}

	// If texts are exactly the same, return true immediately
	if strings.TrimSpace(text1) == strings.TrimSpace(text2) {
		return true, nil
	}

	prompt := fmt.Sprintf(isAlmostSamePromptTemplate, text1, text2)

	stream := false
	req := &ollama.GenerateRequest{
		Model:  sc.model,
		Prompt: prompt,
		Stream: &stream,
		Options: &ollama.Options{
			Temperature: func() *float64 { t := 0.1; return &t }(), // Low temperature for consistent results
		},
	}

	resp, err := sc.client.Generate(ctx, req)
	if err != nil {
		return false, fmt.Errorf("failed to generate similarity judgment: %w", err)
	}

	// Parse the response
	result := strings.TrimSpace(strings.ToLower(resp.Response))

	// Handle cases where the model might return "true" or "false" with extra text
	if strings.Contains(result, "true") {
		return true, nil
	}

	if strings.Contains(result, "false") {
		return false, nil
	}

	// Default to false if we can't determine
	return false, nil
}

// IsAlmostSame is a convenience function that creates a temporary SimilarityChecker
// and checks if two texts refer to the same thing using the default model
func IsAlmostSame(ctx context.Context, text1, text2 string) (bool, error) {
	checker, err := NewSimilarityChecker("")
	if err != nil {
		return false, err
	}
	return checker.IsAlmostSame(ctx, text1, text2)
}

// IsAlmostSameWithModel is a convenience function that creates a temporary SimilarityChecker
// and checks if two texts refer to the same thing using the specified model
func IsAlmostSameWithModel(ctx context.Context, text1, text2, model string) (bool, error) {
	checker, err := NewSimilarityChecker(model)
	if err != nil {
		return false, err
	}
	return checker.IsAlmostSame(ctx, text1, text2)
}
