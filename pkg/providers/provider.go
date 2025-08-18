package providers

import (
	"context"
	"time"
)

// Agent interface to avoid circular import
type Agent interface {
	GetName() string
	GetInstructions() string
	GetModel() string
	GetTemperature() float32
	GetMaxTokens() int
	GetTopP() float32
}

// ToolDefinition represents a tool's metadata for providers
type ToolDefinition struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Schema      ParameterSchema           `json:"schema"`
}

// ParameterSchema describes function parameters
type ParameterSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required"`
}

// PropertySchema describes a single parameter
type PropertySchema struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

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

// HandoffRequest represents an agent handoff
type HandoffRequest struct {
	TargetAgent string                 `json:"target_agent"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
}

// Provider represents an LLM provider
type Provider interface {
	// Complete generates a completion for the given agent, messages, and available tools
	Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error)
}

// Completion represents the result of an LLM completion
type Completion struct {
	Message          Message            `json:"message"`
	Usage            Usage              `json:"usage"`
	ToolCalls        []ToolCall         `json:"tool_calls,omitempty"`
	Handoff          *HandoffRequest    `json:"handoff,omitempty"`
	StructuredOutput interface{}        `json:"structured_output,omitempty"`
}

// Usage tracks token consumption
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewDefaultOpenAIProvider creates a default OpenAI provider with no API key
// This is referenced in runner.go but needs to be implemented
func NewDefaultOpenAIProvider() Provider {
	// Return NoOp provider since no config is provided
	return &NoOpProvider{}
}

// NoOpProvider is a placeholder provider for testing
type NoOpProvider struct{}

func (p *NoOpProvider) Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error) {
	// Return a simple completion for testing
	return &Completion{
		Message: Message{
			Role:      "assistant",
			Content:   "Hello from NoOpProvider",
			Timestamp: time.Now(),
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}