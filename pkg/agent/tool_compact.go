package agent

import (
	"strings"

	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

// Tool schemas as signatures.
//
// A schema is sent with every model call, and what makes it large is prose:
// a tool's description written as a paragraph of guidance, a parameter
// described in two sentences. To call a tool the model needs its name, what it
// is for in a line, and its parameters — names, types, which are required, the
// allowed values. Measured on SuperAI's queen, 91 schemas were about 14k tokens
// on every call, most of it description.
//
// With Tooling.CompactSchemas on, a tool keeps the first sentence of its
// description and each parameter keeps a short description; the structure of
// the parameters is left exactly as it was. A tool found through search is
// still returned whole.

const (
	compactToolDescription  = 120
	compactParamDescription = 60
)

// compactToolSchemas returns copies of tools reduced to their signatures.
func compactToolSchemas(tools []domain.ToolDefinition) []domain.ToolDefinition {
	out := make([]domain.ToolDefinition, len(tools))
	for i, t := range tools {
		c := t
		c.Function.Description = shortSentence(t.Function.Description, compactToolDescription)
		if t.Function.Parameters != nil {
			c.Function.Parameters = compactSchema(t.Function.Parameters)
		}
		out[i] = c
	}
	return out
}

// compactSchema copies a JSON schema, shortening every description in it and
// dropping examples, which document a schema rather than define it.
func compactSchema(s map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(s))
	for k, v := range s {
		switch k {
		case "description":
			if d, ok := v.(string); ok {
				if d = shortSentence(d, compactParamDescription); d != "" {
					out[k] = d
				}
				continue
			}
		case "examples", "example":
			continue
		}
		out[k] = compactValue(v)
	}
	return out
}

func compactValue(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		return compactSchema(x)
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, e := range x {
			out[i] = compactValue(e)
		}
		return out
	default:
		return v
	}
}

// shortSentence is the first sentence or line of s, at most max runes.
func shortSentence(s string, max int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	for _, end := range []string{". ", "。", "；", "; "} {
		if i := strings.Index(s, end); i > 0 {
			s = strings.TrimSpace(s[:i+len(end)])
		}
	}
	if r := []rune(s); len(r) > max {
		s = string(r[:max-1]) + "…"
	}
	return s
}
