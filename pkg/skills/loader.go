package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Loader loads skills from SKILL.md files
type Loader struct {
	paths []string
}

// NewLoader creates a new skill loader
func NewLoader(paths []string) *Loader {
	return &Loader{
		paths: paths,
	}
}

// LoadFromPath loads all skills from a given path
func (l *Loader) LoadFromPath(path string) ([]*Skill, error) {
	var skills []*Skill

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist, return empty list
			return nil, nil
		}
		return nil, fmt.Errorf("failed to access path %s: %w", path, err)
	}

	// If it's a single file, load it
	if !info.IsDir() {
		skill, err := l.LoadFromFile(path)
		if err != nil {
			return nil, err
		}
		return []*Skill{skill}, nil
	}

	// Scan directory for skill directories
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(path, entry.Name())
		skill, err := l.LoadFromDir(skillPath)
		if err != nil {
			// Log error but continue loading other skills
			fmt.Printf("Warning: failed to load skill from %s: %v\n", skillPath, err)
			continue
		}

		if skill != nil {
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// LoadFromDir loads a skill from a directory
func (l *Loader) LoadFromDir(dirPath string) (*Skill, error) {
	skillPath := filepath.Join(dirPath, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return nil, nil // Not a skill directory
	}

	return l.LoadFromFile(skillPath)
}

// LoadFromFile loads a skill from a SKILL.md file
func (l *Loader) LoadFromFile(filePath string) (*Skill, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	return l.ParseSkill(filePath, string(content))
}

// ParseSkill parses skill content from a SKILL.md file
func (l *Loader) ParseSkill(filePath, content string) (*Skill, error) {
	// Split frontmatter and content
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid skill format: missing frontmatter")
	}

	// Parse YAML frontmatter
	var metadata struct {
		Name                   string            `yaml:"name"`
		Description            string            `yaml:"description"`
		Version                string            `yaml:"version"`
		Author                 string            `yaml:"author"`
		Category               string            `yaml:"category"`
		Tags                   []string          `yaml:"tags"`
		Command                string            `yaml:"command"`
		ForkMode               bool              `yaml:"fork_mode"`
		UserInvocable          bool              `yaml:"user_invocable"`
		DisableModelInvocation bool              `yaml:"disable_model_invocation"`
		Variables              []map[string]any  `yaml:"variables"`
		Extra                  map[string]string `yaml:",inline"`
	}

	if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse skill metadata: %w", err)
	}

	// Generate skill ID from directory name or name field
	dirName := filepath.Dir(filePath)
	skillID := filepath.Base(dirName)
	if metadata.Name != "" {
		skillID = strings.ToLower(strings.ReplaceAll(metadata.Name, " ", "-"))
	}

	// Parse variables
	variables := make(map[string]VariableDef)
	if len(metadata.Variables) > 0 {
		for _, v := range metadata.Variables {
			vdef := VariableDef{}
			if name, ok := v["name"].(string); ok {
				vdef.Name = name
			}
			if desc, ok := v["description"].(string); ok {
				vdef.Description = desc
			}
			if typ, ok := v["type"].(string); ok {
				vdef.Type = typ
			} else {
				vdef.Type = "string"
			}
			if required, ok := v["bool"].(bool); ok {
				vdef.Required = required
			}
			if def, ok := v["default"]; ok {
				vdef.Default = def
			}
			if pattern, ok := v["pattern"].(string); ok {
				vdef.Pattern = pattern
			}
			if vdef.Name != "" {
				variables[vdef.Name] = vdef
			}
		}
	}

	// Parse steps from content (## headers)
	steps := l.parseSteps(parts[2])

	skill := &Skill{
		ID:                     skillID,
		Name:                   metadata.Name,
		Description:            metadata.Description,
		Version:                metadata.Version,
		Author:                 metadata.Author,
		Category:               metadata.Category,
		Tags:                   metadata.Tags,
		Command:                metadata.Command,
		ForkMode:               metadata.ForkMode,
		UserInvocable:          true, // Default to true
		DisableModelInvocation: metadata.DisableModelInvocation,
		Variables:              variables,
		Steps:                  steps,
		Path:                   filePath,
		Enabled:                true,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	if metadata.UserInvocable {
		skill.UserInvocable = metadata.UserInvocable
	}

	return skill, nil
}

// parseSteps parses skill steps from content
func (l *Loader) parseSteps(content string) []SkillStep {
	var steps []SkillStep

	lines := strings.Split(content, "\n")
	var currentStep *SkillStep
	stepNum := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for step headers (## Step N: or ## N.)
		if strings.HasPrefix(trimmed, "##") {
			// Save previous step
			if currentStep != nil {
				steps = append(steps, *currentStep)
			}

			// Start new step
			stepNum++
			title := strings.TrimPrefix(trimmed, "##")
			title = strings.TrimPrefix(title, "Step")
			title = strings.TrimLeft(title, "0123456789.")
			title = strings.TrimSpace(title)

			currentStep = &SkillStep{
				ID:    fmt.Sprintf("step-%d", stepNum),
				Title: title,
			}
		} else if currentStep != nil {
			// Add content to current step
			if currentStep.Content != "" {
				currentStep.Content += "\n"
			}
			currentStep.Content += line
		}
	}

	// Add last step
	if currentStep != nil {
		steps = append(steps, *currentStep)
	}

	return steps
}

// LoadAll loads all skills from configured paths
func (l *Loader) LoadAll(ctx context.Context) ([]*Skill, error) {
	var allSkills []*Skill

	for _, path := range l.paths {
		skills, err := l.LoadFromPath(path)
		if err != nil {
			fmt.Printf("Warning: failed to load skills from %s: %v\n", path, err)
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	return allSkills, nil
}
