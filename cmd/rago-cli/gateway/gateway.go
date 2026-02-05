package gateway

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	agent    lipgloss.Style
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
	agent:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213")),
}

// GatewayCmd represents the gateway command
var GatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Run an interactive AI gateway with multi-agent support",
	Long: `Gateway mode runs an async AI reception desk with multi-agent support.

Each agent runs in its own goroutine:
- Agents never block each other
- Agents never block the main process
- Switch agents anytime, even while they're processing

Commands:
  /agent list           - List all agents
  /agent new <name>     - Create new agent
  /agent switch <name>  - Switch to agent
  /status               - Show status
  /cancel <agent>       - Cancel agent's current task
  /quit                 - Exit`,
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
// Agent Request/Response
// ========================================

type AgentRequest struct {
	ID        string
	Query     string
	Ctx       context.Context
	Cancel    context.CancelFunc
	Timestamp time.Time
}

type AgentResponse struct {
	RequestID string
	AgentName string
	Content   string
	Error     error
	Done      bool
}

// ========================================
// Agent Instance - Runs in own goroutine
// ========================================

type AgentInstance struct {
	ID         string
	Name       string
	SystemPrompt string
	Service    *agent.Service
	requestCh  chan *AgentRequest
	running    atomic.Bool
	currentReq atomic.Value // *AgentRequest
	createdAt  time.Time
	mu         sync.RWMutex
}

func (a *AgentInstance) Start(ctx context.Context) {
	a.running.Store(true)
	go a.run(ctx)
}

func (a *AgentInstance) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-a.requestCh:
			if !ok {
				return
			}
			if req == nil {
				continue
			}

			a.currentReq.Store(req)
			fmt.Println(styles.dim.Render(fmt.Sprintf("[%s processing %s]", a.Name, req.ID)))

			// Run agent - this blocks but only this agent's goroutine
			result, err := a.Service.Run(req.Ctx, req.Query)

			resp := &AgentResponse{
				RequestID: req.ID,
				AgentName: a.Name,
				Done:      true,
			}

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

			// Send response to gateway
			globalResponseCh <- resp

			a.currentReq.Store((*AgentRequest)(nil))
		}
	}
}

func (a *AgentInstance) Submit(req *AgentRequest) error {
	select {
	case a.requestCh <- req:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("agent %s request channel full", a.Name)
	}
}

func (a *AgentInstance) Cancel() bool {
	if req := a.currentReq.Load(); req != nil {
		if r, ok := req.(*AgentRequest); ok && r != nil && r.Cancel != nil {
			r.Cancel()
			return true
		}
	}
	return false
}

func (a *AgentInstance) IsProcessing() bool {
	return a.currentReq.Load() != nil
}

func (a *AgentInstance) Stop() {
	a.running.Store(false)
	close(a.requestCh)
}

// Global response channel - all agents send responses here
var globalResponseCh = make(chan *AgentResponse, 100)

// ========================================
// Shared Services
// ========================================

type SharedServices struct {
	llmPool      *services.GlobalPoolService
	mcpService   *mcp.Service
	embedService domain.Embedder
	memorySvc    domain.MemoryService
	routerSvc    *router.Service
}

func initSharedServices(ctx context.Context, cfg *config.Config) (*SharedServices, error) {
	globalPool := services.GetGlobalPoolService()

	llmSvc, err := globalPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	embedSvc, err := globalPool.GetEmbeddingService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedder: %w", err)
	}

	// MCP service (shared)
	mcpSvc, err := mcp.NewService(&cfg.MCP, llmSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP service: %w", err)
	}

	if err := mcpSvc.StartServers(ctx, nil); err != nil {
		// Continue anyway
	}

	// Memory service (shared)
	homeDir, _ := os.UserHomeDir()
	memDBPath := filepath.Join(homeDir, ".rago", "data", "memory.db")
	var memSvc domain.MemoryService
	memStore, err := store.NewMemoryStore(memDBPath)
	if err == nil {
		_ = memStore.InitSchema(ctx)
		memSvc = memory.NewService(memStore, llmSvc, embedSvc, memory.DefaultConfig())
	}

	// Router (shared)
	var routerSvc *router.Service
	if rs, err := router.NewService(embedSvc, router.DefaultConfig()); err == nil {
		_ = rs.RegisterDefaultIntents()
		routerSvc = rs
	}

	return &SharedServices{
		llmPool:      globalPool,
		mcpService:   mcpSvc,
		embedService: embedSvc,
		memorySvc:    memSvc,
		routerSvc:    routerSvc,
	}, nil
}

// ========================================
// Gateway
// ========================================

type Gateway struct {
	agents     map[string]*AgentInstance
	currentID  atomic.Value // string
	shared     *SharedServices
	running    atomic.Bool
	mu         sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewGateway(ctx context.Context, cfg *config.Config) (*Gateway, error) {
	shared, err := initSharedServices(ctx, cfg)
	if err != nil {
		return nil, err
	}

	gwCtx, gwCancel := context.WithCancel(ctx)

	gw := &Gateway{
		agents:     make(map[string]*AgentInstance),
		shared:     shared,
		ctx:        gwCtx,
		cancelFunc: gwCancel,
	}
	gw.running.Store(true)

	// Start response listener
	go gw.responseListener()

	// Create default agent
	if _, err := gw.CreateAgent("default", "You are a helpful AI assistant."); err != nil {
		return nil, err
	}

	return gw, nil
}

func (gw *Gateway) CreateAgent(name, systemPrompt string) (*AgentInstance, error) {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	if _, exists := gw.agents[name]; exists {
		return nil, fmt.Errorf("agent %s already exists", name)
	}

	// Create agent service
	llmSvc, err := gw.shared.llmPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	mcpAdapter := &mcpToolAdapter{service: gw.shared.mcpService}

	homeDir, _ := os.UserHomeDir()
	agentDBPath := filepath.Join(homeDir, ".rago", "data", fmt.Sprintf("agent_%s.db", name))

	agentSvc, err := agent.NewService(llmSvc, mcpAdapter, nil, agentDBPath, gw.shared.memorySvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	if gw.shared.routerSvc != nil {
		agentSvc.SetRouter(gw.shared.routerSvc)
	}

	inst := &AgentInstance{
		ID:           fmt.Sprintf("agent_%s_%d", name, time.Now().Unix()),
		Name:         name,
		SystemPrompt: systemPrompt,
		Service:      agentSvc,
		requestCh:    make(chan *AgentRequest, 10),
		createdAt:    time.Now(),
	}

	gw.agents[name] = inst

	// Start agent goroutine
	inst.Start(gw.ctx)

	return inst, nil
}

func (gw *Gateway) GetAgent(name string) (*AgentInstance, bool) {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	a, ok := gw.agents[name]
	return a, ok
}

func (gw *Gateway) ListAgents() []*AgentInstance {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	result := make([]*AgentInstance, 0, len(gw.agents))
	for _, a := range gw.agents {
		result = append(result, a)
	}
	return result
}

func (gw *Gateway) SetCurrent(name string) error {
	gw.mu.RLock()
	_, ok := gw.agents[name]
	gw.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not found", name)
	}

	gw.currentID.Store(name)
	return nil
}

func (gw *Gateway) GetCurrent() *AgentInstance {
	name := gw.currentID.Load()
	if name == nil || name == "" {
		return nil
	}

	agentName := name.(string)
	if a, ok := gw.GetAgent(agentName); ok {
		return a
	}
	return nil
}

func (gw *Gateway) Submit(ctx context.Context, query string) error {
	current := gw.GetCurrent()
	if current == nil {
		return fmt.Errorf("no current agent")
	}

	reqCtx, cancel := context.WithCancel(ctx)

	req := &AgentRequest{
		ID:        fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Query:     query,
		Ctx:       reqCtx,
		Cancel:    cancel,
		Timestamp: time.Now(),
	}

	return current.Submit(req)
}

func (gw *Gateway) CancelAgent(name string) bool {
	gw.mu.RLock()
	agent, ok := gw.agents[name]
	gw.mu.RUnlock()

	if !ok {
		return false
	}
	return agent.Cancel()
}

func (gw *Gateway) responseListener() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-gw.ctx.Done():
			return
		case resp := <-globalResponseCh:
			gw.printResponse(resp)
		case <-ticker.C:
			// Periodic check if needed
		}
	}
}

func (gw *Gateway) printResponse(resp *AgentResponse) {
	if resp.Error != nil {
		fmt.Println(styles.error.Render(fmt.Sprintf("[%s] Error: %v", resp.AgentName, resp.Error)))
	} else if resp.Content != "" {
		fmt.Printf("%s %s\n", styles.agent.Render("["+resp.AgentName+"]:"), styles.response.Render(resp.Content))
	}
}

// ========================================
// Interactive Loop
// ========================================

func (gw *Gateway) Run(ctx context.Context) error {
	fmt.Println(styles.header.Render("╔════════════════════════════════════════════════════════════╗"))
	fmt.Println(styles.header.Render("║" + styles.title.Render("          RAGO GATEWAY - MULTI-AGENT (PER-AGENT GOROUTINE)      ") + "║"))
	fmt.Println(styles.header.Render("╚════════════════════════════════════════════════════════════╝"))
	fmt.Println()
	gw.printAgentInfo()
	fmt.Println(styles.muted.Render("Type /help for commands"))

	scanner := bufio.NewScanner(os.Stdin)

	for gw.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			prompt := gw.buildPrompt()
			fmt.Print(prompt)

			if !scanner.Scan() {
				return nil
			}

			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}

			if gw.handleCommand(ctx, input) {
				continue
			}

			// Submit to current agent (non-blocking!)
			if err := gw.Submit(ctx, input); err != nil {
				fmt.Println(styles.error.Render(fmt.Sprintf("Error: %v", err)))
			} else {
				fmt.Println(styles.dim.Render("[→ Sent to agent]"))
			}
		}
	}
	return nil
}

func (gw *Gateway) buildPrompt() string {
	current := gw.GetCurrent()
	if current == nil {
		return styles.prompt.Render("> ")
	}

	status := ""
	if current.IsProcessing() {
		status = styles.warning.Render("[busy]")
	} else {
		status = styles.success.Render("[idle]")
	}

	return fmt.Sprintf("%s %s> ", styles.agent.Render(current.Name+":"), status)
}

func (gw *Gateway) printAgentInfo() {
	agents := gw.ListAgents()
	current := gw.GetCurrent()

	fmt.Printf("Agents: %d | ", len(agents))
	if current != nil {
		fmt.Printf("Current: %s", styles.agent.Render(current.Name))
	}
	fmt.Println()
}

func (gw *Gateway) handleCommand(ctx context.Context, input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/quit", "/exit", "/q":
		gw.running.Store(false)
		gw.cancelFunc()
		fmt.Println(styles.success.Render("Goodbye!"))
		os.Exit(0)
		return true

	case "/cancel", "/c":
		if len(args) > 0 {
			if gw.CancelAgent(args[0]) {
				fmt.Println(styles.warning.Render(fmt.Sprintf("[%s cancelled]", args[0])))
			} else {
				fmt.Println(styles.dim.Render(fmt.Sprintf("[Agent %s not found or idle]", args[0])))
			}
		} else {
			current := gw.GetCurrent()
			if current != nil && current.Cancel() {
				fmt.Println(styles.warning.Render(fmt.Sprintf("[%s cancelled]", current.Name)))
			} else {
				fmt.Println(styles.dim.Render("[No active task]"))
			}
		}
		return true

	case "/status", "/s":
		gw.printStatus()
		return true

	case "/agent", "/a":
		gw.handleAgentCommand(ctx, args)
		return true

	case "/help", "/h", "?":
		printHelp()
		return true

	default:
		return false
	}
}

func (gw *Gateway) printStatus() {
	agents := gw.ListAgents()
	current := gw.GetCurrent()

	fmt.Println(styles.header.Render("\nStatus:"))
	for _, a := range agents {
		prefix := "  "
		if current != nil && current.Name == a.Name {
			prefix = styles.success.Render("* ")
		}

		status := styles.success.Render("idle")
		if a.IsProcessing() {
			status = styles.warning.Render("busy")
		}

		fmt.Printf("%s%s: %s\n", prefix, styles.agent.Render(a.Name), status)
	}
	fmt.Println()
}

func (gw *Gateway) handleAgentCommand(ctx context.Context, args []string) {
	if len(args) == 0 {
		gw.printAgentList()
		return
	}

	subCmd := args[0]

	switch subCmd {
	case "list", "ls":
		gw.printAgentList()

	case "new", "create", "n":
		if len(args) < 2 {
			fmt.Println(styles.error.Render("Usage: /agent new <name> [prompt]"))
			return
		}
		name := args[1]
		prompt := "You are a helpful AI assistant."
		if len(args) > 2 {
			prompt = strings.Join(args[2:], " ")
		}
		_, err := gw.CreateAgent(name, prompt)
		if err != nil {
			fmt.Println(styles.error.Render(fmt.Sprintf("Failed: %v", err)))
			return
		}
		fmt.Println(styles.success.Render(fmt.Sprintf("[Agent '%s' created (goroutine started)]", name)))

	case "switch", "use", "s":
		if len(args) < 2 {
			fmt.Println(styles.error.Render("Usage: /agent switch <name>"))
			return
		}
		name := args[1]
		if err := gw.SetCurrent(name); err != nil {
			fmt.Println(styles.error.Render(fmt.Sprintf("Failed: %v", err)))
			return
		}
		fmt.Println(styles.success.Render(fmt.Sprintf("[Switched to '%s']", name)))

	case "current", "curr":
		current := gw.GetCurrent()
		if current == nil {
			fmt.Println(styles.dim.Render("[No current agent]"))
		} else {
			status := "idle"
			if current.IsProcessing() {
				status = "busy"
			}
			fmt.Printf("Current: %s | Status: %s\n",
				styles.agent.Render(current.Name),
				status)
		}

	default:
		fmt.Println(styles.error.Render(fmt.Sprintf("Unknown command: %s", subCmd)))
	}
}

func (gw *Gateway) printAgentList() {
	agents := gw.ListAgents()
	current := gw.GetCurrent()

	fmt.Println(styles.header.Render("\nAgents:"))
	for _, a := range agents {
		prefix := "  "
		if current != nil && current.Name == a.Name {
			prefix = styles.success.Render("* ")
		}

		status := styles.success.Render("idle")
		if a.IsProcessing() {
			status = styles.warning.Render("busy")
		}

		fmt.Printf("%s%s: %s\n", prefix, styles.agent.Render(a.Name), status)
	}
	fmt.Println()
}

func printHelp() {
	fmt.Println(styles.header.Render("\nCommands:"))
	fmt.Println("  /help, /h, ?              - Show this help")
	fmt.Println("  /status, /s                - Show all agents status")
	fmt.Println("  /cancel [agent]            - Cancel agent's task")
	fmt.Println()
	fmt.Println(styles.header.Render("Agent Commands:"))
	fmt.Println("  /agent list, /a ls         - List all agents")
	fmt.Println("  /agent new <name> [prompt] - Create new agent (new goroutine)")
	fmt.Println("  /agent switch <name>       - Switch to agent")
	fmt.Println("  /agent current             - Show current agent")
	fmt.Println()
	fmt.Println(styles.header.Render("Other:"))
	fmt.Println("  /quit, /q                  - Exit gateway")
	fmt.Println()
	fmt.Println(styles.muted.Render("Each agent runs in its own goroutine - they never block each other!"))
	fmt.Println()
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
		gw.cancelFunc()
		fmt.Println("\n" + styles.success.Render("Shutting down..."))
		os.Exit(0)
	}()

	return gw.Run(ctx)
}
