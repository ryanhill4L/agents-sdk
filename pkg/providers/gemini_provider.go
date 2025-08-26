package providers

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/genai"
	"github.com/ryanhill4L/agents-sdk/pkg/logging"
)

// GeminiProvider implements the Provider interface using Google's Gemini API
type GeminiProvider struct {
	config *GeminiConfig
	client *genai.Client
	logger logging.Logger
}

// NewGeminiProvider creates a new Gemini provider instance
func NewGeminiProvider(config *GeminiConfig) (*GeminiProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize logger
	logger := config.Logger
	if logger == nil {
		logger = logging.NewConsoleLogger(config.LogLevel, config.Verbose)
	}

	// Initialize Gemini client
	clientConfig := &genai.ClientConfig{
		APIKey:  config.APIKey,
		Backend: genai.BackendGeminiAPI,
	}

	// Use Vertex AI backend if project ID is provided
	if config.ProjectID != "" {
		clientConfig.Project = config.ProjectID
		clientConfig.Location = config.Location
		clientConfig.Backend = genai.BackendVertexAI
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	provider := &GeminiProvider{
		config: config,
		client: client,
		logger: logger.With(logging.String("provider", "gemini")),
	}

	backend := "gemini_api"
	if config.ProjectID != "" {
		backend = "vertex_ai"
	}

	provider.logger.Info(context.Background(), "Gemini provider initialized", 
		logging.String("backend", backend),
		logging.String("project_id", config.ProjectID),
		logging.String("location", config.Location),
		logging.String("log_level", config.LogLevel.String()),
		logging.Bool("verbose", config.Verbose))

	return provider, nil
}

// Complete implements the Provider interface for Gemini
func (p *GeminiProvider) Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error) {
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
	
	logger.Info(ctx, "Starting Gemini completion request")
	
	// Default model if not specified
	model := agent.GetModel()
	if model == "" {
		model = "gemini-1.5-pro"
		logger.Debug(ctx, "Using default model", logging.String("model", model))
	}
	
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

	// Convert messages to Gemini content format
	contents := make([]*genai.Content, 0)

	// Build the conversation history
	var allText string

	// Add system instructions if present
	if instructions := agent.GetInstructions(); instructions != "" {
		allText += "System: " + instructions + "\n\n"
	}

	// Convert messages to text format (simplified approach)
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			allText += "User: " + msg.Content + "\n"
		case "assistant":
			allText += "Assistant: " + msg.Content + "\n"

			// Handle tool calls if present
			for _, toolCall := range msg.ToolCalls {
				toolCallText := fmt.Sprintf("Tool Call [%s]: %s with args: %v\n",
					toolCall.ID, toolCall.Name, toolCall.Arguments)
				allText += toolCallText
			}
		case "tool":
			// Handle tool responses
			if msg.Metadata != nil {
				if toolCallID, ok := msg.Metadata["tool_call_id"].(string); ok {
					toolResponseText := fmt.Sprintf("Tool Result [%s]: %s\n", toolCallID, msg.Content)
					allText += toolResponseText
				}
			}
		}
	}

	// Add prompt to generate assistant response
	allText += "Assistant: "

	// Create content with text part
	content := &genai.Content{
		Parts: []*genai.Part{
			{Text: allText},
		},
	}
	contents = append(contents, content)

	// Create generation options
	opts := &genai.GenerateContentConfig{}

	// Set temperature if specified
	if temp := agent.GetTemperature(); temp > 0 {
		opts.Temperature = &temp
	}

	// Set max tokens if specified
	if maxTokens := agent.GetMaxTokens(); maxTokens > 0 {
		maxTokensVal := int32(maxTokens)
		opts.MaxOutputTokens = maxTokensVal
	}

	// Set top_p if specified
	if topP := agent.GetTopP(); topP > 0 {
		opts.TopP = &topP
	}

	logger.Debug(ctx, "Making Gemini API request")
	
	// Make API call to generate content
	response, err := p.client.Models.GenerateContent(ctx, model, contents, opts)
	requestDuration := time.Since(startTime)
	
	if err != nil {
		logger.Error(ctx, "Gemini API call failed", 
			logging.Error(err),
			logging.Duration("duration", requestDuration),
		)
		return nil, fmt.Errorf("Gemini API call failed: %w", err)
	}

	// Extract content from response
	var responseContent string
	var toolCalls []ToolCall

	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no response candidates returned from Gemini")
	}

	candidate := response.Candidates[0]

	// Process content parts
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			if responseContent != "" {
				responseContent += "\n"
			}
			responseContent += part.Text
		}
		// Note: Tool calling would be handled differently in production
		// This is a simplified implementation
	}

	// Simple tool call parsing from text (basic implementation)
	toolCalls = p.parseToolCallsFromText(responseContent)

	// Extract usage information if available
	var usage Usage
	if response.UsageMetadata != nil {
		usage = Usage{
			PromptTokens:     int(response.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(response.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(response.UsageMetadata.TotalTokenCount),
		}
	}

	// Log API response details
	logger.Info(ctx, "Gemini API request completed",
		logging.Duration("duration", requestDuration),
		logging.Int("prompt_tokens", usage.PromptTokens),
		logging.Int("completion_tokens", usage.CompletionTokens),
		logging.Int("total_tokens", usage.TotalTokens),
		logging.Int("candidates", len(response.Candidates)),
	)
	
	if p.config.Verbose {
		logger.Debug(ctx, "Response message",
			logging.String("content", responseContent[:min(200, len(responseContent))]),
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
			Content:   responseContent,
			ToolCalls: toolCalls,
			Timestamp: time.Now(),
		},
		Usage:     usage,
		ToolCalls: toolCalls,
	}

	return result, nil
}

// parseToolCallsFromText is a simple helper to extract tool calls from text
// This is a basic implementation - in practice, you would use Gemini's
// structured function calling capabilities
func (p *GeminiProvider) parseToolCallsFromText(content string) []ToolCall {
	var toolCalls []ToolCall

	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Use Gemini's function calling features
	// 2. Parse structured function call responses
	// 3. Handle the tool execution workflow properly

	// For now, return empty slice - tool calling will be handled
	// in text format by the agent framework
	return toolCalls
}

// Close cleans up the provider resources
func (p *GeminiProvider) Close() error {
	// Gemini client doesn't require explicit cleanup
	return nil
}
