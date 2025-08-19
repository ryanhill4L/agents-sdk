package agents

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

type SchedulerAgent struct {
	agent *agents.Agent
	db    *sqlx.DB
}

func NewSchedulerAgent(db *sqlx.DB) *SchedulerAgent {
	queryTool, err := tools.NewFunctionTool("query_events", "Query the events database with SQL", createQueryHandler(db))
	if err != nil {
		panic(fmt.Sprintf("Failed to create query tool: %v", err))
	}

	agent := agents.NewAgent("Event Scheduler",
		agents.WithInstructions(`You are an event scheduling assistant. 
You have access to a database of events and user schedules.
You can help users find scheduling conflicts, available time slots, 
and provide insights about event overlaps.
Always be specific about dates, times, and user names when reporting conflicts.`),
		agents.WithModel("gpt-4"),
		agents.WithTools(queryTool),
	)

	return &SchedulerAgent{
		agent: agent,
		db:    db,
	}
}

func createQueryHandler(db *sqlx.DB) func(ctx context.Context, query string) (interface{}, error) {
	return func(ctx context.Context, query string) (interface{}, error) {
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

func (s *SchedulerAgent) GetAgent() *agents.Agent {
	return s.agent
}