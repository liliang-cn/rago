package gateway

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/gateway"
	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	verbose bool
	gwLib   *gateway.Gateway
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

This CLI uses the pkg/gateway library - you can use it in your own Go programs!`,
	RunE: runGateway,
}

func init() {
	GatewayCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
}

func SetSharedVariables(c *config.Config, v bool) {
	cfg = c
	verbose = v
}

// CLIGateway wraps the library gateway for CLI use
type CLIGateway struct {
	gw      *gateway.Gateway
	running atomic.Bool
	mu      sync.RWMutex
	respCh  chan *gateway.Response
}

func NewCLIGateway(ctx context.Context, cfg *config.Config) (*CLIGateway, error) {
	respCh := make(chan *gateway.Response, 100)

	gw, err := gateway.New(ctx, cfg,
		gateway.WithResponseCallback(func(resp *gateway.Response) {
			select {
			case respCh <- resp:
			default:
			}
		}),
	)
	if err != nil {
		return nil, err
	}

	cliGw := &CLIGateway{
		gw:     gw,
		respCh: respCh,
	}
	cliGw.running.Store(true)

	// Create default agent
	if _, err := gw.CreateAgent("default", "You are a helpful AI assistant."); err != nil {
		return nil, err
	}
	gw.SetCurrent("default")

	// Start response listener
	go cliGw.responseListener(ctx)

	return cliGw, nil
}

func (cg *CLIGateway) responseListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case resp := <-cg.respCh:
			cg.printResponse(resp)
		}
	}
}

func (cg *CLIGateway) printResponse(resp *gateway.Response) {
	if resp.Error != nil {
		fmt.Println(styles.error.Render(fmt.Sprintf("[%s] Error: %v", resp.AgentName, resp.Error)))
	} else if resp.Content != "" {
		fmt.Printf("%s %s\n", styles.agent.Render("["+resp.AgentName+"]:"), styles.response.Render(resp.Content))
	}
}

func (cg *CLIGateway) Run(ctx context.Context) error {
	fmt.Println(styles.header.Render("╔════════════════════════════════════════════════════════════╗"))
	fmt.Println(styles.header.Render("║" + styles.title.Render("       RAGO GATEWAY - using pkg/gateway lib        ") + "║"))
	fmt.Println(styles.header.Render("╚════════════════════════════════════════════════════════════╝"))
	fmt.Println()
	cg.printInfo()
	fmt.Println(styles.muted.Render("Type /help for commands"))

	scanner := bufio.NewScanner(os.Stdin)

	for cg.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			prompt := cg.buildPrompt()
			fmt.Print(prompt)

			if !scanner.Scan() {
				return nil
			}

			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}

			if cg.handleCommand(ctx, input) {
				continue
			}

			// Non-blocking query to current agent
			if _, err := cg.gw.Query(ctx, input); err != nil {
				fmt.Println(styles.error.Render(fmt.Sprintf("Error: %v", err)))
			} else {
				fmt.Println(styles.dim.Render("[→ Sent]"))
			}
		}
	}
	return nil
}

func (cg *CLIGateway) buildPrompt() string {
	current := cg.gw.GetCurrent()
	if current == nil {
		return styles.prompt.Render("> ")
	}

	status := styles.success.Render("[idle]")
	if current.IsBusy() {
		status = styles.warning.Render("[busy]")
	}

	return fmt.Sprintf("%s %s> ", styles.agent.Render(current.Name()+":"), status)
}

func (cg *CLIGateway) printInfo() {
	agents := cg.gw.ListAgents()
	current := cg.gw.GetCurrent()

	fmt.Printf("Agents: %d | ", len(agents))
	if current != nil {
		fmt.Printf("Current: %s", styles.agent.Render(current.Name()))
	}
	fmt.Println()
}

func (cg *CLIGateway) handleCommand(ctx context.Context, input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/quit", "/exit", "/q":
		cg.running.Store(false)
		_ = cg.gw.Close()
		fmt.Println(styles.success.Render("Goodbye!"))
		os.Exit(0)
		return true

	case "/cancel", "/c":
		if len(args) > 0 {
			if agent, ok := cg.gw.GetAgent(args[0]); ok && agent.Cancel() {
				fmt.Println(styles.warning.Render(fmt.Sprintf("[%s cancelled]", args[0])))
			}
		} else if current := cg.gw.GetCurrent(); current != nil && current.Cancel() {
			fmt.Println(styles.warning.Render("[Cancelled]"))
		}
		return true

	case "/status", "/s":
		cg.printStatus()
		return true

	case "/agent", "/a":
		cg.handleAgentCommand(ctx, args)
		return true

	case "/help", "/h", "?":
		printHelp()
		return true

	default:
		return false
	}
}

func (cg *CLIGateway) printStatus() {
	fmt.Println(styles.header.Render("\nStatus:"))
	for _, agent := range cg.gw.ListAgents() {
		prefix := "  "
		if current := cg.gw.GetCurrent(); current != nil && current.ID() == agent.ID() {
			prefix = styles.success.Render("* ")
		}

		status := styles.success.Render("idle")
		if agent.IsBusy() {
			status = styles.warning.Render("busy")
		}

		fmt.Printf("%s%s: %s\n", prefix, styles.agent.Render(agent.Name()), status)
	}
	fmt.Println()
}

func (cg *CLIGateway) handleAgentCommand(ctx context.Context, args []string) {
	if len(args) == 0 {
		cg.printAgentList()
		return
	}

	subCmd := args[0]

	switch subCmd {
	case "list", "ls":
		cg.printAgentList()

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
		if _, err := cg.gw.CreateAgent(name, prompt); err != nil {
			fmt.Println(styles.error.Render(fmt.Sprintf("Failed: %v", err)))
		} else {
			fmt.Println(styles.success.Render(fmt.Sprintf("[Agent '%s' created]", name)))
		}

	case "switch", "use", "s":
		if len(args) < 2 {
			fmt.Println(styles.error.Render("Usage: /agent switch <name>"))
			return
		}
		if err := cg.gw.SetCurrent(args[1]); err != nil {
			fmt.Println(styles.error.Render(fmt.Sprintf("Failed: %v", err)))
		} else {
			fmt.Println(styles.success.Render(fmt.Sprintf("[Switched to '%s']", args[1])))
		}

	case "current", "curr":
		if current := cg.gw.GetCurrent(); current != nil {
			fmt.Printf("Current: %s\n", styles.agent.Render(current.Name()))
		}
	}
}

func (cg *CLIGateway) printAgentList() {
	fmt.Println(styles.header.Render("\nAgents:"))
	current := cg.gw.GetCurrent()
	for _, agent := range cg.gw.ListAgents() {
		prefix := "  "
		if current != nil && current.ID() == agent.ID() {
			prefix = styles.success.Render("* ")
		}

		status := styles.success.Render("idle")
		if agent.IsBusy() {
			status = styles.warning.Render("busy")
		}

		fmt.Printf("%s%s: %s\n", prefix, styles.agent.Render(agent.Name()), status)
	}
	fmt.Println()
}

func printHelp() {
	fmt.Println(styles.header.Render("\nCommands:"))
	fmt.Println("  /help, /h              - Show help")
	fmt.Println("  /status, /s            - Show status")
	fmt.Println("  /cancel [agent]        - Cancel agent task")
	fmt.Println()
	fmt.Println(styles.header.Render("Agent Commands:"))
	fmt.Println("  /agent list            - List agents")
	fmt.Println("  /agent new <name>      - Create agent")
	fmt.Println("  /agent switch <name>   - Switch agent")
	fmt.Println()
	fmt.Println(styles.header.Render("Other:"))
	fmt.Println("  /quit                  - Exit")
	fmt.Println()
	fmt.Println(styles.muted.Render("Powered by pkg/gateway library"))
	fmt.Println()
}

func runGateway(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cliGw, err := NewCLIGateway(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cliGw.running.Store(false)
		_ = cliGw.gw.Close()
		fmt.Println("\n" + styles.success.Render("Shutting down..."))
		os.Exit(0)
	}()

	return cliGw.Run(ctx)
}
