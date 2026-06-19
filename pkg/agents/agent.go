package agents

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/ryanhill4L/agents-sdk/pkg/guardrails"
	"github.com/ryanhill4L/agents-sdk/pkg/skills"
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
	Skills     []skills.Skill

	// Provider names the LLM backend (openai, anthropic, gemini, ollama).
	// It is advisory: a Runner with an explicit provider always wins.
	Provider string

	// Dir records the directory an agent was loaded from, when applicable.
	Dir string

	// MCPServers names the MCP servers declared for this agent (for reporting).
	MCPServers []string

	// Configuration
	OutputType  OutputSchema
	Temperature float32
	MaxTokens   int
	TopP        float32

	// Runtime
	handoffMap   map[string]*Agent
	handoffModes map[string]HandoffMode
	closers      []io.Closer
}

// AddCloser registers a resource (e.g. an MCP connection) to be released when
// the agent is closed.
func (a *Agent) AddCloser(c io.Closer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closers = append(a.closers, c)
}

// Close releases the agent's resources and those of its subagents. It is safe
// to call on agents with nothing to close, and on graphs that share or cycle
// through subagents (each agent is closed at most once).
func (a *Agent) Close() error {
	return a.closeTree(make(map[*Agent]bool))
}

func (a *Agent) closeTree(visited map[*Agent]bool) error {
	if visited[a] {
		return nil
	}
	visited[a] = true

	a.mu.Lock()
	closers := a.closers
	a.closers = nil
	handoffs := a.Handoffs
	a.mu.Unlock()

	var firstErr error
	for _, c := range closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, h := range handoffs {
		if err := h.closeTree(visited); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// NewAgent creates a new agent with the given name and options
func NewAgent(name string, opts ...AgentOption) *Agent {
	agent := &Agent{
		Name:         name,
		Model:        "gpt-4",
		Temperature:  0.7,
		MaxTokens:    2000,
		TopP:         1.0,
		handoffMap:   make(map[string]*Agent),
		handoffModes: make(map[string]HandoffMode),
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
	if len(a.Skills) == 0 {
		return a.Instructions
	}
	catalog := skills.Catalog(a.Skills)
	if a.Instructions == "" {
		return catalog
	}
	return strings.TrimRight(a.Instructions, "\n") + "\n\n" + catalog
}

// EffectiveTools returns the agent's tools plus the builtins the Runner exposes
// to the model: load_skill (when the agent has skills) and one handoff tool per
// subagent. The Runner intercepts handoff tool calls.
func (a *Agent) EffectiveTools() []tools.Tool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	present := make(map[string]bool, len(a.Tools))
	for _, t := range a.Tools {
		present[t.Name()] = true
	}

	effective := make([]tools.Tool, len(a.Tools))
	copy(effective, a.Tools)

	if len(a.Skills) > 0 && !present[skills.LoadSkillToolName] {
		effective = append(effective, skills.NewLoadSkillTool(a.Skills))
	}

	for _, h := range a.Handoffs {
		name := handoffToolName(h.Name)
		if present[name] {
			continue
		}
		effective = append(effective, newHandoffTool(h, a.handoffModes[h.Name]))
	}

	return effective
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
		Provider:     a.Provider,
		Dir:          a.Dir,
		OutputType:   a.OutputType,
		Temperature:  a.Temperature,
		MaxTokens:    a.MaxTokens,
		TopP:         a.TopP,
		handoffMap:   make(map[string]*Agent),
		handoffModes: make(map[string]HandoffMode),
	}

	clone.Skills = make([]skills.Skill, len(a.Skills))
	copy(clone.Skills, a.Skills)

	for k, v := range a.handoffModes {
		clone.handoffModes[k] = v
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
