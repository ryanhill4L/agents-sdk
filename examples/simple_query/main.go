package main

import (
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

	result, err := claudecode.Query(
		"What is the capital of France? Give me a brief answer.",
		claudecode.WithAPIKey(apiKey),
		claudecode.WithModel("claude-3-5-sonnet-20241022"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response: %s\n", result.Content)
	fmt.Printf("Model: %s\n", result.Model)
	fmt.Printf("Tokens: %d\n", result.Usage.TotalTokens)
	fmt.Printf("Duration: %v\n", result.Duration)
}