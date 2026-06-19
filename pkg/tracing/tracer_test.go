package tracing

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRecorderCapturesDepthAndAttrs(t *testing.T) {
	rec := NewRecorder()
	ctx := context.Background()

	ctx, root := Start(ctx, rec, "run", A("agent", "a"))
	_, child := Start(ctx, rec, "turn", A("n", 0))
	child.SetAttributes(A("extra", true))
	child.End()
	root.SetError(errors.New("boom"))
	root.End()

	records := rec.Records()
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	// Ordered by start time: run first, then turn.
	if records[0].Name != "run" || records[0].Depth != 0 {
		t.Errorf("record0 = %+v", records[0])
	}
	if records[0].Error != "boom" {
		t.Errorf("expected error captured, got %q", records[0].Error)
	}
	if records[1].Name != "turn" || records[1].Depth != 1 {
		t.Errorf("record1 = %+v", records[1])
	}
}

func TestTeeFansOut(t *testing.T) {
	a, b := NewRecorder(), NewRecorder()
	tee := NewTee(a, b, NoOpTracer{}, nil)

	_, span := Start(context.Background(), tee, "x")
	span.End()

	if len(a.Records()) != 1 || len(b.Records()) != 1 {
		t.Errorf("expected both recorders to capture the span")
	}
}

func TestNewTeeCollapses(t *testing.T) {
	if _, ok := NewTee(nil, NoOpTracer{}).(NoOpTracer); !ok {
		t.Error("NewTee with no real tracers should be NoOpTracer")
	}
	rec := NewRecorder()
	if NewTee(rec) != Tracer(rec) {
		t.Error("NewTee with one tracer should return it directly")
	}
}

func TestConsoleTracerOutput(t *testing.T) {
	var buf bytes.Buffer
	tr := NewConsoleTracerTo(&buf)

	ctx, root := Start(context.Background(), tr, "run", A("agent", "demo"))
	_, child := Start(ctx, tr, "tool.execute", A("tool", "add"))
	child.End()
	root.End()

	out := buf.String()
	if !strings.Contains(out, "run") || !strings.Contains(out, "agent=demo") {
		t.Errorf("missing root span line: %q", out)
	}
	if !strings.Contains(out, "  tool.execute") || !strings.Contains(out, "tool=add") {
		t.Errorf("missing indented child span line: %q", out)
	}
}

func TestFormatValueTruncates(t *testing.T) {
	long := strings.Repeat("x", 200)
	got, ok := formatValue(long).(string)
	if !ok || len(got) <= 120 || !strings.HasSuffix(got, "…") {
		t.Errorf("expected truncation, got len %d", len(got))
	}
}
