package agent

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

func TestACompactSchemaIsTheSignature(t *testing.T) {
	long := "Keep a reply as a dashboard the user can reopen from the Dashboards panel. For \"save that\" pass only name and prompt.\n\nImproving a board is a save onto that board."
	tool := domain.ToolDefinition{Type: "function", Function: domain.ToolFunction{
		Name:        "dashboard_save",
		Description: long,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Short label, e.g. 我的美股收益"},
				"mode": map[string]interface{}{"type": "string", "enum": []interface{}{"a", "b"},
					"description": "How to save it. Leave this out to save the reply already on screen, which is almost always what is wanted, and only pass it otherwise."},
				"rows": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object",
					"properties": map[string]interface{}{"k": map[string]interface{}{"type": "integer", "description": "A key. It is a number."}}}},
			},
			"required": []interface{}{"name"},
			"examples": []interface{}{map[string]interface{}{"name": "x"}},
		},
	}}
	got := compactToolSchemas([]domain.ToolDefinition{tool})[0]

	if got.Function.Name != "dashboard_save" {
		t.Fatalf("name changed: %q", got.Function.Name)
	}
	if got.Function.Description != "Keep a reply as a dashboard the user can reopen from the Dashboards panel." {
		t.Fatalf("description = %q, want its first sentence", got.Function.Description)
	}
	p := got.Function.Parameters
	if _, ok := p["examples"]; ok {
		t.Fatal("examples kept")
	}
	if !reflect.DeepEqual(p["required"], []interface{}{"name"}) {
		t.Fatalf("required lost: %v", p["required"])
	}
	props := p["properties"].(map[string]interface{})
	mode := props["mode"].(map[string]interface{})
	if !reflect.DeepEqual(mode["enum"], []interface{}{"a", "b"}) || mode["type"] != "string" {
		t.Fatalf("a parameter's allowed values or type changed: %v", mode)
	}
	if d := mode["description"].(string); len([]rune(d)) > compactParamDescription || !strings.HasPrefix(d, "How to save it.") {
		t.Fatalf("parameter description = %q", d)
	}
	k := props["rows"].(map[string]interface{})["items"].(map[string]interface{})["properties"].(map[string]interface{})["k"].(map[string]interface{})
	if k["type"] != "integer" || k["description"] != "A key." {
		t.Fatalf("a nested parameter was not kept or not shortened: %v", k)
	}

	// The original is untouched: the registry still holds the whole schema,
	// which is what a search hands back.
	if tool.Function.Description != long {
		t.Fatal("compacting changed the registered tool")
	}
	before, _ := json.Marshal(tool)
	after, _ := json.Marshal(got)
	if len(after) >= len(before)*2/3 {
		t.Fatalf("not much smaller: %d → %d bytes", len(before), len(after))
	}
}
