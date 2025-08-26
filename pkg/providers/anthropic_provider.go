package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
	"github.com/ryanhill4L/agents-sdk/pkg/logging"
)

// AnthropicProvider implements the Provider interface using Anthropic's Claude API
type AnthropicProvider struct {
	config *AnthropicConfig
	client *anthropic.Client
	logger logging.Logger
}

// NewAnthropicProvider creates a new Anthropic provider instance
func NewAnthropicProvider(config *AnthropicConfig) (*AnthropicProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize logger
	logger := config.Logger
	if logger == nil {
		logger = logging.NewConsoleLogger(config.LogLevel, config.Verbose)
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

	provider := &AnthropicProvider{
		config: config,
		client: &client,
		logger: logger.With(logging.String("provider", "anthropic")),
	}

	provider.logger.Info(context.Background(), "Anthropic provider initialized", 
		logging.String("base_url", config.BaseURL),
		logging.String("version", config.Version),
		logging.String("log_level", config.LogLevel.String()),
		logging.Bool("verbose", config.Verbose))

	return provider, nil
}

// Complete implements the Provider interface for Anthropic
func (p *AnthropicProvider) Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error) {
	startTime := time.Now()
	requestID := logging.NewRequestID()
	
	// Add request context to logger
	ctx = logging.WithRequestID(ctx, requestID)
	logger := p.logger.With(
		logging.String("request_id", requestID),
		logging.String("agent", agent.GetName()),
		logging.String("model", agent.GetModel()),
		logging.Int("input_messages", len(messages)),
		logging.Int("available_tools", len(tools)),
	)
	
	logger.Info(ctx, "Starting Anthropic completion request")
	
	if p.config.Verbose {
		logger.Debug(ctx, "Request details",
			logging.Float64("temperature", float64(agent.GetTemperature())),
			logging.Int("max_tokens", agent.GetMaxTokens()),
			logging.Float64("top_p", float64(agent.GetTopP())),
		)
		
		// Log message content in verbose mode
		for i, msg := range messages {
			logger.Debug(ctx, "Input message",
				logging.Int("index", i),
				logging.String("role", msg.Role),
				logging.String("content", msg.Content[:min(200, len(msg.Content))]), // Truncate for readability
			)
		}
	}
	
	// Convert messages to Anthropic format
	var systemPrompt *string
	claudeMessages := make([]anthropic.MessageParam, 0, len(messages))
	
	// Extract system instructions
	if instructions := agent.GetInstructions(); instructions != "" {
		systemPrompt = &instructions
		logger.Debug(ctx, "Added system instructions", logging.Int("length", len(instructions)))
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
		logger.Debug(ctx, "Converting tools to Anthropic format", logging.Int("tool_count", len(tools)))
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
	
	logger.Debug(ctx, "Making Anthropic API request")
	
	// Make API call
	response, err := p.client.Messages.New(ctx, params)
	requestDuration := time.Since(startTime)
	
	if err != nil {
		logger.Error(ctx, "Anthropic API call failed", 
			logging.Error(err),
			logging.Duration("duration", requestDuration),
		)
		return nil, fmt.Errorf("Anthropic API call failed: %w", err)
	}
	
	// Log API response details
	logger.Info(ctx, "Anthropic API request completed",
		logging.Duration("duration", requestDuration),
		logging.Int("prompt_tokens", int(response.Usage.InputTokens)),
		logging.Int("completion_tokens", int(response.Usage.OutputTokens)),
		logging.Int("total_tokens", int(response.Usage.InputTokens + response.Usage.OutputTokens)),
		logging.String("stop_reason", string(response.StopReason)),
	)
	
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
				logger.Warn(ctx, "Failed to parse tool call input", 
					logging.Error(err),
					logging.String("tool_id", contentBlock.ID),
					logging.String("tool_name", contentBlock.Name),
				)
				continue
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        contentBlock.ID,
				Name:      contentBlock.Name,
				Arguments: inputMap,
			})
		}
	}
	
	if p.config.Verbose {
		logger.Debug(ctx, "Response message",
			logging.String("content", content[:min(200, len(content))]),
			logging.Int("tool_calls", len(toolCalls)),
		)
	}
	
	if len(toolCalls) > 0 {
		logger.Debug(ctx, "Processed tool calls", logging.Int("count", len(toolCalls)))
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


