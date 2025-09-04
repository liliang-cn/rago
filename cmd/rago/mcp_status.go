package rago

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// MCPEnvironmentStatus represents the status of MCP runtime environments
type MCPEnvironmentStatus struct {
	Python struct {
		Available bool
		Version   string
		UV        bool
		UVX       bool
		PipX      bool
	}
	NodeJS struct {
		Available bool
		Version   string
		NPM       bool
		NPX       bool
		Yarn      bool
	}
	Servers map[string]ServerToolStatus
}

// ServerToolStatus represents the availability of a server's tools
type ServerToolStatus struct {
	Name        string
	Runtime     string // "python" or "nodejs"
	Available   bool
	InstallCmd  string
	Description string
}

// CheckMCPEnvironment checks for Python and Node.js tool availability
func CheckMCPEnvironment() *MCPEnvironmentStatus {
	status := &MCPEnvironmentStatus{
		Servers: make(map[string]ServerToolStatus),
	}

	// Check Python environment
	checkPythonEnvironment(status)

	// Check Node.js environment
	checkNodeEnvironment(status)

	// Define default servers and their requirements
	defineDefaultServers(status)

	return status
}

func checkPythonEnvironment(status *MCPEnvironmentStatus) {
	// Check Python
	if cmd, err := exec.LookPath("python3"); err == nil {
		status.Python.Available = true
		if output, err := exec.Command(cmd, "--version").Output(); err == nil {
			status.Python.Version = strings.TrimSpace(string(output))
		}
	} else if cmd, err := exec.LookPath("python"); err == nil {
		status.Python.Available = true
		if output, err := exec.Command(cmd, "--version").Output(); err == nil {
			status.Python.Version = strings.TrimSpace(string(output))
		}
	}

	// Check UV (Rust-based Python package manager)
	if _, err := exec.LookPath("uv"); err == nil {
		status.Python.UV = true
	}

	// Check UVX (UV's tool runner)
	if _, err := exec.LookPath("uvx"); err == nil {
		status.Python.UVX = true
	}

	// Check pipx (Python application installer)
	if _, err := exec.LookPath("pipx"); err == nil {
		status.Python.PipX = true
	}
}

func checkNodeEnvironment(status *MCPEnvironmentStatus) {
	// Check Node.js
	if cmd, err := exec.LookPath("node"); err == nil {
		status.NodeJS.Available = true
		if output, err := exec.Command(cmd, "--version").Output(); err == nil {
			status.NodeJS.Version = strings.TrimSpace(string(output))
		}
	}

	// Check npm
	if _, err := exec.LookPath("npm"); err == nil {
		status.NodeJS.NPM = true
	}

	// Check npx
	if _, err := exec.LookPath("npx"); err == nil {
		status.NodeJS.NPX = true
	}

	// Check yarn
	if _, err := exec.LookPath("yarn"); err == nil {
		status.NodeJS.Yarn = true
	}
}

func defineDefaultServers(status *MCPEnvironmentStatus) {
	// Define Python-based servers
	pythonServers := map[string]ServerToolStatus{
		"fetch-python": {
			Name:        "fetch (Python)",
			Runtime:     "python",
			Available:   status.Python.UVX,
			InstallCmd:  "uvx install mcp-server-fetch",
			Description: "HTTP/HTTPS fetch operations (Python implementation)",
		},
		"git": {
			Name:        "git",
			Runtime:     "python",
			Available:   status.Python.UVX,
			InstallCmd:  "uvx install mcp-server-git",
			Description: "Git repository operations",
		},
		"github": {
			Name:        "github",
			Runtime:     "python",
			Available:   status.Python.UVX,
			InstallCmd:  "uvx install mcp-server-github",
			Description: "GitHub API integration",
		},
	}

	// Define Node.js-based servers
	nodeServers := map[string]ServerToolStatus{
		"filesystem": {
			Name:        "filesystem",
			Runtime:     "nodejs",
			Available:   status.NodeJS.NPX,
			InstallCmd:  "npm install -g @modelcontextprotocol/server-filesystem",
			Description: "File system operations with sandboxing",
		},
		"fetch-nodejs": {
			Name:        "fetch (Node.js)",
			Runtime:     "nodejs",
			Available:   status.NodeJS.NPX,
			InstallCmd:  "npm install -g @modelcontextprotocol/server-fetch",
			Description: "HTTP/HTTPS fetch operations (Node.js implementation)",
		},
		"memory": {
			Name:        "memory",
			Runtime:     "nodejs",
			Available:   status.NodeJS.NPX,
			InstallCmd:  "npm install -g @modelcontextprotocol/server-memory",
			Description: "In-memory key-value storage",
		},
		"sequential-thinking": {
			Name:        "sequential-thinking",
			Runtime:     "nodejs",
			Available:   status.NodeJS.NPX,
			InstallCmd:  "npm install -g @modelcontextprotocol/server-sequential-thinking",
			Description: "Enhanced reasoning through step-by-step decomposition",
		},
		"time": {
			Name:        "time",
			Runtime:     "nodejs",
			Available:   status.NodeJS.NPX,
			InstallCmd:  "npm install -g @modelcontextprotocol/server-time",
			Description: "Time and date utilities",
		},
	}

	// Merge servers
	for k, v := range pythonServers {
		status.Servers[k] = v
	}
	for k, v := range nodeServers {
		status.Servers[k] = v
	}
}

// PrintMCPEnvironmentStatus prints a formatted status report
func PrintMCPEnvironmentStatus(status *MCPEnvironmentStatus) {
	fmt.Println("\n🔍 MCP Runtime Environment Check")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Python status
	fmt.Println("\n🐍 Python Environment:")
	if status.Python.Available {
		fmt.Printf("   ✅ Python: %s\n", status.Python.Version)
	} else {
		fmt.Println("   ❌ Python: Not found")
	}

	printToolStatus("   ", "uv", status.Python.UV, "Fast Python package manager")
	printToolStatus("   ", "uvx", status.Python.UVX, "Run Python tools without installation")
	printToolStatus("   ", "pipx", status.Python.PipX, "Install Python applications")

	// Node.js status
	fmt.Println("\n📦 Node.js Environment:")
	if status.NodeJS.Available {
		fmt.Printf("   ✅ Node.js: %s\n", status.NodeJS.Version)
	} else {
		fmt.Println("   ❌ Node.js: Not found")
	}

	printToolStatus("   ", "npm", status.NodeJS.NPM, "Node package manager")
	printToolStatus("   ", "npx", status.NodeJS.NPX, "Run Node packages without installation")
	printToolStatus("   ", "yarn", status.NodeJS.Yarn, "Alternative package manager")

	// Server availability
	fmt.Println("\n🚀 MCP Server Availability:")

	pythonCount := 0
	nodeCount := 0

	for _, server := range status.Servers {
		if server.Runtime == "python" && server.Available {
			pythonCount++
		}
		if server.Runtime == "nodejs" && server.Available {
			nodeCount++
		}
	}

	fmt.Printf("   Python servers: %d available\n", pythonCount)
	fmt.Printf("   Node.js servers: %d available\n", nodeCount)

	// Recommendations
	fmt.Println("\n💡 Recommendations:")

	if !status.Python.UVX && !status.NodeJS.NPX {
		fmt.Println("   ⚠️  No zero-install tools available!")
		fmt.Println("   Install either:")
		fmt.Println("     - Python: curl -LsSf https://astral.sh/uv/install.sh | sh")
		fmt.Println("     - Node.js: https://nodejs.org/ (includes npx)")
	} else {
		if !status.Python.UVX {
			fmt.Println("   📌 For Python servers, install uv:")
			fmt.Println("     curl -LsSf https://astral.sh/uv/install.sh | sh")
		}
		if !status.NodeJS.NPX {
			fmt.Println("   📌 For Node.js servers, install Node.js:")
			fmt.Println("     https://nodejs.org/ (includes npx)")
		}
	}

	// Platform-specific notes
	if runtime.GOOS == "darwin" {
		fmt.Println("\n   macOS users can also use Homebrew:")
		if !status.Python.UV {
			fmt.Println("     brew install uv")
		}
		if !status.NodeJS.Available {
			fmt.Println("     brew install node")
		}
	}
}

func printToolStatus(prefix, name string, available bool, description string) {
	status := "❌"
	if available {
		status = "✅"
	}
	fmt.Printf("%s%s %s: %s\n", prefix, status, name, description)
}

// GetRecommendedInstallCommands returns installation commands for missing tools
func GetRecommendedInstallCommands(status *MCPEnvironmentStatus) []string {
	var commands []string

	if !status.Python.UV && !status.Python.UVX {
		commands = append(commands, "curl -LsSf https://astral.sh/uv/install.sh | sh")
	}

	if !status.NodeJS.Available {
		switch runtime.GOOS {
		case "darwin":
			commands = append(commands, "brew install node")
		case "linux":
			commands = append(commands, "curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash - && sudo apt-get install -y nodejs")
		default:
			commands = append(commands, "Download Node.js from https://nodejs.org/")
		}
	}

	return commands
}
