package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/memory"
	"github.com/liliang-cn/rago/v2/pkg/store"
	"github.com/spf13/cobra"
)

// CommandOptions holds the options for memory commands
type CommandOptions struct {
	dbPath string
}

// NewCommand creates the memory command
func NewCommand(opts *CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Manage long-term agent memories",
		Long: `Manage long-term agent memories for context-aware interactions.

Memory types:
  - fact: Factual information
  - skill: Skills and procedures
  - pattern: Patterns and trends
  - context: Contextual information
  - preference: User preferences`,
	}

	// Add subcommands
	cmd.AddCommand(newSearchCommand(opts))
	cmd.AddCommand(newGetCommand(opts))
	cmd.AddCommand(newAddCommand(opts))
	cmd.AddCommand(newUpdateCommand(opts))
	cmd.AddCommand(newListCommand(opts))
	cmd.AddCommand(newDeleteCommand(opts))

	return cmd
}

// newSearchCommand creates the search subcommand
func newSearchCommand(opts *CommandOptions) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search memories by query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := createMemoryService(opts)
			if err != nil {
				return err
			}

			memories, err := svc.Search(cmd.Context(), args[0], limit)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			if len(memories) == 0 {
				fmt.Println("No memories found.")
				return nil
			}

			fmt.Printf("Found %d memories:\n\n", len(memories))
			for i, mem := range memories {
				fmt.Printf("[%d] %s (score: %.2f)\n", i+1, mem.Type, mem.Score)
				fmt.Printf("    ID: %s\n", mem.ID)
				fmt.Printf("    Content: %s\n", mem.Content)
				fmt.Printf("    Importance: %.2f | Access Count: %d\n", mem.Importance, mem.AccessCount)
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 5, "Maximum number of results")

	return cmd
}

// newGetCommand creates the get subcommand
func newGetCommand(opts *CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <memory-id>",
		Short: "Get a memory by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := createMemoryService(opts)
			if err != nil {
				return err
			}

			mem, err := svc.Get(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get failed: %w", err)
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(mem)
		},
	}

	return cmd
}

// newAddCommand creates the add subcommand
func newAddCommand(opts *CommandOptions) *cobra.Command {
	var (
		memType    string
		importance float64
		sessionID  string
	)

	cmd := &cobra.Command{
		Use:   "add <content>",
		Short: "Add a new memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := createMemoryService(opts)
			if err != nil {
				return err
			}

			if memType == "" {
				return fmt.Errorf("--type is required")
			}

			if !isValidMemoryType(memType) {
				return fmt.Errorf("invalid memory type: %s (must be one of: fact, skill, pattern, context, preference)", memType)
			}

			mem := &domain.Memory{
				ID:         uuid.New().String(),
				Type:       domain.MemoryType(memType),
				Content:    args[0],
				Importance: importance,
				SessionID:  sessionID,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			if err := svc.Add(cmd.Context(), mem); err != nil {
				return fmt.Errorf("add failed: %w", err)
			}

			fmt.Printf("Memory added successfully:\nID: %s\nType: %s\n", mem.ID, mem.Type)

			return nil
		},
	}

	cmd.Flags().StringVarP(&memType, "type", "t", "", "Memory type (fact, skill, pattern, context, preference)")
	cmd.Flags().Float64VarP(&importance, "importance", "i", 0.5, "Importance score (0-1)")
	cmd.Flags().StringVarP(&sessionID, "session", "s", "", "Session ID to associate with")

	_ = cmd.MarkFlagRequired("type")

	return cmd
}

// newUpdateCommand creates the update subcommand
func newUpdateCommand(opts *CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <memory-id> <instruction>",
		Short: "Update a memory using LLM",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := createMemoryService(opts)
			if err != nil {
				return err
			}

			if err := svc.Update(cmd.Context(), args[0], args[1]); err != nil {
				return fmt.Errorf("update failed: %w", err)
			}

			fmt.Println("Memory updated successfully.")

			return nil
		},
	}

	return cmd
}

// newListCommand creates the list subcommand
func newListCommand(opts *CommandOptions) *cobra.Command {
	var (
		limit  int
		offset int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all memories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := createMemoryService(opts)
			if err != nil {
				return err
			}

			memories, total, err := svc.List(cmd.Context(), limit, offset)
			if err != nil {
				return fmt.Errorf("list failed: %w", err)
			}

			fmt.Printf("Memories (%d total, showing %d):\n\n", total, len(memories))

			for i, mem := range memories {
				fmt.Printf("[%d] %s\n", offset+i+1, mem.Type)
				fmt.Printf("    ID: %s\n", mem.ID)
				fmt.Printf("    Content: %s\n", truncateString(mem.Content, 100))
				fmt.Printf("    Importance: %.2f | Created: %s\n", mem.Importance, mem.CreatedAt.Format("2006-01-02"))
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of memories to show")
	cmd.Flags().IntVarP(&offset, "offset", "o", 0, "Number of memories to skip")

	return cmd
}

// newDeleteCommand creates the delete subcommand
func newDeleteCommand(opts *CommandOptions) *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <memory-id>",
		Short: "Delete a memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := createMemoryService(opts)
			if err != nil {
				return err
			}

			if !confirm {
				fmt.Printf("Are you sure you want to delete memory %s? (y/N): ", args[0])
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := svc.Delete(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("delete failed: %w", err)
			}

			fmt.Println("Memory deleted successfully.")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "Skip confirmation")

	return cmd
}

// createMemoryService creates a memory service with default settings
func createMemoryService(opts *CommandOptions) (*memory.Service, error) {
	// For CLI commands, we need a simple memory service without LLM/embedder
	// Create store only
	path := opts.dbPath
	if path == "" {
		path = "./.rago/data/memory.db"
	}

	db, err := store.NewMemoryStore(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory store: %w", err)
	}

	// Create service with nil LLM/embedder (will only work for basic operations)
	return memory.NewService(db, nil, nil, memory.DefaultConfig()), nil
}

// isValidMemoryType checks if the memory type is valid
func isValidMemoryType(t string) bool {
	switch t {
	case "fact", "skill", "pattern", "context", "preference":
		return true
	default:
		return false
	}
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
