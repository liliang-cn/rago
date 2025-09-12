package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowSpec_JSONSerialization(t *testing.T) {
	spec := WorkflowSpec{
		Steps: []WorkflowStep{
			{
				ID:   "step1",
				Name: "First Step",
				Type: StepTypeTool,
				Tool: "test_tool",
				Inputs: map[string]interface{}{
					"param1": "value1",
					"param2": 42,
				},
				Outputs: map[string]string{
					"result": "output_var",
				},
				Timeout: 30 * time.Second,
			},
			{
				ID:        "step2",
				Name:      "Second Step",
				Type:      StepTypeDelay,
				DependsOn: []string{"step1"},
				Inputs: map[string]interface{}{
					"duration": "5s",
				},
			},
		},
		Triggers: []Trigger{
			{
				ID:   "trigger1",
				Type: TriggerTypeManual,
				Name: "Manual Trigger",
			},
		},
		Variables: map[string]interface{}{
			"global_var": "global_value",
		},
		ErrorPolicy: ErrorPolicy{
			Strategy:   ErrorStrategyFail,
			MaxRetries: 3,
			RetryDelay: 5 * time.Second,
		},
		Timeout: 10 * time.Minute,
		Metadata: WorkflowMetadata{
			Author:      "Test Author",
			Version:     "1.0.0",
			Description: "Test workflow",
			Tags:        []string{"test", "example"},
			Category:    "testing",
			Labels: map[string]string{
				"env": "test",
			},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(spec)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled WorkflowSpec
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Compare fields
	assert.Equal(t, len(spec.Steps), len(unmarshaled.Steps))
	assert.Equal(t, spec.Steps[0].ID, unmarshaled.Steps[0].ID)
	assert.Equal(t, spec.Steps[0].Name, unmarshaled.Steps[0].Name)
	assert.Equal(t, spec.Steps[0].Type, unmarshaled.Steps[0].Type)
	assert.Equal(t, spec.Steps[0].Tool, unmarshaled.Steps[0].Tool)
	assert.Equal(t, spec.Variables, unmarshaled.Variables)
	assert.Equal(t, spec.ErrorPolicy, unmarshaled.ErrorPolicy)
	assert.Equal(t, spec.Timeout, unmarshaled.Timeout)
	assert.Equal(t, spec.Metadata, unmarshaled.Metadata)
}

func TestWorkflowStep_Creation(t *testing.T) {
	step := WorkflowStep{
		ID:          "test-step-1",
		Name:        "Test Step",
		Description: "A test step for validation",
		Type:        StepTypeTool,
		Tool:        "filesystem",
		Inputs: map[string]interface{}{
			"path":      "/tmp/test",
			"operation": "read",
			"recursive": true,
		},
		Outputs: map[string]string{
			"content": "file_content",
			"size":    "file_size",
		},
		Conditions: []Condition{
			{
				Field:    "file_exists",
				Operator: "eq",
				Value:    true,
			},
		},
		Retry: RetryPolicy{
			Enabled:     true,
			MaxRetries:  3,
			Delay:       2 * time.Second,
			BackoffType: BackoffExponential,
			MaxDelay:    30 * time.Second,
		},
		Timeout:   60 * time.Second,
		DependsOn: []string{"setup-step"},
	}

	assert.Equal(t, "test-step-1", step.ID)
	assert.Equal(t, "Test Step", step.Name)
	assert.Equal(t, StepTypeTool, step.Type)
	assert.Equal(t, "filesystem", step.Tool)
	assert.Len(t, step.Inputs, 3)
	assert.Len(t, step.Outputs, 2)
	assert.Len(t, step.Conditions, 1)
	assert.True(t, step.Retry.Enabled)
	assert.Len(t, step.DependsOn, 1)
}

func TestStepType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		stepType StepType
		expected string
	}{
		{"Tool Step", StepTypeTool, "tool"},
		{"Condition Step", StepTypeCondition, "condition"},
		{"Loop Step", StepTypeLoop, "loop"},
		{"Parallel Step", StepTypeParallel, "parallel"},
		{"Delay Step", StepTypeDelay, "delay"},
		{"Variable Step", StepTypeVariable, "variable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.stepType))
		})
	}
}

func TestTrigger_Creation(t *testing.T) {
	trigger := Trigger{
		ID:   "cron-trigger-1",
		Type: TriggerTypeSchedule,
		Name: "Daily Backup",
		Conditions: []Condition{
			{
				Field:    "backup_enabled",
				Operator: "eq",
				Value:    true,
			},
		},
		Schedule: "0 2 * * *", // Daily at 2 AM
		Config: map[string]interface{}{
			"timezone": "UTC",
			"retries":  3,
		},
	}

	assert.Equal(t, "cron-trigger-1", trigger.ID)
	assert.Equal(t, TriggerTypeSchedule, trigger.Type)
	assert.Equal(t, "Daily Backup", trigger.Name)
	assert.Len(t, trigger.Conditions, 1)
	assert.Equal(t, "0 2 * * *", trigger.Schedule)
	assert.NotNil(t, trigger.Config)
}

func TestTriggerType_Constants(t *testing.T) {
	tests := []struct {
		name        string
		triggerType TriggerType
		expected    string
	}{
		{"Manual", TriggerTypeManual, "manual"},
		{"Schedule", TriggerTypeSchedule, "schedule"},
		{"Event", TriggerTypeEvent, "event"},
		{"Webhook", TriggerTypeWebhook, "webhook"},
		{"File Watch", TriggerTypeFileWatch, "file_watch"},
		{"API", TriggerTypeAPI, "api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.triggerType))
		})
	}
}

func TestCondition_Creation(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
	}{
		{
			name: "String equality condition",
			condition: Condition{
				Field:    "status",
				Operator: "eq",
				Value:    "active",
				LogicOp:  "and",
			},
		},
		{
			name: "Numeric comparison condition",
			condition: Condition{
				Field:    "count",
				Operator: "gt",
				Value:    10,
				LogicOp:  "or",
			},
		},
		{
			name: "Contains condition",
			condition: Condition{
				Field:    "tags",
				Operator: "contains",
				Value:    "important",
			},
		},
		{
			name: "Exists condition",
			condition: Condition{
				Field:    "optional_field",
				Operator: "exists",
				Value:    nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.condition.Field)
			assert.NotEmpty(t, tt.condition.Operator)
		})
	}
}

func TestErrorPolicy_Creation(t *testing.T) {
	policy := ErrorPolicy{
		Strategy:   ErrorStrategyRetry,
		MaxRetries: 5,
		RetryDelay: 10 * time.Second,
		ContinueOn: []string{"network_error", "timeout"},
		NotifyOn:   []string{"critical_error", "data_loss"},
	}

	assert.Equal(t, ErrorStrategyRetry, policy.Strategy)
	assert.Equal(t, 5, policy.MaxRetries)
	assert.Equal(t, 10*time.Second, policy.RetryDelay)
	assert.Len(t, policy.ContinueOn, 2)
	assert.Len(t, policy.NotifyOn, 2)
}

func TestErrorStrategy_Constants(t *testing.T) {
	tests := []struct {
		name     string
		strategy ErrorStrategy
		expected string
	}{
		{"Fail", ErrorStrategyFail, "fail"},
		{"Continue", ErrorStrategyContinue, "continue"},
		{"Retry", ErrorStrategyRetry, "retry"},
		{"Skip", ErrorStrategySkip, "skip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.strategy))
		})
	}
}

func TestRetryPolicy_Creation(t *testing.T) {
	policy := RetryPolicy{
		Enabled:     true,
		MaxRetries:  3,
		Delay:       2 * time.Second,
		BackoffType: BackoffExponential,
		MaxDelay:    60 * time.Second,
	}

	assert.True(t, policy.Enabled)
	assert.Equal(t, 3, policy.MaxRetries)
	assert.Equal(t, 2*time.Second, policy.Delay)
	assert.Equal(t, BackoffExponential, policy.BackoffType)
	assert.Equal(t, 60*time.Second, policy.MaxDelay)
}

func TestBackoffType_Constants(t *testing.T) {
	tests := []struct {
		name        string
		backoffType BackoffType
		expected    string
	}{
		{"Fixed", BackoffFixed, "fixed"},
		{"Exponential", BackoffExponential, "exponential"},
		{"Linear", BackoffLinear, "linear"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.backoffType))
		})
	}
}

func TestWorkflowMetadata_Creation(t *testing.T) {
	metadata := WorkflowMetadata{
		Author:      "John Doe",
		Version:     "2.1.0",
		Description: "Advanced workflow for document processing",
		Tags:        []string{"document", "processing", "ai", "automation"},
		Category:    "document-processing",
		Labels: map[string]string{
			"environment": "production",
			"team":        "ai-research",
			"priority":    "high",
		},
	}

	assert.Equal(t, "John Doe", metadata.Author)
	assert.Equal(t, "2.1.0", metadata.Version)
	assert.NotEmpty(t, metadata.Description)
	assert.Len(t, metadata.Tags, 4)
	assert.Equal(t, "document-processing", metadata.Category)
	assert.Len(t, metadata.Labels, 3)
}

func TestToolChain_Creation(t *testing.T) {
	chain := ToolChain{
		ID:   "document-analysis-chain",
		Name: "Document Analysis Chain",
		Steps: []ChainStep{
			{
				ID:       "extract",
				Name:     "Extract Text",
				ToolName: "pdf_extractor",
				Inputs: map[string]interface{}{
					"file_path": "{{input.document}}",
					"format":    "text",
				},
				Outputs: map[string]string{
					"text_content": "extracted_text",
				},
				Retry: RetryPolicy{
					Enabled:    true,
					MaxRetries: 2,
					Delay:      1 * time.Second,
				},
				Timeout: 30 * time.Second,
			},
			{
				ID:       "analyze",
				Name:     "Analyze Content",
				ToolName: "text_analyzer",
				Inputs: map[string]interface{}{
					"text": "{{steps.extract.text_content}}",
					"type": "summary",
				},
				Outputs: map[string]string{
					"summary": "document_summary",
				},
				Timeout: 60 * time.Second,
			},
		},
		Variables: map[string]interface{}{
			"output_format": "json",
			"language":      "en",
		},
		Parallel: false,
	}

	assert.Equal(t, "document-analysis-chain", chain.ID)
	assert.Equal(t, "Document Analysis Chain", chain.Name)
	assert.Len(t, chain.Steps, 2)
	assert.NotNil(t, chain.Variables)
	assert.False(t, chain.Parallel)
}

func TestChainStep_Creation(t *testing.T) {
	step := ChainStep{
		ID:       "validation-step",
		Name:     "Validate Input",
		ToolName: "validator",
		Inputs: map[string]interface{}{
			"data":   "{{input.data}}",
			"schema": "user-schema.json",
		},
		Outputs: map[string]string{
			"is_valid":    "validation_result",
			"error_count": "errors",
		},
		Conditions: []Condition{
			{
				Field:    "input.data",
				Operator: "exists",
				Value:    nil,
			},
		},
		Retry: RetryPolicy{
			Enabled:     true,
			MaxRetries:  1,
			Delay:       500 * time.Millisecond,
			BackoffType: BackoffFixed,
		},
		Timeout: 10 * time.Second,
	}

	assert.Equal(t, "validation-step", step.ID)
	assert.Equal(t, "Validate Input", step.Name)
	assert.Equal(t, "validator", step.ToolName)
	assert.Len(t, step.Inputs, 2)
	assert.Len(t, step.Outputs, 2)
	assert.Len(t, step.Conditions, 1)
	assert.True(t, step.Retry.Enabled)
	assert.Equal(t, 10*time.Second, step.Timeout)
}

func TestLoopDefinition_Creation(t *testing.T) {
	loop := LoopDefinition{
		ID:   "process-files-loop",
		Type: LoopTypeFor,
		Condition: Condition{
			Field:    "files",
			Operator: "exists",
			Value:    nil,
		},
		Iterator:      "files",
		MaxIterations: 100,
		Steps:         []string{"process-file", "validate-output"},
	}

	assert.Equal(t, "process-files-loop", loop.ID)
	assert.Equal(t, LoopTypeFor, loop.Type)
	assert.Equal(t, "files", loop.Iterator)
	assert.Equal(t, 100, loop.MaxIterations)
	assert.Len(t, loop.Steps, 2)
}

func TestLoopType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		loopType LoopType
		expected string
	}{
		{"While Loop", LoopTypeWhile, "while"},
		{"For Loop", LoopTypeFor, "for"},
		{"Count Loop", LoopTypeCount, "count"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.loopType))
		})
	}
}

func TestParallelGroup_Creation(t *testing.T) {
	group := ParallelGroup{
		ID:      "parallel-analysis",
		Name:    "Parallel Document Analysis",
		Steps:   []string{"extract-text", "extract-images", "extract-metadata"},
		WaitAll: true,
	}

	assert.Equal(t, "parallel-analysis", group.ID)
	assert.Equal(t, "Parallel Document Analysis", group.Name)
	assert.Len(t, group.Steps, 3)
	assert.True(t, group.WaitAll)
}

func TestWorkflowSpec_ComplexWorkflow(t *testing.T) {
	// Test a more complex workflow with multiple step types
	spec := WorkflowSpec{
		Steps: []WorkflowStep{
			{
				ID:   "init",
				Name: "Initialize Variables",
				Type: StepTypeVariable,
				Inputs: map[string]interface{}{
					"counter": 0,
					"status":  "starting",
				},
			},
			{
				ID:        "validate",
				Name:      "Validate Input",
				Type:      StepTypeCondition,
				DependsOn: []string{"init"},
				Conditions: []Condition{
					{
						Field:    "input.file",
						Operator: "exists",
						Value:    nil,
					},
				},
			},
			{
				ID:        "process",
				Name:      "Process File",
				Type:      StepTypeTool,
				Tool:      "file_processor",
				DependsOn: []string{"validate"},
				Inputs: map[string]interface{}{
					"file": "{{input.file}}",
				},
			},
			{
				ID:        "wait",
				Name:      "Wait for Processing",
				Type:      StepTypeDelay,
				DependsOn: []string{"process"},
				Inputs: map[string]interface{}{
					"duration": "2s",
				},
			},
		},
		Variables: map[string]interface{}{
			"max_retries": 3,
			"timeout":     "5m",
		},
		ErrorPolicy: ErrorPolicy{
			Strategy:   ErrorStrategyRetry,
			MaxRetries: 3,
			RetryDelay: 5 * time.Second,
		},
	}

	assert.Len(t, spec.Steps, 4)
	assert.Equal(t, StepTypeVariable, spec.Steps[0].Type)
	assert.Equal(t, StepTypeCondition, spec.Steps[1].Type)
	assert.Equal(t, StepTypeTool, spec.Steps[2].Type)
	assert.Equal(t, StepTypeDelay, spec.Steps[3].Type)

	// Verify dependencies
	assert.Len(t, spec.Steps[1].DependsOn, 1)
	assert.Contains(t, spec.Steps[1].DependsOn, "init")
	assert.Len(t, spec.Steps[2].DependsOn, 1)
	assert.Contains(t, spec.Steps[2].DependsOn, "validate")
}
