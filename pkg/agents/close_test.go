package agents

import (
	"sync/atomic"
	"testing"
)

type countingCloser struct{ closes int32 }

func (c *countingCloser) Close() error {
	atomic.AddInt32(&c.closes, 1)
	return nil
}

func TestCloseDiamondClosesOnce(t *testing.T) {
	// shared is reachable through two parents (a diamond). Its closer must run
	// exactly once, and Close must not recurse forever.
	shared := NewAgent("shared")
	closer := &countingCloser{}
	shared.AddCloser(closer)

	left := NewAgent("left", WithHandoffs(shared))
	right := NewAgent("right", WithHandoffs(shared))
	root := NewAgent("root", WithHandoffs(left, right))

	if err := root.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := atomic.LoadInt32(&closer.closes); got != 1 {
		t.Errorf("shared closer ran %d times, want 1", got)
	}
}
