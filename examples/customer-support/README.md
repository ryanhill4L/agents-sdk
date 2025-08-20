# Customer Support Multi-Agent System

This example demonstrates a sophisticated multi-agent customer support system that uses intelligent triage and dynamic workflow routing. It showcases how agents can analyze requests and hand off to specialized agents when needed.

## Architecture

### Agents Overview

1. **Triage Agent** (Orchestrator)
   - First point of contact for all customer requests
   - Analyzes requests to determine complexity and category
   - Routes to appropriate specialists or handles simple queries directly
   - Has access to FAQ tool for basic information

2. **Technical Support Specialist**
   - Handles technical issues, errors, and system problems
   - Tools: `check_system_status`, `analyze_error_logs`
   - Expertise: API issues, error troubleshooting, system diagnostics

3. **Billing Specialist**
   - Manages account, payment, and subscription issues
   - Tools: `check_account_status`, `process_refund`, `update_payment_method`
   - Expertise: Billing inquiries, refunds, payment troubleshooting

4. **Product Information Specialist**
   - Provides product details, pricing, and plan comparisons
   - Tools: `get_product_info`, `compare_plans`
   - Expertise: Feature explanations, upgrade recommendations

## Workflow Patterns

### Single Agent Flow
Simple queries that can be handled directly by the triage agent:
- Business hours, contact information
- Basic company information
- Simple FAQ questions

### Multi-Agent Flow
Complex issues requiring specialist expertise:
- Technical problems → Technical Support Specialist
- Billing issues → Billing Specialist
- Product questions → Product Information Specialist
- Complex multi-domain issues → Multiple handoffs

## Test Scenarios

The example includes 5 different scenarios that demonstrate various workflow patterns:

1. **Simple FAQ Query**: "What are your business hours?"
   - Flow: Triage agent handles directly using FAQ tool
   - Demonstrates: Single-agent resolution

2. **Technical Support Issue**: "I'm getting error E001 when trying to make a payment"
   - Flow: Triage → Technical Support Specialist
   - Demonstrates: Technical issue routing and error analysis

3. **Billing Inquiry**: "I need a refund for my last payment"
   - Flow: Triage → Billing Specialist
   - Demonstrates: Billing handoff and refund processing

4. **Product Comparison**: "What's the difference between Pro and Enterprise plans?"
   - Flow: Triage → Product Information Specialist
   - Demonstrates: Product information routing

5. **Complex Multi-Handoff**: "I want to upgrade but getting an error"
   - Flow: Triage → Technical Support → Billing Specialist
   - Demonstrates: Complex multi-agent collaboration

## Running the Example

### Prerequisites
- Go 1.24.3 or later
- Anthropic API key (optional - runs in demo mode without key)

### Setup
```bash
# Set API key (optional)
export ANTHROPIC_API_KEY="your-anthropic-api-key"

# From the project root
cd examples/customer-support

# Run the example
go run main.go
```

### From Root Directory
```bash
# Add to main Makefile and run
make run-customer-support-example
```

## Key Features Demonstrated

- **Dynamic Workflow Routing**: Intelligent analysis determines the appropriate handling path
- **Context Preservation**: Information flows seamlessly between agents during handoffs
- **Specialized Tools**: Each agent has domain-specific capabilities
- **Graceful Fallbacks**: Simple queries handled efficiently without unnecessary handoffs
- **Multi-Turn Conversations**: Complex issues resolved through collaborative agent work
- **Performance Metrics**: Token usage, duration, and handoff chain tracking

## Sample Output

```
🎯 SCENARIO: Simple FAQ Query
============================================================
👤 Customer: What are your business hours?

🤖 Final Response: Our support team is available Monday-Friday, 9 AM to 6 PM EST. Emergency support is available 24/7 for Enterprise customers.

📊 Metrics:
   • Total Turns: 1
   • Duration: 2.3s
   • Tokens Used: 156
```

## Extending the Example

You can easily extend this example by:

1. **Adding New Specialists**: Create agents for specific domains (e.g., Sales, Legal)
2. **Enhanced Tools**: Add database connections, external API integrations
3. **Complex Routing Logic**: Implement priority routing, escalation paths
4. **Multi-Language Support**: Add language detection and routing
5. **Sentiment Analysis**: Route based on customer emotion/urgency

## Code Structure

- `main.go`: Complete example with all agents and test scenarios
- Tool functions implement mock business logic for demonstration
- Agent instructions define clear handoff criteria and expertise areas
- Test scenarios cover both simple and complex workflow patterns