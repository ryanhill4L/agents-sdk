package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

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
		claudecode.WithSystemPrompt("You are a helpful AI assistant."),
		claudecode.WithMaxTurns(20),
		claudecode.WithSession("./sessions.db", "demo-session"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("Interactive Claude Client")
	fmt.Println("Type 'quit' to exit, 'clear' to reset conversation")
	fmt.Println("----------------------------------------")

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print("\nYou: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		input = strings.TrimSpace(input)

		if input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if input == "clear" {
			client.ClearConversation()
			fmt.Println("Conversation cleared.")
			continue
		}

		response, err := client.SendMessage(ctx, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Print("\nClaude: ")
		for _, block := range response.GetContent() {
			if textBlock, ok := block.(*types.TextBlock); ok {
				fmt.Println(textBlock.Text)
			}
		}
	}
}