package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIProvider implements the Provider interface using OpenAI's API
type OpenAIProvider struct {
	config *OpenAIConfig
	client *openai.Client
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider(config *OpenAIConfig) (*OpenAIProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
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

	return &OpenAIProvider{
		config: config,
		client: &client,
	}, nil
}

// Complete implements the Provider interface for OpenAI
func (p *OpenAIProvider) Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error) {
	// Convert messages to OpenAI format
	chatMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	
	// Add system message if agent has instructions
	if instructions := agent.GetInstructions(); instructions != "" {
		chatMessages = append(chatMessages, openai.SystemMessage(instructions))
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
		// TODO: Implement proper tool calling once SDK issues are resolved
		// For now, we'll proceed without tools to get the basic functionality working
	}
	
	// Make API call
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}
	
	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned from OpenAI")
	}
	
	choice := completion.Choices[0]
	message := choice.Message
	
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
	}
	
	return result, nil
}



