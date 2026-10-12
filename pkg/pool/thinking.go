package pool

import "strings"

// A provider-wide thinking switch.
//
// DeepSeek v4 reasons before every answer unless the request says
// {"thinking": {"type": "disabled"}}. For a call that only has to fill in a
// JSON object, that reasoning is most of the wait. Thinking set on a provider
// sends the switch on every chat completion that does not carry its own, and
// only to that provider: a pool that fails over to another model does not
// send a DeepSeek field to it. An upstream that refuses the field is asked
// once and then no more.

// SetThinking sets the thinking type this client sends ("disabled",
// "enabled"). Empty sends nothing, which is the default.
func (c *Client) SetThinking(thinkingType string) {
	c.thinking = strings.TrimSpace(thinkingType)
}

// withThinking is body with the client's thinking switch added, when there is
// one to add. The caller's map is left as it was.
func (c *Client) withThinking(path string, body interface{}) (interface{}, bool) {
	if c.thinking == "" || c.thinkingRejected.Load() || path != "/chat/completions" {
		return body, false
	}
	m, ok := body.(map[string]interface{})
	if !ok {
		return body, false
	}
	if _, set := m["thinking"]; set {
		return body, false
	}
	out := make(map[string]interface{}, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	out["thinking"] = map[string]interface{}{"type": c.thinking}
	return out, true
}
