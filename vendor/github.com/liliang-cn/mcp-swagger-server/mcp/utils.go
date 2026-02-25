package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-openapi/spec"
	"gopkg.in/yaml.v3"
)

// ParseSwaggerSpec parses a Swagger/OpenAPI spec from JSON or YAML
func ParseSwaggerSpec(data []byte) (*spec.Swagger, error) {
	var swagger spec.Swagger

	// Try JSON first
	err := json.Unmarshal(data, &swagger)
	if err == nil {
		return &swagger, nil
	}

	// Try YAML - first convert to map then to JSON
	var yamlData map[string]interface{}
	err = yaml.Unmarshal(data, &yamlData)
	if err == nil {
		// Convert YAML to JSON
		jsonData, err := json.Marshal(yamlData)
		if err != nil {
			return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
		}
		
		// Parse with go-openapi/spec
		swagger = spec.Swagger{}
		if err := json.Unmarshal(jsonData, &swagger); err != nil {
			return nil, fmt.Errorf("failed to parse converted spec: %w", err)
		}
		
		return &swagger, nil
	}

	return nil, fmt.Errorf("failed to parse spec as JSON or YAML")
}

// FetchSwaggerFromURL downloads a Swagger/OpenAPI spec from a URL
func FetchSwaggerFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spec from URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch spec, status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// readFile reads a file from disk
func readFile(filepath string) ([]byte, error) {
	return os.ReadFile(filepath)
}