package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCronParser(t *testing.T) {
	parser := NewCronParser()
	require.NotNil(t, parser)
	assert.NotNil(t, parser.parser)
}

func TestCronParserValidate(t *testing.T) {
	parser := NewCronParser()

	tests := []struct {
		name     string
		schedule string
		wantErr  bool
	}{
		{"Valid every minute", "* * * * *", false},
		{"Valid hourly", "0 * * * *", false},
		{"Valid daily at midnight", "0 0 * * *", false},
		{"Valid weekly on Sunday", "0 0 * * 0", false},
		{"Valid monthly", "0 0 1 * *", false},
		{"Valid with ranges", "0-30 8-17 * * 1-5", false},
		{"Valid with steps", "*/5 * * * *", false},
		{"Valid with lists", "0,15,30,45 * * * *", false},
		{"Valid empty (one-time)", "", false}, // Empty is valid for one-time execution
		{"Invalid too few fields", "* * * *", true},
		{"Valid with seconds (6 fields)", "0 * * * * *", false}, // SecondOptional parser accepts this
		{"Invalid minute", "60 * * * *", true},
		{"Invalid hour", "0 25 * * *", true},
		{"Invalid day", "0 0 32 * *", true},
		{"Invalid month", "0 0 1 13 *", true},
		{"Invalid weekday", "0 0 * * 8", true},
		{"Invalid syntax", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.Validate(tt.schedule)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCronParserParseAndNext(t *testing.T) {
	parser := NewCronParser()

	// Fixed base time for consistent testing
	baseTime := time.Date(2025, 9, 12, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		schedule     string
		expectedNext time.Time
	}{
		{
			name:         "Every minute",
			schedule:     "* * * * *",
			expectedNext: time.Date(2025, 9, 12, 10, 31, 0, 0, time.UTC),
		},
		{
			name:         "Every hour at minute 0",
			schedule:     "0 * * * *",
			expectedNext: time.Date(2025, 9, 12, 11, 0, 0, 0, time.UTC),
		},
		{
			name:         "Daily at noon",
			schedule:     "0 12 * * *",
			expectedNext: time.Date(2025, 9, 12, 12, 0, 0, 0, time.UTC),
		},
		{
			name:         "Every 5 minutes",
			schedule:     "*/5 * * * *",
			expectedNext: time.Date(2025, 9, 12, 10, 35, 0, 0, time.UTC),
		},
		{
			name:         "At 10:45",
			schedule:     "45 10 * * *",
			expectedNext: time.Date(2025, 9, 12, 10, 45, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextRun, err := parser.ParseAndNext(tt.schedule, baseTime)
			require.NoError(t, err)
			require.NotNil(t, nextRun)
			assert.Equal(t, tt.expectedNext, *nextRun)
		})
	}
}

func TestCronParserInvalidSchedule(t *testing.T) {
	parser := NewCronParser()
	baseTime := time.Now()

	// Test invalid schedule
	nextRun, err := parser.ParseAndNext("invalid", baseTime)
	assert.Error(t, err)
	assert.Nil(t, nextRun)

	// Test empty schedule (valid for one-time)
	nextRun, err = parser.ParseAndNext("", baseTime)
	assert.NoError(t, err)
	assert.Nil(t, nextRun) // Returns nil for one-time execution
}

func TestCronParserMultipleNextRuns(t *testing.T) {
	parser := NewCronParser()

	// Test that successive calls produce correct next times
	schedule := "*/15 * * * *" // Every 15 minutes
	baseTime := time.Date(2025, 9, 12, 10, 0, 0, 0, time.UTC)

	expectedTimes := []time.Time{
		time.Date(2025, 9, 12, 10, 15, 0, 0, time.UTC),
		time.Date(2025, 9, 12, 10, 30, 0, 0, time.UTC),
		time.Date(2025, 9, 12, 10, 45, 0, 0, time.UTC),
		time.Date(2025, 9, 12, 11, 0, 0, 0, time.UTC),
	}

	currentTime := baseTime
	for i, expected := range expectedTimes {
		nextRun, err := parser.ParseAndNext(schedule, currentTime)
		require.NoError(t, err, "Failed at iteration %d", i)
		require.NotNil(t, nextRun)
		assert.Equal(t, expected, *nextRun, "Mismatch at iteration %d", i)
		currentTime = *nextRun
	}
}

func TestCronParserWeekdaySchedules(t *testing.T) {
	parser := NewCronParser()

	tests := []struct {
		name     string
		schedule string
		baseTime time.Time
		expected time.Time
	}{
		{
			name:     "Next Monday",
			schedule: "0 9 * * 1", // Monday at 9am
			baseTime: time.Date(2025, 9, 12, 10, 0, 0, 0, time.UTC), // Friday
			expected: time.Date(2025, 9, 15, 9, 0, 0, 0, time.UTC),  // Next Monday
		},
		{
			name:     "Weekdays only",
			schedule: "0 9 * * 1-5", // Weekdays at 9am
			baseTime: time.Date(2025, 9, 13, 10, 0, 0, 0, time.UTC), // Saturday
			expected: time.Date(2025, 9, 15, 9, 0, 0, 0, time.UTC),  // Monday
		},
		{
			name:     "Weekend only",
			schedule: "0 10 * * 0,6", // Weekend at 10am
			baseTime: time.Date(2025, 9, 12, 10, 0, 0, 0, time.UTC), // Friday
			expected: time.Date(2025, 9, 13, 10, 0, 0, 0, time.UTC), // Saturday
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextRun, err := parser.ParseAndNext(tt.schedule, tt.baseTime)
			require.NoError(t, err)
			require.NotNil(t, nextRun)
			assert.Equal(t, tt.expected, *nextRun)
		})
	}
}

func TestCronParserMonthlySchedules(t *testing.T) {
	parser := NewCronParser()

	tests := []struct {
		name     string
		schedule string
		baseTime time.Time
		expected time.Time
	}{
		{
			name:     "First of month",
			schedule: "0 0 1 * *",
			baseTime: time.Date(2025, 9, 15, 10, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Last day of month (28th)",
			schedule: "0 0 28 * *",
			baseTime: time.Date(2025, 9, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 9, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Specific month",
			schedule: "0 0 15 12 *", // December 15th
			baseTime: time.Date(2025, 9, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextRun, err := parser.ParseAndNext(tt.schedule, tt.baseTime)
			require.NoError(t, err)
			require.NotNil(t, nextRun)
			assert.Equal(t, tt.expected, *nextRun)
		})
	}
}

func TestCronParserEdgeCases(t *testing.T) {
	parser := NewCronParser()

	// Test schedule that's already passed today
	baseTime := time.Date(2025, 9, 12, 15, 0, 0, 0, time.UTC)
	schedule := "0 10 * * *" // 10am daily

	nextRun, err := parser.ParseAndNext(schedule, baseTime)
	require.NoError(t, err)
	require.NotNil(t, nextRun)

	// Should be tomorrow at 10am
	expected := time.Date(2025, 9, 13, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, *nextRun)

	// Test leap year handling
	leapYearBase := time.Date(2024, 2, 28, 10, 0, 0, 0, time.UTC)
	schedule = "0 0 29 2 *" // Feb 29th

	nextRun, err = parser.ParseAndNext(schedule, leapYearBase)
	require.NoError(t, err)
	require.NotNil(t, nextRun)

	// Should be Feb 29, 2024 (leap year)
	expected = time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, *nextRun)
}
