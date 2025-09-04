package rago

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/spf13/cobra"
)

var workflowTemplateCmd = &cobra.Command{
	Use:   "generate-template [description]",
	Short: "Generate a workflow from templates without LLM",
	Long: `Generate a workflow JSON file from templates based on keyword matching.
This command doesn't require LLM services and uses pre-defined templates.

Available templates:
  - Website Monitor: Track changes on websites
  - File Processor: Batch process files in directories
  - Data Fetcher: Retrieve and save data from APIs
  - Report Generator: Analyze data and create reports
  - Backup Automation: Create automated backups

Examples:
  rago agent generate-template "monitor https://example.com for changes"
  rago agent generate-template "backup important files daily"
  rago agent generate-template "process all JSON files"
  rago agent generate-template "fetch data from api"`,
	Aliases: []string{"template", "gen-template"},
	Args:    cobra.ArbitraryArgs,
	RunE:    generateFromTemplate,
}

// WorkflowTemplate represents a workflow pattern
type WorkflowTemplate struct {
	Name        string
	Description string
	Keywords    []string
	Generator   func(params map[string]string) *types.WorkflowSpec
}

var templates = []WorkflowTemplate{
	{
		Name:        "Website Monitor",
		Description: "Monitor a website for changes",
		Keywords:    []string{"monitor", "website", "watch", "changes", "track", "url", "http"},
		Generator:   generateWebsiteMonitor,
	},
	{
		Name:        "File Processor",
		Description: "Process files in a directory",
		Keywords:    []string{"process", "files", "directory", "batch", "transform", "folder"},
		Generator:   generateFileProcessor,
	},
	{
		Name:        "Data Fetcher",
		Description: "Fetch and save data from APIs",
		Keywords:    []string{"fetch", "api", "download", "get", "retrieve", "save", "data"},
		Generator:   generateDataFetcher,
	},
	{
		Name:        "Report Generator",
		Description: "Generate reports from data",
		Keywords:    []string{"report", "analyze", "summary", "generate", "create", "analysis"},
		Generator:   generateReportGenerator,
	},
	{
		Name:        "Backup Automation",
		Description: "Automated backup workflow",
		Keywords:    []string{"backup", "archive", "save", "copy", "store", "preserve"},
		Generator:   generateBackupWorkflow,
	},
}

func init() {
	agentCmd.AddCommand(workflowTemplateCmd)

	workflowTemplateCmd.Flags().StringP("output", "o", "", "Output file path (default: template_name_workflow.json)")
	workflowTemplateCmd.Flags().BoolP("list", "l", false, "List available templates")
	workflowTemplateCmd.Flags().BoolP("execute", "e", false, "Execute the workflow immediately after generation")
}

func generateFromTemplate(cmd *cobra.Command, args []string) error {
	// Check if listing templates
	list, _ := cmd.Flags().GetBool("list")
	if list {
		return listTemplates()
	}

	// Require arguments if not listing
	if len(args) == 0 {
		return fmt.Errorf("please provide a description or use --list to see available templates")
	}

	description := strings.ToLower(strings.Join(args, " "))
	outputPath, _ := cmd.Flags().GetString("output")
	execute, _ := cmd.Flags().GetBool("execute")

	fmt.Printf("🤖 Generating workflow from template for: %s\n\n", description)

	// Find matching template
	template := findBestTemplate(description)
	if template == nil {
		fmt.Println("❌ No matching template found. Using generic workflow...")
		fmt.Println("💡 Tip: Use 'rago agent generate' for LLM-powered generation of custom workflows")
		template = &WorkflowTemplate{
			Name:      "Generic Workflow",
			Generator: generateGenericWorkflow,
		}
	}

	fmt.Printf("📋 Using template: %s\n", template.Name)
	if template.Description != "" {
		fmt.Printf("   Description: %s\n", template.Description)
	}

	// Extract parameters from description
	params := extractParameters(description)

	// Generate workflow
	workflow := template.Generator(params)

	// Determine output path
	if outputPath == "" {
		safeName := strings.ReplaceAll(strings.ToLower(template.Name), " ", "_")
		outputPath = fmt.Sprintf("%s_workflow.json", safeName)
	}

	// Save workflow
	workflowJSON, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	if err := os.WriteFile(outputPath, workflowJSON, 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	fmt.Printf("\n✅ Workflow generated successfully!\n")
	fmt.Printf("📁 Saved to: %s\n", outputPath)

	// Display summary
	fmt.Printf("\n📊 Workflow Summary:\n")
	fmt.Printf("  - Steps: %d\n", len(workflow.Steps))
	for i, step := range workflow.Steps {
		fmt.Printf("    %d. %s (%s)\n", i+1, step.Name, step.Type)
	}

	// Execute if requested
	if execute {
		fmt.Println("\n🚀 Executing generated workflow...")
		return executeGeneratedWorkflow(outputPath)
	}

	fmt.Printf("\n💡 To use this workflow:\n")
	fmt.Printf("  rago agent create --name \"My Agent\" --type workflow --workflow-file %s\n", outputPath)
	fmt.Printf("  rago agent execute [agent-id]\n")

	return nil
}

func listTemplates() error {
	fmt.Println("📚 Available Workflow Templates")
	fmt.Println("=" + strings.Repeat("=", 50))

	for i, template := range templates {
		fmt.Printf("\n%d. %s\n", i+1, template.Name)
		if template.Description != "" {
			fmt.Printf("   %s\n", template.Description)
		}
		fmt.Printf("   Keywords: %s\n", strings.Join(template.Keywords, ", "))
	}

	fmt.Println("\n💡 Usage examples:")
	fmt.Println("  rago agent generate-template \"monitor website changes\"")
	fmt.Println("  rago agent generate-template \"backup files daily\"")
	fmt.Println("  rago agent generate-template \"process JSON files in folder\"")

	return nil
}

func findBestTemplate(description string) *WorkflowTemplate {
	var bestMatch *WorkflowTemplate
	maxScore := 0

	for i := range templates {
		score := 0
		for _, keyword := range templates[i].Keywords {
			if strings.Contains(description, keyword) {
				score++
			}
		}
		if score > maxScore {
			maxScore = score
			bestMatch = &templates[i]
		}
	}

	if maxScore == 0 {
		return nil
	}
	return bestMatch
}

func extractParameters(description string) map[string]string {
	params := make(map[string]string)

	// Extract URLs
	if strings.Contains(description, "http://") || strings.Contains(description, "https://") {
		words := strings.Fields(description)
		for _, word := range words {
			if strings.HasPrefix(word, "http") {
				params["url"] = word
				break
			}
		}
	}

	// Extract file patterns
	patterns := []string{".json", ".txt", ".md", ".go", ".py", ".js", ".csv", ".yaml", ".xml"}
	for _, pattern := range patterns {
		if strings.Contains(description, pattern) {
			params["file_pattern"] = "*" + pattern
			break
		}
	}

	// Extract time intervals
	timeWords := map[string]string{
		"hourly":  "0 * * * *",
		"daily":   "0 0 * * *",
		"weekly":  "0 0 * * 0",
		"monthly": "0 0 1 * *",
		"minute":  "* * * * *",
		"hour":    "0 * * * *",
		"day":     "0 0 * * *",
	}
	for word, cron := range timeWords {
		if strings.Contains(description, word) {
			params["schedule"] = cron
			break
		}
	}

	// Extract directory paths
	if strings.Contains(description, "/") {
		words := strings.Fields(description)
		for _, word := range words {
			if strings.Contains(word, "/") && !strings.HasPrefix(word, "http") {
				params["path"] = word
				break
			}
		}
	}

	return params
}

// Template Generators

func generateWebsiteMonitor(params map[string]string) *types.WorkflowSpec {
	url := params["url"]
	if url == "" {
		url = "{{input.url}}"
	}

	return &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "fetch_page",
				Name: "Fetch Website",
				Type: types.StepTypeTool,
				Tool: "fetch",
				Inputs: map[string]interface{}{
					"url": url,
				},
				Outputs: map[string]string{
					"content": "current_content",
				},
			},
			{
				ID:   "get_previous",
				Name: "Get Previous Version",
				Type: types.StepTypeTool,
				Tool: "memory",
				Inputs: map[string]interface{}{
					"action": "retrieve",
					"key":    fmt.Sprintf("website_%s", strings.ReplaceAll(url, "/", "_")),
				},
				Outputs: map[string]string{
					"value": "previous_content",
				},
			},
			{
				ID:   "compare",
				Name: "Compare Versions",
				Type: types.StepTypeTool,
				Tool: "sequential-thinking",
				Inputs: map[string]interface{}{
					"task":     "Compare these two versions and identify changes",
					"current":  "{{current_content}}",
					"previous": "{{previous_content}}",
				},
				Outputs: map[string]string{
					"changes": "detected_changes",
				},
			},
			{
				ID:   "store_current",
				Name: "Store Current Version",
				Type: types.StepTypeTool,
				Tool: "memory",
				Inputs: map[string]interface{}{
					"action": "store",
					"key":    fmt.Sprintf("website_%s", strings.ReplaceAll(url, "/", "_")),
					"value":  "{{current_content}}",
				},
			},
			{
				ID:   "save_changes",
				Name: "Save Changes Log",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "append",
					"path":    "./website_changes.log",
					"content": "Changes detected:\n{{detected_changes}}\n---\n",
				},
			},
		},
		Variables: map[string]interface{}{
			"url": url,
		},
	}
}

func generateFileProcessor(params map[string]string) *types.WorkflowSpec {
	pattern := params["file_pattern"]
	if pattern == "" {
		pattern = "*.*"
	}

	path := params["path"]
	if path == "" {
		path = "{{input.directory}}"
	}

	return &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "list_files",
				Name: "List Files",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "list",
					"path":    path,
					"pattern": pattern,
				},
				Outputs: map[string]string{
					"files": "file_list",
				},
			},
			{
				ID:   "process_loop",
				Name: "Process Each File",
				Type: types.StepTypeLoop,
				Inputs: map[string]interface{}{
					"items": "{{file_list}}",
				},
			},
			{
				ID:   "read_file",
				Name: "Read File",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action": "read",
					"path":   "{{current_file}}",
				},
				Outputs: map[string]string{
					"content": "file_content",
				},
				DependsOn: []string{"process_loop"},
			},
			{
				ID:   "process_content",
				Name: "Process Content",
				Type: types.StepTypeTool,
				Tool: "sequential-thinking",
				Inputs: map[string]interface{}{
					"task":    "Process and analyze this content",
					"content": "{{file_content}}",
				},
				Outputs: map[string]string{
					"result": "processed_content",
				},
				DependsOn: []string{"read_file"},
			},
			{
				ID:   "save_result",
				Name: "Save Processed Result",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "write",
					"path":    "{{current_file}}.processed",
					"content": "{{processed_content}}",
				},
				DependsOn: []string{"process_content"},
			},
		},
		Variables: map[string]interface{}{
			"directory":    path,
			"file_pattern": pattern,
		},
	}
}

func generateDataFetcher(params map[string]string) *types.WorkflowSpec {
	url := params["url"]
	if url == "" {
		url = "{{input.api_url}}"
	}

	return &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "fetch_data",
				Name: "Fetch Data from API",
				Type: types.StepTypeTool,
				Tool: "fetch",
				Inputs: map[string]interface{}{
					"url":    url,
					"method": "GET",
				},
				Outputs: map[string]string{
					"body":   "api_data",
					"status": "http_status",
				},
			},
			{
				ID:   "process_data",
				Name: "Process API Response",
				Type: types.StepTypeTool,
				Tool: "sequential-thinking",
				Inputs: map[string]interface{}{
					"task": "Extract and format the important data from this API response",
					"data": "{{api_data}}",
				},
				Outputs: map[string]string{
					"processed": "clean_data",
				},
			},
			{
				ID:   "get_timestamp",
				Name: "Get Timestamp",
				Type: types.StepTypeTool,
				Tool: "time",
				Inputs: map[string]interface{}{
					"action": "now",
					"format": "2006-01-02_15-04-05",
				},
				Outputs: map[string]string{
					"time": "timestamp",
				},
			},
			{
				ID:   "save_data",
				Name: "Save Data to File",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "write",
					"path":    "./data/fetch_{{timestamp}}.json",
					"content": "{{clean_data}}",
				},
			},
		},
		Variables: map[string]interface{}{
			"api_url": url,
		},
	}
}

func generateReportGenerator(params map[string]string) *types.WorkflowSpec {
	path := params["path"]
	if path == "" {
		path = "{{input.data_directory}}"
	}

	pattern := params["file_pattern"]
	if pattern == "" {
		pattern = "*.json"
	}

	return &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "collect_data",
				Name: "Collect Data Sources",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "list",
					"path":    path,
					"pattern": pattern,
				},
				Outputs: map[string]string{
					"files": "data_files",
				},
			},
			{
				ID:   "aggregate_data",
				Name: "Aggregate All Data",
				Type: types.StepTypeVariable,
				Inputs: map[string]interface{}{
					"files": "{{data_files}}",
				},
				Outputs: map[string]string{
					"files": "aggregated_list",
				},
			},
			{
				ID:   "analyze_data",
				Name: "Analyze and Generate Report",
				Type: types.StepTypeTool,
				Tool: "sequential-thinking",
				Inputs: map[string]interface{}{
					"task":  "Analyze this data and generate a comprehensive report with insights and recommendations",
					"data":  "{{aggregated_list}}",
					"style": "professional",
				},
				Outputs: map[string]string{
					"report": "final_report",
				},
			},
			{
				ID:   "get_timestamp",
				Name: "Get Report Timestamp",
				Type: types.StepTypeTool,
				Tool: "time",
				Inputs: map[string]interface{}{
					"action": "now",
					"format": "2006-01-02_15-04-05",
				},
				Outputs: map[string]string{
					"time": "timestamp",
				},
			},
			{
				ID:   "save_report",
				Name: "Save Report",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "write",
					"path":    "./reports/report_{{timestamp}}.md",
					"content": "# Generated Report\n\n{{final_report}}\n\n---\nGenerated at {{timestamp}}",
				},
			},
		},
		Variables: map[string]interface{}{
			"data_directory": path,
			"file_pattern":   pattern,
		},
	}
}

func generateBackupWorkflow(params map[string]string) *types.WorkflowSpec {
	sourcePath := params["path"]
	if sourcePath == "" {
		sourcePath = "{{input.source_directory}}"
	}

	return &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "list_files",
				Name: "List Files to Backup",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action": "list",
					"path":   sourcePath,
				},
				Outputs: map[string]string{
					"files": "backup_files",
				},
			},
			{
				ID:   "create_timestamp",
				Name: "Create Backup Timestamp",
				Type: types.StepTypeTool,
				Tool: "time",
				Inputs: map[string]interface{}{
					"action": "now",
					"format": "2006-01-02_15-04-05",
				},
				Outputs: map[string]string{
					"time": "backup_date",
				},
			},
			{
				ID:   "create_backup_dir",
				Name: "Create Backup Directory",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action": "mkdir",
					"path":   "./backups/backup_{{backup_date}}",
				},
			},
			{
				ID:   "copy_files",
				Name: "Copy Files to Backup",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":      "copy",
					"source":      sourcePath,
					"destination": "./backups/backup_{{backup_date}}",
				},
			},
			{
				ID:   "log_backup",
				Name: "Log Backup Completion",
				Type: types.StepTypeTool,
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "append",
					"path":    "./backup.log",
					"content": "Backup completed at {{backup_date}} - Files: {{backup_files}}\n",
				},
			},
		},
		Variables: map[string]interface{}{
			"source_directory": sourcePath,
		},
		Triggers: []types.Trigger{
			{
				Type:     types.TriggerTypeSchedule,
				Schedule: params["schedule"],
				Name:     "Scheduled Backup",
			},
		},
	}
}

func generateGenericWorkflow(params map[string]string) *types.WorkflowSpec {
	return &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "step1",
				Name: "Initialize Process",
				Type: types.StepTypeVariable,
				Inputs: map[string]interface{}{
					"status": "started",
				},
				Outputs: map[string]string{
					"status": "process_status",
				},
			},
			{
				ID:   "step2",
				Name: "Main Processing",
				Type: types.StepTypeTool,
				Tool: "sequential-thinking",
				Inputs: map[string]interface{}{
					"task":  "Process the input according to requirements",
					"input": "{{input.data}}",
				},
				Outputs: map[string]string{
					"result": "processed_data",
				},
			},
			{
				ID:   "step3",
				Name: "Store Results",
				Type: types.StepTypeTool,
				Tool: "memory",
				Inputs: map[string]interface{}{
					"action": "store",
					"key":    "result",
					"value":  "{{processed_data}}",
				},
			},
		},
		Variables: map[string]interface{}{
			"input": "{{user_provided}}",
		},
	}
}
