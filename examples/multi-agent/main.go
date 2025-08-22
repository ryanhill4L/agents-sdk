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

// Tool functions - same as basic example
func add(a, b int) int {
	return a + b
}

func greet(name string) string {
	return fmt.Sprintf("Hello, %s! Nice to meet you.", name)
}

func main() {
	fmt.Println("🤖 Agents SDK - Multi-Agent Handoff Example")
	fmt.Println("============================================")

	// Create tools
	addTool, err := tools.NewFunctionTool("add", "Performs addition of two integer numbers. Use this tool when you need to calculate the sum of two numeric values. Requires two integer parameters (a and b) and returns their mathematical sum.", add)
	if err != nil {
		log.Fatal("Failed to create add tool:", err)
	}

	greetTool, err := tools.NewFunctionTool("greet", "Generates a personalized greeting message for a specified person. Use this tool to create friendly, welcoming messages. Requires a person's name as input and returns a formatted greeting string.", greet)
	if err != nil {
		log.Fatal("Failed to create greet tool:", err)
	}

	// Create specialized agents
	mathAgent := agents.NewAgent("Math Specialist",
		agents.WithInstructions(`System: Role and Objective:
- Serve as a specialized mathematical assistant with precise calculation capabilities.
- Focus exclusively on mathematical operations and arithmetic calculations.

Instructions:
- Use the 'add' tool to perform addition operations with accuracy.
- Provide clear, concise mathematical results.
- Always use the tool rather than attempting manual calculations.
- Return results in a clear, numerical format.

Available Tools:
- 'add': Performs addition of two integer numbers. Use this for all addition requests.

Process:
1. Identify the numbers to be added from the user's request.
2. Use the 'add' tool with the correct parameters.
3. Present the result clearly and concisely.`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(addTool),
		agents.WithTemperature(0.1), // Low temperature for precise math
	)

	greetingAgent := agents.NewAgent("Greeting Specialist",
		agents.WithInstructions(`System: Role and Objective:
- Serve as a specialized greeting assistant focused on creating warm, personalized welcome messages.
- Provide friendly, engaging social interactions and introductions.

Instructions:
- Use the 'greet' tool to generate personalized greeting messages.
- Create warm, welcoming, and appropriate greetings for different contexts.
- Always use the tool to ensure consistent greeting quality.
- Maintain a friendly, professional tone.

Available Tools:
- 'greet': Generates personalized greeting messages. Use this for all greeting requests.

Process:
1. Identify the person's name from the user's request.
2. Use the 'greet' tool with the person's name as parameter.
3. Present the greeting in a warm, friendly manner.`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(greetTool),
		agents.WithTemperature(0.7), // Higher temperature for more creative greetings
	)

	// Create triage agent with handoffs
	triageAgent := agents.NewAgent("Triage Assistant",
		agents.WithInstructions(`System: Role and Objective:
- Serve as an intelligent triage assistant that analyzes user requests and routes them to appropriate specialized agents.
- Determine whether requests are mathematical in nature or social/greeting in nature, then hand off to the appropriate specialist.

Instructions:
- Analyze the user's request to determine the primary intent.
- For mathematical requests (addition, calculations, arithmetic): Hand off to "Math Specialist"
- For greeting requests (saying hello, welcoming someone, introductions): Hand off to "Greeting Specialist"
- Do not attempt to handle requests yourself - always delegate to the appropriate specialist.
- Make handoff decisions based on clear intent patterns.

Available Handoffs:
- "Math Specialist": For all mathematical operations, calculations, and arithmetic requests
- "Greeting Specialist": For all greeting, welcome, and social interaction requests

Decision Process:
1. Analyze the user request for key patterns:
   - Mathematical keywords: "add", "plus", "sum", "calculate", numbers, arithmetic operations
   - Greeting keywords: "hello", "hi", "greet", "welcome", "meet", person names
2. Determine the primary intent of the request.
3. Hand off to the appropriate specialist agent.
4. If unclear, default to the most likely intent based on context clues.

Important: Always hand off to a specialist - do not attempt to handle requests directly.`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithHandoffs(mathAgent, greetingAgent),
		agents.WithTemperature(0.3), // Lower temperature for consistent routing decisions
	)

	// Validate all agents
	if err := mathAgent.Validate(); err != nil {
		log.Fatal("Math agent validation failed:", err)
	}
	if err := greetingAgent.Validate(); err != nil {
		log.Fatal("Greeting agent validation failed:", err)
	}
	if err := triageAgent.Validate(); err != nil {
		log.Fatal("Triage agent validation failed:", err)
	}

	// Test cases to demonstrate handoffs
	testCases := []struct {
		description string
		input       string
		expectedAgent string
	}{
		{
			description: "Mathematical Request",
			input:       "Can you add 15 and 27 for me?",
			expectedAgent: "Math Specialist",
		},
		{
			description: "Greeting Request", 
			input:       "Please greet Sarah and make her feel welcome",
			expectedAgent: "Greeting Specialist",
		},
		{
			description: "Mixed Mathematical Request",
			input:       "Hello! I need help calculating the sum of 8 and 12",
			expectedAgent: "Math Specialist", // Should prioritize the calculation
		},
	}

	ctx := context.Background()

	// Check for API keys
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY") 
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if anthropicKey == "" && openaiKey == "" && geminiKey == "" {
		fmt.Println("⚠️  Warning: No API keys found in environment variables.")
		fmt.Println("Set ANTHROPIC_API_KEY, OPENAI_API_KEY, and/or GEMINI_API_KEY to test real API calls.")
		fmt.Println("Using placeholder keys for demonstration...")
		anthropicKey = "sk-ant-placeholder-key-demo"
		openaiKey = "sk-placeholder-key-demo"
		geminiKey = "placeholder-gemini-key-demo"
	}

	// Run tests with available provider
	var provider providers.Provider
	var providerName string

	if anthropicKey != "" && anthropicKey != "sk-ant-placeholder-key-demo" {
		p, err := providers.NewAnthropicProviderWithKey(anthropicKey)
		if err != nil {
			log.Fatal("Failed to create Anthropic provider:", err)
		}
		provider = p
		providerName = "Anthropic"
	} else if openaiKey != "" && openaiKey != "sk-placeholder-key-demo" {
		p, err := providers.NewOpenAIProviderWithKey(openaiKey)
		if err != nil {
			log.Fatal("Failed to create OpenAI provider:", err)
		}
		provider = p
		providerName = "OpenAI"
		// Update models for OpenAI
		mathAgent = agents.NewAgent("Math Specialist",
			agents.WithInstructions(mathAgent.GetInstructions()),
			agents.WithModel(openai.ChatModelChatgpt4oLatest),
			agents.WithTools(addTool),
			agents.WithTemperature(0.1),
		)
		greetingAgent = agents.NewAgent("Greeting Specialist", 
			agents.WithInstructions(greetingAgent.GetInstructions()),
			agents.WithModel(openai.ChatModelChatgpt4oLatest),
			agents.WithTools(greetTool),
			agents.WithTemperature(0.7),
		)
		triageAgent = agents.NewAgent("Triage Assistant",
			agents.WithInstructions(triageAgent.GetInstructions()),
			agents.WithModel(openai.ChatModelChatgpt4oLatest),
			agents.WithHandoffs(mathAgent, greetingAgent),
			agents.WithTemperature(0.3),
		)
	} else if geminiKey != "" && geminiKey != "placeholder-gemini-key-demo" {
		p, err := providers.NewGeminiProviderWithKey(geminiKey)
		if err != nil {
			log.Fatal("Failed to create Gemini provider:", err)
		}
		provider = p
		providerName = "Gemini"
		// Update models for Gemini
		mathAgent = agents.NewAgent("Math Specialist",
			agents.WithInstructions(mathAgent.GetInstructions()),
			agents.WithModel("gemini-2.0-flash"),
			agents.WithTools(addTool),
			agents.WithTemperature(0.1),
		)
		greetingAgent = agents.NewAgent("Greeting Specialist",
			agents.WithInstructions(greetingAgent.GetInstructions()),
			agents.WithModel("gemini-2.0-flash"),
			agents.WithTools(greetTool),
			agents.WithTemperature(0.7),
		)
		triageAgent = agents.NewAgent("Triage Assistant",
			agents.WithInstructions(triageAgent.GetInstructions()),
			agents.WithModel("gemini-2.0-flash"),
			agents.WithHandoffs(mathAgent, greetingAgent),
			agents.WithTemperature(0.3),
		)
	}

	if provider != nil {
		fmt.Printf("\n🚀 Testing Multi-Agent Handoffs with %s Provider\n", providerName)
		fmt.Println("===============================================")

		// Create runner
		runner := agents.NewRunner(
			agents.WithProvider(provider),
			agents.WithTracer(tracing.NewConsoleTracer()),
			agents.WithMaxTurns(5), // Allow multiple turns for handoffs
		)

		// Run test cases
		for i, testCase := range testCases {
			fmt.Printf("\n📋 Test Case %d: %s\n", i+1, testCase.description)
			fmt.Printf("🤔 User Input: \"%s\"\n", testCase.input)
			fmt.Printf("🎯 Expected Route: %s\n", testCase.expectedAgent)
			fmt.Println("---")

			result, err := runner.Run(ctx, triageAgent, testCase.input)
			if err != nil {
				fmt.Printf("❌ Test failed: %v\n", err)
				continue
			}

			if result != nil {
				fmt.Printf("✅ Response: %s\n", result.FinalOutput)
				fmt.Printf("📊 Tokens Used: %d\n", result.Metrics.TotalTokens)
				fmt.Printf("⏱️  Duration: %v\n", result.Metrics.Duration)
				fmt.Printf("🔄 Turns: %d\n", result.Metrics.TotalTurns)
			}
			
			fmt.Println("===============================================")
		}

		fmt.Println("\n🎉 Multi-Agent Handoff Demonstration Complete!")
		fmt.Println("\n💡 Key Features Demonstrated:")
		fmt.Println("   ✓ Intelligent request triage and routing")
		fmt.Println("   ✓ Specialized agent delegation using handoffs")
		fmt.Println("   ✓ Context-aware decision making")
		fmt.Println("   ✓ Seamless inter-agent communication")

	} else {
		fmt.Println("\n📝 Multi-Agent Handoff Example (Demo Mode)")
		fmt.Println("==========================================")
		fmt.Println("This example demonstrates how agents can intelligently route")
		fmt.Println("requests to specialized agents using the handoff mechanism:")
		fmt.Println("")
		fmt.Println("🏗️  Architecture:")
		fmt.Printf("   📋 Triage Agent: Analyzes requests and routes appropriately\n")
		fmt.Printf("   🧮 Math Specialist: Handles calculations using 'add' tool\n")  
		fmt.Printf("   👋 Greeting Specialist: Handles greetings using 'greet' tool\n")
		fmt.Println("")
		fmt.Println("🔄 Handoff Flow:")
		fmt.Println("   1. User sends request to Triage Agent")
		fmt.Println("   2. Triage analyzes request intent (math vs. greeting)")
		fmt.Println("   3. Triage hands off to appropriate specialist")
		fmt.Println("   4. Specialist processes request using their tools")
		fmt.Println("   5. Result returned to user")
		fmt.Println("")
		fmt.Println("🧪 Test Cases:")
		for i, testCase := range testCases {
			fmt.Printf("   %d. \"%s\" → %s\n", i+1, testCase.input, testCase.expectedAgent)
		}
		fmt.Println("")
		fmt.Println("🔧 To enable real API calls:")
		fmt.Println("   export ANTHROPIC_API_KEY='your-anthropic-key'")
		fmt.Println("   export OPENAI_API_KEY='your-openai-key'")  
		fmt.Println("   export GEMINI_API_KEY='your-gemini-key'")
	}

	fmt.Println("\n✨ Multi-agent handoff system ready!")
	fmt.Println("🌟 This demonstrates the power of specialized agent cooperation!")
}