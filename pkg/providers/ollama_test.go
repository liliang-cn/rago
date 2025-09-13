package providers

import (
	"context"
	"strings"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockOllamaMetadataExtractor simulates LLM metadata extraction for testing
type MockOllamaMetadataExtractor struct {
	mockResponses map[string]*domain.ExtractedMetadata
}

func NewMockOllamaMetadataExtractor() *MockOllamaMetadataExtractor {
	return &MockOllamaMetadataExtractor{
		mockResponses: map[string]*domain.ExtractedMetadata{
			"medical": {
				Summary:      "Medical record with patient information",
				Keywords:     []string{"patient", "diagnosis", "treatment"},
				DocumentType: "Medical Record",
				Collection:   "medical_records",
				TemporalRefs: map[string]string{
					"today":     "2025-09-12",
					"yesterday": "2025-09-11",
				},
				Entities: map[string][]string{
					"person":  {"John Doe", "Dr. Smith"},
					"medical": {"chest pain", "ECG"},
				},
			},
			"meeting": {
				Summary:      "Team meeting notes",
				Keywords:     []string{"meeting", "project", "discussion"},
				DocumentType: "Meeting Notes",
				Collection:   "meeting_notes",
				TemporalRefs: map[string]string{
					"tomorrow": "2025-09-13",
				},
				Entities: map[string][]string{
					"person": {"Alice", "Bob"},
				},
			},
			"code": {
				Summary:      "Code snippet implementation",
				Keywords:     []string{"function", "algorithm", "code"},
				DocumentType: "Code",
				Collection:   "code_snippets",
			},
			"research": {
				Summary:      "Research paper on machine learning",
				Keywords:     []string{"machine learning", "AI", "research"},
				DocumentType: "Article",
				Collection:   "research_papers",
				Entities: map[string][]string{
					"organization": {"MIT", "Stanford"},
				},
			},
			"financial": {
				Summary:      "Financial invoice document",
				Keywords:     []string{"invoice", "payment", "financial"},
				DocumentType: "Invoice",
				Collection:   "financial_reports",
				CreationDate: "2025-09-12",
				Entities: map[string][]string{
					"organization": {"Acme Corp"},
				},
			},
		},
	}
}

func (m *MockOllamaMetadataExtractor) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	// Determine document type based on content keywords
	if containsKeywords(content, []string{"patient", "medical", "diagnosis", "ECG", "doctor"}) {
		return m.mockResponses["medical"], nil
	}
	if containsKeywords(content, []string{"meeting", "discuss", "team", "agenda"}) {
		return m.mockResponses["meeting"], nil
	}
	if containsKeywords(content, []string{"function", "def", "class", "code", "algorithm"}) {
		return m.mockResponses["code"], nil
	}
	if containsKeywords(content, []string{"research", "paper", "abstract", "study"}) {
		return m.mockResponses["research"], nil
	}
	if containsKeywords(content, []string{"invoice", "payment", "bill", "financial"}) {
		return m.mockResponses["financial"], nil
	}
	
	// Default response
	return &domain.ExtractedMetadata{
		Summary:      "General document",
		Keywords:     []string{"document"},
		DocumentType: "Document",
		Collection:   "default",
	}, nil
}

func containsKeywords(text string, keywords []string) bool {
	lowerText := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(lowerText, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func TestMetadataExtractionWithCollection(t *testing.T) {
	tests := []struct {
		name               string
		content            string
		expectedCollection string
		expectedType       string
		checkEntities      bool
		checkTemporal      bool
	}{
		{
			name:               "Medical record should go to medical_records",
			content:            "Patient John Doe visited with chest pain. ECG shows normal rhythm.",
			expectedCollection: "medical_records",
			expectedType:       "Medical Record",
			checkEntities:      true,
			checkTemporal:      true,
		},
		{
			name:               "Meeting notes should go to meeting_notes",
			content:            "Team meeting to discuss Q4 project milestones with Alice and Bob.",
			expectedCollection: "meeting_notes",
			expectedType:       "Meeting Notes",
			checkEntities:      true,
		},
		{
			name:               "Code snippet should go to code_snippets",
			content:            "def fibonacci(n): return n if n <= 1 else fibonacci(n-1) + fibonacci(n-2)",
			expectedCollection: "code_snippets",
			expectedType:       "Code",
		},
		{
			name:               "Research paper should go to research_papers",
			content:            "Abstract: This research paper explores machine learning applications.",
			expectedCollection: "research_papers",
			expectedType:       "Article",
			checkEntities:      true,
		},
		{
			name:               "Invoice should go to financial_reports",
			content:            "Invoice #12345 for Acme Corp. Total payment due: $5000",
			expectedCollection: "financial_reports",
			expectedType:       "Invoice",
			checkEntities:      true,
		},
		{
			name:               "Unknown content should go to default",
			content:            "Random text without clear category",
			expectedCollection: "default",
			expectedType:       "Document",
		},
	}

	extractor := NewMockOllamaMetadataExtractor()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := extractor.ExtractMetadata(ctx, tt.content, "")
			require.NoError(t, err)
			require.NotNil(t, metadata)

			// Check collection assignment
			assert.Equal(t, tt.expectedCollection, metadata.Collection,
				"Document should be assigned to correct collection")

			// Check document type
			assert.Equal(t, tt.expectedType, metadata.DocumentType,
				"Document type should be correctly identified")

			// Check that basic metadata is present
			assert.NotEmpty(t, metadata.Summary, "Should have a summary")
			assert.NotEmpty(t, metadata.Keywords, "Should have keywords")

			// Optional checks based on document type
			if tt.checkEntities {
				assert.NotEmpty(t, metadata.Entities, "Should have extracted entities")
			}

			if tt.checkTemporal {
				assert.NotEmpty(t, metadata.TemporalRefs, "Should have temporal references")
			}
		})
	}
}

func TestCollectionNaming(t *testing.T) {
	// Test that collection names follow consistent snake_case pattern
	validCollectionNames := []string{
		"medical_records",
		"meeting_notes",
		"code_snippets",
		"research_papers",
		"financial_reports",
		"technical_docs",
		"personal_notes",
		"project_docs",
		"legal_documents",
		"customer_feedback",
	}

	for _, name := range validCollectionNames {
		t.Run(name, func(t *testing.T) {
			// Check snake_case format
			assert.Regexp(t, `^[a-z]+(_[a-z]+)*$`, name,
				"Collection name should be in snake_case")
			
			// Check reasonable length
			assert.LessOrEqual(t, len(name), 30,
				"Collection name should not be too long")
			assert.GreaterOrEqual(t, len(name), 3,
				"Collection name should not be too short")
		})
	}
}

func TestMetadataStructureCompleteness(t *testing.T) {
	extractor := NewMockOllamaMetadataExtractor()
	ctx := context.Background()

	// Test medical document for complete metadata structure
	metadata, err := extractor.ExtractMetadata(ctx, 
		"Patient record: John Doe diagnosed with condition", "")
	require.NoError(t, err)
	require.NotNil(t, metadata)

	// Verify all fields are properly initialized
	assert.NotNil(t, metadata.Summary)
	assert.NotNil(t, metadata.Keywords)
	assert.NotNil(t, metadata.DocumentType)
	assert.NotNil(t, metadata.Collection)
	
	// Collection should never be empty
	assert.NotEmpty(t, metadata.Collection, "Collection field must not be empty")
	
	// If temporal refs exist, they should be valid dates
	if metadata.TemporalRefs != nil {
		for key, date := range metadata.TemporalRefs {
			assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, date,
				"Temporal ref %s should be in YYYY-MM-DD format", key)
		}
	}
}

func TestCollectionConsistency(t *testing.T) {
	extractor := NewMockOllamaMetadataExtractor()
	ctx := context.Background()

	// Test that similar documents get the same collection
	medicalDocs := []string{
		"Patient visited emergency room",
		"Medical diagnosis and treatment plan",
		"Doctor's notes on patient condition",
	}

	collections := make([]string, 0)
	for _, doc := range medicalDocs {
		metadata, err := extractor.ExtractMetadata(ctx, doc, "")
		require.NoError(t, err)
		collections = append(collections, metadata.Collection)
	}

	// All medical documents should be in the same collection
	for i := 1; i < len(collections); i++ {
		assert.Equal(t, collections[0], collections[i],
			"Similar documents should be in the same collection")
	}
}