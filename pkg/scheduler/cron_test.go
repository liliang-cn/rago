package scheduler

import (
	"testing"
	"time"
)

func TestNewCronParser(t *testing.T) {
	parser := NewCronParser()
	
	if parser == nil {
		t.Fatal("NewCronParser() returned nil")
	}
	
	// Test that the parser can parse a simple expression to verify it's properly initialized
	_, err := parser.ParseAndNext("* * * * *", time.Now())
	if err != nil {
		t.Errorf("NewCronParser() created parser that can't parse basic expression: %v", err)
	}
}

func TestCronParserValidate(t *testing.T) {
	parser := NewCronParser()
	
	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "Empty expression (valid for one-time)",
			expression: "",
			wantErr:    false,
		},
		{
			name:       "Every minute",
			expression: "* * * * *",
			wantErr:    false,
		},
		{
			name:       "Every 5 minutes",
			expression: "*/5 * * * *",
			wantErr:    false,
		},
		{
			name:       "Daily at midnight",
			expression: "0 0 * * *",
			wantErr:    false,
		},
		{
			name:       "Weekly on Sunday",
			expression: "0 0 * * 0",
			wantErr:    false,
		},
		{
			name:       "Monthly on first",
			expression: "0 0 1 * *",
			wantErr:    false,
		},
		{
			name:       "Yearly descriptor",
			expression: "@yearly",
			wantErr:    false,
		},
		{
			name:       "Daily descriptor",
			expression: "@daily",
			wantErr:    false,
		},
		{
			name:       "Hourly descriptor",
			expression: "@hourly",
			wantErr:    false,
		},
		{
			name:       "With seconds",
			expression: "30 */5 * * * *",
			wantErr:    false,
		},
		{
			name:       "Invalid expression - too few fields",
			expression: "* * *",
			wantErr:    true,
		},
		{
			name:       "Invalid expression - too many fields",
			expression: "* * * * * * *",
			wantErr:    true,
		},
		{
			name:       "Invalid expression - bad range",
			expression: "70 * * * *",
			wantErr:    true,
		},
		{
			name:       "Invalid expression - bad day",
			expression: "0 0 32 * *",
			wantErr:    true,
		},
		{
			name:       "Invalid expression - bad month",
			expression: "0 0 1 13 *",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.Validate(tt.expression)
			
			if tt.wantErr && err == nil {
				t.Errorf("Validate(%s) expected error, got nil", tt.expression)
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("Validate(%s) unexpected error: %v", tt.expression, err)
			}
		})
	}
}

func TestParseAndNext(t *testing.T) {
	parser := NewCronParser()
	baseTime := time.Date(2023, 12, 25, 12, 0, 0, 0, time.UTC) // Monday, Dec 25, 2023 12:00 PM UTC
	
	tests := []struct {
		name       string
		expression string
		wantErr    bool
		checkNext  bool
		expectAfter time.Time // Next execution should be after this time
	}{
		{
			name:        "Empty expression returns nil",
			expression:  "",
			wantErr:     false,
			checkNext:   false,
		},
		{
			name:        "Every minute",
			expression:  "* * * * *",
			wantErr:     false,
			checkNext:   true,
			expectAfter: baseTime,
		},
		{
			name:        "Every 5 minutes",
			expression:  "*/5 * * * *",
			wantErr:     false,
			checkNext:   true,
			expectAfter: baseTime,
		},
		{
			name:        "Daily at midnight",
			expression:  "0 0 * * *",
			wantErr:     false,
			checkNext:   true,
			expectAfter: baseTime,
		},
		{
			name:        "Yearly descriptor",
			expression:  "@yearly",
			wantErr:     false,
			checkNext:   true,
			expectAfter: baseTime,
		},
		{
			name:        "Monthly descriptor", 
			expression:  "@monthly",
			wantErr:     false,
			checkNext:   true,
			expectAfter: baseTime,
		},
		{
			name:        "Weekly descriptor",
			expression:  "@weekly",
			wantErr:     false,
			checkNext:   true,
			expectAfter: baseTime,
		},
		{
			name:        "Daily descriptor",
			expression:  "@daily",
			wantErr:     false,
			checkNext:   true,
			expectAfter: baseTime,
		},
		{
			name:        "Hourly descriptor",
			expression:  "@hourly",
			wantErr:     false,
			checkNext:   true,
			expectAfter: baseTime,
		},
		{
			name:        "Invalid expression",
			expression:  "invalid cron",
			wantErr:     true,
			checkNext:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := parser.ParseAndNext(tt.expression, baseTime)
			
			if tt.wantErr && err == nil {
				t.Errorf("ParseAndNext(%s) expected error, got nil", tt.expression)
				return
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("ParseAndNext(%s) unexpected error: %v", tt.expression, err)
				return
			}
			
			if !tt.checkNext {
				if next != nil {
					t.Errorf("ParseAndNext(%s) expected nil next time, got %v", tt.expression, *next)
				}
				return
			}
			
			if next == nil {
				t.Errorf("ParseAndNext(%s) expected next time, got nil", tt.expression)
				return
			}
			
			if !next.After(tt.expectAfter) && !next.Equal(tt.expectAfter.Add(time.Minute)) {
				t.Errorf("ParseAndNext(%s) next time %v should be after %v", tt.expression, *next, tt.expectAfter)
			}
		})
	}
}

func TestParseAndNextDescriptors(t *testing.T) {
	parser := NewCronParser()
	baseTime := time.Date(2023, 6, 15, 14, 30, 0, 0, time.UTC) // June 15, 2023 2:30 PM
	
	descriptorTests := []struct {
		descriptor string
		expected   string // What the descriptor should resolve to
	}{
		{"@yearly", "0 0 1 1 *"},
		{"@annually", "0 0 1 1 *"},
		{"@monthly", "0 0 1 * *"},
		{"@weekly", "0 0 * * 0"},
		{"@daily", "0 0 * * *"},
		{"@midnight", "0 0 * * *"},
		{"@hourly", "0 * * * *"},
	}
	
	for _, tt := range descriptorTests {
		t.Run(tt.descriptor, func(t *testing.T) {
			// Test descriptor
			next1, err1 := parser.ParseAndNext(tt.descriptor, baseTime)
			if err1 != nil {
				t.Errorf("ParseAndNext(%s) unexpected error: %v", tt.descriptor, err1)
				return
			}
			
			// Test equivalent expression
			next2, err2 := parser.ParseAndNext(tt.expected, baseTime)
			if err2 != nil {
				t.Errorf("ParseAndNext(%s) unexpected error: %v", tt.expected, err2)
				return
			}
			
			if next1 == nil || next2 == nil {
				t.Errorf("Expected both next times to be non-nil, got %v and %v", next1, next2)
				return
			}
			
			if !next1.Equal(*next2) {
				t.Errorf("Descriptor %s and expression %s should produce same next time, got %v and %v", 
					tt.descriptor, tt.expected, *next1, *next2)
			}
		})
	}
}

func TestGetNextExecutions(t *testing.T) {
	parser := NewCronParser()
	baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	
	tests := []struct {
		name       string
		expression string
		count      int
		wantErr    bool
		expectNil  bool
	}{
		{
			name:       "Empty expression",
			expression: "",
			count:      5,
			wantErr:    false,
			expectNil:  true,
		},
		{
			name:       "Zero count",
			expression: "0 * * * *",
			count:      0,
			wantErr:    false,
			expectNil:  true,
		},
		{
			name:       "Negative count",
			expression: "0 * * * *",
			count:      -1,
			wantErr:    false,
			expectNil:  true,
		},
		{
			name:       "Every hour - 3 executions",
			expression: "0 * * * *",
			count:      3,
			wantErr:    false,
			expectNil:  false,
		},
		{
			name:       "Daily - 5 executions",
			expression: "@daily",
			count:      5,
			wantErr:    false,
			expectNil:  false,
		},
		{
			name:       "Invalid expression",
			expression: "invalid",
			count:      3,
			wantErr:    true,
			expectNil:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executions, err := parser.GetNextExecutions(tt.expression, baseTime, tt.count)
			
			if tt.wantErr && err == nil {
				t.Errorf("GetNextExecutions(%s, %d) expected error, got nil", tt.expression, tt.count)
				return
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("GetNextExecutions(%s, %d) unexpected error: %v", tt.expression, tt.count, err)
				return
			}
			
			if tt.expectNil && executions != nil {
				t.Errorf("GetNextExecutions(%s, %d) expected nil, got %v", tt.expression, tt.count, executions)
				return
			}
			
			if !tt.expectNil && !tt.wantErr {
				if executions == nil {
					t.Errorf("GetNextExecutions(%s, %d) expected executions, got nil", tt.expression, tt.count)
					return
				}
				
				if len(executions) != tt.count {
					t.Errorf("GetNextExecutions(%s, %d) expected %d executions, got %d", 
						tt.expression, tt.count, tt.count, len(executions))
					return
				}
				
				// Verify executions are in chronological order
				for i := 1; i < len(executions); i++ {
					if !executions[i].After(executions[i-1]) {
						t.Errorf("GetNextExecutions executions not in chronological order: %v >= %v", 
							executions[i-1], executions[i])
					}
				}
				
				// Verify first execution is after base time
				if !executions[0].After(baseTime) {
					t.Errorf("GetNextExecutions first execution %v should be after base time %v", 
						executions[0], baseTime)
				}
			}
		})
	}
}

func TestDescribeCron(t *testing.T) {
	parser := NewCronParser()
	
	tests := []struct {
		name       string
		expression string
		contains   []string // Substrings that should be in the description
	}{
		{
			name:       "Empty expression",
			expression: "",
			contains:   []string{"One-time execution"},
		},
		{
			name:       "Yearly",
			expression: "@yearly",
			contains:   []string{"Once a year", "January 1st", "midnight"},
		},
		{
			name:       "Annually", 
			expression: "@annually",
			contains:   []string{"Once a year", "January 1st", "midnight"},
		},
		{
			name:       "Monthly",
			expression: "@monthly",
			contains:   []string{"Once a month", "1st day", "midnight"},
		},
		{
			name:       "Weekly",
			expression: "@weekly",
			contains:   []string{"Once a week", "Sunday", "midnight"},
		},
		{
			name:       "Daily",
			expression: "@daily",
			contains:   []string{"Once a day", "midnight"},
		},
		{
			name:       "Midnight",
			expression: "@midnight",
			contains:   []string{"Once a day", "midnight"},
		},
		{
			name:       "Hourly",
			expression: "@hourly",
			contains:   []string{"Once an hour", "top of the hour"},
		},
		{
			name:       "Every 5 minutes",
			expression: "*/5 * * * *",
			contains:   []string{"Every 5 minutes"},
		},
		{
			name:       "Every 15 minutes",
			expression: "*/15 * * * *",
			contains:   []string{"Every 15 minutes"},
		},
		{
			name:       "Every 2 hours",
			expression: "0 */2 * * *",
			contains:   []string{"Every 2 hours"},
		},
		{
			name:       "Custom valid expression",
			expression: "30 14 * * 1",
			contains:   []string{"Next:", "cron:"},
		},
		{
			name:       "Invalid expression",
			expression: "invalid cron",
			contains:   []string{"Invalid cron expression"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := parser.DescribeCron(tt.expression)
			
			if description == "" {
				t.Errorf("DescribeCron(%s) returned empty string", tt.expression)
				return
			}
			
			for _, substr := range tt.contains {
				if len(substr) > 0 && !containsIgnoreCase(description, substr) {
					t.Errorf("DescribeCron(%s) = %q, should contain %q", 
						tt.expression, description, substr)
				}
			}
		})
	}
}

func TestCronParserEdgeCases(t *testing.T) {
	parser := NewCronParser()
	
	t.Run("Parse at year boundary", func(t *testing.T) {
		// Test parsing at year boundary
		yearEnd := time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC)
		next, err := parser.ParseAndNext("0 0 * * *", yearEnd)
		
		if err != nil {
			t.Errorf("ParseAndNext at year boundary unexpected error: %v", err)
			return
		}
		
		if next == nil {
			t.Error("ParseAndNext at year boundary should return next time")
			return
		}
		
		expectedNext := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		if !next.Equal(expectedNext) {
			t.Errorf("ParseAndNext at year boundary expected %v, got %v", expectedNext, *next)
		}
	})
	
	t.Run("Parse with leap year", func(t *testing.T) {
		// Test Feb 29 in leap year
		leapDay := time.Date(2024, 2, 28, 12, 0, 0, 0, time.UTC)
		next, err := parser.ParseAndNext("0 0 * * *", leapDay)
		
		if err != nil {
			t.Errorf("ParseAndNext on leap day unexpected error: %v", err)
			return
		}
		
		if next == nil {
			t.Error("ParseAndNext on leap day should return next time")
			return
		}
		
		expectedNext := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
		if !next.Equal(expectedNext) {
			t.Errorf("ParseAndNext leap day expected %v, got %v", expectedNext, *next)
		}
	})
	
	t.Run("Multiple calls maintain state", func(t *testing.T) {
		baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		
		// First call
		next1, err1 := parser.ParseAndNext("0 * * * *", baseTime)
		if err1 != nil || next1 == nil {
			t.Errorf("First call failed: %v", err1)
			return
		}
		
		// Second call should be independent
		next2, err2 := parser.ParseAndNext("0 * * * *", baseTime)
		if err2 != nil || next2 == nil {
			t.Errorf("Second call failed: %v", err2)
			return
		}
		
		if !next1.Equal(*next2) {
			t.Errorf("Parser calls should be independent, got %v and %v", *next1, *next2)
		}
	})
}

// Helper function for case-insensitive string contains check
func containsIgnoreCase(s, substr string) bool {
	// Simple implementation - convert to lowercase and check
	sLower := ""
	substrLower := ""
	
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			sLower += string(r - 'A' + 'a')
		} else {
			sLower += string(r)
		}
	}
	
	for _, r := range substr {
		if r >= 'A' && r <= 'Z' {
			substrLower += string(r - 'A' + 'a')
		} else {
			substrLower += string(r)
		}
	}
	
	// Check if substr is in s
	if len(substrLower) > len(sLower) {
		return false
	}
	
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		match := true
		for j := 0; j < len(substrLower); j++ {
			if sLower[i+j] != substrLower[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}