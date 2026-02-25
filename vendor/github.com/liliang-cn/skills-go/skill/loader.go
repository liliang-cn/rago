package skill

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Loader loads skills from filesystem
type Loader struct {
	paths []string
	cache *Cache
}

// LoaderOption configures a Loader
type LoaderOption func(*Loader)

// WithPaths adds search paths for skills
func WithPaths(paths ...string) LoaderOption {
	return func(l *Loader) {
		l.paths = append(l.paths, paths...)
	}
}

// WithCache sets the cache for the loader
func WithCache(cache *Cache) LoaderOption {
	return func(l *Loader) {
		l.cache = cache
	}
}

// NewLoader creates a new skill loader
func NewLoader(opts ...LoaderOption) *Loader {
	homeDir, _ := os.UserHomeDir()

	l := &Loader{
		paths: []string{
			".claude/skills",
			filepath.Join(homeDir, ".claude/skills"),
		},
		cache: NewCache(),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// Load loads a single skill from the specified path with full details
func (l *Loader) Load(ctx context.Context, skillPath string) (*Skill, error) {
	return l.LoadWithLevel(ctx, skillPath, LoadLevelFull)
}

// LoadMetadata loads only the metadata and content of a skill, skipping resources (scripts, references, assets)
// This is equivalent to LoadLevelContent (Level 2 in progressive disclosure)
func (l *Loader) LoadMetadata(ctx context.Context, skillPath string) (*Skill, error) {
	return l.LoadWithLevel(ctx, skillPath, LoadLevelContent)
}

// LoadWithLevel loads a skill with a specific level of detail
func (l *Loader) LoadWithLevel(ctx context.Context, skillPath string, level LoadLevel) (*Skill, error) {
	// Check cache first
	if cached, ok := l.cache.Get(skillPath); ok {
		// If cached version has equal or higher level, return it
		if cached.LoadLevel >= level {
			return cached, nil
		}
		// Otherwise we need to reload or upgrade. For simplicity, we proceed to reload.
	}

	// Find SKILL.md
	skillFile := filepath.Join(skillPath, "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		// Maybe skillPath points directly to SKILL.md
		if strings.HasSuffix(skillPath, "SKILL.md") {
			skillFile = skillPath
			skillPath = filepath.Dir(skillPath)
		} else {
			return nil, ErrSkillNotFound
		}
	}

	// Read file
	content, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, err
	}

	// Parse frontmatter and content
	meta, markdown, err := ParseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Use directory name if name is empty
	if meta.Name == "" {
		meta.Name = filepath.Base(skillPath)
		meta.Name = strings.ToLower(meta.Name)
		meta.Name = strings.ReplaceAll(meta.Name, " ", "-")
	}

	skill := &Skill{
		Meta:      *meta,
		Path:      skillPath,
		Name:      meta.Name,
		Content:   markdown,
		Raw:       string(content),
		LoadedAt:  time.Now(),
		LoadLevel: LoadLevelContent, // We at least have content now
	}

	// If level 1 (Metadata only) is requested, we could technically drop Content/Raw to save memory,
	// but since we already read the file, keeping it is usually better unless memory is constrained.
	// However, to strictly follow the pattern, if LoadLevelMetadata is requested, we might want to
	// reflect that in the struct. For this implementation, we treat reading SKILL.md as LoadLevelContent.
	if level == LoadLevelMetadata {
		skill.LoadLevel = LoadLevelMetadata
		// Optionally clear content if strict memory usage is required:
		// skill.Content = ""
		// skill.Raw = ""
	}

	// Load resources if requested
	if level >= LoadLevelFull {
		resources, err := l.loadResources(skillPath)
		if err != nil {
			return nil, err
		}
		skill.Resources = resources
		skill.LoadLevel = LoadLevelFull
	}

	// Cache it
	l.cache.Set(skillPath, skill)

	return skill, nil
}

// EnsureLoaded ensures the skill is loaded to the specified level
func (l *Loader) EnsureLoaded(ctx context.Context, skill *Skill, level LoadLevel) error {
	if skill.LoadLevel >= level {
		return nil
	}

	// Upgrade to Content level (Metadata -> Content)
	// Since LoadWithLevel already loads content in memory even for Metadata level,
	// we just need to update the state.
	if level >= LoadLevelContent && skill.LoadLevel < LoadLevelContent {
		skill.LoadLevel = LoadLevelContent
	}

	// Upgrade to Full level (Content -> Full)
	if level >= LoadLevelFull && skill.LoadLevel < LoadLevelFull {
		resources, err := l.loadResources(skill.Path)
		if err != nil {
			return err
		}
		skill.Resources = resources
		skill.LoadLevel = LoadLevelFull
	}
	
	// Update cache
	l.cache.Set(skill.Path, skill)
	return nil
}

// LoadAll loads all discoverable skills
func (l *Loader) LoadAll(ctx context.Context) ([]*Skill, error) {
	var skills []*Skill
	seen := make(map[string]bool)
	var mu sync.Mutex

	var wg sync.WaitGroup
	errChan := make(chan error, len(l.paths))

	for _, basePath := range l.paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			entries, err := os.ReadDir(path)
			if err != nil {
				if os.IsNotExist(err) {
					return
				}
				errChan <- err
				return
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				skillPath := filepath.Join(path, entry.Name())

				mu.Lock()
				if seen[skillPath] {
					mu.Unlock()
					continue
				}
				seen[skillPath] = true
				mu.Unlock()

				skill, err := l.Load(ctx, skillPath)
				if err != nil {
					continue // Skip invalid skills
				}

				mu.Lock()
				skills = append(skills, skill)
				mu.Unlock()
			}
		}(basePath)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return skills, nil
}

// Discover recursively discovers skills from a starting path
func (l *Loader) Discover(ctx context.Context, startPath string) ([]*Skill, error) {
	var skills []*Skill

	err := filepath.Walk(startPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// Check if contains SKILL.md
			skillFile := filepath.Join(path, "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				skill, err := l.Load(ctx, path)
				if err == nil {
					skills = append(skills, skill)
				}
			}
		}

		return nil
	})

	return skills, err
}

// loadResources loads supporting files from skill directory
func (l *Loader) loadResources(skillPath string) (*Resources, error) {
	r := &Resources{}

	// Load scripts
	if scriptsDir := filepath.Join(skillPath, "scripts"); dirExists(scriptsDir) {
		entries, _ := os.ReadDir(scriptsDir)
		for _, e := range entries {
			if !e.IsDir() {
				r.Scripts = append(r.Scripts, Script{
					Name:     strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())),
					Path:     filepath.Join(scriptsDir, e.Name()),
					Language: strings.TrimPrefix(filepath.Ext(e.Name()), "."),
				})
			}
		}
	}

	// Load references
	if refsDir := filepath.Join(skillPath, "references"); dirExists(refsDir) {
		entries, _ := os.ReadDir(refsDir)
		for _, e := range entries {
			if !e.IsDir() {
				r.References = append(r.References, Reference{
					Name: strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())),
					Path: filepath.Join(refsDir, e.Name()),
				})
			}
		}
	}

	// Load assets
	if assetsDir := filepath.Join(skillPath, "assets"); dirExists(assetsDir) {
		entries, _ := os.ReadDir(assetsDir)
		for _, e := range entries {
			if !e.IsDir() {
				r.Assets = append(r.Assets, Asset{
					Name: e.Name(),
					Path: filepath.Join(assetsDir, e.Name()),
					Type: strings.TrimPrefix(filepath.Ext(e.Name()), "."),
				})
			}
		}
	}

	// Load root directory templates
	entries, _ := os.ReadDir(skillPath)
	for _, e := range entries {
		if !e.IsDir() && e.Name() != "SKILL.md" {
			ext := filepath.Ext(e.Name())
			if ext == ".md" || ext == ".txt" || ext == ".tmpl" {
				r.Templates = append(r.Templates, Template{
					Name: strings.TrimSuffix(e.Name(), ext),
					Path: filepath.Join(skillPath, e.Name()),
				})
			}
		}
	}

	return r, nil
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
