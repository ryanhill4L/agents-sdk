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
		agents.WithInstructions(`You are an event scheduling assistant. You have tools to help with math, greetings, and database queries.

For event-related questions, use these tools:
- query_events: Query the database with SQL. For finding user's events, use: "SELECT e.name, e.date, e.location, e.duration_hours FROM events e JOIN user_schedules us ON e.id = us.event_id WHERE us.user_name = 'Username'"
- detect_user_overlaps: Find users with scheduling conflicts  
- detect_venue_conflicts: Find venues with overlapping bookings

Database schema:
- events table: id, name, date, location, duration_hours
- user_schedules table: id, user_id, user_name, event_id, working_from, working_to

Always use the appropriate tool to answer questions with proper SQL queries.`),
		agents.WithModel(string(anthropic.ModelClaude4Sonnet20250514)),
		agents.WithTools(addTool, greetTool, queryTool, userOverlapTool, venueConflictTool),
		agents.WithTemperature(0.7),
	)

	fmt.Printf("🔍 DEBUG: Agent created successfully: %+v\n", agent)
	return agent
}
