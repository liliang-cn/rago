package agent

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// SystemContext 系统上下文信息
type SystemContext struct {
	Date       string
	Time       string
	Timezone   string
	OS         string
	Arch       string
	Hostname   string
	WorkingDir string
	HomeDir    string
	User       string
	GoVersion  string
	EnvInfo    map[string]string // selected env vars
	Memories   []MemorySummary   // list of available memory entities
	HasMemory  bool              // memory system is enabled
}

// MemorySummary represents a summary of a persistent memory file
type MemorySummary struct {
	ID      string
	Type    string
	Summary string
}

// buildSystemContext 收集系统上下文信息
func (s *Service) buildSystemContext() *SystemContext {
	now := time.Now()
	hostname, _ := os.Hostname()
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	if user == "" {
		user = os.Getenv("LOGNAME")
	}

	// Collect selected useful env vars
	envInfo := make(map[string]string)
	usefulEnvKeys := []string{"SHELL", "PATH", "LANG", "TERM", "EDITOR"}
	for _, key := range usefulEnvKeys {
		if val := os.Getenv(key); val != "" {
			// Truncate long values like PATH
			if len(val) > 100 {
				val = val[:97] + "..."
			}
			envInfo[key] = val
		}
	}

	ctx := &SystemContext{
		Date:       now.Format("2006-01-02"),
		Time:       now.Format("15:04:05"),
		Timezone:   now.Location().String(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Hostname:   hostname,
		WorkingDir: getCwd(),
		HomeDir:    getHomeDir(),
		User:       user,
		GoVersion:  runtime.Version(),
		EnvInfo:    envInfo,
	}

	// 注入记忆地图
	if s.memoryService != nil {
		ctx.HasMemory = true
		// 限制数量，防止上下文溢出
		mems, total, err := s.memoryService.List(context.Background(), 20, 0)
		if err == nil && total > 0 {
			for _, m := range mems {
				// 仅展示有内容的记忆摘要
				if m.Content == "" {
					continue
				}
				// 展示所有类型的长效记忆摘要，以便 Agent 发现线索
				summary := m.Content
				if len(summary) > 80 {
					summary = summary[:77] + "..."
				}
				ctx.Memories = append(ctx.Memories, MemorySummary{
					ID:      m.ID,
					Type:    string(m.Type),
					Summary: summary,
				})
			}
		}
	}

	return ctx
}

// FormatForPrompt 格式化系统上下文为prompt字符串
func (c *SystemContext) FormatForPrompt() string {
	var sb strings.Builder

	sb.WriteString("## SYSTEM CHRONOMETER (CRITICAL)\n")
	fmt.Fprintf(&sb, "- Current Date: %s\n", c.Date)
	fmt.Fprintf(&sb, "- Current Time: %s (%s)\n", c.Time, c.Timezone)
	sb.WriteString("When accessing long-term memories, always interpret their 'updated_at' timestamps relative to the current time above. Recent information should generally override older conflicting data.\n\n")

	sb.WriteString("## System Environment\n")
	fmt.Fprintf(&sb, "- OS: %s/%s\n", c.OS, c.Arch)
	if c.Hostname != "" {
		fmt.Fprintf(&sb, "- Hostname: %s\n", c.Hostname)
	}
	fmt.Fprintf(&sb, "- Working Directory: %s\n", c.WorkingDir)
	if c.User != "" {
		fmt.Fprintf(&sb, "- User: %s\n", c.User)
	}
	if len(c.EnvInfo) > 0 {
		sb.WriteString("Environment: ")
		first := true
		for k, v := range c.EnvInfo {
			if !first {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%s=%s", k, v)
			first = false
		}
		sb.WriteString("\n")
	}

	// Memory tools usage guide (shown when memory is enabled)
	if c.HasMemory {
		sb.WriteString("\n## Memory System (Enabled)\n")
		sb.WriteString("You have access to a persistent memory system. Use these tools:\n\n")
		sb.WriteString("**memory_save(content, type)**: Save important information\n")
		sb.WriteString("- Use when: User shares preferences, facts about themselves, or important context\n")
		sb.WriteString("- Types: fact, preference, skill, pattern, context\n")
		sb.WriteString("- Example: memory_save(\"User prefers dark mode\", \"preference\")\n\n")
		sb.WriteString("**memory_recall(query)**: Search and retrieve memories\n")
		sb.WriteString("- Use when: You need to recall user preferences or past context\n")
		sb.WriteString("- Example: memory_recall(\"user programming language preference\")\n\n")
		sb.WriteString("**memory_update(id, content)**: Update existing memory\n")
		sb.WriteString("- Use when: User corrects or adds to previously saved information\n")
		sb.WriteString("- Example: memory_update(\"mem_123\", \"Updated information\")\n\n")
		sb.WriteString("**memory_delete(id)**: Remove outdated memory\n")
		sb.WriteString("- Use when: User asks to forget something or information is obsolete\n\n")
	}

	if len(c.Memories) > 0 {
		sb.WriteString("### Available Persistent Memories (Memory Map)\n")
		sb.WriteString("The following facts and entities are stored in your long-term memory:\n")
		for _, m := range c.Memories {
			sb.WriteString(fmt.Sprintf("- [%s] (%s): %s\n", m.ID, m.Type, m.Summary))
		}
	}

	return sb.String()
}

// FormatCompact 紧凑格式，适合嵌入到现有prompt中
func (c *SystemContext) FormatCompact() string {
	return fmt.Sprintf("[Context: %s %s, %s/%s, dir=%s]",
		c.Date, c.Time, c.OS, c.Arch, shortPath(c.WorkingDir))
}

func getCwd() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "unknown"
}

func getHomeDir() string {
	// os.UserHomeDir() is available since Go 1.12
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	// Fallback
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home
	}
	return "unknown"
}

func shortPath(path string) string {
	// Shorten home directory to ~
	home := getHomeDir()
	if home != "unknown" && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	// If path is too long, truncate
	if len(path) > 30 {
		return "..." + path[len(path)-27:]
	}
	return path
}
