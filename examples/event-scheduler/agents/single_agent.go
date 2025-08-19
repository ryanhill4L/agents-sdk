package agents

import (
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/jmoiron/sqlx"
	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

// Simple test functions exactly like the basic example
func add(a, b int) int {
	return a + b
}

func greet(name string) string {
	return fmt.Sprintf("Hello, %s! Nice to meet you.", name)
}

// Global database variable for the simple functions
var globalDB *sqlx.DB

func queryEvents(query string) ([]map[string]any, error) {
	fmt.Printf("🔍 DEBUG: Executing SQL query: %s\n", query)
	var results []map[string]any
	rows, err := globalDB.Queryx(query)
	if err != nil {
		fmt.Printf("🔍 DEBUG: Query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		result := make(map[string]any)
		err := rows.MapScan(result)
		if err != nil {
			fmt.Printf("🔍 DEBUG: Row scan error: %v\n", err)
			return nil, err
		}
		results = append(results, result)
	}

	fmt.Printf("🔍 DEBUG: Query returned %d results\n", len(results))
	return results, nil
}

func detectUserOverlaps() (map[string]any, error) {
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

	var results []map[string]any
	rows, err := globalDB.Queryx(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		result := make(map[string]any)
		err := rows.MapScan(result)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return map[string]any{
		"conflicts": results,
		"total":     len(results),
	}, nil
}

func detectVenueConflicts() (map[string]any, error) {
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

	var results []map[string]any
	rows, err := globalDB.Queryx(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		result := make(map[string]any)
		err := rows.MapScan(result)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return map[string]any{
		"venue_conflicts": results,
		"total":           len(results),
	}, nil
}

// NewEventSchedulerAgent creates a single agent with all necessary tools
func NewEventSchedulerAgent(db *sqlx.DB) *agents.Agent {
	fmt.Println("🔍 DEBUG: Creating EventSchedulerAgent with tools...")

	// Set the global DB for the simple functions
	globalDB = db

	// Create tools exactly like the basic example first
	addTool, err := tools.NewFunctionTool("add", "Adds two numbers together", add)
	if err != nil {
		panic(fmt.Sprintf("Failed to create add tool: %v", err))
	}

	greetTool, err := tools.NewFunctionTool("greet", "Greets a person by name", greet)
	if err != nil {
		panic(fmt.Sprintf("Failed to create greet tool: %v", err))
	}

	// Create database tools using the same pattern
	queryTool, err := tools.NewFunctionTool("query_events", "Query the events database with SQL", queryEvents)
	if err != nil {
		panic(fmt.Sprintf("Failed to create query tool: %v", err))
	}

	userOverlapTool, err := tools.NewFunctionTool("detect_user_overlaps", "Find users with conflicting event schedules", detectUserOverlaps)
	if err != nil {
		panic(fmt.Sprintf("Failed to create user overlap tool: %v", err))
	}

	venueConflictTool, err := tools.NewFunctionTool("detect_venue_conflicts", "Find venues with overlapping events", detectVenueConflicts)
	if err != nil {
		panic(fmt.Sprintf("Failed to create venue conflict tool: %v", err))
	}

	// Create the agent with all tools
	fmt.Println("🔍 DEBUG: Creating agent with tools...")
	agent := agents.NewAgent("Event Scheduler Assistant",
		agents.WithInstructions(eventSchedulerInstructions),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithInstructions(eventSchedulerInstructions),
		agents.WithTools(addTool, greetTool, queryTool, userOverlapTool, venueConflictTool),
		agents.WithTemperature(0.7),
	)

	fmt.Printf("🔍 DEBUG: Agent created successfully: %+v\n", agent)
	return agent
}

const eventSchedulerInstructions = `System: Role and Objective:
- Serve as an event scheduling assistant with access to a comprehensive event and user schedule database, providing precise answers by leveraging designated tools.

Instructions:
- Always utilize the available tools to fetch and verify information before responding.
- Never answer user queries about events or schedules without querying the database or running overlap detection tools as appropriate.
- Use only the tools listed in "Available Tools." For routine read-only tasks, call automatically; for destructive or irreversible actions (if applicable in future toolset), require explicit confirmation.

Process Checklist:
Begin with a concise checklist (3-7 bullets) of what you will do:
1. Analyze and categorize the user request.
2. State the purpose and minimal inputs for any significant tool call before invoking it.
3. Invoke the appropriate tool(s) as determined by request type.
4. Validate and summarize the tool output in 1-2 lines; if validation fails, clarify or self-correct.
5. Format and present a concise, user-friendly response with relevant details (dates, times, user names, venues).
6. Mark completion only after correct tool usage and required detail formatting.

Tool Usage Guidelines:
- For event or schedule information: Use 'query_events' with well-structured SQL queries to retrieve relevant data.
- For checking user scheduling conflicts: Use 'detect_user_overlaps' to determine if any users are booked for overlapping events.
- For checking venue booking conflicts: Use 'detect_venue_conflicts' to identify venues with overlapping event reservations.

Available Tools:
- 'get_event_count': Returns the total number of events in the database. For validation or basic metrics. No parameters required.
- 'query_events': Perform specific SQL queries on the event database for detailed information or filtering results. Requires a SQL query string as input.
- 'detect_user_overlaps': Detects users with multiple event commitments at the same time. No parameters required.
- 'detect_venue_conflicts': Detects overlapping bookings for venues. No parameters required.

Verbosity and Output:
- Responses must be clear, concise, and grounded strictly in tool-generated data.
- After each tool invocation, validate the result and proceed or self-correct as needed.
- Mark the task as complete only after ensuring all relevant tool calls have been made and the answer includes all required details.
`
