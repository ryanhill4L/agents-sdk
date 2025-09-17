package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/ryanhill4L/agents-sdk/pkg/errors"
	"github.com/ryanhill4L/agents-sdk/pkg/hooks"
	"github.com/ryanhill4L/agents-sdk/pkg/memory"
	"github.com/ryanhill4L/agents-sdk/pkg/permissions"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/transport"
	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

type Client struct {
	mu           sync.RWMutex
	transport    *transport.AnthropicTransport
	conversation *types.Conversation
	session      memory.Session
	toolExecutor *tools.Executor
	hookManager  *hooks.Manager
	permManager  *permissions.Manager
	options      *types.ClaudeCodeOptions
	turnCount    int
	interrupted  bool
}

func NewClient(opts ...types.OptionFunc) (*Client, error) {
	options := types.DefaultOptions()
	types.ApplyOptions(options, opts...)

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

	client := &Client{
		transport:    transport,
		conversation: types.NewConversation(),
		toolExecutor: toolExecutor,
		hookManager:  hookManager,
		permManager:  permManager,
		options:      options,
		turnCount:    0,
	}

	if options.SessionPath != "" && options.SessionID != "" {
		session, err := memory.NewSQLiteSession(options.SessionPath, options.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		client.session = session

		if err := client.loadSession(); err != nil && err != memory.ErrSessionNotFound {
			return nil, fmt.Errorf("failed to load session: %w", err)
		}
	}

	return client, nil
}

func (c *Client) SendMessage(ctx context.Context, content string) (*types.AssistantMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	c.mu.Lock()
	c.turnCount++
	currentTurn := c.turnCount
	c.mu.Unlock()

	if c.options.MaxTurns > 0 && currentTurn > c.options.MaxTurns {
		return nil, errors.ErrMaxTurnsExceeded
	}

	userMsg := types.NewUserMessage(content)
	c.conversation.AddMessage(userMsg)

	if err := c.hookManager.ExecuteHooks(types.HookPreMessage, &types.HookContext{
		Event:   types.HookPreMessage,
		Context: ctx,
		Message: userMsg,
	}); err != nil {
		return nil, fmt.Errorf("pre-message hook failed: %w", err)
	}

	messages := c.conversation.GetMessages()
	availableTools := c.toolExecutor.GetAvailableTools()

	response, err := c.transport.CreateMessage(ctx, messages, availableTools)
	if err != nil {
		c.hookManager.ExecuteHooks(types.HookError, &types.HookContext{
			Event: types.HookError,
			Error: err,
		})
		return nil, err
	}

	assistantMsg := c.transport.ConvertResponse(response)
	c.conversation.AddMessage(assistantMsg)

	if err := c.processToolCalls(ctx, assistantMsg); err != nil {
		return nil, err
	}

	if err := c.hookManager.ExecuteHooks(types.HookPostMessage, &types.HookContext{
		Event:   types.HookPostMessage,
		Context: ctx,
		Message: assistantMsg,
	}); err != nil {
		return assistantMsg, fmt.Errorf("post-message hook failed: %w", err)
	}

	if c.session != nil {
		if err := c.saveSession(); err != nil {
			return assistantMsg, fmt.Errorf("failed to save session: %w", err)
		}
	}

	return assistantMsg, nil
}

func (c *Client) SendMessageStreaming(ctx context.Context, content string) (<-chan StreamChunk, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !c.options.StreamingMode {
		return nil, fmt.Errorf("streaming mode is not enabled")
	}

	chunkChan := make(chan StreamChunk, 100)

	go func() {
		defer close(chunkChan)

		c.mu.Lock()
		c.turnCount++
		currentTurn := c.turnCount
		c.mu.Unlock()

		if c.options.MaxTurns > 0 && currentTurn > c.options.MaxTurns {
			chunkChan <- StreamChunk{
				Error: errors.ErrMaxTurnsExceeded,
			}
			return
		}

		userMsg := types.NewUserMessage(content)
		c.conversation.AddMessage(userMsg)

		if err := c.hookManager.ExecuteHooks(types.HookStreamStart, &types.HookContext{
			Event:   types.HookStreamStart,
			Context: ctx,
			Message: userMsg,
		}); err != nil {
			chunkChan <- StreamChunk{Error: err}
			return
		}

		messages := c.conversation.GetMessages()
		availableTools := c.toolExecutor.GetAvailableTools()

		stream, err := c.transport.CreateStreamingMessage(ctx, messages, availableTools)
		if err != nil {
			chunkChan <- StreamChunk{Error: err}
			return
		}

		var fullContent []types.ContentBlock
		var accumulated string

		for {
			select {
			case <-ctx.Done():
				chunkChan <- StreamChunk{Error: ctx.Err()}
				return

			default:
				event, err := stream.Recv()
				if err != nil {
					chunkChan <- StreamChunk{Error: err}
					return
				}

				chunk := StreamChunk{
					Type:      string(event.Type),
					Timestamp: time.Now(),
				}

				switch event.Type {
				case anthropic.MessageStreamEventTypeContentBlockStart:
					if event.ContentBlock != nil {
						if event.ContentBlock.Type == "text" {
							chunk.Content = event.ContentBlock.Text
							accumulated += event.ContentBlock.Text
						}
					}

				case anthropic.MessageStreamEventTypeContentBlockDelta:
					if event.Delta != nil && event.Delta.Type == "text_delta" {
						chunk.Content = event.Delta.Text
						accumulated += event.Delta.Text
					}

				case anthropic.MessageStreamEventTypeContentBlockStop:
					if accumulated != "" {
						fullContent = append(fullContent, &types.TextBlock{
							Type: types.ContentTypeText,
							Text: accumulated,
						})
						accumulated = ""
					}

				case anthropic.MessageStreamEventTypeMessageStop:
					assistantMsg := &types.AssistantMessage{
						BaseMessage: types.BaseMessage{
							Role:      types.RoleAssistant,
							Content:   fullContent,
							Timestamp: time.Now(),
						},
					}
					c.conversation.AddMessage(assistantMsg)

					if c.session != nil {
						c.saveSession()
					}

					c.hookManager.ExecuteHooks(types.HookStreamEnd, &types.HookContext{
						Event:   types.HookStreamEnd,
						Context: ctx,
						Message: assistantMsg,
					})

					chunk.Done = true
				}

				c.hookManager.ExecuteHooks(types.HookStreamChunk, &types.HookContext{
					Event:   types.HookStreamChunk,
					Context: ctx,
					Data: map[string]any{
						"chunk": chunk,
					},
				})

				chunkChan <- chunk

				if chunk.Done {
					return
				}
			}
		}
	}()

	return chunkChan, nil
}

func (c *Client) processToolCalls(ctx context.Context, msg *types.AssistantMessage) error {
	var toolCallsToProcess []types.ToolCall

	for _, content := range msg.GetContent() {
		if toolUse, ok := content.(*types.ToolUseBlock); ok {
			toolCallsToProcess = append(toolCallsToProcess, types.ToolCall{
				ID:        toolUse.ID,
				Name:      toolUse.Name,
				Arguments: toolUse.Arguments,
			})
		}
	}

	if len(toolCallsToProcess) == 0 {
		return nil
	}

	for _, toolCall := range toolCallsToProcess {
		if err := c.hookManager.ExecuteHooks(types.HookPreToolUse, &types.HookContext{
			Event:   types.HookPreToolUse,
			Context: ctx,
			Tool:    &toolCall,
		}); err != nil {
			continue
		}

		result, err := c.toolExecutor.ExecuteTool(ctx, toolCall)
		if err != nil {
			result = &types.ToolResult{
				ToolCallID: toolCall.ID,
				IsError:    true,
				Error:      err.Error(),
			}
		}

		toolMsg := types.NewToolMessage(
			toolCall.ID,
			toolCall.Name,
			result.String(),
			result.IsError,
		)
		c.conversation.AddMessage(toolMsg)

		if err := c.hookManager.ExecuteHooks(types.HookPostToolUse, &types.HookContext{
			Event:      types.HookPostToolUse,
			Context:    ctx,
			Tool:       &toolCall,
			ToolResult: result,
		}); err != nil {
			continue
		}
	}

	messages := c.conversation.GetMessages()
	response, err := c.transport.CreateMessage(ctx, messages, nil)
	if err != nil {
		return err
	}

	finalMsg := c.transport.ConvertResponse(response)
	c.conversation.AddMessage(finalMsg)

	return c.processToolCalls(ctx, finalMsg)
}

func (c *Client) RegisterTool(tool types.Tool) error {
	return c.toolExecutor.RegisterTool(tool)
}

func (c *Client) UnregisterTool(name string) error {
	return c.toolExecutor.UnregisterTool(name)
}

func (c *Client) RegisterHook(hook types.Hook) error {
	return c.hookManager.RegisterHook(hook)
}

func (c *Client) UnregisterHook(name string) error {
	return c.hookManager.UnregisterHook(name)
}

func (c *Client) SetPermissionCallback(callback types.PermissionCallback) {
	c.permManager.SetCallback(callback)
}

func (c *Client) SetPermissionMode(mode types.PermissionMode) {
	c.permManager.SetMode(mode)
}

func (c *Client) GetConversation() *types.Conversation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conversation
}

func (c *Client) ClearConversation() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conversation.Clear()
	c.turnCount = 0
}

func (c *Client) GetTurnCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.turnCount
}

func (c *Client) Interrupt() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interrupted = true
}

func (c *Client) IsInterrupted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.interrupted
}

func (c *Client) ResetInterrupt() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interrupted = false
}

func (c *Client) loadSession() error {
	if c.session == nil {
		return nil
	}

	messages, err := c.session.Load()
	if err != nil {
		return err
	}

	for _, msg := range messages {
		c.conversation.AddMessage(msg)
	}

	return nil
}

func (c *Client) saveSession() error {
	if c.session == nil {
		return nil
	}

	return c.session.Save(c.conversation.GetMessages())
}

func (c *Client) Close() error {
	if c.session != nil {
		if err := c.session.Close(); err != nil {
			return err
		}
	}
	return nil
}

type StreamChunk struct {
	Type      string
	Content   string
	Error     error
	Done      bool
	Timestamp time.Time
}