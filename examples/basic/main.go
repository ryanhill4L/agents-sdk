package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// Example tool: simple calculator
func add(a, b int) int {
	return a + b
}

func greet(name string) string {
	return fmt.Sprintf("Hello, %s! Nice to meet you.", name)
}

func main() {
	fmt.Println("🤖 Agents SDK - Provider Comparison Example")
	fmt.Println("==========================================")

	// Create some tools
	addTool, err := tools.NewFunctionTool("add", "Adds two numbers together", add)
	if err != nil {
		log.Fatal("Failed to create add tool:", err)
	}

	greetTool, err := tools.NewFunctionTool("greet", "Greets a person by name", greet)
	if err != nil {
		log.Fatal("Failed to create greet tool:", err)
	}

	// Create agents for different providers
	openaiAgent := agents.NewAgent("OpenAI Assistant",
		agents.WithInstructions("You are a helpful assistant powered by OpenAI."),
		agents.WithModel("gpt-4"),
		agents.WithTools(addTool, greetTool),
		agents.WithTemperature(0.7),
	)

	anthropicAgent := agents.NewAgent("Anthropic Assistant",
		agents.WithInstructions("You are a helpful assistant powered by Anthropic Claude."),
		agents.WithModel("claude-3-5-sonnet"),
		agents.WithTools(addTool, greetTool),
		agents.WithTemperature(0.7),
	)

	// Validate agents
	if err := openaiAgent.Validate(); err != nil {
		log.Fatal("OpenAI agent validation failed:", err)
	}
	if err := anthropicAgent.Validate(); err != nil {
		log.Fatal("Anthropic agent validation failed:", err)
	}

	// Test input
	input := "Hello! Can you add 5 and 3 for me?"
	ctx := context.Background()

	// Test OpenAI Provider
	fmt.Println("\n🔥 Testing OpenAI Provider")
	fmt.Println("==========================")
	
	openaiProvider, err := providers.NewOpenAIProviderWithKey("test-key")
	if err != nil {
		log.Fatal("Failed to create OpenAI provider:", err)
	}

	openaiRunner := agents.NewRunner(
		agents.WithProvider(openaiProvider),
		agents.WithTracer(tracing.NewConsoleTracer()),
		agents.WithMaxTurns(3),
	)

	openaiResult, err := openaiRunner.Run(ctx, openaiAgent, input)
	if err != nil {
		log.Fatal("OpenAI runner failed:", err)
	}

	fmt.Printf("📋 Agent: %s\n", openaiAgent.GetName())
	fmt.Printf("🤖 Model: %s\n", openaiAgent.GetModel())
	fmt.Printf("💬 Response: %s\n", openaiResult.FinalOutput)
	fmt.Printf("📊 Tokens: %d\n", openaiResult.Metrics.TotalTokens)
	fmt.Printf("⏱️  Duration: %v\n", openaiResult.Metrics.Duration)

	// Test Anthropic Provider
	fmt.Println("\n🟣 Testing Anthropic Provider")
	fmt.Println("=============================")
	
	anthropicProvider, err := providers.NewAnthropicProviderWithKey("test-key")
	if err != nil {
		log.Fatal("Failed to create Anthropic provider:", err)
	}

	anthropicRunner := agents.NewRunner(
		agents.WithProvider(anthropicProvider),
		agents.WithTracer(tracing.NewConsoleTracer()),
		agents.WithMaxTurns(3),
	)

	anthropicResult, err := anthropicRunner.Run(ctx, anthropicAgent, input)
	if err != nil {
		log.Fatal("Anthropic runner failed:", err)
	}

	fmt.Printf("📋 Agent: %s\n", anthropicAgent.GetName())
	fmt.Printf("🤖 Model: %s\n", anthropicAgent.GetModel())
	fmt.Printf("💬 Response: %s\n", anthropicResult.FinalOutput)
	fmt.Printf("📊 Tokens: %d\n", anthropicResult.Metrics.TotalTokens)
	fmt.Printf("⏱️  Duration: %v\n", anthropicResult.Metrics.Duration)

	// Compare results
	fmt.Println("\n📈 Comparison")
	fmt.Println("=============")
	fmt.Printf("OpenAI Duration: %v vs Anthropic Duration: %v\n", 
		openaiResult.Metrics.Duration, anthropicResult.Metrics.Duration)
	fmt.Printf("OpenAI Tokens: %d vs Anthropic Tokens: %d\n", 
		openaiResult.Metrics.TotalTokens, anthropicResult.Metrics.TotalTokens)
	
	fmt.Println("\n✅ Provider comparison completed successfully!")
	fmt.Println("💡 Note: These are placeholder implementations.")
	fmt.Println("🔧 Actual API integration requires valid API keys and proper SDK implementation.")
}
