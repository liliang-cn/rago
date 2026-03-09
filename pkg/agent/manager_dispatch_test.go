package agent

import "testing"

func TestExtractDispatchTextPrefersMeaningfulMetadata(t *testing.T) {
	res := &ExecutionResult{
		FinalResult: "The task has been completed.",
		Metadata: map[string]interface{}{
			"dispatch_result": "2024年欧冠冠军是皇家马德里。",
		},
	}

	got := extractDispatchText(res)
	want := "2024年欧冠冠军是皇家马德里。"
	if got != want {
		t.Fatalf("extractDispatchText() = %q, want %q", got, want)
	}
}

func TestExtractDispatchTextFallsBackToFinalResult(t *testing.T) {
	res := &ExecutionResult{FinalResult: "print_current_dir.py has been created successfully."}

	got := extractDispatchText(res)
	want := "print_current_dir.py has been created successfully."
	if got != want {
		t.Fatalf("extractDispatchText() = %q, want %q", got, want)
	}
}
