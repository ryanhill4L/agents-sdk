package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/ryanhill4L/agents-sdk/pkg/errors"
	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

type AnthropicTransport struct {
	client  *anthropic.Client
	options *types.ClaudeCodeOptions
}

func NewAnthropicTransport(opts *types.ClaudeCodeOptions) (*AnthropicTransport, error) {
	if opts == nil {
		opts = types.DefaultOptions()
	}

	if opts.APIKey == "" {
		opts.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	if opts.APIKey == "" {
		return nil, errors.ErrNoAPIKey
	}

	clientOpts := []option.RequestOption{
		option.WithAPIKey(opts.APIKey),
	}

	if opts.BaseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(opts.BaseURL))
	}

	client := anthropic.NewClient(clientOpts...)

	return &AnthropicTransport{
		client:  &client,
		options: opts,
	}, nil
}

func (t *AnthropicTransport) CreateMessage(ctx context.Context, messages []types.Message, tools []types.Tool) (*anthropic.Message, error) {
	params := t.buildMessageParams(messages, tools)

	response, err := t.client.Messages.New(ctx, params)
	if err != nil {
		return nil, t.wrapError(err)
	}

	return &response, nil
}

func (t *AnthropicTransport) CreateStreamingMessage(ctx context.Context, messages []types.Message, tools []types.Tool) (*anthropic.MessageStream, error) {
	params := t.buildMessageParams(messages, tools)

	stream := t.client.Messages.NewStreaming(ctx, params)
	return stream, nil
}

func (t *AnthropicTransport) buildMessageParams(messages []types.Message, tools []types.Tool) anthropic.MessageNewParams {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(t.options.Model),
		MaxTokens: int64(t.options.MaxTokens),
	}

	if t.options.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{
				Type: "text",
				Text: t.options.SystemPrompt,
			},
		}
	}

	if t.options.Temperature != nil {
		params.Temperature = anthropic.Float(*t.options.Temperature)
	}

	if t.options.TopP != nil {
		params.TopP = anthropic.Float(*t.options.TopP)
	}

	if t.options.TopK != nil {
		params.TopK = anthropic.Int(*t.options.TopK)
	}

	params.Messages = t.convertMessages(messages)

	if len(tools) > 0 {
		params.Tools = t.convertTools(tools)
	}

	return params
}

func (t *AnthropicTransport) convertMessages(messages []types.Message) []anthropic.MessageParam {
	var result []anthropic.MessageParam

	for _, msg := range messages {
		switch msg.GetRole() {
		case types.RoleUser:
			content := t.convertContentBlocks(msg.GetContent())
			result = append(result, anthropic.NewUserMessage(content...))

		case types.RoleAssistant:
			content := t.convertContentBlocks(msg.GetContent())
			result = append(result, anthropic.NewAssistantMessage(content...))

		case types.RoleTool:
			if toolMsg, ok := msg.(*types.ToolMessage); ok {
				result = append(result, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(
						toolMsg.ToolCallID,
						t.contentBlocksToString(msg.GetContent()),
						toolMsg.IsError,
					),
				))
			}
		}
	}

	return result
}

func (t *AnthropicTransport) convertContentBlocks(blocks []types.ContentBlock) []anthropic.ContentBlockParamUnion {
	var result []anthropic.ContentBlockParamUnion

	for _, block := range blocks {
		switch b := block.(type) {
		case *types.TextBlock:
			result = append(result, anthropic.NewTextBlock(b.Text))

		case *types.ToolUseBlock:
			inputJSON, _ := json.Marshal(b.Arguments)
			result = append(result, anthropic.NewToolUseBlockParam(
				b.ID,
				inputJSON,
				b.Name,
			))

		case *types.ToolResultBlock:
			result = append(result, anthropic.NewToolResultBlock(
				b.ToolCallID,
				b.Content,
				b.IsError,
			))
		}
	}

	if len(result) == 0 {
		result = append(result, anthropic.NewTextBlock(""))
	}

	return result
}

func (t *AnthropicTransport) contentBlocksToString(blocks []types.ContentBlock) string {
	for _, block := range blocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			return textBlock.Text
		}
	}
	return ""
}

func (t *AnthropicTransport) convertTools(tools []types.Tool) []anthropic.ToolParam {
	var result []anthropic.ToolParam

	for _, tool := range tools {
		schema := tool.Schema()
		inputSchema := make(map[string]interface{})
		inputSchema["type"] = schema.Type
		inputSchema["properties"] = t.convertProperties(schema.Properties)
		if len(schema.Required) > 0 {
			inputSchema["required"] = schema.Required
		}

		result = append(result, anthropic.ToolParam{
			Name:        anthropic.F(tool.Name()),
			Description: anthropic.F(tool.Description()),
			InputSchema: anthropic.F(inputSchema),
		})
	}

	return result
}

func (t *AnthropicTransport) convertProperties(props map[string]types.Property) map[string]interface{} {
	result := make(map[string]interface{})

	for name, prop := range props {
		propMap := make(map[string]interface{})
		propMap["type"] = prop.Type

		if prop.Description != "" {
			propMap["description"] = prop.Description
		}

		if len(prop.Enum) > 0 {
			propMap["enum"] = prop.Enum
		}

		if prop.Items != nil {
			propMap["items"] = map[string]interface{}{
				"type": prop.Items.Type,
			}
		}

		if len(prop.Properties) > 0 {
			propMap["properties"] = t.convertProperties(prop.Properties)
		}

		if len(prop.Required) > 0 {
			propMap["required"] = prop.Required
		}

		result[name] = propMap
	}

	return result
}

func (t *AnthropicTransport) ConvertResponse(response *anthropic.Message) *types.AssistantMessage {
	var contentBlocks []types.ContentBlock
	var toolCalls []types.ToolCall

	for _, content := range response.Content {
		switch content.Type {
		case "text":
			if content.Text != "" {
				contentBlocks = append(contentBlocks, &types.TextBlock{
					Type: types.ContentTypeText,
					Text: content.Text,
				})
			}

		case "tool_use":
			var args map[string]any
			if err := json.Unmarshal(content.Input, &args); err == nil {
				toolCall := types.ToolCall{
					ID:        content.ID,
					Name:      content.Name,
					Arguments: args,
				}
				toolCalls = append(toolCalls, toolCall)

				contentBlocks = append(contentBlocks, &types.ToolUseBlock{
					Type:      types.ContentTypeToolUse,
					ID:        content.ID,
					Name:      content.Name,
					Arguments: args,
				})
			}
		}
	}

	msg := &types.AssistantMessage{
		BaseMessage: types.BaseMessage{
			Role:    types.RoleAssistant,
			Content: contentBlocks,
		},
	}

	return msg
}

func (t *AnthropicTransport) wrapError(err error) error {
	if err == nil {
		return nil
	}

	return errors.NewSDKError("anthropic_error", err.Error(), err)
}

func (t *AnthropicTransport) GetClient() *anthropic.Client {
	return t.client
}

func (t *AnthropicTransport) GetOptions() *types.ClaudeCodeOptions {
	return t.options
}