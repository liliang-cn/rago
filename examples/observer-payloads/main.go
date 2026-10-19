// Package main shows WithObserverPayloads: the request each model call was
// sent, carried to every observer.
//
// By default an observer learns that a model call happened, how long it took,
// what it cost and what came back — and sees nothing of what went in. That is
// the right default for a trace that is mostly about time and tokens, and the
// wrong one on the day a run goes wrong: "what did the model see in round 7"
// then means reproducing round 7.
//
// agent.WithObserverPayloads() puts the request on ModelInfo.Request. With it:
//
//   - agent.NewTraceWriter writes the request on each model_end line, so the
//     line holds both sides of the exchange: messages and tool names in, text
//     length and tokens out. MaxPayloadLen clips message content if a trace
//     is wanted for sizes rather than words; the default keeps it whole.
//   - otelobserver fills llm.input_messages.<i>.message.* and
//     llm.tools.<i>.tool.json_schema on each LLM span, the OpenInference
//     shape Arize Phoenix renders as the conversation. Point the same
//     observer at otelobserver.Phoenix(ctx) and every round's prompt is one
//     click away in the trace view.
//
// The request holds the user's own words and whatever a tool read, and on a
// long run it is most of a trace's bytes. That is why it is opt-in, and why a
// host that turns it on decides where the exporter sends it.
//
// Usage:
//
//	go run ./examples/observer-payloads
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/agent-go/v3/pkg/agent"
)

type clockParams struct{}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	dir, err := os.MkdirTemp("", "agentgo-payloads-*")
	if err != nil {
		log.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	traceFile, err := os.Create(filepath.Join(dir, "run.jsonl"))
	if err != nil {
		log.Fatalf("create trace: %v", err)
	}
	defer traceFile.Close()

	trace := agent.NewTraceWriter(traceFile)
	// trace.MaxPayloadLen = 200 // clip each message's content in the request

	svc, err := agent.New("payloads-demo").
		WithTool(agent.NewTool("clock", "Current time", func(context.Context, *clockParams) (any, error) {
			return map[string]any{"now": time.Now().Format(time.RFC3339)}, nil
		})).
		WithObserver(trace).
		WithObserverPayloads(). // the one line this example is about
		Build()
	if err != nil {
		log.Fatalf("build failed: %v", err)
	}
	defer svc.Close()

	result, err := svc.Run(ctx, "What time is it? Use the clock tool, then answer in one line.")
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}
	fmt.Printf("answer: %s\n\n", result.Text())

	if err := traceFile.Sync(); err != nil {
		log.Fatalf("sync: %v", err)
	}
	raw, err := os.ReadFile(traceFile.Name())
	if err != nil {
		log.Fatalf("read trace: %v", err)
	}

	// Print what each answering turn was sent: role and the start of every
	// message, and the tools offered. This is the part no observer could see
	// before.
	for _, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		var obj struct {
			Event   string `json:"event"`
			Purpose string `json:"purpose"`
			Round   int    `json:"round"`
			Request *struct {
				Prompt   string   `json:"prompt"`
				Tools    []string `json:"tools"`
				Messages []struct {
					Role      string `json:"role"`
					Content   string `json:"content"`
					ToolCalls []struct {
						Name string `json:"name"`
					} `json:"tool_calls"`
				} `json:"messages"`
			} `json:"request"`
		}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			log.Fatalf("trace line is not JSON: %v", err)
		}
		if obj.Event != "model_end" || obj.Request == nil {
			continue
		}
		fmt.Printf("=== %s round %d\n", obj.Purpose, obj.Round)
		if obj.Request.Prompt != "" {
			fmt.Printf("  prompt: %s\n", head(obj.Request.Prompt, 100))
		}
		for _, m := range obj.Request.Messages {
			text := head(m.Content, 80)
			for _, tc := range m.ToolCalls {
				text += fmt.Sprintf(" [call %s]", tc.Name)
			}
			fmt.Printf("  %-9s %s\n", m.Role, text)
		}
		if len(obj.Request.Tools) > 0 {
			fmt.Printf("  tools: %s\n", strings.Join(obj.Request.Tools, ", "))
		}
	}
}

func head(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
