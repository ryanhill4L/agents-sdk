package agents

import (
	"fmt"
	"sync"

	"github.com/ryanhill4L/agents-sdk/pkg/guardrails"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

// Agent represents an AI agent with specific capabilities
type Agent struct {
	mu sync.RWMutex

	// Core properties
	Name         string
	Instructions string
	Model        string

	// Capabilities
	Tools      []tools.Tool
	Handoffs   []*Agent
	Guardrails []guardrails.Guardrail

	// Configuration
	OutputType  OutputSchema
	Temperature float32
	MaxTokens   int
	TopP        float32

	// Runtime
	handoffMap map[string]*Agent
}

// NewAgent creates a new agent with the given name and options
func NewAgent(name string, opts ...AgentOption) *Agent {
	agent := &Agent{
		Name:        name,
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   2000,
		TopP:        1.0,
		handoffMap:  make(map[string]*Agent),
	}

	for _, opt := range opts {
		opt(agent)
	}

	// Build handoff map for quick lookup
	for _, handoff := range agent.Handoffs {
		agent.handoffMap[handoff.Name] = handoff
	}

	return agent
}

// Validate checks if the agent configuration is valid
func (a *Agent) Validate() error {
	if a.Name == "" {
		return ErrInvalidAgentName
	}

	if a.Model == "" {
		return ErrInvalidModel
	}

	// Validate tools
	for _, tool := range a.Tools {
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("invalid tool %s: %w", tool.Name(), err)
		}
	}

	// Validate circular handoffs
	if err := a.validateHandoffs(make(map[string]bool)); err != nil {
		return err
	}

	return nil
}

// validateHandoffs checks for circular dependencies
func (a *Agent) validateHandoffs(visited map[string]bool) error {
	if visited[a.Name] {
		return fmt.Errorf("circular handoff detected: %s", a.Name)
	}

	visited[a.Name] = true
	defer delete(visited, a.Name)

	for _, handoff := range a.Handoffs {
		if err := handoff.validateHandoffs(visited); err != nil {
			return err
		}
	}

	return nil
}

// GetHandoff returns the handoff agent by name
func (a *Agent) GetHandoff(name string) (*Agent, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	agent, ok := a.handoffMap[name]
	return agent, ok
}

// Getter methods to implement providers.Agent interface
func (a *Agent) GetName() string {
	return a.Name
}

func (a *Agent) GetInstructions() string {
	return a.Instructions
}

func (a *Agent) GetModel() string {
	return a.Model
}

func (a *Agent) GetTemperature() float32 {
	return a.Temperature
}

func (a *Agent) GetMaxTokens() int {
	return a.MaxTokens
}

func (a *Agent) GetTopP() float32 {
	return a.TopP
}

// Clone creates a deep copy of the agent
func (a *Agent) Clone() *Agent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	clone := &Agent{
		Name:         a.Name,
		Instructions: a.Instructions,
		Model:        a.Model,
		OutputType:   a.OutputType,
		Temperature:  a.Temperature,
		MaxTokens:    a.MaxTokens,
		TopP:         a.TopP,
		handoffMap:   make(map[string]*Agent),
	}

	// Deep copy tools
	clone.Tools = make([]tools.Tool, len(a.Tools))
	copy(clone.Tools, a.Tools)

	// Deep copy guardrails
	clone.Guardrails = make([]guardrails.Guardrail, len(a.Guardrails))
	copy(clone.Guardrails, a.Guardrails)

	// Note: Handoffs are shared references (agents are immutable once created)
	clone.Handoffs = make([]*Agent, len(a.Handoffs))
	copy(clone.Handoffs, a.Handoffs)

	for k, v := range a.handoffMap {
		clone.handoffMap[k] = v
	}

	return clone
}
