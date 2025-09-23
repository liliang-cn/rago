package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/settings"
	"github.com/spf13/cobra"
)

var (
	cfg       *config.Config
	verbose   bool
	quiet     bool
	settingSvc *settings.Service
)

// SetSharedVariables sets shared variables from the parent command
func SetSharedVariables(c *config.Config, v, q bool) {
	cfg = c
	verbose = v
	quiet = q

	// Initialize settings service
	var err error
	settingSvc, err = settings.NewService(cfg)
	if err != nil {
		fmt.Printf("Warning: failed to initialize settings service: %v\n", err)
	}
}

// ProfileCmd represents the profile command
var ProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage user profiles and LLM settings",
	Long: `Manage user profiles and LLM settings for persistent configuration.
Profiles allow you to maintain different system prompts, LLM parameters,
and conversation contexts across sessions.`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all user profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		if settingSvc == nil {
			return fmt.Errorf("settings service not initialized")
		}

		profiles, err := settingSvc.ListProfiles()
		if err != nil {
			return fmt.Errorf("failed to list profiles: %w", err)
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tACTIVE\tDESCRIPTION\tCREATED")
		fmt.Fprintln(w, "--\t----\t------\t-----------\t-------")

		for _, p := range profiles {
			active := ""
			if p.IsActive {
				active = "✓"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				p.ID[:8], p.Name, active, p.Description, p.CreatedAt.Format("2006-01-02 15:04"))
		}

		return w.Flush()
	},
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new user profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if settingSvc == nil {
			return fmt.Errorf("settings service not initialized")
		}

		name := args[0]
		description, _ := cmd.Flags().GetString("description")
		systemPrompt, _ := cmd.Flags().GetString("system-prompt")
		setActive, _ := cmd.Flags().GetBool("set-active")

		req := settings.CreateProfileRequest{
			Name:                name,
			Description:         description,
			DefaultSystemPrompt: systemPrompt,
			Metadata:            make(map[string]string),
		}

		profile, err := settingSvc.CreateProfile(req)
		if err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}

		if setActive {
			if err := settingSvc.SetActiveProfile(profile.ID); err != nil {
				return fmt.Errorf("failed to set profile as active: %w", err)
			}
		}

		fmt.Printf("Profile created successfully:\n")
		fmt.Printf("  ID: %s\n", profile.ID)
		fmt.Printf("  Name: %s\n", profile.Name)
		fmt.Printf("  Description: %s\n", profile.Description)
		if setActive {
			fmt.Printf("  Set as active profile\n")
		}

		return nil
	},
}

var switchCmd = &cobra.Command{
	Use:   "switch <name-or-id>",
	Short: "Switch to a different profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if settingSvc == nil {
			return fmt.Errorf("settings service not initialized")
		}

		nameOrID := args[0]

		// First try to find by name, then by ID
		profiles, err := settingSvc.ListProfiles()
		if err != nil {
			return fmt.Errorf("failed to list profiles: %w", err)
		}

		var targetProfile *settings.UserProfile
		for _, p := range profiles {
			if p.Name == nameOrID || strings.HasPrefix(p.ID, nameOrID) {
				targetProfile = p
				break
			}
		}

		if targetProfile == nil {
			return fmt.Errorf("profile not found: %s", nameOrID)
		}

		if err := settingSvc.SetActiveProfile(targetProfile.ID); err != nil {
			return fmt.Errorf("failed to switch profile: %w", err)
		}

		fmt.Printf("Switched to profile: %s (%s)\n", targetProfile.Name, targetProfile.ID[:8])
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show [name-or-id]",
	Short: "Show profile details (current profile if no argument)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if settingSvc == nil {
			return fmt.Errorf("settings service not initialized")
		}

		var profile *settings.UserProfile
		var err error

		if len(args) == 0 {
			// Show current active profile
			profile, err = settingSvc.GetActiveProfile()
			if err != nil {
				return fmt.Errorf("failed to get active profile: %w", err)
			}
		} else {
			// Find profile by name or ID
			nameOrID := args[0]
			profiles, err := settingSvc.ListProfiles()
			if err != nil {
				return fmt.Errorf("failed to list profiles: %w", err)
			}

			for _, p := range profiles {
				if p.Name == nameOrID || strings.HasPrefix(p.ID, nameOrID) {
					profile = p
					break
				}
			}

			if profile == nil {
				return fmt.Errorf("profile not found: %s", nameOrID)
			}
		}

		// Get LLM settings for this profile
		llmSettings, err := settingSvc.GetAllLLMSettings(profile.ID)
		if err != nil {
			return fmt.Errorf("failed to get LLM settings: %w", err)
		}

		fmt.Printf("Profile Details:\n")
		fmt.Printf("  ID: %s\n", profile.ID)
		fmt.Printf("  Name: %s\n", profile.Name)
		fmt.Printf("  Description: %s\n", profile.Description)
		fmt.Printf("  Active: %t\n", profile.IsActive)
		fmt.Printf("  Created: %s\n", profile.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Updated: %s\n", profile.UpdatedAt.Format("2006-01-02 15:04:05"))

		if profile.DefaultSystemPrompt != "" {
			fmt.Printf("  Default System Prompt:\n    %s\n", strings.ReplaceAll(profile.DefaultSystemPrompt, "\n", "\n    "))
		}

		if len(profile.Metadata) > 0 {
			fmt.Printf("  Metadata:\n")
			for k, v := range profile.Metadata {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}

		if len(llmSettings) > 0 {
			fmt.Printf("\n  LLM Settings:\n")
			for _, setting := range llmSettings {
				fmt.Printf("    Provider: %s\n", setting.ProviderName)
				if setting.SystemPrompt != "" {
					fmt.Printf("      System Prompt: %s\n", strings.ReplaceAll(setting.SystemPrompt, "\n", "\n        "))
				}
				if setting.Temperature != nil {
					fmt.Printf("      Temperature: %.2f\n", *setting.Temperature)
				}
				if setting.MaxTokens != nil {
					fmt.Printf("      Max Tokens: %d\n", *setting.MaxTokens)
				}
				if len(setting.Settings) > 0 {
					fmt.Printf("      Additional Settings: %v\n", setting.Settings)
				}
				fmt.Println()
			}
		}

		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update <name-or-id>",
	Short: "Update profile information",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if settingSvc == nil {
			return fmt.Errorf("settings service not initialized")
		}

		nameOrID := args[0]

		// Find the profile
		profiles, err := settingSvc.ListProfiles()
		if err != nil {
			return fmt.Errorf("failed to list profiles: %w", err)
		}

		var targetProfile *settings.UserProfile
		for _, p := range profiles {
			if p.Name == nameOrID || strings.HasPrefix(p.ID, nameOrID) {
				targetProfile = p
				break
			}
		}

		if targetProfile == nil {
			return fmt.Errorf("profile not found: %s", nameOrID)
		}

		req := settings.UpdateProfileRequest{}

		// Check for updates
		if cmd.Flags().Changed("name") {
			name, _ := cmd.Flags().GetString("name")
			req.Name = &name
		}
		if cmd.Flags().Changed("description") {
			desc, _ := cmd.Flags().GetString("description")
			req.Description = &desc
		}
		if cmd.Flags().Changed("system-prompt") {
			prompt, _ := cmd.Flags().GetString("system-prompt")
			req.DefaultSystemPrompt = &prompt
		}

		updatedProfile, err := settingSvc.UpdateProfile(targetProfile.ID, req)
		if err != nil {
			return fmt.Errorf("failed to update profile: %w", err)
		}

		fmt.Printf("Profile updated successfully:\n")
		fmt.Printf("  ID: %s\n", updatedProfile.ID)
		fmt.Printf("  Name: %s\n", updatedProfile.Name)
		fmt.Printf("  Description: %s\n", updatedProfile.Description)

		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name-or-id>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if settingSvc == nil {
			return fmt.Errorf("settings service not initialized")
		}

		nameOrID := args[0]

		// Find the profile
		profiles, err := settingSvc.ListProfiles()
		if err != nil {
			return fmt.Errorf("failed to list profiles: %w", err)
		}

		var targetProfile *settings.UserProfile
		for _, p := range profiles {
			if p.Name == nameOrID || strings.HasPrefix(p.ID, nameOrID) {
				targetProfile = p
				break
			}
		}

		if targetProfile == nil {
			return fmt.Errorf("profile not found: %s", nameOrID)
		}

		// Confirmation
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to delete profile '%s' (%s)? [y/N]: ", targetProfile.Name, targetProfile.ID[:8])
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				fmt.Println("Delete cancelled")
				return nil
			}
		}

		if err := settingSvc.DeleteProfile(targetProfile.ID); err != nil {
			return fmt.Errorf("failed to delete profile: %w", err)
		}

		fmt.Printf("Profile '%s' deleted successfully\n", targetProfile.Name)
		return nil
	},
}

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage LLM settings for profiles",
	Long:  `Configure provider-specific LLM settings including system prompts, temperature, and other parameters.`,
}

var llmSetCmd = &cobra.Command{
	Use:   "set <provider>",
	Short: "Set LLM settings for a provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if settingSvc == nil {
			return fmt.Errorf("settings service not initialized")
		}

		provider := args[0]
		profileName, _ := cmd.Flags().GetString("profile")

		// Get target profile
		var profile *settings.UserProfile
		var err error

		if profileName != "" {
			profiles, err := settingSvc.ListProfiles()
			if err != nil {
				return fmt.Errorf("failed to list profiles: %w", err)
			}

			for _, p := range profiles {
				if p.Name == profileName || strings.HasPrefix(p.ID, profileName) {
					profile = p
					break
				}
			}

			if profile == nil {
				return fmt.Errorf("profile not found: %s", profileName)
			}
		} else {
			profile, err = settingSvc.GetActiveProfile()
			if err != nil {
				return fmt.Errorf("failed to get active profile: %w", err)
			}
		}

		req := settings.CreateLLMSettingsRequest{
			ProfileID:    profile.ID,
			ProviderName: provider,
			Settings:     make(map[string]interface{}),
		}

		// Parse command line flags
		if cmd.Flags().Changed("system-prompt") {
			prompt, _ := cmd.Flags().GetString("system-prompt")
			req.SystemPrompt = prompt
		}
		if cmd.Flags().Changed("temperature") {
			temp, _ := cmd.Flags().GetFloat64("temperature")
			req.Temperature = &temp
		}
		if cmd.Flags().Changed("max-tokens") {
			tokens, _ := cmd.Flags().GetInt("max-tokens")
			req.MaxTokens = &tokens
		}

		// Parse additional settings
		settingsStr, _ := cmd.Flags().GetString("settings")
		if settingsStr != "" {
			if err := json.Unmarshal([]byte(settingsStr), &req.Settings); err != nil {
				return fmt.Errorf("failed to parse settings JSON: %w", err)
			}
		}

		setting, err := settingSvc.CreateOrUpdateLLMSettings(req)
		if err != nil {
			return fmt.Errorf("failed to set LLM settings: %w", err)
		}

		fmt.Printf("LLM settings updated for %s in profile '%s':\n", provider, profile.Name)
		if setting.SystemPrompt != "" {
			fmt.Printf("  System Prompt: %s\n", setting.SystemPrompt)
		}
		if setting.Temperature != nil {
			fmt.Printf("  Temperature: %.2f\n", *setting.Temperature)
		}
		if setting.MaxTokens != nil {
			fmt.Printf("  Max Tokens: %d\n", *setting.MaxTokens)
		}

		return nil
	},
}

var llmListCmd = &cobra.Command{
	Use:   "list",
	Short: "List LLM settings for profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		if settingSvc == nil {
			return fmt.Errorf("settings service not initialized")
		}

		profileName, _ := cmd.Flags().GetString("profile")

		var profiles []*settings.UserProfile

		if profileName != "" {
			allProfiles, err := settingSvc.ListProfiles()
			if err != nil {
				return fmt.Errorf("failed to list profiles: %w", err)
			}

			for _, p := range allProfiles {
				if p.Name == profileName || strings.HasPrefix(p.ID, profileName) {
					profiles = append(profiles, p)
					break
				}
			}

			if len(profiles) == 0 {
				return fmt.Errorf("profile not found: %s", profileName)
			}
		} else {
			var err error
			profiles, err = settingSvc.ListProfiles()
			if err != nil {
				return fmt.Errorf("failed to list profiles: %w", err)
			}
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PROFILE\tPROVIDER\tSYSTEM PROMPT\tTEMP\tMAX TOKENS")
		fmt.Fprintln(w, "-------\t--------\t-------------\t----\t----------")

		for _, profile := range profiles {
			settings, err := settingSvc.GetAllLLMSettings(profile.ID)
			if err != nil {
				continue
			}

			if len(settings) == 0 {
				fmt.Fprintf(w, "%s\t-\t-\t-\t-\n", profile.Name)
				continue
			}

			for _, setting := range settings {
				promptPreview := setting.SystemPrompt
				if len(promptPreview) > 50 {
					promptPreview = promptPreview[:47] + "..."
				}

				tempStr := "-"
				if setting.Temperature != nil {
					tempStr = fmt.Sprintf("%.2f", *setting.Temperature)
				}

				tokensStr := "-"
				if setting.MaxTokens != nil {
					tokensStr = strconv.Itoa(*setting.MaxTokens)
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					profile.Name, setting.ProviderName, promptPreview, tempStr, tokensStr)
			}
		}

		return w.Flush()
	},
}

func init() {
	// Create command flags
	createCmd.Flags().String("description", "", "Profile description")
	createCmd.Flags().String("system-prompt", "", "Default system prompt for the profile")
	createCmd.Flags().Bool("set-active", false, "Set this profile as the active one")

	updateCmd.Flags().String("name", "", "New profile name")
	updateCmd.Flags().String("description", "", "New profile description")
	updateCmd.Flags().String("system-prompt", "", "New default system prompt")

	deleteCmd.Flags().Bool("force", false, "Delete without confirmation")

	// LLM settings commands
	llmSetCmd.Flags().String("profile", "", "Target profile (uses active profile if not specified)")
	llmSetCmd.Flags().String("system-prompt", "", "System prompt for this provider")
	llmSetCmd.Flags().Float64("temperature", 0, "Temperature setting")
	llmSetCmd.Flags().Int("max-tokens", 0, "Maximum tokens")
	llmSetCmd.Flags().String("settings", "", "Additional settings as JSON")

	llmListCmd.Flags().String("profile", "", "Show settings for specific profile only")

	// Add subcommands to llm command
	llmCmd.AddCommand(llmSetCmd)
	llmCmd.AddCommand(llmListCmd)

	// Add all commands to the profile command
	ProfileCmd.AddCommand(listCmd)
	ProfileCmd.AddCommand(createCmd)
	ProfileCmd.AddCommand(switchCmd)
	ProfileCmd.AddCommand(showCmd)
	ProfileCmd.AddCommand(updateCmd)
	ProfileCmd.AddCommand(deleteCmd)
	ProfileCmd.AddCommand(llmCmd)
}