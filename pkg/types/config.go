package types

import (
	"time"
)

type ClaudeCodeOptions struct {
	APIKey              string
	BaseURL             string
	SystemPrompt        string
	MaxTurns            int
	MaxTokens           int
	Temperature         *float64
	TopP                *float64
	TopK                *int
	Model               string
	Timeout             time.Duration
	MaxRetries          int
	RetryDelay          time.Duration
	PermissionMode      PermissionMode
	AllowedDirectories  []string
	BlockedDirectories  []string
	RequireConfirmation bool
	StreamingMode       bool
	EnableCaching       bool
	CacheTTL            time.Duration
	SessionID           string
	SessionPath         string
	Debug               bool
	LogLevel            string
}

func DefaultOptions() *ClaudeCodeOptions {
	defaultTemp := 0.7
	defaultTopP := 1.0
	defaultTopK := 40

	return &ClaudeCodeOptions{
		Model:         "claude-3-5-sonnet-20241022",
		MaxTurns:      10,
		MaxTokens:     4096,
		Temperature:   &defaultTemp,
		TopP:          &defaultTopP,
		TopK:          &defaultTopK,
		Timeout:       2 * time.Minute,
		MaxRetries:    3,
		RetryDelay:    time.Second,
		PermissionMode: PermissionDefault,
		StreamingMode: false,
		EnableCaching: true,
		CacheTTL:      15 * time.Minute,
		Debug:         false,
		LogLevel:      "info",
	}
}

type OptionFunc func(*ClaudeCodeOptions)

func WithAPIKey(key string) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.APIKey = key
	}
}

func WithModel(model string) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.Model = model
	}
}

func WithSystemPrompt(prompt string) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.SystemPrompt = prompt
	}
}

func WithMaxTurns(turns int) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.MaxTurns = turns
	}
}

func WithMaxTokens(tokens int) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.MaxTokens = tokens
	}
}

func WithTemperature(temp float64) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.Temperature = &temp
	}
}

func WithStreaming(enabled bool) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.StreamingMode = enabled
	}
}

func WithPermissionMode(mode PermissionMode) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.PermissionMode = mode
	}
}

func WithSession(sessionID, sessionPath string) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.SessionID = sessionID
		o.SessionPath = sessionPath
	}
}

func WithDebug(enabled bool) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.Debug = enabled
		if enabled {
			o.LogLevel = "debug"
		}
	}
}

func WithTimeout(timeout time.Duration) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.Timeout = timeout
	}
}

func WithAllowedDirectories(dirs ...string) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.AllowedDirectories = dirs
	}
}

func WithBlockedDirectories(dirs ...string) OptionFunc {
	return func(o *ClaudeCodeOptions) {
		o.BlockedDirectories = dirs
	}
}

func ApplyOptions(base *ClaudeCodeOptions, opts ...OptionFunc) *ClaudeCodeOptions {
	for _, opt := range opts {
		opt(base)
	}
	return base
}