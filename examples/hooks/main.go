package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	claudecode "github.com/ryanhill4L/agents-sdk"
	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	client, err := claudecode.NewClient(
		claudecode.WithAPIKey(apiKey),
		claudecode.WithModel("claude-3-5-sonnet-20241022"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	messageLogger := types.NewSimpleHook(
		"message_logger",
		[]types.HookEvent{types.HookPreMessage, types.HookPostMessage},
		func(ctx *types.HookContext) error {
			timestamp := time.Now().Format("15:04:05")
			switch ctx.Event {
			case types.HookPreMessage:
				fmt.Printf("[%s] Sending message...\n", timestamp)
			case types.HookPostMessage:
				fmt.Printf("[%s] Response received\n", timestamp)
			}
			return nil
		},
	)

	toolLogger := types.NewSimpleHook(
		"tool_logger",
		[]types.HookEvent{types.HookPreToolUse, types.HookPostToolUse},
		func(ctx *types.HookContext) error {
			if ctx.Tool != nil {
				switch ctx.Event {
				case types.HookPreToolUse:
					fmt.Printf("[TOOL] Executing: %s with args %v\n", ctx.Tool.Name, ctx.Tool.Arguments)
				case types.HookPostToolUse:
					if ctx.ToolResult != nil {
						if ctx.ToolResult.IsError {
							fmt.Printf("[TOOL] Error: %s\n", ctx.ToolResult.Error)
						} else {
							fmt.Printf("[TOOL] Success: %s returned result\n", ctx.Tool.Name)
						}
					}
				}
			}
			return nil
		},
	)

	errorHandler := types.NewSimpleHook(
		"error_handler",
		[]types.HookEvent{types.HookError},
		func(ctx *types.HookContext) error {
			if ctx.Error != nil {
				fmt.Printf("[ERROR] %v\n", ctx.Error)
			}
			return nil
		},
	)

	client.RegisterHook(messageLogger)
	client.RegisterHook(toolLogger)
	client.RegisterHook(errorHandler)

	fmt.Println("Hooks Example")
	fmt.Println("----------------------------------------")

	queries := []string{
		"Hello! How are you today?",
		"What's the weather like?",
		"Can you help me with Python?",
	}

	ctx := context.Background()

	for _, query := range queries {
		fmt.Printf("\nYou: %s\n", query)

		response, err := client.SendMessage(ctx, query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Print("Claude: ")
		for _, block := range response.GetContent() {
			if textBlock, ok := block.(*types.TextBlock); ok {
				fmt.Println(textBlock.Text)
			}
		}

		fmt.Println()
	}
}