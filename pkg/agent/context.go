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
	IsPTC      bool              // PTC mode — skip verbose memory docs (tools listed separately)
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
		IsPTC:      s.isPTCEnabled(),
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

	fmt.Fprintf(&sb, "Date/Time: %s %s (%s) | OS: %s/%s | Dir: %s | User: %s\n",
		c.Date, c.Time, c.Timezone, c.OS, c.Arch, c.WorkingDir, c.User)

	if len(c.EnvInfo) > 0 {
		parts := make([]string, 0, len(c.EnvInfo))
		for k, v := range c.EnvInfo {
			if k == "PATH" {
				continue // PATH is too long and not useful for the LLM
			}
			parts = append(parts, k+"="+v)
		}
		if len(parts) > 0 {
			sb.WriteString("Env: " + strings.Join(parts, ", ") + "\n")
		}
	}

	// Memory tools — show full API docs only in non-PTC mode.
	// In PTC mode, memory tools are already listed in the callTool() tool list.
	if c.HasMemory {
		if c.IsPTC {
			sb.WriteString("Memory: enabled (use memory_save/recall/update/delete via callTool)\n")
		} else {
			sb.WriteString("\n## Memory System\n")
			sb.WriteString("memory_save(content, type) — types: fact|preference|skill|pattern|context\n")
			sb.WriteString("memory_recall(query) — search past context\n")
			sb.WriteString("memory_update(id, content) — update a memory entry\n")
			sb.WriteString("memory_delete(id) — remove a memory entry\n")
		}
	}

	if len(c.Memories) > 0 {
		sb.WriteString("\n## Stored Memories\n")
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
