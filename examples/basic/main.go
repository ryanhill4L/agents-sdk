package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go"
	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// Example tool: simple calculator
func add(a, b int) int {
	return a + b
}

// Example tool: simple greeting
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
		agents.WithModel(openai.ChatModelChatgpt4oLatest),
		agents.WithTools(addTool, greetTool),
		agents.WithTemperature(0.7),
	)

	anthropicAgent := agents.NewAgent("Anthropic Assistant",
		agents.WithInstructions("You are a helpful assistant powered by Anthropic Claude."),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(addTool, greetTool),
		agents.WithTemperature(0.7),
	)

	geminiAgent := agents.NewAgent("Gemini Assistant",
		agents.WithInstructions("You are a helpful assistant powered by Google Gemini."),
		agents.WithModel("gemini-2.0-flash"),
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
	if err := geminiAgent.Validate(); err != nil {
		log.Fatal("Gemini agent validation failed:", err)
	}

	// Test input
	input := "Hello! Can you add 5 and 3 for me?"
	ctx := context.Background()

	// Check for API keys
	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if openaiKey == "" && anthropicKey == "" && geminiKey == "" {
		fmt.Println("⚠️  Warning: No API keys found in environment variables.")
		fmt.Println("Set OPENAI_API_KEY, ANTHROPIC_API_KEY, and/or GEMINI_API_KEY to test real API calls.")
		fmt.Println("Using placeholder keys for demonstration...")
		openaiKey = "sk-placeholder-key-demo"
		anthropicKey = "sk-ant-placeholder-key-demo"
		geminiKey = "placeholder-gemini-key-demo"
	}

	var openaiResult *agents.RunResult
	var anthropicResult *agents.RunResult
	var geminiResult *agents.RunResult

	// Test OpenAI Provider
	if openaiKey != "" {
		fmt.Println("\n🔥 Testing OpenAI Provider")
		fmt.Println("==========================")

		openaiProvider, err := providers.NewOpenAIProviderWithKey(openaiKey)
		if err != nil {
			log.Fatal("Failed to create OpenAI provider:", err)
		}

		openaiRunner := agents.NewRunner(
			agents.WithProvider(openaiProvider),
			agents.WithTracer(tracing.NewConsoleTracer()),
			agents.WithMaxTurns(3),
		)

		openaiResult, err = openaiRunner.Run(ctx, openaiAgent, input)
		if err != nil {
			fmt.Printf("⚠️  OpenAI runner failed: %v\n", err)
			fmt.Println("   This might be due to API quota, invalid model, or network issues.")
		}

		if openaiResult != nil {
			fmt.Printf("📋 Agent: %s\n", openaiAgent.GetName())
			fmt.Printf("🤖 Model: %s\n", openaiAgent.GetModel())
			fmt.Printf("💬 Response: %s\n", openaiResult.FinalOutput)
			fmt.Printf("📊 Tokens: %d\n", openaiResult.Metrics.TotalTokens)
			fmt.Printf("⏱️  Duration: %v\n", openaiResult.Metrics.Duration)
		}
	} else {
		fmt.Println("\n🔥 OpenAI Provider")
		fmt.Println("==========================")
		fmt.Println("⏭️  Skipping OpenAI - no API key provided")
	}

	// Test Anthropic Provider
	if anthropicKey != "" {
		fmt.Println("\n🟣 Testing Anthropic Provider")
		fmt.Println("=============================")

		anthropicProvider, err := providers.NewAnthropicProviderWithKey(anthropicKey)
		if err != nil {
			log.Fatal("Failed to create Anthropic provider:", err)
		}

		anthropicRunner := agents.NewRunner(
			agents.WithProvider(anthropicProvider),
			agents.WithTracer(tracing.NewConsoleTracer()),
			agents.WithMaxTurns(3),
		)

		anthropicResult, err = anthropicRunner.Run(ctx, anthropicAgent, input)
		if err != nil {
			fmt.Printf("⚠️  Anthropic runner failed: %v\n", err)
			fmt.Println("   This might be due to API quota, invalid model, or network issues.")
		}

		if anthropicResult != nil {
			fmt.Printf("📋 Agent: %s\n", anthropicAgent.GetName())
			fmt.Printf("🤖 Model: %s\n", anthropicAgent.GetModel())
			fmt.Printf("💬 Response: %s\n", anthropicResult.FinalOutput)
			fmt.Printf("📊 Tokens: %d\n", anthropicResult.Metrics.TotalTokens)
			fmt.Printf("⏱️  Duration: %v\n", anthropicResult.Metrics.Duration)
		}
	} else {
		fmt.Println("\n🟣 Anthropic Provider")
		fmt.Println("=============================")
		fmt.Println("⏭️  Skipping Anthropic - no API key provided")
	}

	// Test Gemini Provider
	if geminiKey != "" {
		fmt.Println("\n🔷 Testing Gemini Provider")
		fmt.Println("==========================")

		geminiProvider, err := providers.NewGeminiProviderWithKey(geminiKey)
		if err != nil {
			log.Fatal("Failed to create Gemini provider:", err)
		}

		geminiRunner := agents.NewRunner(
			agents.WithProvider(geminiProvider),
			agents.WithTracer(tracing.NewConsoleTracer()),
			agents.WithMaxTurns(3),
		)

		geminiResult, err = geminiRunner.Run(ctx, geminiAgent, input)
		if err != nil {
			fmt.Printf("⚠️  Gemini runner failed: %v\n", err)
			fmt.Println("   This might be due to API quota, invalid model, or network issues.")
		}

		if geminiResult != nil {
			fmt.Printf("📋 Agent: %s\n", geminiAgent.GetName())
			fmt.Printf("🤖 Model: %s\n", geminiAgent.GetModel())
			fmt.Printf("💬 Response: %s\n", geminiResult.FinalOutput)
			fmt.Printf("📊 Tokens: %d\n", geminiResult.Metrics.TotalTokens)
			fmt.Printf("⏱️  Duration: %v\n", geminiResult.Metrics.Duration)
		}
	} else {
		fmt.Println("\n🔷 Gemini Provider")
		fmt.Println("==========================")
		fmt.Println("⏭️  Skipping Gemini - no API key provided")
	}

	// Compare results (if we have multiple)
	resultCount := 0
	if openaiResult != nil {
		resultCount++
	}
	if anthropicResult != nil {
		resultCount++
	}
	if geminiResult != nil {
		resultCount++
	}

	if resultCount > 1 {
		fmt.Println("\n📈 Comparison")
		fmt.Println("=============")
		
		if openaiResult != nil {
			fmt.Printf("OpenAI - Duration: %v, Tokens: %d\n",
				openaiResult.Metrics.Duration, openaiResult.Metrics.TotalTokens)
		}
		if anthropicResult != nil {
			fmt.Printf("Anthropic - Duration: %v, Tokens: %d\n",
				anthropicResult.Metrics.Duration, anthropicResult.Metrics.TotalTokens)
		}
		if geminiResult != nil {
			fmt.Printf("Gemini - Duration: %v, Tokens: %d\n",
				geminiResult.Metrics.Duration, geminiResult.Metrics.TotalTokens)
		}
	}

	fmt.Println("\n✅ Provider demonstration completed successfully!")
	fmt.Println("💡 Note: These are real API implementations using official SDKs.")
	fmt.Println("🔧 To enable real API calls:")
	fmt.Println("   export OPENAI_API_KEY='your-openai-key'")
	fmt.Println("   export ANTHROPIC_API_KEY='your-anthropic-key'")
	fmt.Println("   export GEMINI_API_KEY='your-gemini-key'")
	fmt.Println("🚀 OpenAI, Anthropic, and Gemini integrations are now fully functional!")
}
