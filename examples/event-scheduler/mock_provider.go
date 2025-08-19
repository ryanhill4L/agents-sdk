package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ryanhill4L/agents-sdk/pkg/providers"
)

// SmartMockProvider is a mock provider that intelligently responds to different types of requests
type SmartMockProvider struct{}

func NewSmartMockProvider() *SmartMockProvider {
	return &SmartMockProvider{}
}

func (p *SmartMockProvider) Complete(ctx context.Context, agent providers.Agent, messages []providers.Message, tools []providers.ToolDefinition) (*providers.Completion, error) {
	fmt.Printf("🔍 DEBUG: SmartMockProvider called with %d messages, %d tools\n", len(messages), len(tools))

	// Debug: print all messages
	for i, msg := range messages {
		fmt.Printf("🔍 DEBUG: Message %d [%s]: %s\n", i, msg.Role, msg.Content)
	}

	// Debug: print all available tools
	for i, tool := range tools {
		fmt.Printf("🔍 DEBUG: Tool %d: %s (%s)\n", i, tool.Name, tool.Description)
	}

	// Get the latest user message
	var userMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userMessage = strings.ToLower(messages[i].Content)
			break
		}
	}

	fmt.Printf("🔍 DEBUG: Processing user message: '%s'\n", userMessage)

	// Check if we have tool results in the conversation - if so, provide final answer
	hasToolResults := false
	for _, msg := range messages {
		if msg.Role == "tool" {
			hasToolResults = true
			break
		}
	}

	// Determine if this is a venue overlap request and we need to hand off
	if strings.Contains(userMessage, "venue") || strings.Contains(userMessage, "overlap") || strings.Contains(userMessage, "conflict") {
		fmt.Println("🔍 DEBUG: Detected venue/overlap/conflict request")

		// If we already have tool results, provide final summary
		if hasToolResults {
			fmt.Println("🔍 DEBUG: Tool results found, providing final summary")
			return &providers.Completion{
				Message: providers.Message{
					Role: "assistant",
					Content: `Based on the venue conflict analysis, I found 2 scheduling conflicts:

**Conflict #1: Conference Room A**
- Team Meeting (overlapping with Marketing Standup)

**Conflict #2: Main Hall** 
- Client Presentation (overlapping with Board Meeting)

**Recommendations:**
- Reschedule one of the conflicting events to a different time
- Move one event to a different venue  
- Check if events can be combined or made virtual

These conflicts need immediate attention to avoid double-booking venues.`,
					Timestamp: time.Now(),
				},
				Usage: providers.Usage{
					PromptTokens:     100,
					CompletionTokens: 80,
					TotalTokens:      180,
				},
			}, nil
		}

		// If we have tools, use them directly
		for _, tool := range tools {
			if tool.Name == "detect_venue_conflicts" {
				fmt.Println("🔍 DEBUG: Found venue conflicts tool, calling it")
				return &providers.Completion{
					Message: providers.Message{
						Role:      "assistant",
						Content:   "I'll check for venue conflicts using the detect_venue_conflicts tool.",
						Timestamp: time.Now(),
					},
					ToolCalls: []providers.ToolCall{
						{
							ID:        "call_1",
							Name:      "detect_venue_conflicts",
							Arguments: map[string]interface{}{},
						},
					},
					Usage: providers.Usage{
						PromptTokens:     50,
						CompletionTokens: 20,
						TotalTokens:      70,
					},
				}, nil
			}
		}

		// No tools available, so we need to hand off to Overlap Detector
		fmt.Println("🔍 DEBUG: No tools found, performing handoff to Overlap Detector")
		return &providers.Completion{
			Message: providers.Message{
				Role:      "assistant",
				Content:   "I'll route this request to the Overlap Detector to check for venue conflicts and scheduling issues.",
				Timestamp: time.Now(),
			},
			Handoff: &providers.HandoffRequest{
				TargetAgent: "Overlap Detector",
				Context: map[string]interface{}{
					"request_type": "venue_overlap",
					"user_query":   userMessage,
				},
				Reason: "User requested venue overlap detection",
			},
			Usage: providers.Usage{
				PromptTokens:     40,
				CompletionTokens: 15,
				TotalTokens:      55,
			},
		}, nil
	}

	// Default response for other requests
	return &providers.Completion{
		Message: providers.Message{
			Role:      "assistant",
			Content:   "I understand your request. Let me help you with that.",
			Timestamp: time.Now(),
		},
		Usage: providers.Usage{
			PromptTokens:     30,
			CompletionTokens: 10,
			TotalTokens:      40,
		},
	}, nil
}
