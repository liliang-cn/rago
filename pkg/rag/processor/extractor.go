package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

type EntityExtractor struct {
	generator domain.Generator
}

func NewEntityExtractor(generator domain.Generator) *EntityExtractor {
	return &EntityExtractor{generator: generator}
}

type ExtractedGraphData struct {
	Entities []struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
	} `json:"entities"`
	Relationships []struct {
		Source      string `json:"source"`
		Target      string `json:"target"`
		Type        string `json:"type"`
		Description string `json:"description"`
	} `json:"relationships"`
}

func (e *EntityExtractor) Extract(ctx context.Context, text string) (*ExtractedGraphData, error) {
	prompt := fmt.Sprintf(`You are a knowledge graph expert. Extract entities and relationships from the following text.

Rules:
1. Identify key entities (Person, Organization, Location, Concept, Event, etc.).
2. Identify relationships between these entities.
3. Return valid JSON only. No markdown, no comments.

JSON Schema:
{
  "entities": [
    {"name": "Entity Name", "type": "Type", "description": "Short description"}
  ],
  "relationships": [
    {"source": "Entity Name", "target": "Entity Name", "type": "relationship_type", "description": "Context"}
  ]
}

Text:
%s

JSON Output:`, text)

	// Use low temperature for deterministic output
	opts := &domain.GenerationOptions{
		Temperature: 0.1,
		MaxTokens:   4000,
	}

	response, err := e.generator.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("extraction generation failed: %w", err)
	}

	// Clean up response (remove markdown code blocks if any)
	cleaned := cleanJSON(response)

	var data ExtractedGraphData
	if err := json.Unmarshal([]byte(cleaned), &data); err != nil {
		return nil, fmt.Errorf("failed to parse extraction JSON: %w", err)
	}

	return &data, nil
}

func cleanJSON(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}
