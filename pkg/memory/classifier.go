package memory

import (
	"regexp"
	"strings"
	"unicode"
)

// QueryType represents the classified type of a query
type QueryType int

const (
	QueryTypeGreeting    QueryType = iota // Simple greeting like "hello", "hi"
	QueryTypeCommand                      // Command like "do this", "run that"
	QueryTypeCasual                       // Casual chat like "how are you"
	QueryTypeInformation                  // Information-seeking query (needs memory)
	QueryTypeComplex                      // Complex query (needs memory)
)

// String returns string representation of QueryType
func (t QueryType) String() string {
	switch t {
	case QueryTypeGreeting:
		return "greeting"
	case QueryTypeCommand:
		return "command"
	case QueryTypeCasual:
		return "casual"
	case QueryTypeInformation:
		return "information"
	case QueryTypeComplex:
		return "complex"
	default:
		return "unknown"
	}
}

// ClassifierConfig holds configuration for query classifier
type ClassifierConfig struct {
	Enabled bool // Enable adaptive retrieval

	// Thresholds
	MinQueryLength   int     // Minimum query length to consider for memory retrieval
	KeywordThreshold float64 // Minimum keyword ratio for information query
}

// DefaultClassifierConfig returns default classifier configuration
func DefaultClassifierConfig() *ClassifierConfig {
	return &ClassifierConfig{
		Enabled:          true,
		MinQueryLength:   5,
		KeywordThreshold: 0.3,
	}
}

// QueryClassifier classifies queries to determine if memory retrieval is needed
type QueryClassifier struct {
	config               *ClassifierConfig
	greetingPatterns     []*regexp.Regexp
	commandPatterns      []*regexp.Regexp
	casualPatterns       []*regexp.Regexp
	confirmationPatterns []*regexp.Regexp
}

// NewQueryClassifier creates a new query classifier
func NewQueryClassifier(config *ClassifierConfig) *QueryClassifier {
	if config == nil {
		config = DefaultClassifierConfig()
	}

	c := &QueryClassifier{
		config: config,
	}

	// Initialize greeting patterns
	c.greetingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(hi|hello|hey|greetings|good\s+(morning|afternoon|evening))\s*[!.]?`),
		regexp.MustCompile(`(?i)^(what'?s?\s+up|wassup|howdy)\s*[!.]?`),
		regexp.MustCompile(`(?i)^(你好|您好|早上好|下午好|晚上好)`),
		regexp.MustCompile(`(?i)^(bonjour|hallo|hola|ciao)`),
	}

	// Initialize command patterns
	c.commandPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(please\s+)?(do|run|execute|start|stop|kill|delete|remove|add|create|update|set|get|list|show|find|search)\s+`),
		regexp.MustCompile(`(?i)^(help|version|status|clear|reset|quit|exit)\s*[!.]?$`),
		regexp.MustCompile(`(?i)^(list|show|display|print)\s+(all|the|my|these)`),
		regexp.MustCompile(`(?i)^/(help|version|status|clear|reset)`),
	}

	// Initialize casual patterns
	c.casualPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(how\s+(are|r)\s+(you|u)|how'?s?\s+it\s+going|what'?s?\s+new)`),
		regexp.MustCompile(`(?i)^(thanks|thank\s+you|thx|cheers)\s*[!.]?`),
		regexp.MustCompile(`(?i)^(ok|okay|sure|yes|no|maybe|alright|fine)\s*[!.]?`),
		regexp.MustCompile(`(?i)^(good|great|nice|cool|awesome|excellent)\s*[!.]?`),
		regexp.MustCompile(`(?i)^(bye|goodbye|see\s+you|later|cya)\s*[!.]?`),
	}

	// Initialize confirmation patterns
	c.confirmationPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(got\s+it|understood|roger|copy|acknowledged)\s*[!.]?`),
		regexp.MustCompile(`(?i)^(perfect|wonderful|amazing|fantastic)\s*[!.]?`),
		regexp.MustCompile(`(?i)^(i\s+see|i\s+understand|makes\s+sense)\s*[!.]?`),
	}

	return c
}

// Classify determines the type of a query
func (c *QueryClassifier) Classify(query string) QueryType {
	if !c.config.Enabled {
		return QueryTypeInformation // Default to information if disabled
	}

	query = strings.TrimSpace(query)
	queryLower := strings.ToLower(query)

	// Check minimum length
	if len(query) < c.config.MinQueryLength {
		return QueryTypeGreeting
	}

	// Check greeting patterns
	for _, p := range c.greetingPatterns {
		if p.MatchString(query) {
			return QueryTypeGreeting
		}
	}

	// Check command patterns
	for _, p := range c.commandPatterns {
		if p.MatchString(query) {
			return QueryTypeCommand
		}
	}

	// Check casual patterns
	for _, p := range c.casualPatterns {
		if p.MatchString(query) {
			return QueryTypeCasual
		}
	}

	// Check confirmation patterns
	for _, p := range c.confirmationPatterns {
		if p.MatchString(query) {
			return QueryTypeCasual
		}
	}

	// Check for question words (information seeking)
	questionPatterns := []string{"what", "why", "how", "when", "where", "who", "which", "can you", "could you", "tell me", "explain", "describe"}
	for _, q := range questionPatterns {
		if strings.Contains(queryLower, q) {
			return QueryTypeInformation
		}
	}

	// Check for complex indicators
	complexIndicators := []string{"compare", "analyze", "evaluate", "relationship", "connection", "between", "multiple", "several"}
	for _, ind := range complexIndicators {
		if strings.Contains(queryLower, ind) {
			return QueryTypeComplex
		}
	}

	// Default to information for longer queries or non-ASCII (CJK etc.)
	if len(query) > 20 || isNonASCII(query) {
		return QueryTypeInformation
	}

	return QueryTypeCasual
}

// NeedsMemory determines if a query needs memory retrieval
func (c *QueryClassifier) NeedsMemory(query string) bool {
	if !c.config.Enabled {
		return true // Always retrieve if disabled
	}

	queryType := c.Classify(query)

	switch queryType {
	case QueryTypeGreeting, QueryTypeCasual, QueryTypeCommand:
		return false
	case QueryTypeInformation, QueryTypeComplex:
		return true
	default:
		return true
	}
}

// IsGreeting checks if query is a greeting
func (c *QueryClassifier) IsGreeting(query string) bool {
	return c.Classify(query) == QueryTypeGreeting
}

// IsCommand checks if query is a command
func (c *QueryClassifier) IsCommand(query string) bool {
	return c.Classify(query) == QueryTypeCommand
}

// IsCasual checks if query is casual chat
func (c *QueryClassifier) IsCasual(query string) bool {
	return c.Classify(query) == QueryTypeCasual
}

// IsInformation checks if query is information-seeking
func (c *QueryClassifier) IsInformation(query string) bool {
	qt := c.Classify(query)
	return qt == QueryTypeInformation || qt == QueryTypeComplex
}

// isNonASCII returns true if the string contains any non-ASCII (e.g. CJK) characters.
func isNonASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}
