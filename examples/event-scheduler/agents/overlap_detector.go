package agents

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

type OverlapDetector struct {
	agent *agents.Agent
	db    *sqlx.DB
}

func NewOverlapDetector(db *sqlx.DB) *OverlapDetector {
	userOverlapTool, err := tools.NewFunctionTool("detect_user_overlaps", "Find users with conflicting event schedules", createUserOverlapHandler(db))
	if err != nil {
		panic(fmt.Sprintf("Failed to create user overlap tool: %v", err))
	}

	venueConflictTool, err := tools.NewFunctionTool("detect_venue_conflicts", "Find venues with overlapping events", createVenueConflictHandler(db))
	if err != nil {
		panic(fmt.Sprintf("Failed to create venue conflict tool: %v", err))
	}

	agent := agents.NewAgent("Overlap Detector",
		agents.WithInstructions(`You specialize in finding scheduling conflicts.
Analyze user schedules to find:
1. Users attending multiple events at the same time
2. Events scheduled in the same location at overlapping times
3. Users whose events conflict with their working hours
Provide clear, actionable recommendations to resolve conflicts.`),
		agents.WithModel("gpt-4"),
		agents.WithTools(userOverlapTool, venueConflictTool),
	)

	return &OverlapDetector{
		agent: agent,
		db:    db,
	}
}

func createUserOverlapHandler(db *sqlx.DB) func(ctx context.Context) (interface{}, error) {
	return func(ctx context.Context) (interface{}, error) {
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

func createVenueConflictHandler(db *sqlx.DB) func(ctx context.Context) (interface{}, error) {
	return func(ctx context.Context) (interface{}, error) {
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

func (o *OverlapDetector) GetAgent() *agents.Agent {
	return o.agent
}