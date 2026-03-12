package squad

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/liliang-cn/agent-go/pkg/agent"
	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/spf13/cobra"
)

var SquadCmd = &cobra.Command{
	Use:   "squad",
	Short: "Run squad tasks and manage captains or specialists",
	Long:  `Run squad tasks, inspect members, and control captain lifecycle.`,
	Args:  cobra.NoArgs,
	RunE:  runInteractiveSquad,
}

type delegatedTask struct {
	AgentName   string
	Instruction string
}

var agentMentionPattern = regexp.MustCompile(`^@([^\s@]+)$`)

func init() {
	SquadCmd.AddCommand(goCmd)
	SquadCmd.AddCommand(addCmd)
	SquadCmd.AddCommand(listCmd)
	SquadCmd.AddCommand(memberCmd)
	memberCmd.AddCommand(memberAddCmd)
	memberCmd.AddCommand(memberListCmd)
	memberCmd.AddCommand(memberShowCmd)
	memberCmd.AddCommand(memberStartCmd)
	memberCmd.AddCommand(memberStopCmd)

	addCmd.Flags().StringVar(&squadDescription, "description", "", "squad description")

	memberAddCmd.Flags().StringVar(&memberDescription, "description", "", "member description")
	memberAddCmd.Flags().StringVar(&memberInstructions, "instructions", "", "member system instructions")
	memberAddCmd.Flags().StringVar(&memberKind, "kind", "specialist", "member kind: specialist or captain")
	memberAddCmd.Flags().StringVar(&memberSquadID, "squad-id", "", "target squad ID (defaults to the default squad)")
	memberAddCmd.Flags().StringVar(&memberSquadName, "squad", "", "target squad name (defaults to the default squad)")
	memberAddCmd.Flags().StringVar(&memberModel, "model", "", "preferred provider or model")
}

var (
	squadDescription   string
	memberDescription  string
	memberInstructions string
	memberKind         string
	memberSquadID      string
	memberSquadName    string
	memberModel        string
)

var goCmd = &cobra.Command{
	Use:   "go [task]",
	Short: "Run a squad task",
	Long:  `Run one squad task explicitly, for example: agentgo squad go "@Assistant summarize and implement".`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}
		return runSquadMessage(context.Background(), manager, strings.Join(args, " "))
	},
}

var addCmd = &cobra.Command{
	Use:     "add [name]",
	Aliases: []string{"create"},
	Short:   "Add a squad",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}

		name := strings.TrimSpace(args[0])
		if name == "" {
			return fmt.Errorf("squad name is required")
		}

		description := strings.TrimSpace(squadDescription)
		if description == "" {
			description = name
		}

		squad, err := manager.CreateSquad(context.Background(), &agent.Squad{
			Name:        name,
			Description: description,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Added squad '%s'.\n", squad.Name)
		return nil
	},
}

func getManager() (*agent.SquadManager, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	agentDBPath := cfg.DataDir() + "/agent.db"
	store, err := agent.NewStore(agentDBPath)
	if err != nil {
		return nil, err
	}
	manager := agent.NewSquadManager(store)
	_ = manager.SeedDefaultMembers()
	return manager, nil
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List squads",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}
		squads, err := manager.ListSquads()
		if err != nil {
			return err
		}
		members, err := manager.ListMembers()
		if err != nil {
			return err
		}

		type squadRow struct {
			Name        string
			Captain     string
			Members     int
			Description string
		}

		rows := make([]squadRow, 0, len(squads))
		for _, squad := range squads {
			row := squadRow{Name: squad.Name, Description: squad.Description}
			for _, member := range members {
				if member.TeamID != squad.ID {
					continue
				}
				row.Members++
				if member.Kind == agent.AgentKindCaptain && row.Captain == "" {
					row.Captain = member.Name
				}
			}
			rows = append(rows, row)
		}

		slices.SortFunc(rows, func(a, b squadRow) int {
			switch {
			case a.Name < b.Name:
				return -1
			case a.Name > b.Name:
				return 1
			default:
				return 0
			}
		})

		fmt.Println("Squads")
		if len(rows) == 0 {
			fmt.Println("  (none)")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tCAPTAIN\tMEMBERS\tDESCRIPTION")
		for _, row := range rows {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", row.Name, valueOrDash(row.Captain), row.Members, row.Description)
		}
		w.Flush()
		return nil
	},
}

var memberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage squad members",
}

var memberAddCmd = &cobra.Command{
	Use:     "add [name]",
	Aliases: []string{"create"},
	Short:   "Add a squad member",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}

		name := strings.TrimSpace(args[0])
		if name == "" {
			return fmt.Errorf("agent name is required")
		}

		kind := agent.AgentKind(strings.TrimSpace(memberKind))
		if kind == "" {
			kind = agent.AgentKindSpecialist
		}
		if kind != agent.AgentKindCaptain && kind != agent.AgentKindSpecialist {
			return fmt.Errorf("invalid kind %q: use captain or specialist", kind)
		}

		description := strings.TrimSpace(memberDescription)
		if description == "" {
			description = name
		}
		instructions := strings.TrimSpace(memberInstructions)
		if instructions == "" {
			instructions = description
		}
		squadID, err := resolveMemberSquadID(manager, strings.TrimSpace(memberSquadID), strings.TrimSpace(memberSquadName))
		if err != nil {
			return err
		}

		member, err := manager.CreateMember(context.Background(), &agent.AgentModel{
			Name:         name,
			TeamID:       squadID,
			Kind:         kind,
			Description:  description,
			Instructions: instructions,
			Model:        strings.TrimSpace(memberModel),
		})
		if err != nil {
			return err
		}

		fmt.Printf("Added %s '%s'.\n", strings.ToLower(string(member.Kind)), member.Name)
		return nil
	},
}

var memberListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all captains and specialists",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}
		agentsList, err := manager.ListMembers()
		if err != nil {
			return err
		}

		var captains []*agent.AgentModel
		var specialists []*agent.AgentModel
		for _, a := range agentsList {
			switch a.Kind {
			case agent.AgentKindSpecialist:
				specialists = append(specialists, a)
			default:
				captains = append(captains, a)
			}
		}

		slices.SortFunc(captains, func(a, b *agent.AgentModel) int {
			return compareAgentNames(a.Name, b.Name)
		})
		slices.SortFunc(specialists, func(a, b *agent.AgentModel) int {
			return compareAgentNames(a.Name, b.Name)
		})

		printAgentSection("Captains", captains, true)
		fmt.Println()
		printAgentSection("Specialists", specialists, false)
		return nil
	},
}

var memberShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show detailed agent configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}
		a, err := manager.GetMemberByName(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Name: %s\n", a.Name)
		fmt.Printf("Kind: %s\n", a.Kind)
		if a.Kind == agent.AgentKindCaptain {
			fmt.Printf("Status: %s\n", a.Status)
		} else {
			fmt.Println("Status: reusable specialist")
		}
		fmt.Printf("Model: %s\n", valueOrDash(a.Model))
		fmt.Printf("Description: %s\n", valueOrDash(a.Description))
		fmt.Printf("RAG: %s\n", enabledState(a.EnableRAG))
		fmt.Printf("Memory: %s\n", enabledState(a.EnableMemory))
		fmt.Printf("MCP: %s\n", enabledState(a.EnableMCP))
		fmt.Printf("PTC: %s\n", enabledState(a.EnablePTC))
		fmt.Printf("Skills: %s\n", joinOrDash(a.Skills))
		fmt.Printf("MCP Tools: %s\n", joinOrDash(a.MCPTools))
		fmt.Printf("Created: %s\n", formatTimestamp(a.CreatedAt))
		fmt.Printf("Updated: %s\n", formatTimestamp(a.UpdatedAt))
		if a.Instructions != "" {
			fmt.Printf("\nInstructions:\n%s\n", a.Instructions)
		}
		return nil
	},
}

var memberStartCmd = &cobra.Command{
	Use:     "start [name]",
	Short:   "Enable a captain by name",
	Aliases: []string{"enable"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}
		name := args[0]
		err = manager.EnableCaptain(context.Background(), name)
		if err != nil {
			return err
		}
		fmt.Printf("Captain '%s' enabled.\n", name)
		return nil
	},
}

var memberStopCmd = &cobra.Command{
	Use:     "stop [name]",
	Short:   "Disable a captain by name",
	Aliases: []string{"disable"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}
		name := args[0]
		err = manager.DisableCaptain(context.Background(), name)
		if err != nil {
			return err
		}
		fmt.Printf("Captain '%s' disabled.\n", name)
		return nil
	},
}

func printAgentSection(title string, agentsList []*agent.AgentModel, showStatus bool) {
	fmt.Println(title)
	if len(agentsList) == 0 {
		fmt.Println("  (none)")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if showStatus {
		fmt.Fprintln(w, "NAME\tSTATUS\tDESCRIPTION")
		for _, a := range agentsList {
			fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, a.Status, a.Description)
		}
	} else {
		fmt.Fprintln(w, "NAME\tDESCRIPTION")
		for _, a := range agentsList {
			fmt.Fprintf(w, "%s\t%s\n", a.Name, a.Description)
		}
	}
	w.Flush()
}

func compareAgentNames(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func valueOrDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func enabledState(v bool) string {
	if v {
		return "enabled"
	}
	return "disabled"
}

func formatTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Format(time.RFC3339)
}

func resolveMemberSquadID(manager *agent.SquadManager, squadID, squadName string) (string, error) {
	if squadID != "" && squadName != "" {
		return "", fmt.Errorf("use either --squad-id or --squad, not both")
	}
	if squadID != "" {
		return squadID, nil
	}
	if squadName == "" {
		return "", nil
	}

	squads, err := manager.ListSquads()
	if err != nil {
		return "", err
	}
	for _, squad := range squads {
		if strings.EqualFold(strings.TrimSpace(squad.Name), squadName) {
			return squad.ID, nil
		}
	}
	return "", fmt.Errorf("unknown squad: %s", squadName)
}

func runInteractiveSquad(cmd *cobra.Command, args []string) error {
	manager, err := getManager()
	if err != nil {
		return err
	}
	return runInteractiveSquadChat(context.Background(), manager)
}

func runInteractiveSquadChat(ctx context.Context, manager *agent.SquadManager) error {
	fmt.Println("🤝 AgentGo Squad Mode")
	fmt.Println("💡 Direct requests go to Assistant by default")
	fmt.Println("💡 Use @Assistant or any existing member name to delegate")
	fmt.Println("💡 Type 'quit' or 'exit' to end")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("squad> ")
		if !scanner.Scan() {
			fmt.Println()
			return nil
		}

		input := strings.TrimSpace(scanner.Text())
		switch input {
		case "":
			continue
		case "quit", "exit":
			return nil
		}

		if err := runSquadMessage(ctx, manager, input); err != nil {
			fmt.Printf("Error: %v\n\n", err)
		}
	}
}

func runSquadMessage(ctx context.Context, manager *agent.SquadManager, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}

	fmt.Printf("\n🤔 You: %s\n", message)

	tasks, err := parseDelegatedTasks(message, func(name string) bool {
		_, getErr := manager.GetMemberByName(name)
		return getErr == nil
	})
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		tasks = []delegatedTask{{
			AgentName:   "Assistant",
			Instruction: message,
		}}
	}

	var previousAgent string
	var previousResult string
	for idx, task := range tasks {
		fmt.Printf("\n🚀 Running %d/%d with @%s...\n", idx+1, len(tasks), task.AgentName)
		instruction := task.Instruction
		if idx > 0 && previousResult != "" {
			instruction = buildSequentialInstruction(previousAgent, previousResult, task.Instruction)
		}

		res, dispatchErr := manager.DispatchTask(ctx, task.AgentName, instruction)
		if dispatchErr != nil {
			return fmt.Errorf("task failed for @%s: %w", task.AgentName, dispatchErr)
		}
		fmt.Printf("\n✅ Response from @%s:\n%s\n", task.AgentName, res)
		previousAgent = task.AgentName
		previousResult = strings.TrimSpace(res)
	}

	fmt.Println()
	return nil
}

func buildSequentialInstruction(previousAgent, previousResult, nextInstruction string) string {
	const maxContextChars = 6000

	previousResult = strings.TrimSpace(previousResult)
	if len(previousResult) > maxContextChars {
		previousResult = previousResult[:maxContextChars] + "\n...[truncated]"
	}

	return strings.TrimSpace(
		"Previous result from @" + previousAgent + ":\n" +
			previousResult + "\n\n" +
			"Use that result as input for your step. Complete the following task:\n" +
			strings.TrimSpace(nextInstruction),
	)
}

func parseDelegatedTasks(input string, isKnownAgent func(name string) bool) ([]delegatedTask, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, nil
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return nil, nil
	}

	firstName, ok := parseMentionedAgent(words[0])
	if !ok {
		return nil, nil
	}
	if isKnownAgent != nil && !isKnownAgent(firstName) {
		return nil, fmt.Errorf("unknown agent: %s", firstName)
	}

	// Support leading shared mentions like:
	//   @Assistant @SomeMember summarize the repo and write a note
	// In this form every leading mention receives the same instruction.
	leadingMentions := []string{firstName}
	firstInstructionIndex := 1
	for firstInstructionIndex < len(words) {
		nextName, isMention := parseMentionedAgent(words[firstInstructionIndex])
		if !isMention {
			break
		}
		if isKnownAgent != nil && !isKnownAgent(nextName) {
			return nil, fmt.Errorf("unknown agent: %s", nextName)
		}
		leadingMentions = append(leadingMentions, nextName)
		firstInstructionIndex++
	}
	if len(leadingMentions) > 1 {
		sharedInstruction := strings.TrimSpace(strings.Join(words[firstInstructionIndex:], " "))
		if sharedInstruction == "" {
			return nil, fmt.Errorf("please provide an instruction after the agent mentions")
		}
		tasks := make([]delegatedTask, 0, len(leadingMentions))
		for _, name := range leadingMentions {
			tasks = append(tasks, delegatedTask{
				AgentName:   name,
				Instruction: sharedInstruction,
			})
		}
		return tasks, nil
	}

	tasks := make([]delegatedTask, 0, 2)
	current := delegatedTask{AgentName: firstName}

	for _, word := range words[1:] {
		if nextName, isMention := parseMentionedAgent(word); isMention {
			if isKnownAgent != nil && isKnownAgent(nextName) {
				current.Instruction = strings.TrimSpace(current.Instruction)
				if current.Instruction == "" {
					return nil, fmt.Errorf("please provide an instruction for %s", current.AgentName)
				}
				tasks = append(tasks, current)
				current = delegatedTask{AgentName: nextName}
				continue
			}
		}

		if current.Instruction == "" {
			current.Instruction = word
		} else {
			current.Instruction += " " + word
		}
	}

	current.Instruction = strings.TrimSpace(current.Instruction)
	if current.Instruction == "" {
		return nil, fmt.Errorf("please provide an instruction for %s", current.AgentName)
	}
	tasks = append(tasks, current)
	return tasks, nil
}

func parseMentionedAgent(word string) (string, bool) {
	matches := agentMentionPattern.FindStringSubmatch(word)
	if len(matches) != 2 {
		return "", false
	}
	return matches[1], true
}
