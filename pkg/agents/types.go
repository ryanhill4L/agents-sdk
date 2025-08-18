package agents

import (
	"context"
	"encoding/json"
	"time"
)

// Message represents a conversation message
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	ToolCalls []ToolCall             `json:"tool_calls,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResponse represents a tool execution result
type ToolResponse struct {
	ToolCallID string      `json:"tool_call_id"`
	Content    interface{} `json:"content"`
	Error      error       `json:"error,omitempty"`
}

// HandoffRequest represents an agent handoff
type HandoffRequest struct {
	TargetAgent string                 `json:"target_agent"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
}

// RunContext holds runtime information
type RunContext struct {
	context.Context
	SessionID   string
	TraceID     string
	CurrentTurn int
	MaxTurns    int
	Variables   map[string]interface{}
}

// OutputSchema defines structured output types
type OutputSchema interface {
	Validate(interface{}) error
	Schema() json.RawMessage
}
