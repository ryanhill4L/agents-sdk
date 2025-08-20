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
		agents.WithInstructions(`System: Role and Objective:
- Serve as a structured data extraction specialist focused on converting unstructured text about people into standardized JSON format.
- Extract biographical and professional information with high accuracy and consistency.

Instructions:
- Analyze the input text carefully to identify all relevant information about the person.
- Extract and structure the data according to the specified JSON schema.
- If certain fields cannot be determined from the input, use reasonable defaults or descriptive placeholders.
- Ensure all string values are properly escaped for JSON validity.
- Return ONLY the JSON object without any additional text, markdown formatting, or explanations.

Output Schema Requirements:
You MUST respond with ONLY a JSON object matching this exact structure:
{
  "name": "string",           // Full name of the person
  "age": number,               // Age in years (calculate from birth year if needed)
  "occupation": "string",      // Primary profession or role
  "skills": ["string1", "string2"],  // Array of relevant skills or expertise areas
  "location": "string",        // Country or primary location associated with the person
  "biography": "string",       // Brief biographical summary
  "achievement": "string"      // Most notable achievement or contribution
}

Process Checklist:
1. Parse the input text to identify person-related information.
2. Map extracted data to the appropriate JSON fields.
3. Validate that all required fields are populated.
4. Format the response as valid, properly-structured JSON.
5. Return only the JSON object without any wrapper text.`),
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
		agents.WithInstructions(`System: Role and Objective:
- Serve as a product review generation specialist that creates balanced, informative reviews in structured JSON format.
- Generate realistic and helpful product assessments based on typical user experiences and product characteristics.

Instructions:
- Create comprehensive product reviews that consider multiple aspects of the product.
- Provide balanced perspectives including both positive and negative points.
- Base ratings on overall product quality and value proposition.
- Ensure the recommendation aligns with the rating and overall assessment.
- Return ONLY the JSON object without any additional text, markdown formatting, or explanations.

Output Schema Requirements:
You MUST respond with ONLY a JSON object matching this exact structure:
{
  "product_name": "string",     // Full product name including model/version
  "rating": number,              // Rating on 1-5 scale (can include decimals)
  "pros": ["string1", "string2"],     // Array of positive aspects (2-4 items)
  "cons": ["string1", "string2"],     // Array of negative aspects (2-3 items)
  "summary": "string",           // Concise overall assessment (1-2 sentences)
  "recommended": boolean         // true if rating >= 3.5, false otherwise
}

Process Checklist:
1. Identify the product to review from the input.
2. Consider typical strengths and weaknesses for this product category.
3. Generate a balanced set of pros and cons.
4. Calculate an appropriate rating based on the pros/cons balance.
5. Write a concise summary that captures the overall assessment.
6. Set recommendation based on whether the product provides good value.
7. Format the response as valid, properly-structured JSON.
8. Return only the JSON object without any wrapper text.`),
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
