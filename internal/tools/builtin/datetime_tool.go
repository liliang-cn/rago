package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/internal/tools"
)

// DateTimeTool provides date and time related functionality
type DateTimeTool struct {
	location *time.Location
}

// NewDateTimeTool creates a new datetime tool
func NewDateTimeTool() *DateTimeTool {
	return &DateTimeTool{
		location: time.Local,
	}
}

// NewDateTimeToolWithLocation creates a new datetime tool with specific timezone
func NewDateTimeToolWithLocation(locationName string) (*DateTimeTool, error) {
	loc, err := time.LoadLocation(locationName)
	if err != nil {
		return nil, fmt.Errorf("invalid location %s: %w", locationName, err)
	}
	return &DateTimeTool{
		location: loc,
	}, nil
}

// Name returns the tool name
func (t *DateTimeTool) Name() string {
	return "datetime"
}

// Description returns the tool description
func (t *DateTimeTool) Description() string {
	return "Get current date/time, format dates, calculate date differences, and convert timezones"
}

// Parameters returns the tool parameters schema
func (t *DateTimeTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"action": {
				Type:        "string",
				Description: "The action to perform",
				Enum:        []string{"now", "format", "parse", "add", "diff", "timezone"},
			},
			"date": {
				Type:        "string",
				Description: "Date string (ISO format or other common formats)",
			},
			"format": {
				Type:        "string",
				Description: "Date format (Go format or common names: RFC3339, ISO8601, etc.)",
				Default:     "2006-01-02 15:04:05",
			},
			"duration": {
				Type:        "string",
				Description: "Duration string (e.g., '24h', '7d', '1y')",
			},
			"timezone": {
				Type:        "string",
				Description: "Timezone name (e.g., 'America/New_York', 'UTC', 'Asia/Shanghai')",
			},
			"date2": {
				Type:        "string",
				Description: "Second date for diff operation",
			},
		},
		Required: []string{"action"},
	}
}

// Execute runs the datetime tool
func (t *DateTimeTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	action, ok := args["action"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "action parameter is required",
		}, nil
	}

	switch action {
	case "now":
		return t.getNow(args)
	case "format":
		return t.formatDate(args)
	case "parse":
		return t.parseDate(args)
	case "add":
		return t.addDuration(args)
	case "diff":
		return t.dateDiff(args)
	case "timezone":
		return t.convertTimezone(args)
	default:
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", action),
		}, nil
	}
}

// Validate validates the tool arguments
func (t *DateTimeTool) Validate(args map[string]interface{}) error {
	action, ok := args["action"]
	if !ok {
		return fmt.Errorf("action parameter is required")
	}

	actionStr, ok := action.(string)
	if !ok {
		return fmt.Errorf("action must be a string")
	}

	validActions := []string{"now", "format", "parse", "add", "diff", "timezone"}
	valid := false
	for _, v := range validActions {
		if actionStr == v {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid action: %s", actionStr)
	}

	// Validate specific action requirements
	switch actionStr {
	case "format", "parse", "add":
		if _, ok := args["date"]; !ok && actionStr != "now" {
			return fmt.Errorf("date parameter is required for %s action", actionStr)
		}
	case "diff":
		if _, ok := args["date"]; !ok {
			return fmt.Errorf("date parameter is required for diff action")
		}
		if _, ok := args["date2"]; !ok {
			return fmt.Errorf("date2 parameter is required for diff action")
		}
	case "timezone":
		if _, ok := args["timezone"]; !ok {
			return fmt.Errorf("timezone parameter is required for timezone action")
		}
	}

	return nil
}

// Helper methods

func (t *DateTimeTool) getNow(args map[string]interface{}) (*tools.ToolResult, error) {
	now := time.Now().In(t.location)

	format := t.getFormat(args)
	formatted := t.formatTime(now, format)

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"datetime":  formatted,
			"timestamp": now.Unix(),
			"timezone":  now.Location().String(),
			"iso8601":   now.Format(time.RFC3339),
		},
	}, nil
}

func (t *DateTimeTool) formatDate(args map[string]interface{}) (*tools.ToolResult, error) {
	dateStr, ok := args["date"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "date parameter is required",
		}, nil
	}

	parsedTime, err := t.parseTime(dateStr)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse date: %v", err),
		}, nil
	}

	format := t.getFormat(args)
	formatted := t.formatTime(parsedTime, format)

	return &tools.ToolResult{
		Success: true,
		Data:    formatted,
	}, nil
}

func (t *DateTimeTool) parseDate(args map[string]interface{}) (*tools.ToolResult, error) {
	dateStr, ok := args["date"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "date parameter is required",
		}, nil
	}

	parsedTime, err := t.parseTime(dateStr)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse date: %v", err),
		}, nil
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"year":      parsedTime.Year(),
			"month":     parsedTime.Month().String(),
			"day":       parsedTime.Day(),
			"hour":      parsedTime.Hour(),
			"minute":    parsedTime.Minute(),
			"second":    parsedTime.Second(),
			"weekday":   parsedTime.Weekday().String(),
			"timestamp": parsedTime.Unix(),
			"iso8601":   parsedTime.Format(time.RFC3339),
		},
	}, nil
}

func (t *DateTimeTool) addDuration(args map[string]interface{}) (*tools.ToolResult, error) {
	dateStr, ok := args["date"].(string)
	if !ok {
		// Use current time if date not provided
		dateStr = time.Now().In(t.location).Format(time.RFC3339)
	}

	durationStr, ok := args["duration"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "duration parameter is required",
		}, nil
	}

	parsedTime, err := t.parseTime(dateStr)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse date: %v", err),
		}, nil
	}

	duration, err := t.parseDuration(durationStr)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse duration: %v", err),
		}, nil
	}

	newTime := parsedTime.Add(duration)
	format := t.getFormat(args)

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"original": t.formatTime(parsedTime, format),
			"new":      t.formatTime(newTime, format),
			"duration": durationStr,
		},
	}, nil
}

func (t *DateTimeTool) dateDiff(args map[string]interface{}) (*tools.ToolResult, error) {
	date1Str, ok := args["date"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "date parameter is required",
		}, nil
	}

	date2Str, ok := args["date2"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "date2 parameter is required",
		}, nil
	}

	time1, err := t.parseTime(date1Str)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse date: %v", err),
		}, nil
	}

	time2, err := t.parseTime(date2Str)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse date2: %v", err),
		}, nil
	}

	diff := time2.Sub(time1)

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"seconds": diff.Seconds(),
			"minutes": diff.Minutes(),
			"hours":   diff.Hours(),
			"days":    diff.Hours() / 24,
			"human":   diff.String(),
		},
	}, nil
}

func (t *DateTimeTool) convertTimezone(args map[string]interface{}) (*tools.ToolResult, error) {
	tzName, ok := args["timezone"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "timezone parameter is required",
		}, nil
	}

	targetLoc, err := time.LoadLocation(tzName)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid timezone: %v", err),
		}, nil
	}

	var sourceTime time.Time
	if dateStr, ok := args["date"].(string); ok {
		sourceTime, err = t.parseTime(dateStr)
		if err != nil {
			return &tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to parse date: %v", err),
			}, nil
		}
	} else {
		sourceTime = time.Now().In(t.location)
	}

	convertedTime := sourceTime.In(targetLoc)
	format := t.getFormat(args)

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"original": map[string]interface{}{
				"time":     t.formatTime(sourceTime, format),
				"timezone": sourceTime.Location().String(),
			},
			"converted": map[string]interface{}{
				"time":     t.formatTime(convertedTime, format),
				"timezone": convertedTime.Location().String(),
			},
		},
	}, nil
}

// Utility functions

func (t *DateTimeTool) getFormat(args map[string]interface{}) string {
	if format, ok := args["format"].(string); ok {
		// Handle common format names
		switch format {
		case "RFC3339":
			return time.RFC3339
		case "ISO8601":
			return time.RFC3339
		case "RFC822":
			return time.RFC822
		case "Kitchen":
			return time.Kitchen
		case "Stamp":
			return time.Stamp
		default:
			return format
		}
	}
	return "2006-01-02 15:04:05"
}

func (t *DateTimeTool) formatTime(tm time.Time, format string) string {
	return tm.Format(format)
}

func (t *DateTimeTool) parseTime(dateStr string) (time.Time, error) {
	// Try common formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"15:04:05",
		"Jan 2, 2006",
		"02/01/2006",
		"01/02/2006",
		time.RFC822,
		time.RFC850,
		time.RFC1123,
	}

	for _, format := range formats {
		if tm, err := time.ParseInLocation(format, dateStr, t.location); err == nil {
			return tm, nil
		}
	}

	// Try parse with default location
	for _, format := range formats {
		if tm, err := time.Parse(format, dateStr); err == nil {
			return tm, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func (t *DateTimeTool) parseDuration(durationStr string) (time.Duration, error) {
	// First try standard Go duration parsing
	if d, err := time.ParseDuration(durationStr); err == nil {
		return d, nil
	}

	// Handle custom duration formats like "7d", "1y"
	var multiplier time.Duration
	var value int

	if n, err := fmt.Sscanf(durationStr, "%dd", &value); n == 1 && err == nil {
		// Days
		multiplier = 24 * time.Hour
	} else if n, err := fmt.Sscanf(durationStr, "%dw", &value); n == 1 && err == nil {
		// Weeks
		multiplier = 7 * 24 * time.Hour
	} else if n, err := fmt.Sscanf(durationStr, "%dM", &value); n == 1 && err == nil {
		// Months (approximate)
		multiplier = 30 * 24 * time.Hour
	} else if n, err := fmt.Sscanf(durationStr, "%dy", &value); n == 1 && err == nil {
		// Years (approximate)
		multiplier = 365 * 24 * time.Hour
	} else {
		return 0, fmt.Errorf("invalid duration format: %s", durationStr)
	}

	return time.Duration(value) * multiplier, nil
}
