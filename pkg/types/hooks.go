package types

import (
	"context"
)

type HookEvent string

const (
	HookPreToolUse    HookEvent = "pre_tool_use"
	HookPostToolUse   HookEvent = "post_tool_use"
	HookPreMessage    HookEvent = "pre_message"
	HookPostMessage   HookEvent = "post_message"
	HookPreQuery      HookEvent = "pre_query"
	HookPostQuery     HookEvent = "post_query"
	HookError         HookEvent = "error"
	HookStreamStart   HookEvent = "stream_start"
	HookStreamChunk   HookEvent = "stream_chunk"
	HookStreamEnd     HookEvent = "stream_end"
)

type HookContext struct {
	Event      HookEvent
	Context    context.Context
	Message    Message
	Tool       *ToolCall
	ToolResult *ToolResult
	Error      error
	Data       map[string]any
}

type Hook interface {
	Name() string
	Events() []HookEvent
	Execute(ctx *HookContext) error
	Priority() int
}

type HookFunc func(ctx *HookContext) error

type SimpleHook struct {
	name     string
	events   []HookEvent
	fn       HookFunc
	priority int
}

func NewSimpleHook(name string, events []HookEvent, fn HookFunc) *SimpleHook {
	return &SimpleHook{
		name:     name,
		events:   events,
		fn:       fn,
		priority: 0,
	}
}

func (h *SimpleHook) Name() string          { return h.name }
func (h *SimpleHook) Events() []HookEvent   { return h.events }
func (h *SimpleHook) Execute(ctx *HookContext) error { return h.fn(ctx) }
func (h *SimpleHook) Priority() int         { return h.priority }
func (h *SimpleHook) SetPriority(p int)     { h.priority = p }

type HookManager interface {
	RegisterHook(hook Hook) error
	UnregisterHook(name string) error
	ExecuteHooks(event HookEvent, ctx *HookContext) error
	GetHooks(event HookEvent) []Hook
	ClearHooks()
}