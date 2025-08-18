package agents

import "errors"

var (
	// Agent errors
	ErrInvalidAgentName = errors.New("agent name cannot be empty")
	ErrInvalidModel     = errors.New("model cannot be empty")
	ErrCircularHandoff  = errors.New("circular handoff detected")

	// Runner errors
	ErrMaxTurnsExceeded = errors.New("maximum turns exceeded")
	ErrTimeout          = errors.New("execution timeout")
	ErrNoProvider       = errors.New("no LLM provider configured")

	// Tool errors
	ErrToolNotFound  = errors.New("tool not found")
	ErrToolExecution = errors.New("tool execution failed")

	// Guardrail errors
	ErrGuardrailViolation = errors.New("guardrail violation")
)
