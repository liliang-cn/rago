package gateway

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/liliang-cn/rago/v2/pkg/agent"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/memory"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/store"
	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	verbose bool
)

// Styles
var styles = struct {
	title    lipgloss.Style
	header   lipgloss.Style
	prompt   lipgloss.Style
	response lipgloss.Style
	success  lipgloss.Style
	error    lipgloss.Style
	warning  lipgloss.Style
	dim      lipgloss.Style
	muted    lipgloss.Style
}{
	title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
	header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("38")),
	prompt:   lipgloss.NewStyle().Foreground(lipgloss.Color("99")),
	response: lipgloss.NewStyle().Foreground(lipgloss.Color("43")),
	success:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	error:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
	warning:  lipgloss.NewStyle().Foreground(lipgloss.Color("226")),
	dim:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	muted:    lipgloss.NewStyle().Faint(true),
}

// GatewayCmd represents the gateway command
var GatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Run an interactive AI gateway with async task processing",
	Long: `Gateway mode runs an async AI reception desk.

The LLM worker runs in a goroutine, allowing you to:
  - Send commands at any time (even while agent is running)
  - Cancel running tasks with /cancel
  - Check status with /status
  - Exit with /quit or Ctrl+C`,
	RunE: runGateway,
}

func init() {
	GatewayCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
}

func SetSharedVariables(c *config.Config, v bool) {
	cfg = c
	verbose = v
}

// ========================================
// Request/Response System
// ========================================

type Request struct {
	ID      string
	Query   string
	Ctx     context.Context
	Cancel  context.CancelFunc
	ReplyCh chan *Response
}

type Response struct {
	Content string
	Error   error
	Done    bool
}

// ========================================
// Gateway Service
// ========================================

type Gateway struct {
	agentService *agent.Service
	requestCh    chan *Request
	currentTask  atomic.Value // *Request
	running      atomic.Bool
	mu           sync.RWMutex
}

func NewGateway(ctx context.Context, cfg *config.Config) (*Gateway, error) {
	// Initialize services
	agentSvc, err := initAgentService(ctx, cfg)
	if err != nil {
		return nil, err
	}

	gw := &Gateway{
		agentService: agentSvc,
		requestCh:    make(chan *Request, 10),
	}
	gw.running.Store(true)

	// Start worker goroutine
	go gw.worker(ctx)

	return gw, nil
}

func (gw *Gateway) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-gw.requestCh:
			if req == nil {
				return
			}

			// Process request
			gw.processRequest(req)

			// Clear current task
			gw.currentTask.Store((*Request)(nil))
		}
	}
}

func (gw *Gateway) processRequest(req *Request) {
	result, err := gw.agentService.Run(req.Ctx, req.Query)

	resp := &Response{}
	if err != nil {
		resp.Error = err
	} else if result.FinalResult != nil {
		if str, ok := result.FinalResult.(string); ok {
			resp.Content = str
		} else {
			resp.Content = fmt.Sprintf("%v", result.FinalResult)
		}
	} else {
		resp.Content = "Done"
	}
	resp.Done = true

	select {
	case req.ReplyCh <- resp:
	case <-req.Ctx.Done():
	}
}

func (gw *Gateway) Submit(ctx context.Context, query string) (*Request, error) {
	reqCtx, cancel := context.WithCancel(ctx)

	req := &Request{
		ID:      fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Query:   query,
		Ctx:     reqCtx,
		Cancel:  cancel,
		ReplyCh: make(chan *Response, 1),
	}

	// Cancel current task if any
	if current := gw.currentTask.Load(); current != nil {
		if currReq, ok := current.(*Request); ok && currReq != nil {
			currReq.Cancel()
			fmt.Println(styles.warning.Render("[Cancelling previous task...]"))
		}
	}

	gw.currentTask.Store(req)

	select {
	case gw.requestCh <- req:
		return req, nil
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	}
}

func (gw *Gateway) Cancel() bool {
	if current := gw.currentTask.Load(); current != nil {
		if currReq, ok := current.(*Request); ok && currReq != nil {
			currReq.Cancel()
			return true
		}
	}
	return false
}

func (gw *Gateway) IsProcessing() bool {
	return gw.currentTask.Load() != nil
}

// ========================================
// Interactive Loop
// ========================================

func (gw *Gateway) Run(ctx context.Context) error {
	fmt.Println(styles.header.Render("═══════════════════════════════════════"))
	fmt.Println(styles.title.Render("        RAGO GATEWAY"))
	fmt.Println(styles.header.Render("═══════════════════════════════════════"))
	fmt.Println()
	fmt.Println(styles.muted.Render("Worker running. Type your message or /help."))

	// Response listener
	go gw.responseListener(ctx)

	// Input loop (non-blocking)
	scanner := bufio.NewScanner(os.Stdin)
	prompt := styles.prompt.Render("> ")

	for gw.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			fmt.Print(prompt)

			if !scanner.Scan() {
				return nil
			}

			input := scanner.Text()
			if input == "" {
				continue
			}

			if gw.handleCommand(ctx, input) {
				continue
			}

			// Submit request
			req, err := gw.Submit(ctx, input)
			if err != nil {
				fmt.Println(styles.error.Render(fmt.Sprintf("Error: %v", err)))
			} else {
				fmt.Println(styles.dim.Render(fmt.Sprintf("[%s queued]", req.ID)))
			}
		}
	}
	return nil
}

func (gw *Gateway) responseListener(ctx context.Context) {
	// Track active requests for responses
	activeReqs := make(map[string]*Request)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gw.mu.RLock()
			if current := gw.currentTask.Load(); current != nil {
				if req, ok := current.(*Request); ok && req != nil {
					activeReqs[req.ID] = req
				}
			}
			gw.mu.RUnlock()

			// Check for responses
			for id, req := range activeReqs {
				select {
				case resp := <-req.ReplyCh:
					if resp.Error != nil {
						fmt.Println(styles.error.Render(fmt.Sprintf("Error: %v", resp.Error)))
					} else if resp.Content != "" {
						fmt.Println(styles.response.Render(resp.Content))
					}
					delete(activeReqs, id)
				default:
					// No response yet
				}
			}
		}
	}
}

func (gw *Gateway) handleCommand(ctx context.Context, input string) bool {
	switch input {
	case "/quit", "/exit", "/q":
		gw.running.Store(false)
		fmt.Println(styles.success.Render("Goodbye!"))
		os.Exit(0)
		return true

	case "/cancel", "/c":
		if gw.Cancel() {
			fmt.Println(styles.warning.Render("[Task cancelled]"))
		} else {
			fmt.Println(styles.dim.Render("[No active task]"))
		}
		return true

	case "/status", "/s":
		if gw.IsProcessing() {
			fmt.Println(styles.warning.Render("[Processing...]"))
		} else {
			fmt.Println(styles.success.Render("[Idle]"))
		}
		return true

	case "/help", "/h":
		printHelp()
		return true

	default:
		return false
	}
}

func printHelp() {
	fmt.Println(styles.header.Render("\nCommands:"))
	fmt.Println("  /help, /h      - Show this help")
	fmt.Println("  /status, /s    - Show current status")
	fmt.Println("  /cancel, /c    - Cancel current task")
	fmt.Println("  /quit, /q      - Exit gateway")
	fmt.Println()
}

// ========================================
// Service Initialization
// ========================================

func initAgentService(ctx context.Context, cfg *config.Config) (*agent.Service, error) {
	globalPool := services.GetGlobalPoolService()

	llmSvc, err := globalPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	embedSvc, err := globalPool.GetEmbeddingService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedder: %w", err)
	}

	// MCP service
	mcpSvc, err := mcp.NewService(&cfg.MCP, llmSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP service: %w", err)
	}

	if err := mcpSvc.StartServers(ctx, nil); err != nil {
		// Continue anyway
	}

	mcpAdapter := &mcpToolAdapter{service: mcpSvc}

	// Memory service
	homeDir, _ := os.UserHomeDir()
	memDBPath := filepath.Join(homeDir, ".rago", "data", "memory.db")
	memStore, err := store.NewMemoryStore(memDBPath)
	if err == nil {
		_ = memStore.InitSchema(ctx)
		memSvc := memory.NewService(memStore, llmSvc, embedSvc, memory.DefaultConfig())

		// Agent service
		agentDBPath := filepath.Join(homeDir, ".rago", "data", "agent.db")
		agentSvc, err := agent.NewService(llmSvc, mcpAdapter, nil, agentDBPath, memSvc)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent: %w", err)
		}

		// Router
		if routerSvc, err := router.NewService(embedSvc, router.DefaultConfig()); err == nil {
			_ = routerSvc.RegisterDefaultIntents()
			agentSvc.SetRouter(routerSvc)
		}

		return agentSvc, nil
	}

	// Fallback without memory
	agentDBPath := filepath.Join(homeDir, ".rago", "data", "agent.db")
	return agent.NewService(llmSvc, mcpAdapter, nil, agentDBPath, nil)
}

// ========================================
// MCP Tool Adapter
// ========================================

type mcpToolAdapter struct {
	service *mcp.Service
}

func (a *mcpToolAdapter) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	result, err := a.service.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("MCP tool error: %s", result.Error)
	}
	return result.Data, nil
}

func (a *mcpToolAdapter) ListTools() []domain.ToolDefinition {
	tools := a.service.GetAvailableTools(context.Background())
	result := make([]domain.ToolDefinition, 0, len(tools))

	for _, t := range tools {
		var parameters map[string]interface{}
		if t.InputSchema != nil && len(t.InputSchema) > 0 {
			parameters = t.InputSchema
		} else {
			parameters = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"arguments": map[string]interface{}{
						"type":        "object",
						"description": "Tool arguments",
					},
				},
			}
		}

		result = append(result, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  parameters,
			},
		})
	}
	return result
}

// ========================================
// Command Entry Point
// ========================================

func runGateway(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	gw, err := NewGateway(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		gw.running.Store(false)
		fmt.Println("\n" + styles.success.Render("Shutting down..."))
		os.Exit(0)
	}()

	return gw.Run(ctx)
}
