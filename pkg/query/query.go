package query

import (
	"context"
	"fmt"
	"time"

	"github.com/ryanhill4L/agents-sdk/pkg/errors"
	"github.com/ryanhill4L/agents-sdk/pkg/hooks"
	"github.com/ryanhill4L/agents-sdk/pkg/permissions"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/transport"
	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

type QueryResult struct {
	Content      string
	ToolCalls    []types.ToolCall
	ToolResults  []types.ToolResult
	Usage        Usage
	Duration     time.Duration
	Model        string
}

type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	CacheCreated int
	CacheRead    int
}

func Query(ctx context.Context, prompt string, opts ...types.OptionFunc) (*QueryResult, error) {
	start := time.Now()

	options := types.DefaultOptions()
	types.ApplyOptions(options, opts...)

	if ctx == nil {
		ctx = context.Background()
	}

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	transport, err := transport.NewAnthropicTransport(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	hookManager := hooks.NewManager()
	permManager := permissions.NewManager(options.PermissionMode)
	toolExecutor := tools.NewExecutor(permManager, hookManager)

	for _, dir := range options.AllowedDirectories {
		permManager.AddAllowedPath(dir)
	}
	for _, dir := range options.BlockedDirectories {
		permManager.AddBlockedPath(dir)
	}

	messages := []types.Message{
		types.NewUserMessage(prompt),
	}

	if err := hookManager.ExecuteHooks(types.HookPreQuery, &types.HookContext{
		Event:   types.HookPreQuery,
		Context: ctx,
		Message: messages[0],
		Data:    map[string]any{"prompt": prompt},
	}); err != nil {
		return nil, fmt.Errorf("pre-query hook failed: %w", err)
	}

	availableTools := toolExecutor.GetAvailableTools()

	response, err := transport.CreateMessage(ctx, messages, availableTools)
	if err != nil {
		hookManager.ExecuteHooks(types.HookError, &types.HookContext{
			Event: types.HookError,
			Error: err,
		})
		return nil, err
	}

	assistantMsg := transport.ConvertResponse(response)
	messages = append(messages, assistantMsg)

	var toolCalls []types.ToolCall
	var toolResults []types.ToolResult

	for _, content := range assistantMsg.GetContent() {
		if toolUse, ok := content.(*types.ToolUseBlock); ok {
			toolCall := types.ToolCall{
				ID:        toolUse.ID,
				Name:      toolUse.Name,
				Arguments: toolUse.Arguments,
			}
			toolCalls = append(toolCalls, toolCall)

			if err := hookManager.ExecuteHooks(types.HookPreToolUse, &types.HookContext{
				Event:   types.HookPreToolUse,
				Context: ctx,
				Tool:    &toolCall,
			}); err != nil {
				continue
			}

			result, err := toolExecutor.ExecuteTool(ctx, toolCall)
			if err != nil {
				result = &types.ToolResult{
					ToolCallID: toolCall.ID,
					IsError:    true,
					Error:      err.Error(),
				}
			}

			toolResults = append(toolResults, *result)

			toolMsg := types.NewToolMessage(
				toolCall.ID,
				toolCall.Name,
				result.String(),
				result.IsError,
			)
			messages = append(messages, toolMsg)

			if err := hookManager.ExecuteHooks(types.HookPostToolUse, &types.HookContext{
				Event:      types.HookPostToolUse,
				Context:    ctx,
				Tool:       &toolCall,
				ToolResult: result,
			}); err != nil {
				continue
			}
		}
	}

	if len(toolCalls) > 0 {
		finalResponse, err := transport.CreateMessage(ctx, messages, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get final response after tool use: %w", err)
		}

		assistantMsg = transport.ConvertResponse(finalResponse)
		response = finalResponse
	}

	var content string
	for _, block := range assistantMsg.GetContent() {
		if textBlock, ok := block.(*types.TextBlock); ok {
			if content != "" {
				content += "\n"
			}
			content += textBlock.Text
		}
	}

	result := &QueryResult{
		Content:     content,
		ToolCalls:   toolCalls,
		ToolResults: toolResults,
		Usage: Usage{
			InputTokens:  int(response.Usage.InputTokens),
			OutputTokens: int(response.Usage.OutputTokens),
			TotalTokens:  int(response.Usage.InputTokens + response.Usage.OutputTokens),
			CacheCreated: int(response.Usage.CacheCreationInputTokens),
			CacheRead:    int(response.Usage.CacheReadInputTokens),
		},
		Duration: time.Since(start),
		Model:    string(response.Model),
	}

	if err := hookManager.ExecuteHooks(types.HookPostQuery, &types.HookContext{
		Event:   types.HookPostQuery,
		Context: ctx,
		Data: map[string]any{
			"result": result,
		},
	}); err != nil {
		return result, fmt.Errorf("post-query hook failed: %w", err)
	}

	return result, nil
}

func QueryWithTools(ctx context.Context, prompt string, userTools []types.Tool, opts ...types.OptionFunc) (*QueryResult, error) {
	options := types.DefaultOptions()
	types.ApplyOptions(options, opts...)

	if ctx == nil {
		ctx = context.Background()
	}

	transport, err := transport.NewAnthropicTransport(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	hookManager := hooks.NewManager()
	permManager := permissions.NewManager(options.PermissionMode)
	toolExecutor := tools.NewExecutor(permManager, hookManager)

	for _, tool := range userTools {
		if err := toolExecutor.RegisterTool(tool); err != nil {
			return nil, fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}

	return performQuery(ctx, prompt, transport, toolExecutor, hookManager, options)
}

func performQuery(
	ctx context.Context,
	prompt string,
	transport *transport.AnthropicTransport,
	toolExecutor *tools.Executor,
	hookManager *hooks.Manager,
	options *types.ClaudeCodeOptions,
) (*QueryResult, error) {
	start := time.Now()

	messages := []types.Message{
		types.NewUserMessage(prompt),
	}

	availableTools := toolExecutor.GetAvailableTools()

	response, err := transport.CreateMessage(ctx, messages, availableTools)
	if err != nil {
		return nil, err
	}

	assistantMsg := transport.ConvertResponse(response)

	var content string
	for _, block := range assistantMsg.GetContent() {
		if textBlock, ok := block.(*types.TextBlock); ok {
			if content != "" {
				content += "\n"
			}
			content += textBlock.Text
		}
	}

	return &QueryResult{
		Content: content,
		Usage: Usage{
			InputTokens:  int(response.Usage.InputTokens),
			OutputTokens: int(response.Usage.OutputTokens),
			TotalTokens:  int(response.Usage.InputTokens + response.Usage.OutputTokens),
		},
		Duration: time.Since(start),
		Model:    string(response.Model),
	}, nil
}