package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// CronParser handles cron expression parsing and next time calculation
type CronParser struct {
	parser cron.Parser
}

// NewCronParser creates a new cron parser
func NewCronParser() *CronParser {
	// Use standard cron parser with optional seconds field
	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)

	return &CronParser{
		parser: parser,
	}
}

// ParseAndNext parses a cron expression and returns the next execution time
func (c *CronParser) ParseAndNext(cronExpr string, from time.Time) (*time.Time, error) {
	if cronExpr == "" {
		// Empty schedule means one-time execution
		return nil, nil
	}

	// Handle special descriptors
	switch cronExpr {
	case "@yearly", "@annually":
		cronExpr = "0 0 1 1 *"
	case "@monthly":
		cronExpr = "0 0 1 * *"
	case "@weekly":
		cronExpr = "0 0 * * 0"
	case "@daily", "@midnight":
		cronExpr = "0 0 * * *"
	case "@hourly":
		cronExpr = "0 * * * *"
	}

	schedule, err := c.parser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression '%s': %w", cronExpr, err)
	}

	next := schedule.Next(from)
	return &next, nil
}

// Validate checks if a cron expression is valid
func (c *CronParser) Validate(cronExpr string) error {
	if cronExpr == "" {
		return nil // Empty is valid (one-time)
	}

	_, err := c.ParseAndNext(cronExpr, time.Now())
	return err
}

// GetNextExecutions returns the next N execution times for a cron expression
func (c *CronParser) GetNextExecutions(cronExpr string, from time.Time, count int) ([]time.Time, error) {
	if cronExpr == "" || count <= 0 {
		return nil, nil
	}

	schedule, err := c.parser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression '%s': %w", cronExpr, err)
	}

	var executions []time.Time
	current := from

	for i := 0; i < count; i++ {
		next := schedule.Next(current)
		executions = append(executions, next)
		current = next
	}

	return executions, nil
}

// DescribeCron returns a human-readable description of a cron expression
func (c *CronParser) DescribeCron(cronExpr string) string {
	if cronExpr == "" {
		return "One-time execution"
	}

	// Handle common patterns
	switch cronExpr {
	case "@yearly", "@annually", "0 0 1 1 *":
		return "Once a year (January 1st at midnight)"
	case "@monthly", "0 0 1 * *":
		return "Once a month (1st day at midnight)"
	case "@weekly", "0 0 * * 0":
		return "Once a week (Sunday at midnight)"
	case "@daily", "@midnight", "0 0 * * *":
		return "Once a day (at midnight)"
	case "@hourly", "0 * * * *":
		return "Once an hour (at the top of the hour)"
	case "*/5 * * * *":
		return "Every 5 minutes"
	case "*/10 * * * *":
		return "Every 10 minutes"
	case "*/15 * * * *":
		return "Every 15 minutes"
	case "*/30 * * * *":
		return "Every 30 minutes"
	case "0 */2 * * *":
		return "Every 2 hours"
	case "0 */6 * * *":
		return "Every 6 hours"
	case "0 */12 * * *":
		return "Every 12 hours"
	}

	// Try to parse and get next execution for description
	next, err := c.ParseAndNext(cronExpr, time.Now())
	if err != nil {
		return fmt.Sprintf("Invalid cron expression: %s", cronExpr)
	}

	if next != nil {
		return fmt.Sprintf("Next: %s (cron: %s)", next.Format("2006-01-02 15:04:05"), cronExpr)
	}

	return fmt.Sprintf("Cron: %s", cronExpr)
}
