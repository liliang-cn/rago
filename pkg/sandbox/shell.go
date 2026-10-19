package sandbox

import "runtime"

// Windows has no sh. Its shell that is always there is Windows PowerShell,
// which writes pipes in the console's code page unless told otherwise, so
// every script it runs first switches its output to UTF-8: Chinese file names
// and messages would otherwise come back as mojibake.
const (
	windowsShell   = "powershell.exe"
	windowsUTF8Out = "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; "
)

// HostShellCommand is how one command line runs on this machine: sh -c on
// Unix, PowerShell -Command on Windows.
func HostShellCommand(script string) (string, []string) {
	return shellCommandFor(runtime.GOOS, script)
}

// ShellCommand is how one command line runs in sb. A LocalSandbox is this
// machine and runs its shell; any other sandbox (a Docker container) is
// Linux whatever the host is, and runs sh.
func ShellCommand(sb Sandbox, script string) (string, []string) {
	return shellCommandFor(sandboxOS(sb), script)
}

// ShellName names the shell ShellCommand runs in sb, for a tool's
// description: the model writes PowerShell or sh accordingly.
func ShellName(sb Sandbox) string {
	if sandboxOS(sb) == "windows" {
		return "PowerShell"
	}
	return "sh"
}

func sandboxOS(sb Sandbox) string {
	if _, local := sb.(*LocalSandbox); local {
		return runtime.GOOS
	}
	return "linux"
}

func shellCommandFor(goos, script string) (string, []string) {
	if goos == "windows" {
		return windowsShell, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", windowsUTF8Out + script}
	}
	return "sh", []string{"-c", script}
}

// HasSessions says whether Shell can start a session here: it needs a PTY,
// and there is none on Windows.
func HasSessions() bool {
	return hasSessionsOn(runtime.GOOS)
}

func hasSessionsOn(goos string) bool {
	return goos != "windows"
}
