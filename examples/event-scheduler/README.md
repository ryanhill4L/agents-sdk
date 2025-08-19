# Event Scheduler Agent Example

This example demonstrates building an AI agent system that manages event scheduling and detects overlaps in user schedules using the Go Agents SDK.

## Features

- **Multi-agent system** with specialized roles
- **Database integration** with SQLite for event and schedule storage  
- **Tool execution** for database queries and conflict detection
- **Agent handoffs** between scheduler and overlap detector agents
- **Guardrails** for privacy and security protection
- **Interactive CLI** for testing the agents

## Architecture

### Agents

1. **Triage Agent** - Main coordinator that routes requests to specialists
2. **Scheduler Agent** - Handles general event and schedule queries  
3. **Overlap Detector Agent** - Specialized in finding scheduling conflicts

### Tools

- `query_events` - Execute SQL queries against the events database
- `detect_user_overlaps` - Find users with conflicting schedules
- `detect_venue_conflicts` - Find venues with overlapping bookings

### Database Schema

- `events` table - Stores event information (name, date, location, duration)
- `user_schedules` table - Links users to events with working hours

## Prerequisites

- Go 1.21 or later
- OpenAI API key

## Setup

1. **Set your OpenAI API key:**
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

2. **Install dependencies:**
   ```bash
   go mod tidy
   ```

3. **Run the example:**
   ```bash
   go run main.go
   ```

## Usage

The application creates a SQLite database with sample data and provides an interactive CLI. Try these example queries:

- `Show me all scheduled events`
- `Find scheduling conflicts for users`  
- `Check for venue overlaps`
- `What events is Alice attending?`
- `Who is attending the Team Meeting?`

Type `exit` to quit the application.

## Sample Output

```
🗓️  Event Scheduling Assistant Ready!
=====================================
Try asking:
  - 'Show me all scheduled events'
  - 'Find scheduling conflicts for users'
  - 'Check for venue overlaps'
  - 'What events is Alice attending?'

Type 'exit' to quit

> Show me all scheduled events

🤖 Here are all the scheduled events:

1. **Team Meeting** - Tomorrow at Conference Room A (2 hours)
2. **Project Review** - Tomorrow at Conference Room B (1 hour) 
3. **Client Presentation** - Day after tomorrow at Main Hall (3 hours)
4. **Training Session** - In 3 days at Training Room (4 hours)

📊 Metrics: 2 turns, 1 tool calls, 1 handoffs, 245 tokens, 1.2s duration

> Find scheduling conflicts for users

🤖 I found the following scheduling conflicts:

**Alice** has overlapping events:
- Team Meeting and Project Review are both scheduled for tomorrow

**Charlie** also has conflicts:
- Project Review and Client Presentation have overlapping time slots

I recommend rescheduling one of Alice's events and adjusting Charlie's schedule.

📊 Metrics: 3 turns, 1 tool calls, 1 handoffs, 312 tokens, 1.8s duration
```

## Key Implementation Details

### Agent Creation with Handoffs

```go
triageAgent := agents.NewAgent("Scheduling Triage",
    agents.WithInstructions(`You are the main scheduling coordinator...`),
    agents.WithModel("gpt-4"),
    agents.WithHandoffs(scheduler.GetAgent(), overlapDetector.GetAgent()),
)
```

### Custom Tools with Database Access

```go
queryTool, err := tools.NewFunctionTool("query_events", 
    "Query the events database with SQL", 
    createQueryHandler(db))
```

### Guardrails for Security

```go
privacyGuardrail := NewPrivacyGuardrail()
agent = agents.NewAgent(name,
    agents.WithGuardrails(privacyGuardrail),
    // other options...
)
```

### Provider Configuration

```go
provider, err := providers.NewOpenAIProviderFromEnv()
runner := agents.NewRunner(
    agents.WithProvider(provider),
    agents.WithTracer(tracing.NewConsoleTracer()),
    agents.WithMaxTurns(5),
)
```

## Next Steps

This example can be extended with:

- More sophisticated guardrails and validation
- Additional AI providers (Anthropic Claude, Google Gemini)
- REST API endpoints for web integration
- Advanced scheduling algorithms
- Calendar system integrations
- Email notifications for conflicts
- Multi-tenant support

For more examples and documentation, see the [Go Agents SDK repository](https://github.com/ryanhill4L/agents-sdk).