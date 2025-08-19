package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
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

	// Initialize Anthropic client with official SDK
	var opts []option.RequestOption
	if config.APIKey != "" {
		opts = append(opts, option.WithAPIKey(config.APIKey))
	}
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}
	
	client := anthropic.NewClient(opts...)

	return &AnthropicProvider{
		config: config,
		client: &client,
	}, nil
}

// Complete implements the Provider interface for Anthropic
func (p *AnthropicProvider) Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error) {
	// Convert messages to Anthropic format
	var systemPrompt *string
	claudeMessages := make([]anthropic.MessageParam, 0, len(messages))
	
	// Extract system instructions
	if instructions := agent.GetInstructions(); instructions != "" {
		systemPrompt = &instructions
	}
	
	// Convert messages (skip system messages as they're handled separately)
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			claudeMessages = append(claudeMessages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case "assistant":
			// Handle assistant messages with potential tool calls
			content := []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(msg.Content),
			}
			
			// Add tool calls if present
			for _, toolCall := range msg.ToolCalls {
				content = append(content, anthropic.NewToolUseBlock(
					toolCall.ID,
					toolCall.Arguments,
					toolCall.Name,
				))
			}
			
			claudeMessages = append(claudeMessages, anthropic.NewAssistantMessage(content...))
		case "tool":
			// Handle tool result messages properly for Anthropic
			if msg.Metadata != nil {
				if toolCallID, ok := msg.Metadata["tool_call_id"].(string); ok {
					claudeMessages = append(claudeMessages, anthropic.NewUserMessage(
						anthropic.NewToolResultBlock(toolCallID, msg.Content, false),
					))
				}
			}
		}
	}
	
	// Prepare request parameters
	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest, // Default model
		Messages:  claudeMessages,
		MaxTokens: 1000, // Default max tokens
	}
	
	// Override model if specified by agent
	if model := agent.GetModel(); model != "" {
		params.Model = anthropic.Model(model)
	}
	
	// Override max tokens if specified by agent
	if maxTokens := agent.GetMaxTokens(); maxTokens > 0 {
		params.MaxTokens = int64(maxTokens)
	}
	
	// Set system message if available
	if systemPrompt != nil {
		params.System = []anthropic.TextBlockParam{
			{Type: "text", Text: *systemPrompt},
		}
	}
	
	// Set temperature if specified
	if temp := agent.GetTemperature(); temp > 0 {
		params.Temperature = anthropic.Float(float64(temp))
	}
	
	// Set top_p if specified
	if topP := agent.GetTopP(); topP > 0 {
		params.TopP = anthropic.Float(float64(topP))
	}
	
	// Convert tools to Anthropic format
	if len(tools) > 0 {
		anthropicTools := make([]anthropic.ToolUnionParam, 0, len(tools))
		for _, tool := range tools {
			// Convert our schema to Anthropic's expected format
			inputSchema := anthropic.ToolInputSchemaParam{
				Type:       constant.Object("object"), // Always "object" for function tools
				Properties: tool.Schema.Properties,
				Required:   tool.Schema.Required,
			}
			
			anthropicTools = append(anthropicTools, anthropic.ToolUnionParamOfTool(
				inputSchema,
				tool.Name,
			))
		}
		params.Tools = anthropicTools
	}
	
	// Make API call
	response, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Anthropic API call failed: %w", err)
	}
	
	// Extract content and tool calls from response
	var content string
	var toolCalls []ToolCall
	
	for _, contentBlock := range response.Content {
		switch contentBlock.Type {
		case "text":
			if contentBlock.Text != "" {
				if content != "" {
					content += "\n"
				}
				content += contentBlock.Text
			}
		case "tool_use":
			// Parse the JSON input to map[string]interface{}
			var inputMap map[string]interface{}
			if err := json.Unmarshal(contentBlock.Input, &inputMap); err != nil {
				// If unmarshaling fails, skip this tool call
				continue
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        contentBlock.ID,
				Name:      contentBlock.Name,
				Arguments: inputMap,
			})
		}
	}
	
	// Convert response
	result := &Completion{
		Message: Message{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
			Timestamp: time.Now(),
		},
		Usage: Usage{
			PromptTokens:     int(response.Usage.InputTokens),
			CompletionTokens: int(response.Usage.OutputTokens),
			TotalTokens:      int(response.Usage.InputTokens + response.Usage.OutputTokens),
		},
		ToolCalls: toolCalls,
	}
	
	return result, nil
}


