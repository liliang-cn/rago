package prompt

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestPromptManager(t *testing.T) {
	m := NewManager()

	// 1. Test default registration and getting
	m.RegisterDefault("test.hello", "Hello {{.Name}}")
	assert.Equal(t, "Hello {{.Name}}", m.Get("test.hello"))

	// 2. Test rendering
	rendered, err := m.Render("test.hello", map[string]string{"Name": "RAGO"})
	assert.NoError(t, err)
	assert.Equal(t, "Hello RAGO", rendered)

	// 3. Test override
	m.SetPrompt("test.hello", "Hi {{.Name}}, how are you?")
	assert.Equal(t, "Hi {{.Name}}, how are you?", m.Get("test.hello"))

	rendered, err = m.Render("test.hello", map[string]string{"Name": "Li"})
	assert.NoError(t, err)
	assert.Equal(t, "Hi Li, how are you?", rendered)

	// 4. Test missing key
	_, err = m.Render("missing.key", nil)
	assert.Error(t, err)
}

func TestDefaultPromptsExist(t *testing.T) {
	m := NewManager()
	
	// Ensure core prompts are loaded by default
	assert.NotEmpty(t, m.Get(PlannerIntentRecognition))
	assert.NotEmpty(t, m.Get(PlannerSystemPrompt))
	assert.NotEmpty(t, m.Get(AgentVerification))
	assert.NotEmpty(t, m.Get(AgentSystemPrompt))
}
