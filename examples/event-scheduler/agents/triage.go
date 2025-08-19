package agents

import (
	"github.com/jmoiron/sqlx"
	"github.com/ryanhill4L/agents-sdk/pkg/agents"
)

func NewTriageAgent(db *sqlx.DB) *agents.Agent {
	scheduler := NewSchedulerAgent(db)
	overlapDetector := NewOverlapDetector(db)

	triageAgent := agents.NewAgent("Scheduling Triage",
		agents.WithInstructions(`You are the main scheduling coordinator.
Analyze user requests and route them to the appropriate specialist:
- Use the Event Scheduler for general queries about events and schedules
- Use the Overlap Detector for finding conflicts and scheduling issues
Always provide a brief summary of what you're doing.`),
		agents.WithModel("gpt-4"),
		agents.WithHandoffs(scheduler.GetAgent(), overlapDetector.GetAgent()),
	)

	return triageAgent
}