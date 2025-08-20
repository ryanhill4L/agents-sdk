package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// Customer Support Tools

// Technical Support Tools
func checkSystemStatus(service string) string {
	statuses := map[string]string{
		"payment":      "operational",
		"api":          "degraded",
		"dashboard":    "operational",
		"notification": "maintenance",
	}
	
	if status, exists := statuses[service]; exists {
		return fmt.Sprintf("Service '%s' status: %s", service, status)
	}
	return fmt.Sprintf("Service '%s' not found. Available services: payment, api, dashboard, notification", service)
}

func analyzeErrorLogs(errorCode string) string {
	errors := map[string]string{
		"E001": "Payment gateway timeout. Recommend retrying in 5 minutes or using alternative payment method.",
		"E002": "Invalid API token. User needs to regenerate their API key from settings.",
		"E003": "Rate limit exceeded. User should wait 1 hour or upgrade plan for higher limits.",
		"E404": "Resource not found. Check if the requested item still exists or if user has proper permissions.",
	}
	
	if solution, exists := errors[errorCode]; exists {
		return fmt.Sprintf("Error %s: %s", errorCode, solution)
	}
	return fmt.Sprintf("Unknown error code %s. Please provide more details about when this error occurs.", errorCode)
}

// Billing Tools
func checkAccountStatus(customerID string) string {
	// Mock account data
	accounts := map[string]string{
		"CUST001": "Active - Pro Plan ($49/month) - Next billing: 2024-02-15",
		"CUST002": "Past Due - Basic Plan ($19/month) - Payment failed on 2024-01-15",
		"CUST003": "Active - Enterprise Plan ($199/month) - Annual billing",
	}
	
	if account, exists := accounts[customerID]; exists {
		return fmt.Sprintf("Account %s: %s", customerID, account)
	}
	return fmt.Sprintf("Customer ID %s not found. Please verify the ID is correct.", customerID)
}

func processRefund(customerID, amount, reason string) string {
	return fmt.Sprintf("Refund processed for customer %s: $%s refunded. Reason: %s. Refund will appear in 3-5 business days.", customerID, amount, reason)
}

func updatePaymentMethod(customerID, paymentMethod string) string {
	return fmt.Sprintf("Payment method updated for customer %s to %s. Next charge will use the new method.", customerID, paymentMethod)
}

// Product Information Tools
func getProductInfo(product string) string {
	products := map[string]string{
		"basic":      "Basic Plan: $19/month - 1,000 API calls, email support, basic analytics",
		"pro":        "Pro Plan: $49/month - 10,000 API calls, priority support, advanced analytics, custom integrations",
		"enterprise": "Enterprise Plan: $199/month - Unlimited API calls, dedicated support, white-label option, SLA guarantee",
	}
	
	if info, exists := products[product]; exists {
		return info
	}
	return "Available plans: basic, pro, enterprise. Please specify which plan you'd like information about."
}

func comparePlans(plan1, plan2 string) string {
	return fmt.Sprintf("Comparing %s vs %s: The main differences are API limits, support level, and advanced features. Pro plan includes priority support and custom integrations not available in Basic.", plan1, plan2)
}

// FAQ Tool for simple queries
func getFAQ(question string) string {
	faqs := map[string]string{
		"hours":     "Our support team is available Monday-Friday, 9 AM to 6 PM EST. Emergency support is available 24/7 for Enterprise customers.",
		"contact":   "You can reach us at support@example.com, through the in-app chat, or by calling 1-800-SUPPORT.",
		"api-docs":  "API documentation is available at docs.example.com. It includes getting started guides, endpoint references, and code examples.",
		"downtime":  "We maintain 99.9% uptime. Scheduled maintenance is announced 48 hours in advance via email and status page.",
		"security":  "We use industry-standard encryption, SOC 2 compliance, and regular security audits to protect your data.",
	}
	
	for key, answer := range faqs {
		if question == key {
			return answer
		}
	}
	return ""
}

func createSupportAgents() (*agents.Agent, *agents.Agent, *agents.Agent, *agents.Agent) {
	// Create tools for each agent
	
	// Technical Support Tools
	systemStatusTool, _ := tools.NewFunctionTool("check_system_status", 
		"Check the current operational status of various services (payment, api, dashboard, notification). Use this to diagnose service-related issues.", 
		checkSystemStatus)
	
	errorAnalysisTool, _ := tools.NewFunctionTool("analyze_error_logs", 
		"Analyze error codes and provide troubleshooting solutions. Provide the error code to get detailed resolution steps.", 
		analyzeErrorLogs)
	
	// Billing Tools
	accountStatusTool, _ := tools.NewFunctionTool("check_account_status", 
		"Check a customer's current account status, billing plan, and payment information. Requires customer ID.", 
		checkAccountStatus)
	
	refundTool, _ := tools.NewFunctionTool("process_refund", 
		"Process a refund for a customer. Requires customer ID, refund amount, and reason for the refund.", 
		processRefund)
	
	paymentUpdateTool, _ := tools.NewFunctionTool("update_payment_method", 
		"Update a customer's payment method. Requires customer ID and new payment method details.", 
		updatePaymentMethod)
	
	// Product Information Tools
	productInfoTool, _ := tools.NewFunctionTool("get_product_info", 
		"Get detailed information about our product plans (basic, pro, enterprise) including pricing and features.", 
		getProductInfo)
	
	compareTool, _ := tools.NewFunctionTool("compare_plans", 
		"Compare two different product plans to help customers understand the differences and make informed decisions.", 
		comparePlans)
	
	// FAQ Tool
	faqTool, _ := tools.NewFunctionTool("get_faq", 
		"Get answers to frequently asked questions about hours, contact, api-docs, downtime, security.", 
		getFAQ)

	// Create specialized agents
	technicalAgent := agents.NewAgent("Technical Support Specialist",
		agents.WithInstructions(`You are a Technical Support Specialist focused on resolving technical issues and system problems.

Your expertise includes:
- Diagnosing system status and service issues
- Analyzing error codes and providing solutions
- Troubleshooting API, payment gateway, and platform issues
- Providing step-by-step technical guidance

Always:
- Use the available tools to check system status and analyze errors
- Provide clear, actionable troubleshooting steps
- Escalate complex issues when necessary
- Be specific about timelines and next steps`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(systemStatusTool, errorAnalysisTool),
		agents.WithTemperature(0.3),
	)

	billingAgent := agents.NewAgent("Billing Specialist",
		agents.WithInstructions(`You are a Billing Specialist focused on account, payment, and subscription issues.

Your expertise includes:
- Account status and billing inquiries
- Processing refunds and payment updates
- Subscription changes and upgrades
- Payment troubleshooting

Always:
- Verify customer identity before accessing account information
- Use tools to check account status and process transactions
- Explain billing cycles and payment processes clearly
- Offer solutions for payment issues
- Be empathetic about billing concerns`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(accountStatusTool, refundTool, paymentUpdateTool),
		agents.WithTemperature(0.4),
	)

	productAgent := agents.NewAgent("Product Information Specialist",
		agents.WithInstructions(`You are a Product Information Specialist focused on helping customers understand our offerings.

Your expertise includes:
- Detailed product and plan information
- Feature comparisons and recommendations
- Pricing and upgrade guidance
- Helping customers choose the right plan

Always:
- Use tools to provide accurate, up-to-date product information
- Make clear comparisons between different plans
- Recommend the best fit based on customer needs
- Explain features and benefits clearly
- Be helpful in guiding purchase decisions`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(productInfoTool, compareTool),
		agents.WithTemperature(0.5),
	)

	// Create triage agent with handoffs to specialists
	triageAgent := agents.NewAgent("Customer Support Triage",
		agents.WithInstructions(`You are the first point of contact for customer support. Your role is to:

1. ANALYZE the customer's request to understand the nature of their issue
2. DETERMINE if you can handle it directly (simple FAQ) or if it needs a specialist
3. ROUTE appropriately:
   - Technical issues (errors, system problems, API issues) → Hand off to "Technical Support Specialist"
   - Billing/payment issues (refunds, account status, subscriptions) → Hand off to "Billing Specialist"  
   - Product questions (features, pricing, comparisons) → Hand off to "Product Information Specialist"
   - Simple FAQ questions (hours, contact info, basic info) → Handle directly with the FAQ tool

For HANDOFFS:
- Always explain to the customer that you're connecting them with a specialist
- Provide a brief summary of their issue when handing off
- Only hand off when the issue clearly falls into a specialist's domain

For DIRECT HANDLING:
- Use the FAQ tool for simple questions about business hours, contact info, etc.
- Keep responses helpful and professional
- If the FAQ tool returns empty, consider if a specialist would be better

Categories:
- TECHNICAL: Error codes, system status, API problems, platform issues, bugs
- BILLING: Account status, payments, refunds, subscription changes, billing questions  
- PRODUCT: Plan features, pricing, comparisons, recommendations, upgrades
- SIMPLE: Business hours, contact information, general company info`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(faqTool),
		agents.WithHandoffs(technicalAgent, billingAgent, productAgent),
		agents.WithTemperature(0.6),
	)

	return triageAgent, technicalAgent, billingAgent, productAgent
}

func runScenario(ctx context.Context, runner *agents.Runner, triageAgent *agents.Agent, scenario, input string) {
	fmt.Printf("\n" + "="*60 + "\n")
	fmt.Printf("🎯 SCENARIO: %s\n", scenario)
	fmt.Printf("="*60 + "\n")
	fmt.Printf("👤 Customer: %s\n\n", input)

	result, err := runner.Run(ctx, triageAgent, input)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("🤖 Final Response: %s\n", result.FinalOutput)
	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   • Total Turns: %d\n", result.Metrics.TotalTurns)
	fmt.Printf("   • Duration: %v\n", result.Metrics.Duration)
	fmt.Printf("   • Tokens Used: %d\n", result.Metrics.TotalTokens)
	
	if len(result.HandoffChain) > 0 {
		fmt.Printf("   • Agents Used: ")
		for i, agent := range result.HandoffChain {
			if i > 0 {
				fmt.Printf(" → ")
			}
			fmt.Printf("%s", agent)
		}
		fmt.Printf("\n")
	}
}

func main() {
	fmt.Println("🏢 Customer Support Multi-Agent System")
	fmt.Println("=====================================")
	fmt.Println("Demonstrating dynamic workflow with triage and specialist agents")

	// Create all support agents
	triageAgent, technicalAgent, billingAgent, productAgent := createSupportAgents()

	// Validate agents
	agents := []*agents.Agent{triageAgent, technicalAgent, billingAgent, productAgent}
	for _, agent := range agents {
		if err := agent.Validate(); err != nil {
			log.Fatalf("Agent validation failed for %s: %v", agent.GetName(), err)
		}
	}

	// Set up provider
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		fmt.Println("⚠️  Warning: No ANTHROPIC_API_KEY found. Using placeholder key for demo.")
		anthropicKey = "sk-ant-placeholder-demo"
	}

	provider, err := providers.NewAnthropicProviderWithKey(anthropicKey)
	if err != nil {
		log.Fatal("Failed to create Anthropic provider:", err)
	}

	// Create runner
	runner := agents.NewRunner(
		agents.WithProvider(provider),
		agents.WithTracer(tracing.NewConsoleTracer()),
		agents.WithMaxTurns(10),
		agents.WithParallelTools(true),
	)

	ctx := context.Background()

	// Test scenarios demonstrating different workflow paths

	// Scenario 1: Simple FAQ - Single agent handles directly
	runScenario(ctx, runner, triageAgent,
		"Simple FAQ Query", 
		"What are your business hours?")

	// Scenario 2: Technical Issue - Multi-agent handoff
	runScenario(ctx, runner, triageAgent,
		"Technical Support Issue",
		"I'm getting error E001 when trying to make a payment. The payment button just keeps loading.")

	// Scenario 3: Billing Issue - Handoff to billing specialist
	runScenario(ctx, runner, triageAgent,
		"Billing Inquiry",
		"I need a refund for my last payment. My customer ID is CUST002 and I was charged $19 but the service wasn't working.")

	// Scenario 4: Product Information - Handoff to product specialist
	runScenario(ctx, runner, triageAgent,
		"Product Comparison",
		"I'm currently on the basic plan but need more API calls. What's the difference between Pro and Enterprise plans?")

	// Scenario 5: Complex Multi-Agent - Technical + Billing
	runScenario(ctx, runner, triageAgent,
		"Complex Multi-Handoff",
		"I want to upgrade to Pro plan but when I click the upgrade button I get error E002. Can you help me upgrade and also tell me when I'll be charged?")

	fmt.Println("\n✅ Customer Support demonstration completed!")
	fmt.Printf("\n💡 Key Features Demonstrated:\n")
	fmt.Printf("   • Dynamic workflow routing based on request analysis\n")
	fmt.Printf("   • Intelligent triage agent that routes to specialists\n")
	fmt.Printf("   • Direct handling of simple FAQ queries\n")
	fmt.Printf("   • Multi-agent handoffs for complex issues\n")
	fmt.Printf("   • Specialized agents with domain-specific tools\n")
	fmt.Printf("   • Context preservation across handoffs\n")
}