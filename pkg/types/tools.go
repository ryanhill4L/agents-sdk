package types

import (
	"context"
	"encoding/json"
)

type Tool interface {
	Name() string
	Description() string
	Schema() ToolSchema
	Execute(ctx context.Context, args map[string]any) (any, error)
	RequiresPermission() bool
}

type ToolSchema struct {
	Type        string                `json:"type"`
	Properties  map[string]Property   `json:"properties"`
	Required    []string              `json:"required,omitempty"`
	Description string                `json:"description,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Items       *Property `json:"items,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Required    []string `json:"required,omitempty"`
}

type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    any    `json:"content"`
	IsError    bool   `json:"is_error"`
	Error      string `json:"error,omitempty"`
}

func (tr *ToolResult) String() string {
	if tr.IsError {
		return tr.Error
	}

	switch v := tr.Content.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

type ToolDefinition struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Schema      ToolSchema `json:"input_schema"`
}

func ToolToDefinition(tool Tool) ToolDefinition {
	return ToolDefinition{
		Name:        tool.Name(),
		Description: tool.Description(),
		Schema:      tool.Schema(),
	}
}

type ToolPermission struct {
	ToolName   string
	Operation  string
	Arguments  map[string]any
	Allowed    bool
	Reason     string
}

type ToolExecutor interface {
	ExecuteTool(ctx context.Context, toolCall ToolCall) (*ToolResult, error)
	GetAvailableTools() []Tool
	RegisterTool(tool Tool) error
	UnregisterTool(name string) error
}