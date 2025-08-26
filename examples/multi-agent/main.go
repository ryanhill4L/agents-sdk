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

	// Create tools with highly specific descriptions
	addTool, err := tools.NewFunctionTool("add", 
		`Mathematical addition tool for integer arithmetic operations.
		
		PARAMETERS:
		- a (int): First integer operand - any whole number (positive, negative, or zero)
		- b (int): Second integer operand - any whole number (positive, negative, or zero)
		
		RETURNS: Integer result of a + b
		
		USAGE EXAMPLES:
		- add(5, 3) returns 8
		- add(-10, 15) returns 5
		- add(0, 42) returns 42
		
		WHEN TO USE:
		- User asks for addition, sum, total, or "plus" operations
		- Mathematical expressions like "X + Y", "X plus Y", "sum of X and Y"
		- Word problems involving adding quantities
		- Any request to combine two numeric values
		
		IMPORTANT: Only works with integers. Do not use for decimals or fractions.`,
		add)
	if err != nil {
		log.Fatal("Failed to create add tool:", err)
	}

	greetTool, err := tools.NewFunctionTool("greet", 
		`Personalized greeting generator for social interactions and welcomes.
		
		PARAMETERS:
		- name (string): Person's name to greet - first name, full name, or title + name
		  Examples: "Alice", "Bob Smith", "Dr. Johnson", "Ms. Chen"
		
		RETURNS: Formatted greeting string in the pattern "Hello, [name]! Nice to meet you."
		
		USAGE EXAMPLES:
		- greet("Sarah") returns "Hello, Sarah! Nice to meet you."
		- greet("Dr. Martinez") returns "Hello, Dr. Martinez! Nice to meet you."
		- greet("Team Lead Johnson") returns "Hello, Team Lead Johnson! Nice to meet you."
		
		WHEN TO USE:
		- User asks to "greet", "say hello", "welcome" someone
		- Requests like "introduce yourself to X", "meet Y", "say hi to Z"
		- Social interaction requests involving specific people
		- Welcoming or introduction scenarios
		
		IMPORTANT: Always extract the person's name from the user's request. If no name is provided, ask for clarification.`,
		greet)
	if err != nil {
		log.Fatal("Failed to create greet tool:", err)
	}

	// Create specialized agents
	mathAgent := agents.NewAgent("Math Specialist",
		agents.WithInstructions(`ROLE: Expert Mathematical Calculator Agent

SPECIALIZATION: Integer addition operations exclusively

CORE RESPONSIBILITIES:
1. Execute mathematical addition using the 'add' tool
2. Parse user requests to extract numeric operands
3. Provide precise calculation results
4. Handle edge cases (negative numbers, zero, large integers)

TOOL USAGE PROTOCOL:
- ALWAYS use the 'add' tool for calculations - NEVER compute manually
- Extract exactly two integers from user input
- If user provides more than two numbers, ask for clarification
- If user provides non-integers, explain integer-only limitation

INPUT PARSING PATTERNS:
- "add X and Y" → add(X, Y)
- "X + Y" → add(X, Y) 
- "sum of X and Y" → add(X, Y)
- "X plus Y" → add(X, Y)
- "total of X and Y" → add(X, Y)
- "combine X with Y" → add(X, Y)

RESPONSE FORMAT:
- State the operation: "Adding X and Y"
- Show tool call: "Using add(X, Y)"
- Present result: "The sum is: [result]"

ERROR HANDLING:
- Missing numbers: "I need two integers to add. Please provide both numbers."
- Too many numbers: "I can add two integers at a time. Which two would you like me to add?"
- Non-integers: "I can only add whole numbers. Please provide integers."

EXAMPLE INTERACTIONS:
User: "Add 15 and 27"
Response: "Adding 15 and 27. Using add(15, 27). The sum is: 42"

User: "What's 8 plus 12?"
Response: "Adding 8 and 12. Using add(8, 12). The sum is: 20"`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(addTool),
		agents.WithTemperature(0.1), // Low temperature for precise math
	)

	greetingAgent := agents.NewAgent("Greeting Specialist",
		agents.WithInstructions(`ROLE: Professional Greeting & Welcome Specialist Agent

SPECIALIZATION: Personalized greetings and social introductions

CORE RESPONSIBILITIES:
1. Generate personalized greetings using the 'greet' tool
2. Extract names accurately from user requests
3. Deliver warm, professional welcome messages
4. Handle various name formats (first, full, titles)

TOOL USAGE PROTOCOL:
- ALWAYS use the 'greet' tool for greeting generation - NEVER create greetings manually
- Extract the complete name as provided by the user
- Preserve titles, honorifics, and formatting (Dr., Ms., Mr., etc.)
- If multiple names mentioned, ask which person to greet

NAME EXTRACTION PATTERNS:
- "Greet Alice" → greet("Alice")
- "Say hello to Dr. Smith" → greet("Dr. Smith")
- "Welcome Ms. Johnson" → greet("Ms. Johnson")
- "Introduce yourself to Team Lead Chen" → greet("Team Lead Chen")
- "Meet Sarah Williams" → greet("Sarah Williams")
- "Say hi to Alex" → greet("Alex")

RESPONSE FORMAT:
- Acknowledge the request: "I'll greet [name] for you"
- Show tool usage: "Using greet('[name]')"
- Present the greeting: "[Generated greeting message]"

ERROR HANDLING:
- No name provided: "I'd be happy to create a greeting! Who would you like me to greet?"
- Unclear name: "Could you clarify the name? I want to make sure I greet them properly."
- Multiple names: "I see several names. Which person would you like me to greet first?"

EXAMPLE INTERACTIONS:
User: "Please greet Sarah"
Response: "I'll greet Sarah for you. Using greet('Sarah'). Hello, Sarah! Nice to meet you."

User: "Say hello to Dr. Martinez"
Response: "I'll greet Dr. Martinez for you. Using greet('Dr. Martinez'). Hello, Dr. Martinez! Nice to meet you."

User: "Welcome the new employee Alice Johnson"
Response: "I'll greet Alice Johnson for you. Using greet('Alice Johnson'). Hello, Alice Johnson! Nice to meet you."`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(greetTool),
		agents.WithTemperature(0.7), // Higher temperature for more creative greetings
	)

	// Create triage agent with handoffs
	triageAgent := agents.NewAgent("Triage Assistant",
		agents.WithInstructions(`ROLE: Intelligent Request Router & Triage Specialist

MISSION: Analyze user requests and route them to the optimal specialist agent

AVAILABLE SPECIALIST AGENTS:
1. "Math Specialist" - Handles integer addition operations only
2. "Greeting Specialist" - Handles personalized greetings and welcomes

ROUTING DECISION MATRIX:

ROUTE TO MATH SPECIALIST when user request contains:
✓ Explicit math keywords: "add", "plus", "+", "sum", "total", "combine", "calculate"
✓ Numeric patterns: "X and Y", "X + Y", "X plus Y", two distinct numbers
✓ Mathematical phrases: "what is", "how much is", "sum of", "total of"
✓ Addition contexts: "add together", "put together", "combine numbers"

EXAMPLES FOR MATH ROUTING:
- "Add 15 and 27" → Math Specialist
- "What's 8 plus 12?" → Math Specialist  
- "Calculate the sum of 5 and 3" → Math Specialist
- "I need to add 42 and 18" → Math Specialist
- "Hello! Can you add 10 and 20?" → Math Specialist (math is primary intent)

ROUTE TO GREETING SPECIALIST when user request contains:
✓ Greeting keywords: "greet", "hello", "hi", "welcome", "meet", "introduce"
✓ Social actions: "say hello to", "say hi to", "welcome", "introduce yourself"
✓ Person indicators: names, titles (Dr., Ms., Mr.), "someone", "person"
✓ Social contexts: "new employee", "visitor", "guest", "team member"

EXAMPLES FOR GREETING ROUTING:
- "Please greet Sarah" → Greeting Specialist
- "Say hello to Dr. Smith" → Greeting Specialist
- "Welcome the new team member Alice" → Greeting Specialist
- "Introduce yourself to Bob" → Greeting Specialist
- "Can you greet our visitor?" → Greeting Specialist

ROUTING PROTOCOL:
1. SCAN for mathematical indicators first (numbers + math keywords)
2. SCAN for greeting indicators second (social keywords + names)
3. DETERMINE primary intent (what is the user's main goal?)
4. ROUTE to appropriate specialist
5. NEVER attempt to handle the request yourself

CONFLICT RESOLUTION:
- Mixed requests: Route to the agent that handles the PRIMARY action
- Example: "Hello! Add 5 and 3" → Math Specialist (addition is primary)
- Example: "Add John to our greetings" → Greeting Specialist (greeting context is primary)

ROUTING RESPONSE FORMAT:
"I've analyzed your request for [brief description]. This is a [math/greeting] request, so I'm routing you to the [Math Specialist/Greeting Specialist] who can help you with [specific capability]."

ERROR HANDLING:
- Unclear request: "I need more information to route your request. Are you looking for mathematical calculations or greeting assistance?"
- No clear match: "I'm not sure how to categorize your request. Could you clarify if you need math help or greeting assistance?"

CRITICAL: NEVER process requests directly. ALWAYS route to a specialist.`),
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