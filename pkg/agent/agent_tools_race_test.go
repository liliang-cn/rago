package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// Tools can be added while a turn reads them; run with -race.
func TestAddToolWhileReading(t *testing.T) {
	a := NewAgent("a")
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			a.AddToolWithMetadata(fmt.Sprintf("t%d", i), "", nil,
				func(context.Context, map[string]interface{}) (interface{}, error) { return nil, nil }, ToolMetadata{ReadOnly: true})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			_ = a.Tools()
			_ = a.HasTool("t1")
			_, _ = a.GetHandler("t1")
			_ = a.MetadataOf("t1")
			_ = a.GetToolNames()
		}
	}()
	wg.Wait()
	if got := len(a.Tools()); got != 200 {
		t.Fatalf("tools = %d, want 200", got)
	}
}
