package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	claudecode "github.com/ryanhill4L/agents-sdk"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

func add(a, b float64) float64 {
	return a + b
}

func multiply(a, b float64) float64 {
	return a * b
}

func sqrt(n float64) (float64, error) {
	if n < 0 {
		return 0, fmt.Errorf("cannot take square root of negative number")
	}
	return math.Sqrt(n), nil
}

func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	addTool := tools.Function("add", "Add two numbers", add)
	multiplyTool := tools.Function("multiply", "Multiply two numbers", multiply)
	sqrtTool := tools.Function("sqrt", "Calculate square root of a number", sqrt)
	timeTool := tools.Function("getCurrentTime", "Get the current time", getCurrentTime)

	client, err := claudecode.NewClient(
		claudecode.WithAPIKey(apiKey),
		claudecode.WithModel("claude-3-5-sonnet-20241022"),
		claudecode.WithSystemPrompt("You are a helpful math assistant with access to calculation tools."),
		claudecode.WithPermissionMode(types.PermissionAcceptAll),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	client.RegisterTool(addTool)
	client.RegisterTool(multiplyTool)
	client.RegisterTool(sqrtTool)
	client.RegisterTool(timeTool)

	fmt.Println("Math Assistant with Tools")
	fmt.Println("----------------------------------------")

	queries := []string{
		"What is 15 + 27?",
		"Calculate the square root of 144",
		"What is 12 multiplied by 8?",
		"What time is it right now?",
		"Calculate (10 + 5) * 3 and then find the square root of the result",
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
	}
}