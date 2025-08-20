package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	client *api.Client
}

// NewOllamaProvider creates a new Ollama provider instance
func NewOllamaProvider(host string) (*OllamaProvider, error) {
	var client *api.Client
	var err error
	if host == "" {
		client, err = api.ClientFromEnvironment()
		if err != nil {
			return nil, err
		}
		return &OllamaProvider{client: client}, nil
	}

	// Parse and validate the host URL
	parsedURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid host URL: %w", err)
	}

	client = api.NewClient(parsedURL, http.DefaultClient)

	return &OllamaProvider{
		client: client,
	}, nil
}

// Complete implements the Provider interface for Ollama
func (p *OllamaProvider) Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error) {
	// Convert our messages to Ollama format
	ollamaMessages := []api.Message{}

	// Add system message with agent instructions and tool definitions
	systemMessage := agent.GetInstructions()
	if systemMessage != "" {
		ollamaMessages = append(ollamaMessages, api.Message{
			Role:    "system",
			Content: systemMessage,
		})
	}

	// Add conversation messages
	for _, msg := range messages {
		ollamaMessages = append(ollamaMessages, api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	req := &api.ChatRequest{
		Model:    agent.GetModel(),
		Messages: ollamaMessages,
		Options: map[string]any{
			"temperature": agent.GetTemperature(),
			"top_p":       agent.GetTopP(),
		},
		Stream: new(bool), // Disable streaming to simplify response handling
		Think: &api.ThinkValue{ // Disable thinking since it makes the response more complicated
			Value: false,
		},
		Tools: toolsToOllamaTools(tools),
	}

	var resp api.ChatResponse
	err := p.client.Chat(ctx, req, func(cr api.ChatResponse) error {
		resp = cr
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	toolCalls := p.convertOllamaToolCalls(resp.Message.ToolCalls)

	// Create completion response
	completion := &Completion{
		Message: Message{
			Role:      "assistant",
			Content:   resp.Message.Content,
			Timestamp: time.Now(),
		},
		ToolCalls: toolCalls,
		// Handoff:   handoff, // Handoff not currently supported
		Usage: Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.EvalCount + resp.PromptEvalCount,
		},
	}

	return completion, nil
}

func toolsToOllamaTools(tools []ToolDefinition) []api.Tool {
	result := make([]api.Tool, len(tools))
	for i, t := range tools {
		result[i] = api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters: ToolFunctionParameters{
					Type:       "object",
					Properties: parameterSchemaToToolProperties(t.Schema),
					Required:   t.Schema.Required,
				}.ToAPI(),
			},
		}
	}
	return result
}

func parameterSchemaToToolProperties(schema ParameterSchema) map[string]api.ToolProperty {
	properties := make(map[string]api.ToolProperty)

	for name, prop := range schema.Properties {
		properties[name] = api.ToolProperty{
			Type:        api.PropertyType{prop.Type},
			Description: prop.Description,
		}
	}
	return properties
}

// convertOllamaToolCalls converts Ollama's tool calls to our internal format
func (p *OllamaProvider) convertOllamaToolCalls(ollamaCalls []api.ToolCall) []ToolCall {
	if len(ollamaCalls) == 0 {
		return nil
	}

	toolCalls := make([]ToolCall, len(ollamaCalls))
	for i, call := range ollamaCalls {
		toolCalls[i] = ToolCall{
			Name:      call.Function.Name,
			ID:        fmt.Sprint(call.Function.Index),
			Arguments: call.Function.Arguments,
		}
	}
	return toolCalls
}

type ToolFunctionParameters struct {
	Type       string                      `json:"type"`
	Defs       any                         `json:"$defs,omitempty"`
	Items      any                         `json:"items,omitempty"`
	Required   []string                    `json:"required"`
	Properties map[string]api.ToolProperty `json:"properties"`
}

func (t ToolFunctionParameters) ToAPI() struct {
	Type       string                      `json:"type"`
	Defs       any                         `json:"$defs,omitempty"`
	Items      any                         `json:"items,omitempty"`
	Required   []string                    `json:"required"`
	Properties map[string]api.ToolProperty `json:"properties"`
} {
	return struct {
		Type       string                      `json:"type"`
		Defs       any                         `json:"$defs,omitempty"`
		Items      any                         `json:"items,omitempty"`
		Required   []string                    `json:"required"`
		Properties map[string]api.ToolProperty `json:"properties"`
	}{
		Type:       t.Type,
		Defs:       t.Defs,
		Items:      t.Items,
		Required:   t.Required,
		Properties: t.Properties,
	}
}
