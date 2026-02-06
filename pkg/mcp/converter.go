package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liliang-cn/skills-go/mcp"
	"github.com/liliang-cn/skills-go/skill"
)

// ConverterConfig configures the MCP to Skill converter
type ConverterConfig struct {
	OutputDir    string // Directory to write converted skills
	UseLLM       bool   // Use LLM for enhanced conversion
	LLMAPIKey    string // API key for LLM
	LLMBaseURL   string // Base URL for LLM API
	LLMModel     string // Model to use (default: gpt-4o)
}

// DefaultConverterConfig returns default converter config
func DefaultConverterConfig() *ConverterConfig {
	homeDir, _ := os.UserHomeDir()
	return &ConverterConfig{
		OutputDir:  filepath.Join(homeDir, ".rago", "skills"),
		UseLLM:     false,
		LLMModel:   "gpt-4o",
	}
}

// Converter converts MCP servers to Skills
type Converter struct {
	config  *ConverterConfig
	mcpSvc  *Service
	conv    *mcp.Converter
}

// NewConverter creates a new MCP to Skill converter
func NewConverter(cfg *ConverterConfig, mcpSvc *Service) (*Converter, error) {
	if cfg == nil {
		cfg = DefaultConverterConfig()
	}

	c := &Converter{
		config: cfg,
		mcpSvc: mcpSvc,
	}

	// Initialize skills-go converter with optional LLM
	if cfg.UseLLM && cfg.LLMAPIKey != "" {
		opts := []mcp.ConverterOption{
			mcp.WithLLM(cfg.LLMAPIKey, cfg.LLMBaseURL),
		}
		if cfg.LLMModel != "" {
			opts = append(opts, mcp.WithLLMModel(cfg.LLMModel))
		}
		c.conv = mcp.NewConverter(opts...)
	} else {
		c.conv = mcp.NewConverter()
	}

	return c, nil
}

// ConvertServer converts a single MCP server to a Skill
func (c *Converter) ConvertServer(ctx context.Context, serverName string) (*skill.Skill, error) {
	// Get server config from MCP service
	servers := c.mcpSvc.mcpConfig.GetLoadedServers()
	var serverConfig *ServerConfig
	for _, s := range servers {
		if s.Name == serverName {
			serverConfig = &s
			break
		}
	}
	if serverConfig == nil {
		return nil, fmt.Errorf("server not found: %s", serverName)
	}

	// Build converter config
	svrCfg := &mcp.ServerConfig{
		Name: serverName,
		Include: mcp.IncludeConfig{
			Tools:     true,
			Resources: true,
			Prompts:   true,
		},
	}

	// Sanitize description - remove colons or escape them
	if serverConfig.Description != "" {
		svrCfg.Description = sanitizeDescription(serverConfig.Description)
	}

	// Map command
	if len(serverConfig.Command) > 0 {
		svrCfg.Command = serverConfig.Command
		if len(serverConfig.Args) > 0 {
			svrCfg.Command = append(svrCfg.Command, serverConfig.Args...)
		}
	} else if serverConfig.Type == ServerTypeStdio {
		// Try to use command from combined Command/Args
		cmdStr := strings.Join(serverConfig.Command, " ")
		if cmdStr != "" {
			svrCfg.Command = []string{cmdStr}
		}
	}

	// Set URL for HTTP servers
	if serverConfig.Type == ServerTypeHTTP && serverConfig.URL != "" {
		svrCfg.URL = serverConfig.URL
	}

	// Convert
	var sk *skill.Skill
	var err error

	if c.config.UseLLM {
		sk, err = c.conv.ConvertWithLLM(ctx, svrCfg, c.config.OutputDir)
	} else {
		sk, err = c.conv.Convert(ctx, svrCfg, c.config.OutputDir)
	}

	if err != nil {
		return nil, fmt.Errorf("conversion failed: %w", err)
	}

	// Post-process: fix YAML frontmatter
	if sk != nil && sk.Path != "" {
		if err := fixSkillFrontmatter(sk.Path); err != nil {
			// Log but don't fail
			fmt.Printf("⚠️  Warning: Failed to fix skill frontmatter: %v\n", err)
		}
	}

	return sk, nil
}

// sanitizeDescription removes or replaces characters that cause YAML parsing issues
func sanitizeDescription(desc string) string {
	// Replace colons with dashes to avoid YAML key-value interpretation
	desc = strings.ReplaceAll(desc, ": ", " - ")
	desc = strings.ReplaceAll(desc, ":", "-")
	return desc
}

// fixSkillFrontmatter quotes the description value in the YAML frontmatter
func fixSkillFrontmatter(skillPath string) error {
	skillFile := filepath.Join(skillPath, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// Fix: quote the description value if it contains special characters
	// Find "description: <value>" and quote the value
	frontmatterEnd := strings.Index(contentStr, "---")
	if frontmatterEnd == -1 {
		return nil
	}

	// Get the frontmatter section (between first and second ---)
	frontmatterStart := strings.Index(contentStr, "---")
	if frontmatterStart == -1 {
		return nil
	}

	secondDash := strings.Index(contentStr[frontmatterStart+3:], "---")
	if secondDash == -1 {
		return nil
	}
	secondDash += frontmatterStart + 3

	frontmatter := contentStr[:secondDash]

	// Fix description line
	lines := strings.Split(frontmatter, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "description:") && !strings.Contains(line, "description: \"") {
			// Extract the value
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				if value != "" {
					lines[i] = "description: \"" + value + "\""
				}
			}
		}
	}

	newFrontmatter := strings.Join(lines, "\n")
	newContent := newFrontmatter + contentStr[secondDash:]

	return os.WriteFile(skillFile, []byte(newContent), 0644)
}

// ConvertAllServers converts all configured MCP servers to Skills
func (c *Converter) ConvertAllServers(ctx context.Context) ([]*skill.Skill, error) {
	servers := c.mcpSvc.mcpConfig.GetLoadedServers()

	var skills []*skill.Skill
	var errs []string

	for _, s := range servers {
		sk, err := c.ConvertServer(ctx, s.Name)
		if err != nil {
			errs = append(errs, fmt.Sprintf("  - %s: %v", s.Name, err))
			continue
		}
		skills = append(skills, sk)
	}

	if len(errs) > 0 {
		return skills, fmt.Errorf("some conversions failed:\n%s", strings.Join(errs, "\n"))
	}

	return skills, nil
}

// DiscoverServer discovers capabilities of an MCP server without converting
func (c *Converter) DiscoverServer(ctx context.Context, serverName string) (*mcp.ServerCapabilities, error) {
	// Get server config from MCP service
	servers := c.mcpSvc.mcpConfig.GetLoadedServers()
	var serverConfig *ServerConfig
	for _, s := range servers {
		if s.Name == serverName {
			serverConfig = &s
			break
		}
	}
	if serverConfig == nil {
		return nil, fmt.Errorf("server not found: %s", serverName)
	}

	// Build converter config
	svrCfg := &mcp.ServerConfig{
		Name: serverName,
		Include: mcp.IncludeConfig{
			Tools:     true,
			Resources: true,
			Prompts:   true,
		},
	}

	if len(serverConfig.Command) > 0 {
		svrCfg.Command = serverConfig.Command
		if len(serverConfig.Args) > 0 {
			svrCfg.Command = append(svrCfg.Command, serverConfig.Args...)
		}
	}

	if serverConfig.Type == ServerTypeHTTP && serverConfig.URL != "" {
		svrCfg.URL = serverConfig.URL
	}

	return c.conv.Discover(ctx, svrCfg)
}
