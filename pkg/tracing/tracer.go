package tracing

import (
	"context"
	"time"
)

// Tracer provides distributed tracing capabilities
type Tracer interface {
	// StartSpan starts a new span with the given name
	StartSpan(ctx context.Context, name string) (context.Context, Span)

	// EndSpan ends the given span
	EndSpan(span Span)
}

// Span represents a single trace span
type Span interface {
	// SetAttribute sets an attribute on the span
	SetAttribute(key string, value interface{})

	// SetError marks the span as having an error
	SetError(err error)

	// End ends the span
	End()
}

// NewNoOpTracer creates a tracer that does nothing
// This is referenced in runner.go
func NewNoOpTracer() Tracer {
	return &NoOpTracer{}
}

// NoOpTracer is a no-operation tracer for when tracing is disabled
type NoOpTracer struct{}

func (t *NoOpTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return ctx, &NoOpSpan{}
}

func (t *NoOpTracer) EndSpan(span Span) {
	// No-op
}

// NoOpSpan is a no-operation span
type NoOpSpan struct{}

func (s *NoOpSpan) SetAttribute(key string, value interface{}) {
	// No-op
}

func (s *NoOpSpan) SetError(err error) {
	// No-op
}

func (s *NoOpSpan) End() {
	// No-op
}

// ConsoleTracer is a simple tracer that logs to stdout
type ConsoleTracer struct{}

func NewConsoleTracer() Tracer {
	return &ConsoleTracer{}
}

func (t *ConsoleTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	span := &ConsoleSpan{
		Name:      name,
		StartTime: time.Now(),
	}
	return ctx, span
}

func (t *ConsoleTracer) EndSpan(span Span) {
	if cs, ok := span.(*ConsoleSpan); ok {
		duration := time.Since(cs.StartTime)
		println("TRACE:", cs.Name, "completed in", duration.String())
	}
}

// ConsoleSpan logs span information to stdout
type ConsoleSpan struct {
	Name      string
	StartTime time.Time
}

func (s *ConsoleSpan) SetAttribute(key string, value interface{}) {
	println("TRACE:", s.Name, "attribute:", key, "=", value)
}

func (s *ConsoleSpan) SetError(err error) {
	println("TRACE:", s.Name, "error:", err.Error())
}

func (s *ConsoleSpan) End() {
	duration := time.Since(s.StartTime)
	println("TRACE:", s.Name, "ended after", duration.String())
}