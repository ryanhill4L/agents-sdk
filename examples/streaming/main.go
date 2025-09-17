package main

import (
	"context"
	"fmt"
	"log"
	"os"

	claudecode "github.com/ryanhill4L/agents-sdk"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	client, err := claudecode.NewClient(
		claudecode.WithAPIKey(apiKey),
		claudecode.WithModel("claude-3-5-sonnet-20241022"),
		claudecode.WithStreaming(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("Streaming Example")
	fmt.Println("----------------------------------------")

	ctx := context.Background()
	prompt := "Write a short story about a robot learning to paint. Make it creative and engaging."

	fmt.Printf("You: %s\n\n", prompt)
	fmt.Print("Claude: ")

	chunks, err := client.SendMessageStreaming(ctx, prompt)
	if err != nil {
		log.Fatal(err)
	}

	for chunk := range chunks {
		if chunk.Error != nil {
			fmt.Printf("\nError: %v\n", chunk.Error)
			break
		}

		if chunk.Content != "" {
			fmt.Print(chunk.Content)
		}

		if chunk.Done {
			fmt.Println("\n\n[Stream completed]")
			break
		}
	}
}