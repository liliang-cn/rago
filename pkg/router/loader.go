package router

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// IntentDefinition represents the structure of an intent Markdown file's frontmatter
type IntentDefinition struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Utterances  []string          `yaml:"utterances"`
	ToolMapping string            `yaml:"tool_mapping"`
	Metadata    map[string]string `yaml:"metadata"`
}

// LoadIntentsFromDir scans a directory for .md files and parses them into intent definitions
func LoadIntentsFromDir(dir string) ([]*IntentDefinition, error) {
	var definitions []*IntentDefinition

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // Return empty if dir doesn't exist
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read intents directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, file.Name())
		def, err := parseIntentFile(path)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to parse intent file %s: %v\n", path, err)
			continue
		}

		if def != nil {
			definitions = append(definitions, def)
		}
	}

	return definitions, nil
}

// parseIntentFile parses a single Markdown file with YAML frontmatter
func parseIntentFile(path string) (*IntentDefinition, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Simple frontmatter parser
	strContent := string(content)
	if !strings.HasPrefix(strContent, "---") {
		return nil, nil // Not a valid intent file
	}

	parts := strings.SplitN(strContent, "---", 3)
	if len(parts) < 3 {
		return nil, nil // Incomplete frontmatter
	}

	var def IntentDefinition
	if err := yaml.Unmarshal([]byte(parts[1]), &def); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	if def.Name == "" {
		// Use filename as default name if not specified
		base := filepath.Base(path)
		def.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return &def, nil
}