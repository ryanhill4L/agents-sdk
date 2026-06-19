package tracing

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ConsoleTracer renders spans as an indented tree to a writer (stderr by
// default), one line per span when it ends, showing duration and attributes.
type ConsoleTracer struct {
	mu sync.Mutex
	w  io.Writer
}

// NewConsoleTracer writes traces to stderr.
func NewConsoleTracer() Tracer { return &ConsoleTracer{w: os.Stderr} }

// NewConsoleTracerTo writes traces to w.
func NewConsoleTracerTo(w io.Writer) Tracer { return &ConsoleTracer{w: w} }

// StartSpan begins a console span at the current context depth.
func (t *ConsoleTracer) StartSpan(ctx context.Context, name string, attrs []Attr) Span {
	return &consoleSpan{
		tracer: t,
		name:   name,
		depth:  Depth(ctx),
		start:  time.Now(),
		attrs:  append([]Attr(nil), attrs...),
	}
}

func (t *ConsoleTracer) write(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprintln(t.w, line)
}

type consoleSpan struct {
	tracer *ConsoleTracer
	name   string
	depth  int
	start  time.Time
	attrs  []Attr
	err    error
	ended  bool
}

func (s *consoleSpan) SetAttributes(attrs ...Attr) { s.attrs = append(s.attrs, attrs...) }
func (s *consoleSpan) SetError(err error)          { s.err = err }

func (s *consoleSpan) End() {
	if s.ended {
		return
	}
	s.ended = true
	dur := time.Since(s.start)

	var b strings.Builder
	b.WriteString("[trace] ")
	b.WriteString(strings.Repeat("  ", s.depth))
	b.WriteString(s.name)
	fmt.Fprintf(&b, " (%s)", dur.Round(time.Microsecond))
	for _, a := range s.attrs {
		fmt.Fprintf(&b, " %s=%v", a.Key, formatValue(a.Value))
	}
	if s.err != nil {
		fmt.Fprintf(&b, " error=%q", s.err.Error())
	}
	s.tracer.write(b.String())
}

// formatValue keeps trace lines readable by truncating long string values.
func formatValue(v any) any {
	if str, ok := v.(string); ok {
		const max = 120
		str = strings.ReplaceAll(str, "\n", "\\n")
		if len(str) > max {
			return str[:max] + "…"
		}
		return str
	}
	return v
}
