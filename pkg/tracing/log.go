package tracing

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// DiscardLogger returns a *slog.Logger that drops all records. It is the safe
// default when no logger is configured.
func DiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// OrDiscard returns logger, or a DiscardLogger when logger is nil.
func OrDiscard(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return DiscardLogger()
	}
	return logger
}

// NewLogger builds a text logger writing to w (stderr if nil) at the given
// level.
func NewLogger(w io.Writer, level slog.Level) *slog.Logger {
	if w == nil {
		w = os.Stderr
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}

// ParseLevel maps a string (debug/info/warn/error) to an slog.Level, defaulting
// to info.
func ParseLevel(s string) slog.Level {
	switch s {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "warn", "WARN", "warning":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SlogTracer renders spans as slog records (one debug record per span end,
// including duration and attributes).
type SlogTracer struct {
	logger *slog.Logger
}

// NewSlogTracer returns a tracer that emits spans to logger (or a discard
// logger when nil).
func NewSlogTracer(logger *slog.Logger) Tracer {
	return &SlogTracer{logger: OrDiscard(logger)}
}

// StartSpan begins an slog-backed span.
func (t *SlogTracer) StartSpan(ctx context.Context, name string, attrs []Attr) Span {
	return &slogSpan{
		logger: t.logger,
		ctx:    ctx,
		name:   name,
		depth:  Depth(ctx),
		start:  time.Now(),
		attrs:  append([]Attr(nil), attrs...),
	}
}

type slogSpan struct {
	logger *slog.Logger
	ctx    context.Context
	name   string
	depth  int
	start  time.Time
	attrs  []Attr
	err    error
	ended  bool
}

func (s *slogSpan) SetAttributes(attrs ...Attr) { s.attrs = append(s.attrs, attrs...) }
func (s *slogSpan) SetError(err error)          { s.err = err }

func (s *slogSpan) End() {
	if s.ended {
		return
	}
	s.ended = true

	args := []any{
		slog.String("span", s.name),
		slog.Int("depth", s.depth),
		slog.Duration("duration", time.Since(s.start)),
	}
	for _, a := range s.attrs {
		args = append(args, slog.Any(a.Key, a.Value))
	}
	level := slog.LevelDebug
	if s.err != nil {
		args = append(args, slog.String("error", s.err.Error()))
		level = slog.LevelError
	}
	s.logger.Log(s.ctx, level, "span", args...)
}
