package types

import (
	"encoding/json"
	"time"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

type Message interface {
	GetRole() Role
	GetContent() []ContentBlock
	GetTimestamp() time.Time
}

type BaseMessage struct {
	Role      Role           `json:"role"`
	Content   []ContentBlock `json:"content"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func (m BaseMessage) GetRole() Role             { return m.Role }
func (m BaseMessage) GetContent() []ContentBlock { return m.Content }
func (m BaseMessage) GetTimestamp() time.Time   { return m.Timestamp }

type UserMessage struct {
	BaseMessage
}

func NewUserMessage(content string) *UserMessage {
	return &UserMessage{
		BaseMessage: BaseMessage{
			Role: RoleUser,
			Content: []ContentBlock{
				&TextBlock{Text: content},
			},
			Timestamp: time.Now(),
		},
	}
}

type AssistantMessage struct {
	BaseMessage
	ThinkingContent []ContentBlock `json:"thinking_content,omitempty"`
}

func NewAssistantMessage(content string) *AssistantMessage {
	return &AssistantMessage{
		BaseMessage: BaseMessage{
			Role: RoleAssistant,
			Content: []ContentBlock{
				&TextBlock{Text: content},
			},
			Timestamp: time.Now(),
		},
	}
}

type SystemMessage struct {
	BaseMessage
}

func NewSystemMessage(content string) *SystemMessage {
	return &SystemMessage{
		BaseMessage: BaseMessage{
			Role: RoleSystem,
			Content: []ContentBlock{
				&TextBlock{Text: content},
			},
			Timestamp: time.Now(),
		},
	}
}

type ToolMessage struct {
	BaseMessage
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	IsError    bool   `json:"is_error,omitempty"`
}

func NewToolMessage(toolCallID, toolName, content string, isError bool) *ToolMessage {
	return &ToolMessage{
		BaseMessage: BaseMessage{
			Role: RoleTool,
			Content: []ContentBlock{
				&TextBlock{Text: content},
			},
			Timestamp: time.Now(),
		},
		ToolCallID: toolCallID,
		ToolName:   toolName,
		IsError:    isError,
	}
}

type ContentBlockType string

const (
	ContentTypeText     ContentBlockType = "text"
	ContentTypeThinking ContentBlockType = "thinking"
	ContentTypeToolUse  ContentBlockType = "tool_use"
	ContentTypeToolResult ContentBlockType = "tool_result"
)

type ContentBlock interface {
	GetType() ContentBlockType
	json.Marshaler
	json.Unmarshaler
}

type TextBlock struct {
	Type ContentBlockType `json:"type"`
	Text string          `json:"text"`
}

func (t *TextBlock) GetType() ContentBlockType { return ContentTypeText }

func (t *TextBlock) MarshalJSON() ([]byte, error) {
	type Alias TextBlock
	return json.Marshal(&struct {
		Type ContentBlockType `json:"type"`
		*Alias
	}{
		Type:  ContentTypeText,
		Alias: (*Alias)(t),
	})
}

func (t *TextBlock) UnmarshalJSON(data []byte) error {
	type Alias TextBlock
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.Type = ContentTypeText
	return nil
}

type ThinkingBlock struct {
	Type    ContentBlockType `json:"type"`
	Content string          `json:"content"`
}

func (t *ThinkingBlock) GetType() ContentBlockType { return ContentTypeThinking }

func (t *ThinkingBlock) MarshalJSON() ([]byte, error) {
	type Alias ThinkingBlock
	return json.Marshal(&struct {
		Type ContentBlockType `json:"type"`
		*Alias
	}{
		Type:  ContentTypeThinking,
		Alias: (*Alias)(t),
	})
}

func (t *ThinkingBlock) UnmarshalJSON(data []byte) error {
	type Alias ThinkingBlock
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.Type = ContentTypeThinking
	return nil
}

type ToolUseBlock struct {
	Type      ContentBlockType `json:"type"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments map[string]any  `json:"arguments"`
}

func (t *ToolUseBlock) GetType() ContentBlockType { return ContentTypeToolUse }

func (t *ToolUseBlock) MarshalJSON() ([]byte, error) {
	type Alias ToolUseBlock
	return json.Marshal(&struct {
		Type ContentBlockType `json:"type"`
		*Alias
	}{
		Type:  ContentTypeToolUse,
		Alias: (*Alias)(t),
	})
}

func (t *ToolUseBlock) UnmarshalJSON(data []byte) error {
	type Alias ToolUseBlock
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.Type = ContentTypeToolUse
	return nil
}

type ToolResultBlock struct {
	Type       ContentBlockType `json:"type"`
	ToolCallID string          `json:"tool_call_id"`
	Content    string          `json:"content"`
	IsError    bool            `json:"is_error,omitempty"`
}

func (t *ToolResultBlock) GetType() ContentBlockType { return ContentTypeToolResult }

func (t *ToolResultBlock) MarshalJSON() ([]byte, error) {
	type Alias ToolResultBlock
	return json.Marshal(&struct {
		Type ContentBlockType `json:"type"`
		*Alias
	}{
		Type:  ContentTypeToolResult,
		Alias: (*Alias)(t),
	})
}

func (t *ToolResultBlock) UnmarshalJSON(data []byte) error {
	type Alias ToolResultBlock
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.Type = ContentTypeToolResult
	return nil
}

type Conversation struct {
	Messages []Message `json:"messages"`
}

func NewConversation() *Conversation {
	return &Conversation{
		Messages: make([]Message, 0),
	}
}

func (c *Conversation) AddMessage(msg Message) {
	c.Messages = append(c.Messages, msg)
}

func (c *Conversation) GetMessages() []Message {
	return c.Messages
}

func (c *Conversation) Clear() {
	c.Messages = c.Messages[:0]
}

func (c *Conversation) Len() int {
	return len(c.Messages)
}