// Package tracing provides span-based tracing and structured logging for agent
// runs. A Tracer renders spans (to the console, to slog, or into a recorder for
// programmatic access); span depth is tracked through context.Context so traces
// form a tree. Use Start to open an instrumented span:
//
//	ctx, span := tracing.Start(ctx, tracer, "tool.execute", tracing.A("name", "add"))
//	defer span.End()
package tracing

import "context"

// Attr is a key/value attribute attached to a span.
type Attr struct {
	Key   string
	Value any
}

// A is shorthand for constructing an Attr.
func A(key string, value any) Attr { return Attr{Key: key, Value: value} }

// Tracer renders spans. Implementations must be safe for concurrent use.
type Tracer interface {
	// StartSpan begins a span. The returned Span is ended by the caller. Depth
	// for tree rendering is read from ctx via Depth.
	StartSpan(ctx context.Context, name string, attrs []Attr) Span
}

// Span represents an in-progress operation.
type Span interface {
	// SetAttributes adds attributes to the span.
	SetAttributes(attrs ...Attr)
	// SetError marks the span as failed.
	SetError(err error)
	// End completes the span.
	End()
}

// Start opens a span on tr and returns a child context whose depth is one
// greater, so nested Start calls form a tree. A nil tracer is treated as NoOp.
func Start(ctx context.Context, tr Tracer, name string, attrs ...Attr) (context.Context, Span) {
	if tr == nil {
		tr = NoOpTracer{}
	}
	span := tr.StartSpan(ctx, name, attrs)
	return withDepth(ctx, Depth(ctx)+1), span
}

type depthKey struct{}

// Depth returns the current span depth carried by ctx (0 at the root).
func Depth(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	if d, ok := ctx.Value(depthKey{}).(int); ok {
		return d
	}
	return 0
}

func withDepth(ctx context.Context, d int) context.Context {
	return context.WithValue(ctx, depthKey{}, d)
}

// NoOpTracer discards all spans. It is the default.
type NoOpTracer struct{}

// NewNoOpTracer returns a tracer that does nothing.
func NewNoOpTracer() Tracer { return NoOpTracer{} }

// StartSpan returns a no-op span.
func (NoOpTracer) StartSpan(context.Context, string, []Attr) Span { return noOpSpan{} }

type noOpSpan struct{}

func (noOpSpan) SetAttributes(...Attr) {}
func (noOpSpan) SetError(error)        {}
func (noOpSpan) End()                  {}

// Tee fans a span out to several tracers at once (e.g. console + recorder).
type Tee []Tracer

// NewTee combines tracers, dropping nils. If only one remains it is returned
// directly; if none, a NoOpTracer is returned.
func NewTee(tracers ...Tracer) Tracer {
	var kept Tee
	for _, t := range tracers {
		if t != nil {
			if _, isNoop := t.(NoOpTracer); isNoop {
				continue
			}
			kept = append(kept, t)
		}
	}
	switch len(kept) {
	case 0:
		return NoOpTracer{}
	case 1:
		return kept[0]
	default:
		return kept
	}
}

// StartSpan starts the span on every underlying tracer.
func (t Tee) StartSpan(ctx context.Context, name string, attrs []Attr) Span {
	spans := make([]Span, 0, len(t))
	for _, tr := range t {
		spans = append(spans, tr.StartSpan(ctx, name, attrs))
	}
	return teeSpan(spans)
}

type teeSpan []Span

func (s teeSpan) SetAttributes(attrs ...Attr) {
	for _, sp := range s {
		sp.SetAttributes(attrs...)
	}
}

func (s teeSpan) SetError(err error) {
	for _, sp := range s {
		sp.SetError(err)
	}
}

func (s teeSpan) End() {
	for _, sp := range s {
		sp.End()
	}
}
