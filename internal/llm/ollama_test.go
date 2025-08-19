package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liliang-cn/ollama-go"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractMetadata(t *testing.T) {
	validMetadata := domain.ExtractedMetadata{
		Summary:      "This is a test summary.",
		Keywords:     []string{"test", "metadata", "extraction"},
		DocumentType: "Test Document",
		CreationDate: "2024-01-01",
	}

	validResponse, err := json.Marshal(ollama.GenerateResponse{
		Response: `{"summary":"This is a test summary.","keywords":["test","metadata","extraction"],"document_type":"Test Document","creation_date":"2024-01-01"}`,
	})
	require.NoError(t, err)

	invalidJsonResponse, err := json.Marshal(ollama.GenerateResponse{
		Response: `{"summary":"This is a test summary", "keywords":}`,
	})
	require.NoError(t, err)

	testCases := []struct {
		name             string
		handler          http.HandlerFunc
		inputContent     string
		model            string
		expectedMetadata *domain.ExtractedMetadata
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name: "Success Case",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/generate", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(validResponse)
			},
			inputContent:     "This is a test document.",
			model:            "test-model",
			expectedMetadata: &validMetadata,
			expectError:      false,
		},
		{
			name: "Error Case - Invalid JSON Response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(invalidJsonResponse)
			},
			inputContent:     "This content leads to invalid json.",
			model:            "test-model",
			expectedMetadata: nil,
			expectError:      true,
			expectedErrorMsg: "failed to unmarshal metadata response",
		},
		{
			name: "Error Case - Server Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			inputContent:     "This content causes a server error.",
			model:            "test-model",
			expectedMetadata: nil,
			expectError:      true,
			expectedErrorMsg: "metadata extraction failed",
		},
		{
			name:             "Error Case - Empty Input Content",
			handler:          func(w http.ResponseWriter, r *http.Request) {},
			inputContent:     "",
			model:            "test-model",
			expectedMetadata: nil,
			expectError:      true,
			expectedErrorMsg: "content cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			// Set the environment variable to point to the test server.
			t.Setenv("OLLAMA_HOST", server.URL)

			// We create a new service that will use the test server's URL via environment variable.
			service, err := NewOllamaService(server.URL, tc.model)
			require.NoError(t, err)

			metadata, err := service.ExtractMetadata(context.Background(), tc.inputContent, tc.model)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tc.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedMetadata, metadata)
			}
		})
	}
}
