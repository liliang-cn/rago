package agent

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// SystemContext 系统上下文信息
type SystemContext struct {
	Date         string
	Time         string
	Timezone     string
	OS           string
	Arch         string
	Hostname     string
	WorkingDir   string
	HomeDir      string
	User         string
	GoVersion    string
	EnvInfo      map[string]string // selected env vars
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

	return &SystemContext{
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
}

// FormatForPrompt 格式化系统上下文为prompt字符串
func (c *SystemContext) FormatForPrompt() string {
	var sb strings.Builder

	sb.WriteString("--- System Context ---\n")
	fmt.Fprintf(&sb, "Date: %s\n", c.Date)
	fmt.Fprintf(&sb, "Time: %s\n", c.Time)
	if c.Timezone != "" && c.Timezone != "UTC" {
		fmt.Fprintf(&sb, "Timezone: %s\n", c.Timezone)
	}
	fmt.Fprintf(&sb, "OS: %s/%s\n", c.OS, c.Arch)
	if c.Hostname != "" {
		fmt.Fprintf(&sb, "Hostname: %s\n", c.Hostname)
	}
	fmt.Fprintf(&sb, "Working Directory: %s\n", c.WorkingDir)
	if c.User != "" {
		fmt.Fprintf(&sb, "User: %s\n", c.User)
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
