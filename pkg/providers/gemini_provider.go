package providers

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/genai"
)

// GeminiProvider implements the Provider interface using Google's Gemini API
type GeminiProvider struct {
	config *GeminiConfig
	client *genai.Client
}

// NewGeminiProvider creates a new Gemini provider instance
func NewGeminiProvider(config *GeminiConfig) (*GeminiProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
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

	return &GeminiProvider{
		config: config,
		client: client,
	}, nil
}

// Complete implements the Provider interface for Gemini
func (p *GeminiProvider) Complete(ctx context.Context, agent Agent, messages []Message, tools []ToolDefinition) (*Completion, error) {
	// Default model if not specified
	model := agent.GetModel()
	if model == "" {
		model = "gemini-1.5-pro"
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

	// Make API call to generate content
	response, err := p.client.Models.GenerateContent(ctx, model, contents, opts)
	if err != nil {
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
