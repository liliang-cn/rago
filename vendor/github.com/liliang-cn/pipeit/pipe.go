// Package pipe provides a high-level API for capturing and interacting with
// process input and output across different platforms.
//
// It supports both pseudo-terminal (PTY) for interactive programs (like shells
// or REPLs) and standard pipes for non-interactive commands.
package pipe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// OutputHandler is a callback function type used to process output data
// received from the managed process's stdout or stderr.
type OutputHandler func([]byte)

// ProcessManager handles the lifecycle and IO of a system process.
// It manages the execution, provides methods for writing to stdin,
// and uses handlers to capture stdout and stderr.
type ProcessManager struct {
	cmd       *exec.Cmd
	pty       *os.File
	ctx       context.Context
	cancel    context.CancelFunc
	stdinPipe io.WriteCloser
	onOutput  OutputHandler
	onError   OutputHandler
	mu        sync.Mutex
	running   bool
}

// Config specifies the parameters for creating a new ProcessManager.
type Config struct {
	// Command is the name or path of the executable.
	Command string
	// Args is the list of arguments for the command.
	Args []string
	// Env specifies the environment variables for the process.
	// If nil, the current process environment is used.
	Env []string
	// OnOutput is the handler for stdout data.
	OnOutput OutputHandler
	// OnError is the handler for stderr data.
	OnError OutputHandler
}

// New creates a new ProcessManager for the given command and arguments.
// It uses default environment variables and provides no initial handlers.
func New(command string, args ...string) *ProcessManager {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = os.Environ()

	return &ProcessManager{
		cmd:    cmd,
		ctx:    ctx,
		cancel: cancel,
	}
}

// NewWithConfig creates a ProcessManager using the provided Config.
func NewWithConfig(cfg Config) *ProcessManager {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)

	if len(cfg.Env) > 0 {
		cmd.Env = append(os.Environ(), cfg.Env...)
	} else {
		cmd.Env = os.Environ()
	}

	return &ProcessManager{
		cmd:      cmd,
		ctx:      ctx,
		cancel:   cancel,
		onOutput: cfg.OnOutput,
		onError:  cfg.OnError,
	}
}

// SetOutputHandler sets or updates the callback for stdout data.
func (p *ProcessManager) SetOutputHandler(handler OutputHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onOutput = handler
}

// SetErrorHandler sets or updates the callback for stderr data.
func (p *ProcessManager) SetErrorHandler(handler OutputHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onError = handler
}

// StartWithPTY starts the process attached to a pseudo-terminal (PTY).
// This is required for interactive programs like shells, Python REPL, etc.
func (p *ProcessManager) StartWithPTY() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	p.pty, err = pty.Start(p.cmd)
	if err != nil {
		return fmt.Errorf("start PTY failed: %w", err)
	}
	p.running = true

	go p.readOutput()
	return nil
}

// StartWithPipes starts the process using standard OS pipes for stdin/stdout/stderr.
// This is suitable for non-interactive batch commands.
func (p *ProcessManager) StartWithPipes() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}
	p.stdinPipe = stdin

	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := p.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("start command: %w", err)
	}
	p.running = true

	go p.readFromReader(stdout, p.onOutput)
	go p.readFromReader(stderr, p.onError)
	return nil
}

// readOutput is an internal goroutine that reads from the PTY.
func (p *ProcessManager) readOutput() {
	buf := make([]byte, 4096)
	for {
		n, err := p.pty.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			p.mu.Lock()
			handler := p.onOutput
			p.mu.Unlock()

			if handler != nil {
				handler(data)
			}
		}
		if err != nil {
			p.mu.Lock()
			handler := p.onError
			p.mu.Unlock()

			// Check for EIO on Linux which indicates PTY closed
			if err != io.EOF && !errors.Is(err, syscall.EIO) && handler != nil {
				handler([]byte(fmt.Sprintf("\n[Read Error]: %v\n", err)))
			}
			break
		}
	}
}

// readFromReader is an internal helper to stream data from a reader to a handler.
func (p *ProcessManager) readFromReader(r io.Reader, handler OutputHandler) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			if handler != nil {
				handler(data)
			}
		}
		if err != nil {
			if err != io.EOF && handler != nil {
				handler([]byte(fmt.Sprintf("[Read Error]: %v\n", err)))
			}
			break
		}
	}
}

// Write sends raw bytes to the process's standard input.
func (p *ProcessManager) Write(data []byte) (n int, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.pty != nil {
		return p.pty.Write(data)
	}
	if p.stdinPipe != nil {
		return p.stdinPipe.Write(data)
	}
	return 0, fmt.Errorf("no input pipe available")
}

// WriteString sends a string to the process's standard input.
func (p *ProcessManager) WriteString(s string) error {
	_, err := p.Write([]byte(s))
	return err
}

// Writef formats a string and sends it to the process's standard input.
func (p *ProcessManager) Writef(format string, args ...any) error {
	_, err := p.Write([]byte(fmt.Sprintf(format, args...)))
	return err
}

// Writeln sends a string followed by a newline to the process's standard input.
func (p *ProcessManager) Writeln(s string) error {
	return p.WriteString(s + "\n")
}

// IsRunning returns true if the process is currently active.
func (p *ProcessManager) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// Stop terminates the process and closes associated pipes or PTY.
func (p *ProcessManager) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.cancel()
	p.running = false

	if p.pty != nil {
		p.pty.Close()
	}
	if p.stdinPipe != nil {
		p.stdinPipe.Close()
	}

	if p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

// Wait blocks until the managed process exits.
func (p *ProcessManager) Wait() error {
	return p.cmd.Wait()
}

// Pid returns the process ID of the managed process, or -1 if not started.
func (p *ProcessManager) Pid() int {
	if p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return -1
}

// Session returns the underlying PTY file, if one is in use.
// This allows for advanced terminal operations like setting window size.
func (p *ProcessManager) Session() *os.File {
	return p.pty
}
