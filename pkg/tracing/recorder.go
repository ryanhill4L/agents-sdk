package tracing

import (
	"context"
	"sort"
	"sync"
	"time"
)

// SpanRecord is a completed span captured by a Recorder.
type SpanRecord struct {
	Name       string
	Depth      int
	Attributes []Attr
	Error      string
	StartedAt  time.Time
	Duration   time.Duration
}

// Recorder is a Tracer that captures spans in memory for programmatic access
// (e.g. RunResult.Traces) and tests.
type Recorder struct {
	mu      sync.Mutex
	records []SpanRecord
}

// NewRecorder returns an empty recorder.
func NewRecorder() *Recorder { return &Recorder{} }

// StartSpan begins a recorded span.
func (r *Recorder) StartSpan(ctx context.Context, name string, attrs []Attr) Span {
	return &recordingSpan{
		rec:   r,
		depth: Depth(ctx),
		start: time.Now(),
		record: SpanRecord{
			Name:       name,
			Depth:      Depth(ctx),
			Attributes: append([]Attr(nil), attrs...),
		},
	}
}

// Records returns the captured spans, ordered by start time.
func (r *Recorder) Records() []SpanRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]SpanRecord, len(r.records))
	copy(out, r.records)
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	return out
}

type recordingSpan struct {
	rec    *Recorder
	depth  int
	start  time.Time
	record SpanRecord
	ended  bool
}

func (s *recordingSpan) SetAttributes(attrs ...Attr) {
	s.record.Attributes = append(s.record.Attributes, attrs...)
}

func (s *recordingSpan) SetError(err error) {
	if err != nil {
		s.record.Error = err.Error()
	}
}

func (s *recordingSpan) End() {
	if s.ended {
		return
	}
	s.ended = true
	s.record.StartedAt = s.start
	s.record.Duration = time.Since(s.start)

	s.rec.mu.Lock()
	s.rec.records = append(s.rec.records, s.record)
	s.rec.mu.Unlock()
}
