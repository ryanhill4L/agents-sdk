package agents

import (
	"context"
	"sync"
	"testing"

	"github.com/ryanhill4L/agents-sdk/pkg/providers"
)

// scriptedProvider returns canned completions keyed by agent name and call
// count, and records the messages each agent was invoked with.
type scriptedProvider struct {
	mu       sync.Mutex
	calls    map[string]int
	seenMsgs map[string][]providers.Message
}

func newScriptedProvider() *scriptedProvider {
	return &scriptedProvider{calls: map[string]int{}, seenMsgs: map[string][]providers.Message{}}
}

func (p *scriptedProvider) Complete(_ context.Context, agent providers.Agent, messages []providers.Message, _ []providers.ToolDefinition) (*providers.Completion, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	name := agent.GetName()
	n := p.calls[name]
	p.calls[name]++
	if _, ok := p.seenMsgs[name]; !ok {
		p.seenMsgs[name] = messages
	}

	switch name {
	case "router":
		if n == 0 {
			return &providers.Completion{
				Message: providers.Message{Role: "assistant"},
				ToolCalls: []providers.ToolCall{{
					ID:        "call_1",
					Name:      "handoff_worker",
					Arguments: map[string]interface{}{"task": "do the thing"},
				}},
			}, nil
		}
		return &providers.Completion{Message: providers.Message{Role: "assistant", Content: "router final"}}, nil
	case "worker":
		return &providers.Completion{Message: providers.Message{Role: "assistant", Content: "worker handled it"}}, nil
	default:
		return &providers.Completion{Message: providers.Message{Role: "assistant", Content: "?"}}, nil
	}
}

func runHandoff(t *testing.T, mode HandoffMode) (*RunResult, *scriptedProvider) {
	t.Helper()
	worker := NewAgent("worker")
	router := NewAgent("router", WithHandoff(worker, mode))

	provider := newScriptedProvider()
	runner := NewRunner(WithProvider(provider))

	result, err := runner.Run(context.Background(), router, "start")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return result, provider
}

func TestHandoffFreshContext(t *testing.T) {
	result, provider := runHandoff(t, ContextFresh)

	if result.FinalOutput != "router final" {
		t.Errorf("final output = %v, want 'router final'", result.FinalOutput)
	}
	if result.Metrics.Handoffs == 0 {
		t.Error("expected at least one handoff in metrics")
	}

	// Fresh: the worker sees only the task, not the parent's history.
	seen := provider.seenMsgs["worker"]
	if len(seen) != 1 || seen[0].Content != "do the thing" {
		t.Errorf("worker context = %+v, want single 'do the thing' message", seen)
	}
}

func TestHandoffForkedContext(t *testing.T) {
	_, provider := runHandoff(t, ContextForked)

	// Forked: the worker sees a copy of the parent history plus the task.
	seen := provider.seenMsgs["worker"]
	if len(seen) != 2 {
		t.Fatalf("worker context length = %d, want 2; msgs=%+v", len(seen), seen)
	}
	if seen[0].Content != "start" {
		t.Errorf("forked context[0] = %q, want 'start'", seen[0].Content)
	}
	if seen[1].Content != "do the thing" {
		t.Errorf("forked context[1] = %q, want 'do the thing'", seen[1].Content)
	}
}

func TestHandoffSharedContextTransfersControl(t *testing.T) {
	result, _ := runHandoff(t, ContextShared)

	// Shared: control transfers to the worker, whose output becomes the result.
	if result.FinalOutput != "worker handled it" {
		t.Errorf("final output = %v, want 'worker handled it'", result.FinalOutput)
	}
	if result.Agent == nil || result.Agent.Name != "worker" {
		t.Errorf("final agent = %v, want worker", result.Agent)
	}
}

func TestHandoffToolExposed(t *testing.T) {
	worker := NewAgent("worker")
	router := NewAgent("router", WithHandoff(worker, ContextFresh))

	var found bool
	for _, tool := range router.EffectiveTools() {
		if tool.Name() == "handoff_worker" {
			found = true
		}
	}
	if !found {
		t.Error("expected handoff_worker tool in router.EffectiveTools()")
	}
}

func TestParseHandoffMode(t *testing.T) {
	cases := map[string]HandoffMode{"": ContextShared, "shared": ContextShared, "fresh": ContextFresh, "forked": ContextForked}
	for in, want := range cases {
		got, err := ParseHandoffMode(in)
		if err != nil || got != want {
			t.Errorf("ParseHandoffMode(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseHandoffMode("bogus"); err == nil {
		t.Error("expected error for bogus mode")
	}
}
