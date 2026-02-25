# pipeit

`pipeit` is a Go library designed to simplify cross-platform process IO capture and interaction. It provides a high-level API to spawn processes, capture their standard output and standard error, and write to their standard input.

Key features include:
*   **PTY Support**: seamless interaction with interactive CLI tools (like shells, REPLs) using pseudo-terminals.
*   **Standard Pipes**: Support for non-interactive commands using standard OS pipes.
*   **Bidirectional Communication**: Write to a process's stdin and read from its stdout/stderr in real-time.
*   **Simple API**: Easy-to-use methods for starting, stopping, and managing processes.

## Installation

```bash
go get github.com/liliang-cn/pipeit
```

*(Note: If you are developing locally, ensure your `go.mod` is set up correctly to resolve the package).*

## Usage

Here is a simple example of how to use `pipeit` to run a bash command and capture its output.

```go
package main

import (
	"fmt"
	"time"

	"github.com/liliang-cn/pipeit"
)

func main() {
	// 1. Create a new process manager for 'bash'
	pm := pipe.New("bash", "--norc")

	// 2. Set a handler to process output (stdout/stderr)
	pm.SetOutputHandler(func(data []byte) {
		fmt.Printf("[Output]: %s", string(data))
	})

	// 3. Start the process with a PTY (useful for interactive commands)
	if err := pm.StartWithPTY(); err != nil {
		panic(err)
	}
	defer pm.Stop()

	// 4. Send commands to the process
	pm.Writeln("echo 'Hello from pipeit!'")
	time.Sleep(100 * time.Millisecond) // Give it a moment to process

	// 5. Exit the process
	pm.Writeln("exit")
	pm.Wait()
}
```

### Advanced Configuration

You can use `NewWithConfig` for more control, such as setting environment variables:

```go
config := pipe.Config{
    Command: "python3",
    Args:    []string{"-q"},
    Env:     []string{"PYTHONUNBUFFERED=1"},
    OnOutput: func(data []byte) {
        fmt.Print(string(data))
    },
}

pm := pipe.NewWithConfig(config)
pm.StartWithPTY()
// ... interact with python ...
```

## Examples

Check the `examples/` directory for more comprehensive demos:
*   `cli/main.go`: A CLI tool demonstrating bash, zsh, and python interactions.
*   `lib/main.go`: A collection of library usage patterns (streaming, error handling, etc.).

To run the CLI demo:

```bash
# Run bash example
EXAMPLE=bash go run examples/cli/main.go

# Run python example
EXAMPLE=python go run examples/cli/main.go
```
