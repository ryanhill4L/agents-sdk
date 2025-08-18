package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ryanhill4L/agents-sdk/pkg/memory"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
	"golang.org/x/sync/errgroup"
)

// Runner executes agent workflows
type Runner struct {
	provider providers.Provider
	tracer   tracing.Tracer
	session  memory.Session

	maxTurns      int
	timeout       time.Duration
	parallelTools bool
}

// RunResult contains the execution results
type RunResult struct {
	FinalOutput interface{}    `json:"final_output"`
	Messages    []Message      `json:"messages"`
	Agent       *Agent         `json:"-"`
	Traces      []tracing.Span `json:"traces,omitempty"`
	Metrics     RunMetrics     `json:"metrics"`
}

// RunMetrics contains execution metrics
type RunMetrics struct {
	TotalTurns  int           `json:"total_turns"`
	TotalTokens int           `json:"total_tokens"`
	Duration    time.Duration `json:"duration"`
	ToolCalls   int           `json:"tool_calls"`
	Handoffs    int           `json:"handoffs"`
}

// NewRunner creates a new runner with options
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{
		maxTurns:      10,
		timeout:       5 * time.Minute,
		parallelTools: true,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set defaults if not provided
	if r.provider == nil {
		r.provider = providers.NewDefaultOpenAIProvider()
	}

	if r.tracer == nil {
		r.tracer = tracing.NewNoOpTracer()
	}

	return r
}

// Run executes the agent workflow asynchronously
func (r *Runner) Run(ctx context.Context, agent *Agent, input string) (*RunResult, error) {
	// Create run context
	runCtx := &RunContext{
		Context:   ctx,
		SessionID: uuid.New().String(),
		TraceID:   uuid.New().String(),
		MaxTurns:  r.maxTurns,
		Variables: make(map[string]interface{}),
	}

	// Start tracing
	ctx, rootSpan := r.tracer.StartSpan(ctx, "agent.run")
	defer r.tracer.EndSpan(rootSpan)

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Initialize messages
	messages := []Message{
		{
			Role:      "user",
			Content:   input,
			Timestamp: time.Now(),
		},
	}

	// Load session history if available
	if r.session != nil {
		history, err := r.session.GetItems(ctx, 100)
		if err != nil {
			return nil, fmt.Errorf("failed to load session: %w", err)
		}
		messages = append(messagesFromMemory(history), messages...)
	}

	// Execute agent loop
	result, err := r.executeLoop(runCtx, agent, messages)
	if err != nil {
		return nil, err
	}

	// Save to session
	if r.session != nil {
		if err := r.session.AddItems(ctx, messagesToMemory(result.Messages)); err != nil {
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
	}

	return result, nil
}

// executeLoop runs the main agent execution loop
func (r *Runner) executeLoop(ctx *RunContext, agent *Agent, messages []Message) (*RunResult, error) {
	startTime := time.Now()
	metrics := RunMetrics{}
	currentAgent := agent

	for turn := 0; turn < ctx.MaxTurns; turn++ {
		ctx.CurrentTurn = turn

		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled: %w", err)
		}

		// Validate input with guardrails
		if err := r.validateGuardrails(currentAgent, messages); err != nil {
			return nil, fmt.Errorf("guardrail validation failed: %w", err)
		}

		// Get LLM completion
		toolDefs := convertToolsToProviders(currentAgent.Tools)
		completion, err := r.provider.Complete(ctx.Context, currentAgent, messagesToProviders(messages), toolDefs)
		if err != nil {
			return nil, fmt.Errorf("completion failed: %w", err)
		}

		metrics.TotalTokens += completion.Usage.TotalTokens
		messages = append(messages, messageFromProviders(completion.Message))

		// Check for final output
		if currentAgent.OutputType != nil && completion.StructuredOutput != nil {
			metrics.Duration = time.Since(startTime)
			metrics.TotalTurns = turn + 1

			return &RunResult{
				FinalOutput: completion.StructuredOutput,
				Messages:    messages,
				Agent:       currentAgent,
				Metrics:     metrics,
			}, nil
		}

		// Handle handoffs
		if completion.Handoff != nil {
			metrics.Handoffs++

			newAgent, ok := currentAgent.GetHandoff(completion.Handoff.TargetAgent)
			if !ok {
				return nil, fmt.Errorf("handoff agent not found: %s", completion.Handoff.TargetAgent)
			}

			currentAgent = newAgent
			continue
		}

		// Handle tool calls
		if len(completion.ToolCalls) > 0 {
			metrics.ToolCalls += len(completion.ToolCalls)

			toolResponses, err := r.executeTools(ctx, currentAgent, toolCallsFromProviders(completion.ToolCalls))
			if err != nil {
				return nil, fmt.Errorf("tool execution failed: %w", err)
			}

			// Add tool responses as messages
			for _, resp := range toolResponses {
				messages = append(messages, Message{
					Role:      "tool",
					Content:   fmt.Sprintf("%v", resp.Content),
					Timestamp: time.Now(),
					Metadata: map[string]interface{}{
						"tool_call_id": resp.ToolCallID,
					},
				})
			}

			continue
		}

		// If no tools, handoffs, or structured output, we have final output
		if currentAgent.OutputType == nil {
			metrics.Duration = time.Since(startTime)
			metrics.TotalTurns = turn + 1

			return &RunResult{
				FinalOutput: completion.Message.Content,
				Messages:    messages,
				Agent:       currentAgent,
				Metrics:     metrics,
			}, nil
		}
	}

	return nil, ErrMaxTurnsExceeded
}

// executeTools runs tool calls in parallel or sequence
func (r *Runner) executeTools(ctx *RunContext, agent *Agent, toolCalls []ToolCall) ([]ToolResponse, error) {
	responses := make([]ToolResponse, len(toolCalls))

	if r.parallelTools && len(toolCalls) > 1 {
		// Execute tools in parallel
		g, gCtx := errgroup.WithContext(ctx.Context)

		for i, call := range toolCalls {
			i, call := i, call // capture loop variables

			g.Go(func() error {
				tool := r.findTool(agent, call.Name)
				if tool == nil {
					responses[i] = ToolResponse{
						ToolCallID: call.ID,
						Error:      fmt.Errorf("tool not found: %s", call.Name),
					}
					return nil
				}

				result, err := tool.Execute(gCtx, call.Arguments)
				responses[i] = ToolResponse{
					ToolCallID: call.ID,
					Content:    result,
					Error:      err,
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}
	} else {
		// Execute tools sequentially
		for i, call := range toolCalls {
			tool := r.findTool(agent, call.Name)
			if tool == nil {
				responses[i] = ToolResponse{
					ToolCallID: call.ID,
					Error:      fmt.Errorf("tool not found: %s", call.Name),
				}
				continue
			}

			result, err := tool.Execute(ctx.Context, call.Arguments)
			responses[i] = ToolResponse{
				ToolCallID: call.ID,
				Content:    result,
				Error:      err,
			}
		}
	}

	return responses, nil
}

// findTool locates a tool by name
func (r *Runner) findTool(agent *Agent, name string) tools.Tool {
	for _, tool := range agent.Tools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

// validateGuardrails runs all guardrail checks
func (r *Runner) validateGuardrails(agent *Agent, messages []Message) error {
	if len(messages) == 0 || len(agent.Guardrails) == 0 {
		return nil
	}

	lastMessage := messages[len(messages)-1]

	for _, guardrail := range agent.Guardrails {
		if err := guardrail.Validate(lastMessage.Content); err != nil {
			return fmt.Errorf("guardrail %T failed: %w", guardrail, err)
		}
	}

	return nil
}

// RunSync provides a synchronous interface
func RunSync(ctx context.Context, agent *Agent, input string, opts ...RunnerOption) (*RunResult, error) {
	runner := NewRunner(opts...)
	return runner.Run(ctx, agent, input)
}

// Conversion functions to handle type differences between packages

// messagesToProviders converts agents.Message to providers.Message
func messagesToProviders(msgs []Message) []providers.Message {
	result := make([]providers.Message, len(msgs))
	for i, msg := range msgs {
		result[i] = providers.Message{
			Role:      msg.Role,
			Content:   msg.Content,
			ToolCalls: toolCallsToProviders(msg.ToolCalls),
			Metadata:  msg.Metadata,
			Timestamp: msg.Timestamp,
		}
	}
	return result
}

// toolCallsToProviders converts agents.ToolCall to providers.ToolCall
func toolCallsToProviders(calls []ToolCall) []providers.ToolCall {
	result := make([]providers.ToolCall, len(calls))
	for i, call := range calls {
		result[i] = providers.ToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
		}
	}
	return result
}

// toolCallsFromProviders converts providers.ToolCall to agents.ToolCall
func toolCallsFromProviders(calls []providers.ToolCall) []ToolCall {
	result := make([]ToolCall, len(calls))
	for i, call := range calls {
		result[i] = ToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
		}
	}
	return result
}

// messageFromProviders converts providers.Message to agents.Message
func messageFromProviders(msg providers.Message) Message {
	return Message{
		Role:      msg.Role,
		Content:   msg.Content,
		ToolCalls: toolCallsFromProviders(msg.ToolCalls),
		Metadata:  msg.Metadata,
		Timestamp: msg.Timestamp,
	}
}

// messagesToMemory converts agents.Message to memory.Message
func messagesToMemory(msgs []Message) []memory.Message {
	result := make([]memory.Message, len(msgs))
	for i, msg := range msgs {
		result[i] = memory.Message{
			Role:      msg.Role,
			Content:   msg.Content,
			Metadata:  msg.Metadata,
			Timestamp: msg.Timestamp,
		}
	}
	return result
}

// messagesFromMemory converts memory.Message to agents.Message
func messagesFromMemory(msgs []memory.Message) []Message {
	result := make([]Message, len(msgs))
	for i, msg := range msgs {
		result[i] = Message{
			Role:      msg.Role,
			Content:   msg.Content,
			Metadata:  msg.Metadata,
			Timestamp: msg.Timestamp,
		}
	}
	return result
}

// convertToolsToProviders converts agents tools to provider tool definitions
func convertToolsToProviders(tools []tools.Tool) []providers.ToolDefinition {
	result := make([]providers.ToolDefinition, len(tools))
	for i, tool := range tools {
		schema := tool.Schema()
		result[i] = providers.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Schema: providers.ParameterSchema{
				Type:       schema.Type,
				Properties: convertProperties(schema.Properties),
				Required:   schema.Required,
			},
		}
	}
	return result
}

// convertProperties converts tools.PropertySchema to providers.PropertySchema
func convertProperties(props map[string]tools.PropertySchema) map[string]providers.PropertySchema {
	result := make(map[string]providers.PropertySchema)
	for name, prop := range props {
		result[name] = providers.PropertySchema{
			Type:        prop.Type,
			Description: prop.Description,
		}
	}
	return result
}

// RunAsync provides a channel-based async interface
func RunAsync(ctx context.Context, agent *Agent, input string, opts ...RunnerOption) <-chan *RunResult {
	resultChan := make(chan *RunResult, 1)

	go func() {
		defer close(resultChan)

		runner := NewRunner(opts...)
		result, err := runner.Run(ctx, agent, input)

		if err != nil {
			// Include error in result
			result = &RunResult{
				FinalOutput: fmt.Sprintf("Error: %v", err),
			}
		}

		resultChan <- result
	}()

	return resultChan
}
