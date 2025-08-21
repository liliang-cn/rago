package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRealOllamaIntegration(t *testing.T) {
	// This test requires a running Ollama instance with specific models.

	// 1. Setup Environment
	tempDir, err := os.MkdirTemp("", "rago-real-integration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a temporary config file pointing to the real Ollama instance
	configPath := filepath.Join(tempDir, "config.toml")
	configContent := fmt.Sprintf(`
[ollama]
base_url = "http://localhost:11434"
embedding_model = "nomic-embed-text"
llm_model = "qwen3"

[sqvect]
db_path = "%s"

[ingest.metadata_extraction]
enable = true
llm_model = "qwen3"
`, filepath.Join(tempDir, "test.db"))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create a sample document
	docPath := filepath.Join(tempDir, "testdoc.md")
	docContent := "RAGO is a Retrieval-Augmented Generation system written in the Go programming language."
	require.NoError(t, os.WriteFile(docPath, []byte(docContent), 0644))

	// 2. Run Ingest Command
	originalArgs := os.Args
	os.Args = []string{"rago", "--config", configPath, "ingest", docPath}
	defer func() { os.Args = originalArgs }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = run()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	var ingestOutput bytes.Buffer
	io.Copy(&ingestOutput, r)

	require.Contains(t, ingestOutput.String(), "Successfully ingested documents")

	// 3. Run Query Command with a Filter
	// The filter value is set to "Technical Manual" to match what the LLM extracts.
	os.Args = []string{"rago", "--config", configPath, "query", "What is RAGO written in?", "--filter", "document_type=Technical Manual", "--stream=false"}

	r, w, _ = os.Pipe()
	os.Stdout = w

	err = run()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	var queryOutput bytes.Buffer
	io.Copy(&queryOutput, r)

	// Now that sqvect filtering is fixed, we can assert the result.
	// We expect the answer to contain "Go programming language".
	fmt.Println("Query Answer with Filter:", queryOutput.String())
	require.Contains(t, queryOutput.String(), "Go programming language")
}