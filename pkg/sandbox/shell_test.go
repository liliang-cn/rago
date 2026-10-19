package sandbox

import (
	"context"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestWindowsRunsACommandLineInPowerShell(t *testing.T) {
	name, args := shellCommandFor("windows", "Get-Date")
	if name != "powershell.exe" {
		t.Fatalf("shell = %q, want powershell.exe", name)
	}
	if !slices.Contains(args, "-Command") || !strings.HasSuffix(args[len(args)-1], "Get-Date") {
		t.Fatalf("args = %q, want -Command ending in the script", args)
	}
	if !strings.Contains(args[len(args)-1], "UTF8") {
		t.Fatalf("script %q does not switch the output to UTF-8", args[len(args)-1])
	}

	name, args = shellCommandFor("linux", "date")
	if name != "sh" || !slices.Equal(args, []string{"-c", "date"}) {
		t.Fatalf("linux: %q %q, want sh -c date", name, args)
	}
}

func TestADockerSandboxIsLinuxWhateverTheHost(t *testing.T) {
	var docker Sandbox = &DockerSandbox{}
	if name, _ := ShellCommand(docker, "ls"); name != "sh" {
		t.Fatalf("docker shell = %q, want sh", name)
	}
	if got := ShellName(docker); got != "sh" {
		t.Fatalf("docker shell name = %q, want sh", got)
	}

	want := "sh"
	if runtime.GOOS == "windows" {
		want = "PowerShell"
	}
	if got := ShellName(&LocalSandbox{}); got != want {
		t.Fatalf("local shell name = %q, want %q", got, want)
	}
}

func TestWindowsHasNoShellSessions(t *testing.T) {
	if hasSessionsOn("windows") {
		t.Fatal("Windows claims shell sessions, but it has no PTY")
	}
	if !hasSessionsOn("linux") || !hasSessionsOn("darwin") {
		t.Fatal("Unix lost its shell sessions")
	}
}

func TestOnWindowsACommandRunsInPowerShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("needs Windows")
	}
	sb := newTestLocal(t)
	name, args := ShellCommand(sb, "Write-Output '你好'; exit 3")
	res, err := sb.Exec(context.Background(), ExecRequest{Command: name, Args: args, Timeout: time.Minute})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !strings.Contains(res.Stdout, "你好") || res.ExitCode != 3 {
		t.Fatalf("stdout %q exit %d, want 你好 and 3 (stderr %q)", res.Stdout, res.ExitCode, res.Stderr)
	}
}
