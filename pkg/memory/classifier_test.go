package memory

import (
	"testing"
)

func TestQueryClassifier_Classify(t *testing.T) {
	classifier := NewQueryClassifier(nil)

	tests := []struct {
		query       string
		expectedType QueryType
	}{
		// Greetings
		{"hello", QueryTypeGreeting},
		{"Hi there!", QueryTypeGreeting},
		{"Hey", QueryTypeGreeting},
		{"Good morning", QueryTypeGreeting},
		{"What's up", QueryTypeGreeting},

		// Commands
		{"list all files", QueryTypeCommand},
		{"show me the data", QueryTypeCommand},
		{"run the test", QueryTypeCommand},

		// Casual
		{"thanks", QueryTypeCasual},
		{"goodbye", QueryTypeCasual},
		{"how are you", QueryTypeCasual},

		// Information
		{"what is machine learning?", QueryTypeInformation},
		{"explain how to use this feature", QueryTypeInformation},
		{"tell me about the project", QueryTypeInformation},
		{"why did this happen?", QueryTypeInformation},
		{"can you help me understand the code?", QueryTypeInformation},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := classifier.Classify(tt.query)
			if result != tt.expectedType {
				t.Errorf("Classify(%q) = %v, want %v", tt.query, result, tt.expectedType)
			}
		})
	}
}

func TestQueryClassifier_NeedsMemory(t *testing.T) {
	classifier := NewQueryClassifier(nil)

	tests := []struct {
		query          string
		expectedResult bool
	}{
		// Should NOT need memory
		{"hello", false},
		{"hi", false},
		{"thanks", false},
		{"ok", false},
		{"goodbye", false},
		{"status", false},
		{"help", false},

		// Should need memory
		{"what is the architecture of this system?", true},
		{"explain the memory system", true},
		{"how do I configure the database?", true},
		{"tell me about the features", true},
		{"why was this design chosen?", true},
		{"compare the performance of these approaches", true},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := classifier.NeedsMemory(tt.query)
			if result != tt.expectedResult {
				t.Errorf("NeedsMemory(%q) = %v, want %v", tt.query, result, tt.expectedResult)
			}
		})
	}
}

func TestQueryClassifier_Disabled(t *testing.T) {
	config := &ClassifierConfig{
		Enabled: false,
	}
	classifier := NewQueryClassifier(config)

	// When disabled, should always need memory
	if !classifier.NeedsMemory("hello") {
		t.Error("NeedsMemory() should return true when disabled")
	}
	if !classifier.NeedsMemory("what is this?") {
		t.Error("NeedsMemory() should return true when disabled")
	}
}
