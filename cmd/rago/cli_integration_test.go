package rago

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liliang-cn/ollama-go"
	"github.com/stretchr/testify/require"
)

// TestMain sets up a global mock server for all integration tests in this package.
var mockServer *httptest.Server

func TestMain(m *testing.M) {
	mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req ollama.GenerateRequest
		_ = json.Unmarshal(body, &req)

		switch r.URL.Path {
		case "/api/show":
			w.Header().Set("Content-Type", "application/json")
			// Pretend the model exists
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name":"test-model"}`))

		case "/api/embeddings":
			w.Header().Set("Content-Type", "application/json")
			resp := ollama.EmbeddingsResponse{
				Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
			}
			json.NewEncoder(w).Encode(resp)

		case "/api/generate":
			w.Header().Set("Content-Type", "application/json")
			var resp ollama.GenerateResponse
			// Metadata extraction request
			if strings.Contains(req.Prompt, "You are an expert data extractor") {
				resp.Response = `{"summary":"A test document about Go.","keywords":["go","testing","cli"],"document_type":"Test Report","creation_date":"2024-08-20"}`
			} else {
				// Regular query request
				resp.Response = "The document is a Test Report about Go programming."
			}
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	code := m.Run()
	mockServer.Close()
	os.Exit(code)
}

func TestMetadataExtractionIntegration(t *testing.T) {
	// Set the environment variable to point to the global mock server for this test.
	t.Setenv("OLLAMA_HOST", mockServer.URL)

	// 1. Setup Environment
	tempDir, err := os.MkdirTemp("", "rago-integration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a temporary config file
	configPath := filepath.Join(tempDir, "config.toml")
	configContent := fmt.Sprintf(`
[ollama]
base_url = "%s"
embedding_model = "test-embed-model"
llm_model = "test-llm-model"

[sqvect]
db_path = "%s"

[ingest.metadata_extraction]
enable = false # Default is false, we will override with flag
llm_model = "test-llm-model"
`, mockServer.URL, filepath.Join(tempDir, "test.db"))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create a sample document
	docPath := filepath.Join(tempDir, "testdoc.md")
	docContent := "This is a test document about the Go programming language."
	require.NoError(t, os.WriteFile(docPath, []byte(docContent), 0644))

	// 2. Run Ingest Command
	ctx := context.Background()
	rootCmd.SetArgs([]string{
		"--config", configPath,
		"ingest", docPath,
		"--extract-metadata", // Enable the feature
	})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = rootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	var ingestOutput bytes.Buffer
	io.Copy(&ingestOutput, r)

	require.Contains(t, ingestOutput.String(), "Successfully ingested documents")

	// 3. Run Query Command with Correct Filter
	rootCmd.SetArgs([]string{
		"--config", configPath,
		"query", "What is the document about?",
		"--filter", "document_type=Test Report", // Filter by extracted metadata
		"--stream=false", // Disable stream for easier output capture
	})

	r, w, _ = os.Pipe()
	os.Stdout = w

	err = rootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	var queryOutput bytes.Buffer
	io.Copy(&queryOutput, r)

	// Assert that the answer is what our mock server provided
	require.Contains(t, queryOutput.String(), "The document is a Test Report about Go programming.")

	// 4. Run Query Command with Incorrect Filter
	rootCmd.SetArgs([]string{
		"--config", configPath,
		"query", "What is the document about?",
		"--filter", "document_type=NonExistentType", // This should not match
		"--stream=false",
	})

	r, w, _ = os.Pipe()
	os.Stdout = w

	err = rootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	var failedQueryOutput bytes.Buffer
	io.Copy(&failedQueryOutput, r)

	// Assert that no results were found
	require.Contains(t, failedQueryOutput.String(), "我在知识库中找不到相关信息来回答您的问题")
}
