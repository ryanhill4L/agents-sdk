# Go Agents SDK Quickstart

Build an AI agent that manages event scheduling and detects overlaps in user schedules.

## Prerequisites

### Install Go
Ensure you have Go 1.21 or later installed:
```bash
go version
```

### Create a new project
```bash
mkdir event-scheduler-agent
cd event-scheduler-agent
go mod init event-scheduler
```

### Install the Agents SDK
```bash
go get github.com/ryanhill4L/agents-sdk
```

### Install database dependencies
```bash
go get github.com/mattn/go-sqlite3
go get github.com/jmoiron/sqlx
```

### Set your OpenAI API key
```bash
export OPENAI_API_KEY="your-api-key-here"
```

## Set up the database

First, let's create our database schema and populate it with sample data.

```go
// db/setup.go
package db

import (
    "log"
    "time"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/mattn/go-sqlite3"
)

type Event struct {
    ID       int       `db:"id"`
    Name     string    `db:"name"`
    Date     time.Time `db:"date"`
    Location string    `db:"location"`
    Duration int       `db:"duration_hours"`
}

type UserSchedule struct {
    ID          int    `db:"id"`
    UserID      int    `db:"user_id"`
    UserName    string `db:"user_name"`
    EventID     int    `db:"event_id"`
    WorkingFrom string `db:"working_from"`
    WorkingTo   string `db:"working_to"`
}

func InitDB() (*sqlx.DB, error) {
    db, err := sqlx.Open("sqlite3", "./events.db")
    if err != nil {
        return nil, err
    }
    
    schema := `
    CREATE TABLE IF NOT EXISTS events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        date DATETIME NOT NULL,
        location TEXT NOT NULL,
        duration_hours INTEGER DEFAULT 1
    );
    
    CREATE TABLE IF NOT EXISTS user_schedules (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        user_name TEXT NOT NULL,
        event_id INTEGER NOT NULL,
        working_from TEXT NOT NULL,
        working_to TEXT NOT NULL,
        FOREIGN KEY (event_id) REFERENCES events(id)
    );`
    
    db.MustExec(schema)
    
    // Add sample data
    seedData(db)
    
    return db, nil
}

func seedData(db *sqlx.DB) {
    // Clear existing data
    db.MustExec("DELETE FROM user_schedules")
    db.MustExec("DELETE FROM events")
    
    // Insert sample events
    events := []Event{
        {Name: "Team Meeting", Date: time.Now().Add(24 * time.Hour), Location: "Conference Room A", Duration: 2},
        {Name: "Project Review", Date: time.Now().Add(24 * time.Hour), Location: "Conference Room B", Duration: 1},
        {Name: "Client Presentation", Date: time.Now().Add(48 * time.Hour), Location: "Main Hall", Duration: 3},
        {Name: "Training Session", Date: time.Now().Add(72 * time.Hour), Location: "Training Room", Duration: 4},
    }
    
    for _, e := range events {
        db.MustExec(
            "INSERT INTO events (name, date, location, duration_hours) VALUES (?, ?, ?, ?)",
            e.Name, e.Date, e.Location, e.Duration,
        )
    }
    
    // Insert sample user schedules
    schedules := []UserSchedule{
        {UserID: 1, UserName: "Alice", EventID: 1, WorkingFrom: "09:00", WorkingTo: "17:00"},
        {UserID: 1, UserName: "Alice", EventID: 2, WorkingFrom: "09:00", WorkingTo: "17:00"},
        {UserID: 2, UserName: "Bob", EventID: 1, WorkingFrom: "10:00", WorkingTo: "18:00"},
        {UserID: 2, UserName: "Bob", EventID: 3, WorkingFrom: "10:00", WorkingTo: "18:00"},
        {UserID: 3, UserName: "Charlie", EventID: 2, WorkingFrom: "08:00", WorkingTo: "16:00"},
        {UserID: 3, UserName: "Charlie", EventID: 3, WorkingFrom: "08:00", WorkingTo: "16:00"},
    }
    
    for _, s := range schedules {
        db.MustExec(
            "INSERT INTO user_schedules (user_id, user_name, event_id, working_from, working_to) VALUES (?, ?, ?, ?, ?)",
            s.UserID, s.UserName, s.EventID, s.WorkingFrom, s.WorkingTo,
        )
    }
    
    log.Println("Database initialized with sample data")
}
```

## Create your first agent

Let's create a scheduling agent that can query the database and analyze events.

```go
// agents/scheduler.go
package agents

import (
    "context"
    "fmt"
    "log"
    
    "github.com/jmoiron/sqlx"
    "github.com/ryanhill4L/agents-sdk"
)

type SchedulerAgent struct {
    agent *agents.Agent
    db    *sqlx.DB
}

func NewSchedulerAgent(db *sqlx.DB) *SchedulerAgent {
    agent := agents.NewAgent(
        agents.WithName("Event Scheduler"),
        agents.WithInstructions(`You are an event scheduling assistant. 
            You have access to a database of events and user schedules.
            You can help users find scheduling conflicts, available time slots, 
            and provide insights about event overlaps.
            Always be specific about dates, times, and user names when reporting conflicts.`),
        agents.WithModel("gpt-4-turbo-preview"),
    )
    
    // Add database query tool
    agent.AddTool(agents.Tool{
        Name:        "query_events",
        Description: "Query the events database",
        Handler:     createQueryHandler(db),
    })
    
    return &SchedulerAgent{
        agent: agent,
        db:    db,
    }
}

func createQueryHandler(db *sqlx.DB) agents.ToolHandler {
    return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
        query, ok := args["query"].(string)
        if !ok {
            return nil, fmt.Errorf("query parameter required")
        }
        
        var results []map[string]interface{}
        rows, err := db.Queryx(query)
        if err != nil {
            return nil, err
        }
        defer rows.Close()
        
        for rows.Next() {
            result := make(map[string]interface{})
            err := rows.MapScan(result)
            if err != nil {
                return nil, err
            }
            results = append(results, result)
        }
        
        return results, nil
    }
}
```

## Add specialized agents

Create agents for specific scheduling tasks.

```go
// agents/overlap_detector.go
package agents

import (
    "github.com/jmoiron/sqlx"
    "github.com/ryanhill4L/agents-sdk"
)

type OverlapDetector struct {
    agent *agents.Agent
    db    *sqlx.DB
}

func NewOverlapDetector(db *sqlx.DB) *OverlapDetector {
    agent := agents.NewAgent(
        agents.WithName("Overlap Detector"),
        agents.WithHandoffDescription("Specialist for detecting scheduling conflicts and overlaps"),
        agents.WithInstructions(`You specialize in finding scheduling conflicts.
            Analyze user schedules to find:
            1. Users attending multiple events at the same time
            2. Events scheduled in the same location at overlapping times
            3. Users whose events conflict with their working hours
            Provide clear, actionable recommendations to resolve conflicts.`),
    )
    
    // Add overlap detection tool
    agent.AddTool(agents.Tool{
        Name:        "detect_user_overlaps",
        Description: "Find users with conflicting event schedules",
        Handler:     createOverlapHandler(db),
    })
    
    agent.AddTool(agents.Tool{
        Name:        "detect_venue_conflicts",
        Description: "Find venues with overlapping events",
        Handler:     createVenueConflictHandler(db),
    })
    
    return &OverlapDetector{
        agent: agent,
        db:    db,
    }
}

func createOverlapHandler(db *sqlx.DB) agents.ToolHandler {
    return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
        query := `
        SELECT 
            us1.user_name,
            e1.name as event1_name,
            e1.date as event1_date,
            e2.name as event2_name,
            e2.date as event2_date
        FROM user_schedules us1
        JOIN user_schedules us2 ON us1.user_id = us2.user_id AND us1.event_id < us2.event_id
        JOIN events e1 ON us1.event_id = e1.id
        JOIN events e2 ON us2.event_id = e2.id
        WHERE datetime(e1.date) < datetime(e2.date, '+' || e2.duration_hours || ' hours')
          AND datetime(e2.date) < datetime(e1.date, '+' || e1.duration_hours || ' hours')`
        
        var results []map[string]interface{}
        rows, err := db.Queryx(query)
        if err != nil {
            return nil, err
        }
        defer rows.Close()
        
        for rows.Next() {
            result := make(map[string]interface{})
            err := rows.MapScan(result)
            if err != nil {
                return nil, err
            }
            results = append(results, result)
        }
        
        return map[string]interface{}{
            "conflicts": results,
            "total":     len(results),
        }, nil
    }
}

func createVenueConflictHandler(db *sqlx.DB) agents.ToolHandler {
    return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
        query := `
        SELECT 
            e1.location,
            e1.name as event1_name,
            e1.date as event1_date,
            e2.name as event2_name,
            e2.date as event2_date
        FROM events e1
        JOIN events e2 ON e1.location = e2.location AND e1.id < e2.id
        WHERE datetime(e1.date) < datetime(e2.date, '+' || e2.duration_hours || ' hours')
          AND datetime(e2.date) < datetime(e1.date, '+' || e1.duration_hours || ' hours')`
        
        var results []map[string]interface{}
        rows, err := db.Queryx(query)
        if err != nil {
            return nil, err
        }
        defer rows.Close()
        
        for rows.Next() {
            result := make(map[string]interface{})
            err := rows.MapScan(result)
            if err != nil {
                return nil, err
            }
            results = append(results, result)
        }
        
        return map[string]interface{}{
            "venue_conflicts": results,
            "total":           len(results),
        }, nil
    }
}
```

## Define your handoffs

Create a triage agent that routes between specialized agents.

```go
// agents/triage.go
package agents

import (
    "github.com/jmoiron/sqlx"
    "github.com/ryanhill4L/agents-sdk"
)

func NewTriageAgent(db *sqlx.DB) *agents.Agent {
    // Create specialized agents
    scheduler := NewSchedulerAgent(db)
    overlapDetector := NewOverlapDetector(db)
    
    // Create triage agent with handoffs
    triageAgent := agents.NewAgent(
        agents.WithName("Scheduling Triage"),
        agents.WithInstructions(`You are the main scheduling coordinator.
            Analyze user requests and route them to the appropriate specialist:
            - Use the Event Scheduler for general queries about events and schedules
            - Use the Overlap Detector for finding conflicts and scheduling issues
            Always provide a brief summary of what you're doing.`),
        agents.WithHandoffs([]agents.Agent{
            scheduler.agent,
            overlapDetector.agent,
        }),
    )
    
    return triageAgent
}
```

## Run the agent orchestration

Put it all together and run your scheduling agent system.

```go
// main.go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "strings"
    
    "event-scheduler/agents"
    "event-scheduler/db"
    
    "github.com/ryanhill4L/agents-sdk"
)

func main() {
    // Initialize database
    database, err := db.InitDB()
    if err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    defer database.Close()
    
    // Create the triage agent
    triageAgent := agents.NewTriageAgent(database)
    
    // Create the runner
    runner := agents.NewRunner(
        agents.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        agents.WithTracing(true), // Enable tracing for debugging
    )
    
    // Interactive loop
    fmt.Println("Event Scheduling Assistant Ready!")
    fmt.Println("Try asking:")
    fmt.Println("  - 'Show me all scheduled events'")
    fmt.Println("  - 'Find scheduling conflicts for users'")
    fmt.Println("  - 'Check for venue overlaps'")
    fmt.Println("  - 'What events is Alice attending?'")
    fmt.Println("\nType 'exit' to quit\n")
    
    scanner := bufio.NewScanner(os.Stdin)
    
    for {
        fmt.Print("> ")
        scanner.Scan()
        input := scanner.Text()
        
        if strings.ToLower(input) == "exit" {
            break
        }
        
        // Run the agent
        ctx := context.Background()
        result, err := runner.Run(ctx, triageAgent, input)
        if err != nil {
            log.Printf("Error: %v\n", err)
            continue
        }
        
        fmt.Printf("\n%s\n\n", result.Output)
    }
}
```

## Add guardrails

Implement guardrails to ensure data privacy and appropriate access.

```go
// agents/guardrails.go
package agents

import (
    "context"
    "strings"
    
    "github.com/ryanhill4L/agents-sdk"
)

type AccessCheckOutput struct {
    Allowed bool   `json:"allowed"`
    Reason  string `json:"reason"`
}

func PrivacyGuardrail() agents.GuardrailFunc {
    // Create a guardrail agent
    guardAgent := agents.NewAgent(
        agents.WithName("Privacy Guard"),
        agents.WithInstructions(`Check if the request involves sensitive information.
            Block requests that:
            - Try to access personal data without authorization
            - Attempt to modify schedules without permission
            - Request bulk export of all user data
            Return allowed=false for sensitive requests.`),
        agents.WithOutputType(AccessCheckOutput{}),
    )
    
    return func(ctx context.Context, agent agents.Agent, input string) (agents.GuardrailOutput, error) {
        runner := agents.NewRunner()
        result, err := runner.Run(ctx, guardAgent, input)
        if err != nil {
            return agents.GuardrailOutput{}, err
        }
        
        output := result.OutputAs(AccessCheckOutput{})
        
        return agents.GuardrailOutput{
            Info:              output,
            TripwireTriggered: !output.Allowed,
            Message:           output.Reason,
        }, nil
    }
}

// Update your triage agent to include guardrails
func NewSecureTriageAgent(db *sqlx.DB) *agents.Agent {
    triageAgent := NewTriageAgent(db)
    
    // Add input guardrail
    triageAgent.AddInputGuardrail(agents.InputGuardrail{
        Name:     "privacy_check",
        Function: PrivacyGuardrail(),
    })
    
    return triageAgent
}
```

## Complete example

Here's the full working example with all components integrated:

```go
// complete_example.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "event-scheduler/agents"
    "event-scheduler/db"
    
    "github.com/ryanhill4L/agents-sdk"
)

func runSchedulingExample() {
    // Initialize database
    database, err := db.InitDB()
    if err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    defer database.Close()
    
    // Create secure triage agent with guardrails
    triageAgent := agents.NewSecureTriageAgent(database)
    
    // Create runner with configuration
    runner := agents.NewRunner(
        agents.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        agents.WithTracing(true),
        agents.WithTimeout(30), // 30 second timeout
    )
    
    // Example queries
    queries := []string{
        "Show me all events happening tomorrow",
        "Find any scheduling conflicts for users",
        "Which users have overlapping events?",
        "Check if Conference Room A has any booking conflicts",
        "What's Alice's schedule for this week?",
    }
    
    ctx := context.Background()
    
    for _, query := range queries {
        fmt.Printf("\nQuery: %s\n", query)
        fmt.Println(strings.Repeat("-", 50))
        
        result, err := runner.Run(ctx, triageAgent, query)
        if err != nil {
            log.Printf("Error: %v\n", err)
            continue
        }
        
        fmt.Printf("Response: %s\n", result.Output)
        
        // Show handoff chain if any
        if len(result.HandoffChain) > 0 {
            fmt.Printf("Agents involved: ")
            for _, agent := range result.HandoffChain {
                fmt.Printf("%s -> ", agent)
            }
            fmt.Println("Done")
        }
    }
}

func main() {
    runSchedulingExample()
}
```

## View your traces

To debug and monitor your agent runs:

```go
// Enable detailed logging
runner := agents.NewRunner(
    agents.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    agents.WithTracing(true),
    agents.WithDebugMode(true),
    agents.WithLogLevel("debug"),
)

// Access trace data
result, _ := runner.Run(ctx, agent, input)
fmt.Printf("Trace ID: %s\n", result.TraceID)
fmt.Printf("Tokens used: %d\n", result.TokensUsed)
fmt.Printf("Latency: %v\n", result.Latency)
```

## Next steps

Now that you have a working event scheduling agent, you can:

1. **Extend the database schema** - Add more fields like attendee limits, recurring events, or event categories
2. **Create more specialized agents** - Build agents for specific tasks like finding optimal meeting times or suggesting alternative venues
3. **Implement advanced tools** - Add calendar integrations, email notifications, or export capabilities
4. **Add more guardrails** - Implement rate limiting, data validation, or authentication checks
5. **Build a REST API** - Expose your agent system through HTTP endpoints for integration with other services

### Learn more
- Explore advanced [Agent configurations](https://github.com/ryanhill4L/agents-sdk/docs/agents.md)
- Implement custom [Tools and Functions](https://github.com/ryanhill4L/agents-sdk/docs/tools.md)
- Set up comprehensive [Guardrails](https://github.com/ryanhill4L/agents-sdk/docs/guardrails.md)
- Configure different [AI Models](https://github.com/ryanhill4L/agents-sdk/docs/models.md)
- Build complex [Agent Orchestration](https://github.com/ryanhill4L/agents-sdk/docs/orchestration.md)