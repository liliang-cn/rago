package domain

import (
	"strings"

	"github.com/google/uuid"
)

// NormalizeToolCallID coerces tool call ids into a format accepted by
// OpenAI-compatible providers that require ids beginning with "fc".
func NormalizeToolCallID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "fc_" + uuid.NewString()
	}
	if strings.HasPrefix(strings.ToLower(id), "fc") {
		return id
	}
	return "fc_" + id
}
