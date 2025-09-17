package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/ryanhill4L/agents-sdk/pkg/errors"
	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

type Executor struct {
	mu              sync.RWMutex
	tools           map[string]types.Tool
	permManager     types.PermissionManager
	hookManager     types.HookManager
}

func NewExecutor(permManager types.PermissionManager, hookManager types.HookManager) *Executor {
	return &Executor{
		tools:       make(map[string]types.Tool),
		permManager: permManager,
		hookManager: hookManager,
	}
}

func (e *Executor) RegisterTool(tool types.Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool must have a name")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.tools[name]; exists {
		return fmt.Errorf("tool with name '%s' already registered", name)
	}

	e.tools[name] = tool
	return nil
}

func (e *Executor) UnregisterTool(name string) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.tools[name]; !exists {
		return fmt.Errorf("tool with name '%s' not found", name)
	}

	delete(e.tools, name)
	return nil
}

func (e *Executor) ExecuteTool(ctx context.Context, toolCall types.ToolCall) (*types.ToolResult, error) {
	e.mu.RLock()
	tool, exists := e.tools[toolCall.Name]
	e.mu.RUnlock()

	if !exists {
		return &types.ToolResult{
			ToolCallID: toolCall.ID,
			IsError:    true,
			Error:      fmt.Sprintf("tool '%s' not found", toolCall.Name),
		}, errors.ErrToolNotFound
	}

	if tool.RequiresPermission() && e.permManager != nil {
		permReq := types.PermissionRequest{
			Tool:      toolCall.Name,
			Operation: types.PermissionExecute,
			Arguments: toolCall.Arguments,
			Context:   ctx,
		}

		resp := e.permManager.CheckPermission(permReq)
		if !resp.Allowed {
			return &types.ToolResult{
				ToolCallID: toolCall.ID,
				IsError:    true,
				Error:      fmt.Sprintf("permission denied: %s", resp.Reason),
			}, errors.NewPermissionError(toolCall.Name, "execute", "", resp.Reason)
		}

		if resp.Modified != nil {
			toolCall.Arguments = resp.Modified
		}
	}

	result, err := tool.Execute(ctx, toolCall.Arguments)
	if err != nil {
		return &types.ToolResult{
			ToolCallID: toolCall.ID,
			IsError:    true,
			Error:      err.Error(),
		}, err
	}

	return &types.ToolResult{
		ToolCallID: toolCall.ID,
		Content:    result,
		IsError:    false,
	}, nil
}

func (e *Executor) GetAvailableTools() []types.Tool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tools := make([]types.Tool, 0, len(e.tools))
	for _, tool := range e.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (e *Executor) GetTool(name string) (types.Tool, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tool, exists := e.tools[name]
	return tool, exists
}

func (e *Executor) HasTool(name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	_, exists := e.tools[name]
	return exists
}

func (e *Executor) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tools = make(map[string]types.Tool)
}

func (e *Executor) ToolCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return len(e.tools)
}

func (e *Executor) GetToolDefinitions() []types.ToolDefinition {
	e.mu.RLock()
	defer e.mu.RUnlock()

	definitions := make([]types.ToolDefinition, 0, len(e.tools))
	for _, tool := range e.tools {
		definitions = append(definitions, types.ToolToDefinition(tool))
	}
	return definitions
}