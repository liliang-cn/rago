package agent

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// ChecklistItemStatus represents the status of a checklist item
type ChecklistItemStatus string

const (
	ChecklistItemPending    ChecklistItemStatus = "pending"
	ChecklistItemDone       ChecklistItemStatus = "done"
	ChecklistItemInProgress ChecklistItemStatus = "in_progress"
)

// ChecklistItem represents a single item in HEARTBEAT.md
type ChecklistItem struct {
	ID               string              `json:"id"`
	Description      string              `json:"description"`
	Status           ChecklistItemStatus `json:"status"`
	RequiresApproval bool                `json:"requires_approval"`
	Priority         int                 `json:"priority"`
	CreatedAt        time.Time           `json:"created_at"`
}

// Checklist represents the parsed HEARTBEAT.md
type Checklist struct {
	Items      []ChecklistItem `json:"items"`
	Updated    time.Time       `json:"updated"`
	RawContent string          `json:"raw_content,omitempty"`
}

// ChecklistParser parses HEARTBEAT.md files
type ChecklistParser struct{}

// NewChecklistParser creates a new parser
func NewChecklistParser() *ChecklistParser {
	return &ChecklistParser{}
}

// Parse parses HEARTBEAT.md content
func (p *ChecklistParser) Parse(content string) *Checklist {
	items := []ChecklistItem{}
	scanner := bufio.NewScanner(strings.NewReader(content))

	// Regex patterns
	// - [ ] Task description
	// - [x] Done task
	// - [!] Requires approval
	checklistRegex := regexp.MustCompile(`^\s*-\s*\[([ x!~])\]\s*(.+)$`)

	itemID := 0
	for scanner.Scan() {
		line := scanner.Text()

		matches := checklistRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		itemID++
		status := matches[1]
		description := strings.TrimSpace(matches[2])

		item := ChecklistItem{
			ID:          fmt.Sprintf("item_%d", itemID),
			Description: description,
			CreatedAt:   time.Now(),
		}

		switch status {
		case " ":
			item.Status = ChecklistItemPending
		case "x":
			item.Status = ChecklistItemDone
		case "!":
			item.Status = ChecklistItemPending
			item.RequiresApproval = true
		case "~":
			item.Status = ChecklistItemInProgress
		}

		items = append(items, item)
	}

	return &Checklist{
		Items:      items,
		Updated:    time.Now(),
		RawContent: content,
	}
}

// ParseFile parses a HEARTBEAT.md file
func (p *ChecklistParser) ParseFile(path string) (*Checklist, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return p.Parse(string(content)), nil
}

// readChecklist reads and parses the HEARTBEAT.md file
func (s *LongRunService) readChecklist() (*Checklist, error) {
	parser := NewChecklistParser()
	return parser.ParseFile(s.config.HeartbeatFile)
}

// updateChecklistItem updates a checklist item's status
func (s *LongRunService) updateChecklistItem(itemID string, status ChecklistItemStatus) error {
	content, err := os.ReadFile(s.config.HeartbeatFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	itemNum := 0
	checklistRegex := regexp.MustCompile(`^(\s*-\s*\[)([ x!~])(\]\s*)(.+)$`)

	for i, line := range lines {
		matches := checklistRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		itemNum++
		currentID := fmt.Sprintf("item_%d", itemNum)

		if currentID == itemID {
			var newStatus string
			switch status {
			case ChecklistItemDone:
				newStatus = "x"
			case ChecklistItemInProgress:
				newStatus = "~"
			default:
				newStatus = " "
			}

			// Preserve requires_approval marker if present
			originalStatus := matches[2]
			if originalStatus == "!" && status != ChecklistItemDone {
				newStatus = "!"
			}

			lines[i] = matches[1] + newStatus + matches[3] + matches[4]
			break
		}
	}

	return os.WriteFile(s.config.HeartbeatFile, []byte(strings.Join(lines, "\n")), 0644)
}

// FormatChecklistForLLM formats checklist for LLM context
func FormatChecklistForLLM(checklist *Checklist) string {
	if checklist == nil || len(checklist.Items) == 0 {
		return "No pending tasks."
	}

	var sb strings.Builder
	sb.WriteString("# Current Tasks\n\n")

	for _, item := range checklist.Items {
		status := "⬜"
		if item.Status == ChecklistItemDone {
			status = "✅"
		} else if item.Status == ChecklistItemInProgress {
			status = "🔄"
		}

		approval := ""
		if item.RequiresApproval {
			approval = " ⚠️(requires approval)"
		}

		sb.WriteString(fmt.Sprintf("%s %s%s\n", status, item.Description, approval))
	}

	return sb.String()
}
