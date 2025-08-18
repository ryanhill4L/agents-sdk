package agents

import (
	"time"

	"github.com/ryanhill4L/agents-sdk/pkg/guardrails"
	"github.com/ryanhill4L/agents-sdk/pkg/memory"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// AgentOption configures an Agent
type AgentOption func(*Agent)

// WithInstructions sets the agent's instructions
func WithInstructions(instructions string) AgentOption {
	return func(a *Agent) {
		a.Instructions = instructions
	}
}

// WithModel sets the LLM model
func WithModel(model string) AgentOption {
	return func(a *Agent) {
		a.Model = model
	}
}

// WithTools adds tools to the agent
func WithTools(tools ...tools.Tool) AgentOption {
	return func(a *Agent) {
		a.Tools = append(a.Tools, tools...)
	}
}

// WithHandoffs adds handoff agents
func WithHandoffs(agents ...*Agent) AgentOption {
	return func(a *Agent) {
		a.Handoffs = append(a.Handoffs, agents...)

		// Rebuild handoff map
		a.handoffMap = make(map[string]*Agent)
		for _, handoff := range a.Handoffs {
			a.handoffMap[handoff.Name] = handoff
		}
	}
}

// WithGuardrails adds guardrails
func WithGuardrails(guardrails ...guardrails.Guardrail) AgentOption {
	return func(a *Agent) {
		a.Guardrails = append(a.Guardrails, guardrails...)
	}
}

// WithOutputType sets structured output schema
func WithOutputType(schema OutputSchema) AgentOption {
	return func(a *Agent) {
		a.OutputType = schema
	}
}

// WithTemperature sets the temperature parameter
func WithTemperature(temp float32) AgentOption {
	return func(a *Agent) {
		a.Temperature = temp
	}
}

// WithMaxTokens sets the max tokens parameter
func WithMaxTokens(tokens int) AgentOption {
	return func(a *Agent) {
		a.MaxTokens = tokens
	}
}

// RunnerOption configures a Runner
type RunnerOption func(*Runner)

// WithProvider sets the LLM provider
func WithProvider(provider providers.Provider) RunnerOption {
	return func(r *Runner) {
		r.provider = provider
	}
}

// WithTracer sets the tracer
func WithTracer(tracer tracing.Tracer) RunnerOption {
	return func(r *Runner) {
		r.tracer = tracer
	}
}

// WithSession sets the session memory
func WithSession(session memory.Session) RunnerOption {
	return func(r *Runner) {
		r.session = session
	}
}

// WithMaxTurns sets the maximum turns
func WithMaxTurns(turns int) RunnerOption {
	return func(r *Runner) {
		r.maxTurns = turns
	}
}

// WithTimeout sets the execution timeout
func WithTimeout(timeout time.Duration) RunnerOption {
	return func(r *Runner) {
		r.timeout = timeout
	}
}

// WithParallelTools enables/disables parallel tool execution
func WithParallelTools(parallel bool) RunnerOption {
	return func(r *Runner) {
		r.parallelTools = parallel
	}
}
