package skill

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/pipeit"
)

// Executor executes scripts
type Executor struct {
	timeout time.Duration
	env     []string
}

// ExecutorOption configures an Executor
type ExecutorOption func(*Executor)

// WithTimeout sets the execution timeout
func WithTimeout(d time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.timeout = d
	}
}

// WithEnv sets environment variables for script execution
func WithEnv(env []string) ExecutorOption {
	return func(e *Executor) {
		e.env = env
	}
}

// NewExecutor creates a new script executor
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Result is the result of a script execution
type Result struct {
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Error    error         `json:"error,omitempty"`
}

// Execute runs a script by name
func (e *Executor) Execute(ctx context.Context, skill *Skill, scriptName string, args ...string) (*Result, error) {
	if skill.Resources == nil {
		return nil, fmt.Errorf("skill has no resources")
	}

	// Find the script
	var script *Script
	for _, s := range skill.Resources.Scripts {
		if s.Name == scriptName {
			script = &s
			break
		}
	}
	if script == nil {
		return nil, fmt.Errorf("script not found: %s", scriptName)
	}

	return e.ExecutePathWithDir(ctx, script.Path, skill.Path, args...)
}

// ExecutePath runs a script at the given path
func (e *Executor) ExecutePath(ctx context.Context, scriptPath string, args ...string) (*Result, error) {
	return e.ExecutePathWithDir(ctx, scriptPath, "", args...)
}

// ExecutePathWithDir runs a script at the given path with a specific working directory
func (e *Executor) ExecutePathWithDir(ctx context.Context, scriptPath, workDir string, args ...string) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Determine the command based on file extension
	ext := strings.TrimPrefix(filepath.Ext(scriptPath), ".")
	var cmd *exec.Cmd

	scriptArgs := append([]string{scriptPath}, args...)

	switch ext {
	case "py", "python":
		cmd = exec.CommandContext(ctx, "python3", scriptArgs...)
	case "js":
		cmd = exec.CommandContext(ctx, "node", scriptArgs...)
	case "sh", "bash":
		cmd = exec.CommandContext(ctx, "bash", scriptArgs...)
	case "go":
		runArgs := append([]string{"run", scriptPath}, args...)
		cmd = exec.CommandContext(ctx, "go", runArgs...)
	case "rb":
		cmd = exec.CommandContext(ctx, "ruby", scriptArgs...)
	case "php":
		cmd = exec.CommandContext(ctx, "php", scriptArgs...)
	case "pl":
		cmd = exec.CommandContext(ctx, "perl", scriptArgs...)
	default:
		// Try to execute directly
		cmd = exec.CommandContext(ctx, scriptPath, args...)
	}

	// Set working directory
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Set environment
	if len(e.env) > 0 {
		cmd.Env = append(cmd.Env, e.env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err
	}

		return result, nil

	}

	

	// ExecuteInteractive runs a command interactively using pipeit (PTY support)

	func (e *Executor) ExecuteInteractive(ctx context.Context, command string, args ...string) (*Result, error) {

		var stdout, stderr bytes.Buffer

		

		cfg := pipe.Config{

			Command: command,

			Args:    args,

			Env:     e.env,

			OnOutput: func(data []byte) {

				stdout.Write(data)

			},

			OnError: func(data []byte) {

				stderr.Write(data)

			},

		}

	

		pm := pipe.NewWithConfig(cfg)

		

		start := time.Now()

		// Using StartWithPipes for non-PTY interactive-ish execution if needed, 

		// or StartWithPTY if we want full terminal emulation.

		// For most skills, StartWithPipes is safer unless a PTY is specifically requested.

		if err := pm.StartWithPipes(); err != nil {

			return nil, err

		}

	

		// Create a channel to wait for completion

		done := make(chan error, 1)

		go func() {

			done <- pm.Wait()

		}()

	

		var err error

		select {

		case <-ctx.Done():

			pm.Stop()

			err = ctx.Err()

		case waitErr := <-done:

			err = waitErr

		case <-time.After(e.timeout):

			pm.Stop()

			err = fmt.Errorf("execution timed out after %v", e.timeout)

		}

	

		duration := time.Since(start)

	

		result := &Result{

			Stdout:   stdout.String(),

			Stderr:   stderr.String(),

			Duration: duration,

		}

	

		if err != nil {

			result.Error = err

			// Exit code handling in pipeit might be different, 

			// but pm.Wait() usually returns an error if exit code != 0

		}

	

		return result, nil

	}

	

	// ExecuteShell runs a shell command
func (e *Executor) ExecuteShell(ctx context.Context, command string) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	if len(e.env) > 0 {
		cmd.Env = append(cmd.Env, e.env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err
	}

	return result, nil
}

// ListScripts returns all available scripts in a skill
func ListScripts(skill *Skill) []Script {
	if skill.Resources == nil {
		return nil
	}
	return skill.Resources.Scripts
}

// FindScript finds a script by name in a skill
func FindScript(skill *Skill, name string) *Script {
	if skill.Resources == nil {
		return nil
	}
	for _, s := range skill.Resources.Scripts {
		if s.Name == name {
			return &s
		}
	}
	return nil
}
