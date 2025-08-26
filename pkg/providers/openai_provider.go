package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/ryanhill4L/agents-sdk/pkg/logging"
)

// OpenAIProvider implements the Provider interface using OpenAI's API
type OpenAIProvider struct {
	config *OpenAIConfig
	client *openai.Client
	logger logging.Logger
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider(config *OpenAIConfig) (*OpenAIProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize logger
	logger := config.Logger
	if logger == nil {
		logger = logging.NewConsoleLogger(config.LogLevel, config.Verbose)
	}

	// Initialize OpenAI client
	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}
	
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}
	
	if config.Organization != "" {
		opts = append(opts, option.WithHeader("OpenAI-Organization", config.Organization))
	}
	
	if config.Project != "" {
		opts = append(opts, option.WithHeader("OpenAI-Project", config.Project))
	}

	client := openai.NewClient(opts...)

	provider := &OpenAIProvider{
		config: config,
		client: &client,
		logger: logger.With(logging.String("provider", "openai")),
	}

	provider.logger.Info(context.Background(), "OpenAI provider initialized", 
		logging.String("base_url", config.BaseURL),
		logging.Bool("has_organization", config.Organization != ""),
		logging.Bool("has_project", config.Project != ""),
		logging.String("log_level", config.LogLevel.String()),
		logging.Bool("verbose", config.Verbose))

	return provider, nil
}

// Complete implements the Provider interface for OpenAI
func (p *OpenAIProvider) Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error) {
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
	
	logger.Info(ctx, "Starting OpenAI completion request")
	
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
	
	// Convert messages to OpenAI format
	chatMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	
	// Add system message if agent has instructions
	if instructions := agent.GetInstructions(); instructions != "" {
		chatMessages = append(chatMessages, openai.SystemMessage(instructions))
		logger.Debug(ctx, "Added system instructions", logging.Int("length", len(instructions)))
	}
	
	// Convert messages
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			chatMessages = append(chatMessages, openai.UserMessage(msg.Content))
		case "assistant":
			chatMessages = append(chatMessages, openai.AssistantMessage(msg.Content))
		case "tool":
			// Handle tool responses
			if toolCallID, ok := msg.Metadata["tool_call_id"].(string); ok {
				chatMessages = append(chatMessages, openai.ToolMessage(toolCallID, msg.Content))
			}
		}
	}
	
	// Prepare chat completion request
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(agent.GetModel()),
		Messages: chatMessages,
	}
	
	// Set temperature if specified
	if temp := agent.GetTemperature(); temp > 0 {
		params.Temperature = openai.Float(float64(temp))
	}
	
	// Convert tools if provided
	// Note: Tool calling implementation is disabled for now due to OpenAI SDK complexity
	// The basic chat completion will work without tools
	if len(tools) > 0 {
		logger.Debug(ctx, "Tools available but not yet implemented", logging.Int("tool_count", len(tools)))
		// TODO: Implement proper tool calling once SDK issues are resolved
		// For now, we'll proceed without tools to get the basic functionality working
	}
	
	logger.Debug(ctx, "Making OpenAI API request")
	
	// Make API call
	completion, err := p.client.Chat.Completions.New(ctx, params)
	requestDuration := time.Since(startTime)
	
	if err != nil {
		logger.Error(ctx, "OpenAI API call failed", 
			logging.Error(err),
			logging.Duration("duration", requestDuration),
		)
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}
	
	if len(completion.Choices) == 0 {
		logger.Error(ctx, "No completion choices returned from OpenAI")
		return nil, fmt.Errorf("no completion choices returned from OpenAI")
	}
	
	choice := completion.Choices[0]
	message := choice.Message
	
	// Log API response details
	logger.Info(ctx, "OpenAI API request completed",
		logging.Duration("duration", requestDuration),
		logging.Int("prompt_tokens", int(completion.Usage.PromptTokens)),
		logging.Int("completion_tokens", int(completion.Usage.CompletionTokens)),
		logging.Int("total_tokens", int(completion.Usage.TotalTokens)),
		logging.String("finish_reason", string(choice.FinishReason)),
	)
	
	if p.config.Verbose {
		logger.Debug(ctx, "Response message",
			logging.String("content", message.Content[:min(200, len(message.Content))]),
			logging.Int("tool_calls", len(message.ToolCalls)),
		)
	}
	
	// Convert response
	result := &Completion{
		Message: Message{
			Role:      "assistant",
			Content:   message.Content,
			Timestamp: time.Now(),
		},
		Usage: Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
		},
	}
	
	// Handle tool calls
	if len(message.ToolCalls) > 0 {
		toolCalls := make([]ToolCall, 0, len(message.ToolCalls))
		for _, tc := range message.ToolCalls {
			if tc.Function.Name != "" {
				toolCalls = append(toolCalls, ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: map[string]interface{}{
						"raw": tc.Function.Arguments,
					},
				})
			}
		}
		result.ToolCalls = toolCalls
		logger.Debug(ctx, "Processed tool calls", logging.Int("count", len(toolCalls)))
	}
	
	return result, nil
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}



