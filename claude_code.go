package claudecode

import (
	"context"

	"github.com/ryanhill4L/agents-sdk/pkg/client"
	"github.com/ryanhill4L/agents-sdk/pkg/query"
	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

var (
	DefaultOptions = types.DefaultOptions
	WithAPIKey = types.WithAPIKey
	WithModel = types.WithModel
	WithSystemPrompt = types.WithSystemPrompt
	WithMaxTurns = types.WithMaxTurns
	WithMaxTokens = types.WithMaxTokens
	WithTemperature = types.WithTemperature
	WithStreaming = types.WithStreaming
	WithPermissionMode = types.WithPermissionMode
	WithSession = types.WithSession
	WithDebug = types.WithDebug
	WithTimeout = types.WithTimeout
	WithAllowedDirectories = types.WithAllowedDirectories
	WithBlockedDirectories = types.WithBlockedDirectories
)

var (
	PermissionDefault     = types.PermissionDefault
	PermissionAcceptAll   = types.PermissionAcceptAll
	PermissionAcceptEdits = types.PermissionAcceptEdits
	PermissionBypass      = types.PermissionBypass
	PermissionRejectAll   = types.PermissionRejectAll
)

func Query(prompt string, opts ...types.OptionFunc) (*query.QueryResult, error) {
	return query.Query(context.Background(), prompt, opts...)
}

func QueryWithContext(ctx context.Context, prompt string, opts ...types.OptionFunc) (*query.QueryResult, error) {
	return query.Query(ctx, prompt, opts...)
}

func NewClient(opts ...types.OptionFunc) (*client.Client, error) {
	return client.NewClient(opts...)
}

type (
	Client = client.Client
	QueryResult = query.QueryResult
	Tool = types.Tool
	Message = types.Message
	UserMessage = types.UserMessage
	AssistantMessage = types.AssistantMessage
	SystemMessage = types.SystemMessage
	ToolMessage = types.ToolMessage
	ContentBlock = types.ContentBlock
	TextBlock = types.TextBlock
	ToolUseBlock = types.ToolUseBlock
	ToolResultBlock = types.ToolResultBlock
	Hook = types.Hook
	HookFunc = types.HookFunc
	HookContext = types.HookContext
	PermissionCallback = types.PermissionCallback
	PermissionRequest = types.PermissionRequest
	PermissionResponse = types.PermissionResponse
)