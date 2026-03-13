package main

import "testing"

func TestBuildDelegatedTaskInstruction(t *testing.T) {
	tasks := []delegatedTask{
		{AgentName: "Assistant", Instruction: "first"},
		{AgentName: "Writer", Instruction: "second"},
	}

	if got := buildDelegatedTaskInstruction(tasks, 0, "ignored"); got != "first" {
		t.Fatalf("first task should keep original instruction, got %q", got)
	}

	got := buildDelegatedTaskInstruction(tasks, 1, "result from first")
	want := "Previous result from @Assistant:\nresult from first\n\nYour task:\nsecond"
	if got != want {
		t.Fatalf("unexpected chained instruction:\nwant: %q\ngot:  %q", want, got)
	}
}
