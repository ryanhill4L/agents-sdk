package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/liushuangls/go-anthropic/v2"
)

// AnthropicProvider implements the Provider interface using Anthropic's Claude API
type AnthropicProvider struct {
	config *AnthropicConfig
	client *anthropic.Client
}

// NewAnthropicProvider creates a new Anthropic provider instance
func NewAnthropicProvider(config *AnthropicConfig) (*AnthropicProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize Anthropic client
	client := anthropic.NewClient(config.APIKey)
	
	// Note: BaseURL configuration would need to be set during client creation
	// For now, we'll use the default Anthropic API endpoint

	return &AnthropicProvider{
		config: config,
		client: client,
	}, nil
}

// Complete implements the Provider interface for Anthropic
func (p *AnthropicProvider) Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error) {
	// Convert messages to Anthropic format
	var systemMessage string
	claudeMessages := make([]anthropic.Message, 0, len(messages))
	
	// Extract system instructions
	if instructions := agent.GetInstructions(); instructions != "" {
		systemMessage = instructions
	}
	
	// Convert messages (skip system messages as they're handled separately)
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			claudeMessages = append(claudeMessages, anthropic.Message{
				Role: anthropic.RoleUser,
				Content: []anthropic.MessageContent{
					anthropic.NewTextMessageContent(msg.Content),
				},
			})
		case "assistant":
			claudeMessages = append(claudeMessages, anthropic.Message{
				Role: anthropic.RoleAssistant,
				Content: []anthropic.MessageContent{
					anthropic.NewTextMessageContent(msg.Content),
				},
			})
		case "tool":
			// Anthropic handles tool responses differently
			// For now, we'll skip tool messages to get basic functionality working
		}
	}
	
	// Prepare request
	request := anthropic.MessagesRequest{
		Model:     anthropic.Model(agent.GetModel()),
		Messages:  claudeMessages,
		MaxTokens: 1000, // Default max tokens
	}
	
	// Set system message if available
	if systemMessage != "" {
		request.System = systemMessage
	}
	
	// Set temperature if specified
	if temp := agent.GetTemperature(); temp > 0 {
		tempFloat := float32(temp)
		request.Temperature = &tempFloat
	}
	
	// Note: Tool calling implementation is disabled for now to get basic functionality working
	// TODO: Implement proper tool calling once SDK issues are resolved
	
	// Make API call
	response, err := p.client.CreateMessages(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("Anthropic API call failed: %w", err)
	}
	
	// Extract content from response
	var content string
	if len(response.Content) > 0 {
		// Handle the first content item (assuming it's text)
		if response.Content[0].Type == "text" && response.Content[0].Text != nil {
			content = *response.Content[0].Text
		}
	}
	
	// Convert response
	result := &Completion{
		Message: Message{
			Role:      "assistant",
			Content:   content,
			Timestamp: time.Now(),
		},
		Usage: Usage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.InputTokens + response.Usage.OutputTokens,
		},
	}
	
	return result, nil
}


