package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	logger   *slog.Logger
	session  memory.Session

	maxTurns      int
	timeout       time.Duration
	parallelTools bool
}

// RunResult contains the execution results
type RunResult struct {
	FinalOutput interface{}          `json:"final_output"`
	Messages    []Message            `json:"messages"`
	Agent       *Agent               `json:"-"`
	Traces      []tracing.SpanRecord `json:"traces,omitempty"`
	Metrics     RunMetrics           `json:"metrics"`
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

	if r.logger == nil {
		r.logger = tracing.DiscardLogger()
	}

	return r
}

// Run executes the agent workflow asynchronously
func (r *Runner) Run(ctx context.Context, agent *Agent, input string) (result *RunResult, err error) {
	sessionID := uuid.New().String()
	traceID := uuid.New().String()

	// Tee the configured tracer with a recorder so the trace tree is also
	// available programmatically via RunResult.Traces.
	recorder := tracing.NewRecorder()
	tracer := tracing.NewTee(r.tracer, recorder)

	// Apply timeout, then open the root span.
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	ctx, rootSpan := tracing.Start(ctx, tracer, "agent.run",
		tracing.A("agent", agent.Name),
		tracing.A("session_id", sessionID),
		tracing.A("input_chars", len(input)),
	)
	// End the root span, then attach the full recorded trace tree to the result.
	defer func() {
		rootSpan.End()
		if result != nil {
			result.Traces = recorder.Records()
		}
	}()

	logger := r.logger.With("agent", agent.Name, "session_id", sessionID, "trace_id", traceID)
	logger.Info("run started", "input_chars", len(input), "max_turns", r.maxTurns)

	runCtx := &RunContext{
		Context:   ctx,
		SessionID: sessionID,
		TraceID:   traceID,
		MaxTurns:  r.maxTurns,
		Variables: make(map[string]interface{}),
	}

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
			rootSpan.SetError(err)
			logger.Error("session load failed", "error", err)
			return nil, fmt.Errorf("failed to load session: %w", err)
		}
		logger.Debug("session history loaded", "messages", len(history))
		messages = append(messagesFromMemory(history), messages...)
	}

	// Execute agent loop
	result, err = r.executeLoop(runCtx, tracer, logger, agent, messages)
	if err != nil {
		rootSpan.SetError(err)
		logger.Error("run failed", "error", err)
		return nil, err
	}

	rootSpan.SetAttributes(
		tracing.A("turns", result.Metrics.TotalTurns),
		tracing.A("tokens", result.Metrics.TotalTokens),
		tracing.A("tool_calls", result.Metrics.ToolCalls),
		tracing.A("handoffs", result.Metrics.Handoffs),
	)
	logger.Info("run completed",
		"turns", result.Metrics.TotalTurns,
		"tokens", result.Metrics.TotalTokens,
		"tool_calls", result.Metrics.ToolCalls,
		"handoffs", result.Metrics.Handoffs,
		"duration", result.Metrics.Duration,
	)

	// Save to session
	if r.session != nil {
		if err := r.session.AddItems(ctx, messagesToMemory(result.Messages)); err != nil {
			rootSpan.SetError(err)
			logger.Error("session save failed", "error", err)
			return nil, fmt.Errorf("failed to save session: %w", err)
		}
	}

	return result, nil
}

// executeLoop runs the main agent execution loop. Each turn is wrapped in a
// span (via a closure so the span is always ended), and emits structured logs.
func (r *Runner) executeLoop(ctx *RunContext, tracer tracing.Tracer, logger *slog.Logger, agent *Agent, messages []Message) (*RunResult, error) {
	startTime := time.Now()
	metrics := RunMetrics{}
	currentAgent := agent

	for turn := 0; turn < ctx.MaxTurns; turn++ {
		ctx.CurrentTurn = turn

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled: %w", err)
		}

		var (
			result    *RunResult
			nextAgent *Agent
		)

		turnErr := func() error {
			turnCtx, turnSpan := tracing.Start(ctx.Context, tracer, "turn",
				tracing.A("n", turn), tracing.A("agent", currentAgent.Name))
			defer turnSpan.End()

			// Guardrails
			if err := r.validateGuardrails(turnCtx, tracer, logger, currentAgent, messages); err != nil {
				turnSpan.SetError(err)
				return fmt.Errorf("guardrail validation failed: %w", err)
			}

			// LLM completion
			toolDefs := convertToolsToProviders(currentAgent.EffectiveTools())
			_, llmSpan := tracing.Start(turnCtx, tracer, "llm.complete",
				tracing.A("model", currentAgent.Model),
				tracing.A("messages", len(messages)),
				tracing.A("tools", len(toolDefs)))
			logger.Debug("llm request", "turn", turn, "agent", currentAgent.Name,
				"model", currentAgent.Model, "messages", len(messages), "tools", len(toolDefs))

			completion, err := r.provider.Complete(turnCtx, currentAgent, messagesToProviders(messages), toolDefs)
			if err != nil {
				llmSpan.SetError(err)
				llmSpan.End()
				turnSpan.SetError(err)
				logger.Error("llm request failed", "turn", turn, "error", err)
				return fmt.Errorf("completion failed: %w", err)
			}
			llmSpan.SetAttributes(
				tracing.A("tokens", completion.Usage.TotalTokens),
				tracing.A("tool_calls", len(completion.ToolCalls)))
			llmSpan.End()
			logger.Debug("llm response", "turn", turn,
				"tokens", completion.Usage.TotalTokens,
				"tool_calls", len(completion.ToolCalls),
				"content_chars", len(completion.Message.Content))

			metrics.TotalTokens += completion.Usage.TotalTokens

			// turnStart marks the history boundary before this assistant turn,
			// used to fork a clean context for delegated subagents.
			turnStart := len(messages)
			messages = append(messages, messageFromProviders(completion.Message))

			// Structured output -> done.
			if currentAgent.OutputType != nil && completion.StructuredOutput != nil {
				metrics.Duration = time.Since(startTime)
				metrics.TotalTurns = turn + 1
				result = &RunResult{FinalOutput: completion.StructuredOutput, Messages: messages, Agent: currentAgent, Metrics: metrics}
				return nil
			}

			// Legacy provider-driven handoff (transfer of control).
			if completion.Handoff != nil {
				newAgent, ok := currentAgent.GetHandoff(completion.Handoff.TargetAgent)
				if !ok {
					return fmt.Errorf("handoff agent not found: %s", completion.Handoff.TargetAgent)
				}
				metrics.Handoffs++
				logger.Info("handoff", "from", currentAgent.Name, "to", newAgent.Name, "mode", "shared")
				nextAgent = newAgent
				return nil
			}

			// Tool calls, separating handoff tool calls from regular ones.
			if len(completion.ToolCalls) > 0 {
				var (
					normalCalls []ToolCall
					handoffList []pendingHandoff
				)
				for _, call := range toolCallsFromProviders(completion.ToolCalls) {
					if sub, mode, ok := currentAgent.HandoffForTool(call.Name); ok {
						task, _ := call.Arguments["task"].(string)
						handoffList = append(handoffList, pendingHandoff{call: call, sub: sub, mode: mode, task: task})
						continue
					}
					normalCalls = append(normalCalls, call)
				}

				if len(normalCalls) > 0 {
					metrics.ToolCalls += len(normalCalls)
					toolResponses, err := r.executeTools(turnCtx, tracer, logger, currentAgent, normalCalls)
					if err != nil {
						turnSpan.SetError(err)
						return fmt.Errorf("tool execution failed: %w", err)
					}
					for _, resp := range toolResponses {
						messages = append(messages, toolMessage(resp.ToolCallID, toolContent(resp)))
					}
				}

				// Only one handoff is acted on per turn; acknowledge extras.
				for i := 1; i < len(handoffList); i++ {
					messages = append(messages, toolMessage(handoffList[i].call.ID,
						"Ignored: only one handoff is processed per turn."))
				}

				if len(handoffList) > 0 {
					h := handoffList[0]
					metrics.Handoffs++

					if h.mode == ContextShared {
						logger.Info("handoff", "from", currentAgent.Name, "to", h.sub.Name, "mode", "shared")
						messages = append(messages, toolMessage(h.call.ID,
							fmt.Sprintf("Transferring to %s.", h.sub.Name)))
						nextAgent = h.sub
						return nil
					}

					// ContextFresh / ContextForked: delegate and return the result.
					delegateCtx, hSpan := tracing.Start(turnCtx, tracer, "handoff.delegate",
						tracing.A("to", h.sub.Name), tracing.A("mode", h.mode.String()), tracing.A("task", h.task))
					logger.Info("delegate", "from", currentAgent.Name, "to", h.sub.Name, "mode", h.mode.String())

					nested := buildHandoffMessages(h.mode, messages[:turnStart], h.task)
					subCtx := *ctx
					subCtx.Context = delegateCtx
					subResult, err := r.executeLoop(&subCtx, tracer, logger.With("parent", currentAgent.Name), h.sub, nested)
					if err != nil {
						hSpan.SetError(err)
						hSpan.End()
						return fmt.Errorf("subagent %q failed: %w", h.sub.Name, err)
					}
					hSpan.SetAttributes(tracing.A("tokens", subResult.Metrics.TotalTokens))
					hSpan.End()
					metrics.TotalTokens += subResult.Metrics.TotalTokens
					metrics.ToolCalls += subResult.Metrics.ToolCalls
					metrics.Handoffs += subResult.Metrics.Handoffs
					messages = append(messages, toolMessage(h.call.ID,
						fmt.Sprintf("%v", subResult.FinalOutput)))
				}
				return nil // continue to next turn
			}

			// Plain text final output.
			if currentAgent.OutputType == nil {
				metrics.Duration = time.Since(startTime)
				metrics.TotalTurns = turn + 1
				result = &RunResult{FinalOutput: completion.Message.Content, Messages: messages, Agent: currentAgent, Metrics: metrics}
			}
			return nil
		}()

		if turnErr != nil {
			return nil, turnErr
		}
		if result != nil {
			return result, nil
		}
		if nextAgent != nil {
			currentAgent = nextAgent
		}
	}

	return nil, ErrMaxTurnsExceeded
}

// executeTools runs tool calls in parallel or sequence, tracing and logging each.
func (r *Runner) executeTools(ctx context.Context, tracer tracing.Tracer, logger *slog.Logger, agent *Agent, toolCalls []ToolCall) ([]ToolResponse, error) {
	responses := make([]ToolResponse, len(toolCalls))

	if r.parallelTools && len(toolCalls) > 1 {
		g, gCtx := errgroup.WithContext(ctx)
		for i, call := range toolCalls {
			i, call := i, call // capture loop variables
			g.Go(func() error {
				responses[i] = r.runTool(gCtx, tracer, logger, agent, call)
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
	} else {
		for i, call := range toolCalls {
			responses[i] = r.runTool(ctx, tracer, logger, agent, call)
		}
	}

	return responses, nil
}

// runTool executes a single tool call within its own span.
func (r *Runner) runTool(ctx context.Context, tracer tracing.Tracer, logger *slog.Logger, agent *Agent, call ToolCall) ToolResponse {
	ctx, span := tracing.Start(ctx, tracer, "tool.execute",
		tracing.A("tool", call.Name),
		tracing.A("args", argsPreview(call.Arguments)))
	defer span.End()

	tool := r.findTool(agent, call.Name)
	if tool == nil {
		err := fmt.Errorf("tool not found: %s", call.Name)
		span.SetError(err)
		logger.Warn("tool not found", "tool", call.Name)
		return ToolResponse{ToolCallID: call.ID, Error: err}
	}

	start := time.Now()
	logger.Debug("tool call", "tool", call.Name)
	result, err := tool.Execute(ctx, call.Arguments)
	if err != nil {
		span.SetError(err)
		logger.Error("tool failed", "tool", call.Name, "error", err, "duration", time.Since(start))
		return ToolResponse{ToolCallID: call.ID, Error: err}
	}

	resultStr := fmt.Sprintf("%v", result)
	span.SetAttributes(tracing.A("result_chars", len(resultStr)))
	logger.Debug("tool result", "tool", call.Name, "duration", time.Since(start), "result_chars", len(resultStr))
	return ToolResponse{ToolCallID: call.ID, Content: result}
}

// argsPreview renders tool arguments compactly for trace attributes.
func argsPreview(args map[string]interface{}) string {
	b, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("%v", args)
	}
	return string(b)
}

// findTool locates a tool by name, including builtins such as load_skill.
func (r *Runner) findTool(agent *Agent, name string) tools.Tool {
	for _, tool := range agent.EffectiveTools() {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

// validateGuardrails runs all guardrail checks for the latest message.
func (r *Runner) validateGuardrails(ctx context.Context, tracer tracing.Tracer, logger *slog.Logger, agent *Agent, messages []Message) error {
	if len(messages) == 0 || len(agent.Guardrails) == 0 {
		return nil
	}

	_, span := tracing.Start(ctx, tracer, "guardrails", tracing.A("count", len(agent.Guardrails)))
	defer span.End()

	lastMessage := messages[len(messages)-1]
	for _, guardrail := range agent.Guardrails {
		if err := guardrail.Validate(lastMessage.Content); err != nil {
			span.SetError(err)
			logger.Warn("guardrail blocked", "guardrail", guardrail.Name(), "error", err)
			return fmt.Errorf("guardrail %s failed: %w", guardrail.Name(), err)
		}
	}
	logger.Debug("guardrails passed", "count", len(agent.Guardrails))
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
