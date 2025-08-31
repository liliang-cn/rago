package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateTimeTool_Name(t *testing.T) {
	tool := NewDateTimeTool()
	assert.Equal(t, "datetime", tool.Name())
}

func TestDateTimeTool_Description(t *testing.T) {
	tool := NewDateTimeTool()
	assert.NotEmpty(t, tool.Description())
}

func TestDateTimeTool_Parameters(t *testing.T) {
	tool := NewDateTimeTool()
	params := tool.Parameters()

	assert.Equal(t, "object", params.Type)
	assert.Contains(t, params.Required, "action")
	assert.Contains(t, params.Properties, "action")
	assert.Contains(t, params.Properties["action"].Enum, "now")
}

func TestDateTimeTool_Validate(t *testing.T) {
	tool := NewDateTimeTool()

	// Valid case
	err := tool.Validate(map[string]interface{}{
		"action": "now",
	})
	assert.NoError(t, err)

	// Missing action
	err = tool.Validate(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action parameter is required")

	// Invalid action
	err = tool.Validate(map[string]interface{}{
		"action": "invalid",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action")

	// Diff action missing date2
	err = tool.Validate(map[string]interface{}{
		"action": "diff",
		"date":   "2023-01-01",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "date2 parameter is required")
}

func TestDateTimeTool_Execute_Now(t *testing.T) {
	tool := NewDateTimeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "now",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Contains(t, data, "datetime")
	assert.Contains(t, data, "timestamp")
	assert.Contains(t, data, "timezone")
	assert.Contains(t, data, "iso8601")
}

func TestDateTimeTool_Execute_Format(t *testing.T) {
	tool := NewDateTimeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "format",
		"date":   "2023-01-01T10:30:00Z",
		"format": "2006-01-02",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "2023-01-01", result.Data)
}

func TestDateTimeTool_Execute_Parse(t *testing.T) {
	tool := NewDateTimeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "parse",
		"date":   "2023-01-01T10:30:00Z",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, 2023, data["year"])
	assert.Equal(t, "January", data["month"])
	assert.Equal(t, 1, data["day"])
	assert.Contains(t, data, "timestamp")
}

func TestDateTimeTool_Execute_Add(t *testing.T) {
	tool := NewDateTimeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "add",
		"date":     "2023-01-01T00:00:00Z",
		"duration": "24h",
		"format":   "2006-01-02",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "2023-01-01", data["original"])
	assert.Equal(t, "2023-01-02", data["new"])
}

func TestDateTimeTool_Execute_Add_CustomDuration(t *testing.T) {
	tool := NewDateTimeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "add",
		"date":     "2023-01-01T00:00:00Z",
		"duration": "7d",
		"format":   "2006-01-02",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "2023-01-01", data["original"])
	assert.Equal(t, "2023-01-08", data["new"])
}

func TestDateTimeTool_Execute_Diff(t *testing.T) {
	tool := NewDateTimeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "diff",
		"date":   "2023-01-01T00:00:00Z",
		"date2":  "2023-01-02T00:00:00Z",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, float64(86400), data["seconds"]) // 24 hours in seconds
	assert.Equal(t, float64(1440), data["minutes"])  // 24 hours in minutes
	assert.Equal(t, float64(24), data["hours"])      // 24 hours
	assert.Equal(t, float64(1), data["days"])        // 1 day
}

func TestDateTimeTool_Execute_Timezone(t *testing.T) {
	tool := NewDateTimeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "timezone",
		"date":     "2023-01-01T00:00:00Z",
		"timezone": "America/New_York",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Contains(t, data, "original")
	assert.Contains(t, data, "converted")
}

func TestDateTimeTool_Execute_InvalidAction(t *testing.T) {
	tool := NewDateTimeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "invalid",
	})

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "unknown action")
}

func TestDateTimeTool_Execute_MissingRequiredParam(t *testing.T) {
	tool := NewDateTimeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "format",
		// missing date parameter
	})

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "date parameter is required")
}

func TestDateTimeTool_ParseDuration(t *testing.T) {
	tool := NewDateTimeTool()

	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1h", time.Hour},
		{"30m", 30 * time.Minute},
		{"1d", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
		{"1y", 365 * 24 * time.Hour},
	}

	for _, test := range tests {
		duration, err := tool.parseDuration(test.input)
		assert.NoError(t, err, "Failed to parse %s", test.input)
		assert.Equal(t, test.expected, duration, "Duration mismatch for %s", test.input)
	}
}

func TestDateTimeTool_ParseTime(t *testing.T) {
	tool := NewDateTimeTool()

	tests := []string{
		"2023-01-01T10:30:00Z",
		"2023-01-01 10:30:00",
		"2023-01-01",
		"Jan 1, 2023",
		"01/01/2023",
	}

	for _, dateStr := range tests {
		_, err := tool.parseTime(dateStr)
		assert.NoError(t, err, "Failed to parse %s", dateStr)
	}
}

func TestNewDateTimeToolWithLocation(t *testing.T) {
	tool, err := NewDateTimeToolWithLocation("America/New_York")
	require.NoError(t, err)
	assert.NotNil(t, tool)

	expectedLoc, _ := time.LoadLocation("America/New_York")
	assert.Equal(t, expectedLoc, tool.location)

	// Test invalid location
	_, err = NewDateTimeToolWithLocation("Invalid/Location")
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkDateTimeTool_Execute_Now(b *testing.B) {
	tool := NewDateTimeTool()
	ctx := context.Background()
	args := map[string]interface{}{
		"action": "now",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args)
	}
}

func BenchmarkDateTimeTool_Execute_Format(b *testing.B) {
	tool := NewDateTimeTool()
	ctx := context.Background()
	args := map[string]interface{}{
		"action": "format",
		"date":   "2023-01-01T10:30:00Z",
		"format": "2006-01-02",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args)
	}
}
