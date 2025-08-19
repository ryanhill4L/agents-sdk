package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// PersonInfo represents structured information about a person
type PersonInfo struct {
	Name        string   `json:"name"`
	Age         int      `json:"age"`
	Occupation  string   `json:"occupation"`
	Skills      []string `json:"skills"`
	Location    string   `json:"location"`
	Biography   string   `json:"biography"`
	Achievement string   `json:"achievement"`
}

// ProductReview represents a structured product review
type ProductReview struct {
	ProductName string   `json:"product_name"`
	Rating      float32  `json:"rating"` // 1-5 scale
	Pros        []string `json:"pros"`
	Cons        []string `json:"cons"`
	Summary     string   `json:"summary"`
	Recommended bool     `json:"recommended"`
}

func main() {
	fmt.Println("🔧 Agents SDK - Structured JSON Output Example")
	fmt.Println("===============================================")

	ctx := context.Background()

	// Check for API keys
	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	// Use the first available provider
	var provider providers.Provider
	var providerName string

	if openaiKey != "" {
		var err error
		provider, err = providers.NewOpenAIProviderWithKey(openaiKey)
		if err != nil {
			log.Fatal("Failed to create OpenAI provider:", err)
		}
		providerName = "OpenAI"
	} else if anthropicKey != "" {
		var err error
		provider, err = providers.NewAnthropicProviderWithKey(anthropicKey)
		if err != nil {
			log.Fatal("Failed to create Anthropic provider:", err)
		}
		providerName = "Anthropic"
	} else if geminiKey != "" {
		var err error
		provider, err = providers.NewGeminiProviderWithKey(geminiKey)
		if err != nil {
			log.Fatal("Failed to create Gemini provider:", err)
		}
		providerName = "Gemini"
	} else {
		fmt.Println("⚠️  Warning: No API keys found in environment variables.")
		fmt.Println("Set OPENAI_API_KEY, ANTHROPIC_API_KEY, or GEMINI_API_KEY to test real API calls.")
		fmt.Println("Using mock responses for demonstration...")

		// Use a mock provider that returns structured JSON
		provider = &MockProvider{}
		providerName = "Mock"
	}

	fmt.Printf("🤖 Using %s provider\n\n", providerName)

	// Example 1: Person Information Extraction
	fmt.Println("📝 Example 1: Extract person information as JSON")
	fmt.Println("================================================")

	personAgent := agents.NewAgent("JSON Person Extractor",
		agents.WithInstructions(`You are a structured data extractor. Your task is to extract information about a person and return it as valid JSON.

You MUST respond with ONLY a JSON object matching this exact structure:
{
  "name": "string",
  "age": number,
  "occupation": "string", 
  "skills": ["string1", "string2"],
  "location": "string",
  "biography": "string",
  "achievement": "string"
}

Do not include any other text, explanations, or formatting - just the JSON object.`),
		agents.WithModel("gpt-4o"),
		agents.WithTemperature(0.3),
	)

	personInput := "Extract information about Albert Einstein. He was a theoretical physicist born in Germany in 1879, known for developing the theory of relativity. He won the Nobel Prize in Physics in 1921."

	personResult, err := runJSONAgent(ctx, provider, personAgent, personInput)
	if err != nil {
		log.Printf("Person extraction failed: %v", err)
	} else {
		fmt.Printf("📋 Input: %s\n", personInput)

		// Convert FinalOutput to string
		responseStr, ok := personResult.FinalOutput.(string)
		if !ok {
			fmt.Printf("❌ FinalOutput is not a string: %T\n", personResult.FinalOutput)
			return
		}

		fmt.Printf("🔄 Raw Response: %s\n", responseStr)

		var person PersonInfo
		if err := parseJSONResponse(responseStr, &person); err != nil {
			fmt.Printf("❌ JSON parsing failed: %v\n", err)
		} else {
			fmt.Printf("✅ Parsed Person Info:\n")
			fmt.Printf("   Name: %s\n", person.Name)
			fmt.Printf("   Age: %d\n", person.Age)
			fmt.Printf("   Occupation: %s\n", person.Occupation)
			fmt.Printf("   Skills: %v\n", person.Skills)
			fmt.Printf("   Location: %s\n", person.Location)
			fmt.Printf("   Achievement: %s\n", person.Achievement)
		}
	}

	fmt.Println()

	// Example 2: Product Review Extraction
	fmt.Println("⭐ Example 2: Generate product review as JSON")
	fmt.Println("=============================================")

	reviewAgent := agents.NewAgent("JSON Review Generator",
		agents.WithInstructions(`You are a product review generator. Your task is to create a structured product review and return it as valid JSON.

You MUST respond with ONLY a JSON object matching this exact structure:
{
  "product_name": "string",
  "rating": number,
  "pros": ["string1", "string2"],
  "cons": ["string1", "string2"], 
  "summary": "string",
  "recommended": boolean
}

Do not include any other text, explanations, or formatting - just the JSON object.`),
		agents.WithModel("gpt-4o"),
		agents.WithTemperature(0.5),
	)

	reviewInput := "Generate a review for the iPhone 15 Pro based on typical user experiences with smartphones."

	reviewResult, err := runJSONAgent(ctx, provider, reviewAgent, reviewInput)
	if err != nil {
		log.Printf("Review generation failed: %v", err)
	} else {
		fmt.Printf("📋 Input: %s\n", reviewInput)

		// Convert FinalOutput to string
		responseStr, ok := reviewResult.FinalOutput.(string)
		if !ok {
			fmt.Printf("❌ FinalOutput is not a string: %T\n", reviewResult.FinalOutput)
			return
		}

		fmt.Printf("🔄 Raw Response: %s\n", responseStr)

		var review ProductReview
		if err := parseJSONResponse(responseStr, &review); err != nil {
			fmt.Printf("❌ JSON parsing failed: %v\n", err)
		} else {
			fmt.Printf("✅ Parsed Product Review:\n")
			fmt.Printf("   Product: %s\n", review.ProductName)
			fmt.Printf("   Rating: %d/5\n", review.Rating)
			fmt.Printf("   Pros: %v\n", review.Pros)
			fmt.Printf("   Cons: %v\n", review.Cons)
			fmt.Printf("   Summary: %s\n", review.Summary)
			fmt.Printf("   Recommended: %t\n", review.Recommended)
		}
	}

	fmt.Println("\n✅ Structured JSON output examples completed!")
	fmt.Println("💡 This demonstrates how to enforce structured responses from AI agents.")
	fmt.Println("🔧 Set environment variables to test with real providers:")
	fmt.Println("   export OPENAI_API_KEY='your-key'")
	fmt.Println("   export ANTHROPIC_API_KEY='your-key'")
	fmt.Println("   export GEMINI_API_KEY='your-key'")
}

// runJSONAgent executes an agent and returns the result
func runJSONAgent(ctx context.Context, provider providers.Provider, agent *agents.Agent, input string) (*agents.RunResult, error) {
	runner := agents.NewRunner(
		agents.WithProvider(provider),
		agents.WithTracer(tracing.NewConsoleTracer()),
		agents.WithMaxTurns(1), // Single turn for JSON output
	)

	return runner.Run(ctx, agent, input)
}

// parseJSONResponse attempts to parse the JSON response into the target struct
func parseJSONResponse(response string, target interface{}) error {
	// Clean the response - remove any markdown formatting
	cleaned := cleanJSONResponse(response)

	return json.Unmarshal([]byte(cleaned), target)
}

// cleanJSONResponse removes common formatting issues from LLM JSON responses
func cleanJSONResponse(response string) string {
	// Remove markdown code blocks if present
	if len(response) > 6 && response[:3] == "```" {
		// Find the first newline after ```
		start := 3
		for i := 3; i < len(response); i++ {
			if response[i] == '\n' {
				start = i + 1
				break
			}
		}

		// Find the ending ```
		end := len(response)
		for i := len(response) - 3; i >= 0; i-- {
			if i+2 < len(response) && response[i:i+3] == "```" {
				end = i
				break
			}
		}

		if start < end {
			response = response[start:end]
		}
	}

	return response
}

// MockProvider provides sample JSON responses for demonstration
type MockProvider struct{}

func (m *MockProvider) Complete(ctx context.Context, agent providers.Agent, messages []providers.Message, tools []providers.ToolDefinition) (*providers.Completion, error) {
	// Return different responses based on agent instructions
	instructions := agent.GetInstructions()

	var response string
	if contains(instructions, "person") || contains(instructions, "Person") {
		response = `{
  "name": "Albert Einstein", 
  "age": 76,
  "occupation": "Theoretical Physicist",
  "skills": ["Physics", "Mathematics", "Philosophy", "Scientific Research"],
  "location": "Germany/USA",
  "biography": "German-born theoretical physicist who developed the theory of relativity",
  "achievement": "Nobel Prize in Physics (1921) for photoelectric effect"
}`
	} else {
		response = `{
  "product_name": "iPhone 15 Pro",
  "rating": 4,
  "pros": ["Excellent camera quality", "Fast performance", "Premium build quality", "Good battery life"],
  "cons": ["Expensive", "USB-C adapter needed", "Can get warm during heavy use"],
  "summary": "A high-quality flagship smartphone with excellent features but at a premium price point",
  "recommended": true
}`
	}

	return &providers.Completion{
		Message: providers.Message{
			Role:    "assistant",
			Content: response,
		},
		Usage: providers.Usage{
			PromptTokens:     50,
			CompletionTokens: 100,
			TotalTokens:      150,
		},
	}, nil
}

// Helper function to check if string contains substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
