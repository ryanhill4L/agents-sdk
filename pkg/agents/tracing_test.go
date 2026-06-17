package agents

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// toolThenText calls a named tool on the first turn, then returns final text.
type toolThenText struct {
	mu       sync.Mutex
	calls    int
	toolName string
}

func (p *toolThenText) Complete(_ context.Context, _ providers.Agent, _ []providers.Message, _ []providers.ToolDefinition) (*providers.Completion, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := p.calls
	p.calls++
	if n == 0 {
		return &providers.Completion{
			Message:   providers.Message{Role: "assistant"},
			ToolCalls: []providers.ToolCall{{ID: "c1", Name: p.toolName, Arguments: map[string]interface{}{}}},
			Usage:     providers.Usage{TotalTokens: 7},
		}, nil
	}
	return &providers.Completion{Message: providers.Message{Role: "assistant", Content: "done"}}, nil
}

func spanNames(records []tracing.SpanRecord) map[string]int {
	counts := map[string]int{}
	for _, r := range records {
		counts[r.Name]++
	}
	return counts
}

func TestRunRecordsTraceTree(t *testing.T) {
	pingTool := tools.NewTool("ping", "pings", func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
		return "pong", nil
	})
	agent := NewAgent("traced", WithTools(pingTool))

	result, err := NewRunner(WithProvider(&toolThenText{toolName: "ping"})).Run(context.Background(), agent, "go")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	counts := spanNames(result.Traces)
	for _, name := range []string{"agent.run", "turn", "llm.complete", "tool.execute"} {
		if counts[name] == 0 {
			t.Errorf("expected a %q span, traces=%v", name, counts)
		}
	}

	// The root run span is at depth 0; tool.execute carries the tool attribute.
	var sawRootDepth0, sawToolAttr bool
	for _, r := range result.Traces {
		if r.Name == "agent.run" && r.Depth == 0 {
			sawRootDepth0 = true
		}
		if r.Name == "tool.execute" {
			for _, a := range r.Attributes {
				if a.Key == "tool" && a.Value == "ping" {
					sawToolAttr = true
				}
			}
		}
	}
	if !sawRootDepth0 {
		t.Error("expected agent.run at depth 0")
	}
	if !sawToolAttr {
		t.Error("expected tool.execute span with tool=ping attribute")
	}
}

func TestRunWithLoggerEmitsRecords(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	agent := NewAgent("logged")
	_, err := NewRunner(
		WithProvider(&toolThenText{toolName: "none"}), // no such tool; returns text on turn 2
		WithLogger(logger),
	).Run(context.Background(), agent, "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	out := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("run started")) {
		t.Errorf("expected 'run started' log, got: %s", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("run completed")) {
		t.Errorf("expected 'run completed' log, got: %s", out)
	}
}
