package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

// pendingHandoff captures a handoff tool call the runner needs to act on.
type pendingHandoff struct {
	call ToolCall
	sub  *Agent
	mode HandoffMode
	task string
}

// toolMessage builds a tool-result message for the given tool call id.
func toolMessage(toolCallID, content string) Message {
	return Message{
		Role:      "tool",
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"tool_call_id": toolCallID},
	}
}

// toolContent renders a tool response (or its error) as message content.
func toolContent(resp ToolResponse) string {
	if resp.Error != nil {
		return fmt.Sprintf("Error: %v", resp.Error)
	}
	return fmt.Sprintf("%v", resp.Content)
}

// buildHandoffMessages constructs the message list a delegated subagent runs
// with. history is the parent conversation up to (but excluding) the assistant
// turn that triggered the handoff.
func buildHandoffMessages(mode HandoffMode, history []Message, task string) []Message {
	switch mode {
	case ContextForked:
		forked := make([]Message, len(history))
		copy(forked, history)
		if task != "" {
			forked = append(forked, Message{Role: "user", Content: task, Timestamp: time.Now()})
		}
		return forked
	default: // ContextFresh
		if task == "" {
			task = lastUserContent(history)
		}
		return []Message{{Role: "user", Content: task, Timestamp: time.Now()}}
	}
}

// lastUserContent returns the content of the most recent user message.
func lastUserContent(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

// HandoffMode controls how context is passed when an agent hands off to a
// subagent, and whether control returns to the parent.
type HandoffMode int

const (
	// ContextShared transfers control to the subagent, which continues with the
	// parent's full live conversation. Control does not return to the parent.
	// This is the default and matches a classic "you take it from here" handoff.
	ContextShared HandoffMode = iota

	// ContextFresh delegates a task to the subagent in a clean context (it sees
	// only the task, no parent history). The subagent's result is returned to
	// the parent, which then resumes. Use when you want isolation with no
	// context bleed.
	ContextFresh

	// ContextForked delegates to the subagent with a copy of the parent's
	// conversation, so it has full context but runs independently. Its result is
	// returned to the parent, which resumes unaffected. Use for independent
	// exploration that shouldn't mutate the parent's state.
	ContextForked
)

// String returns the lowercase name of the mode.
func (m HandoffMode) String() string {
	switch m {
	case ContextShared:
		return "shared"
	case ContextFresh:
		return "fresh"
	case ContextForked:
		return "forked"
	default:
		return "unknown"
	}
}

// ParseHandoffMode converts a string (shared/fresh/forked, empty = shared) to a
// HandoffMode.
func ParseHandoffMode(s string) (HandoffMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "shared":
		return ContextShared, nil
	case "fresh":
		return ContextFresh, nil
	case "forked":
		return ContextForked, nil
	default:
		return ContextShared, fmt.Errorf("unknown handoff mode %q (want shared, fresh, or forked)", s)
	}
}

// handoffToolName derives the deterministic tool name a model calls to hand off
// to the named subagent.
func handoffToolName(agentName string) string {
	var b strings.Builder
	b.WriteString("handoff_")
	for _, r := range strings.ToLower(agentName) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// HandoffForTool resolves a handoff tool-call name to its target subagent and
// mode. It reports false when toolName is not a handoff tool for this agent.
func (a *Agent) HandoffForTool(toolName string) (*Agent, HandoffMode, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, h := range a.Handoffs {
		if handoffToolName(h.Name) == toolName {
			return h, a.handoffModes[h.Name], true
		}
	}
	return nil, ContextShared, false
}

// GetHandoffMode returns the configured mode for a handoff target (defaults to
// ContextShared).
func (a *Agent) GetHandoffMode(name string) HandoffMode {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.handoffModes[name]
}

// newHandoffTool builds the tool a model calls to hand off to sub. The runner
// intercepts calls to this tool, so its handler is only a safe fallback.
func newHandoffTool(sub *Agent, mode HandoffMode) tools.Tool {
	desc := fmt.Sprintf(
		"Hand off to the %q subagent (%s context). Use this when the request is better handled by that agent. Provide the task or reason in 'task'.",
		sub.Name, mode,
	)
	return tools.NewTool(
		handoffToolName(sub.Name),
		desc,
		func(_ context.Context, args map[string]interface{}) (interface{}, error) {
			task, _ := args["task"].(string)
			return fmt.Sprintf("Handing off to %s: %s", sub.Name, task), nil
		},
		tools.Param{
			Name:        "task",
			Type:        "string",
			Description: "The task or reason for the handoff.",
			Required:    true,
		},
	)
}
