// Package channels provides integration adapters that connect an agent to the
// outside world. Each channel turns inbound messages into agent runs and routes
// the output back. The HTTP channel is the default, powering `eve dev`.
package channels

import "context"

// Handler runs an agent for a single inbound message and returns its reply.
type Handler func(ctx context.Context, input string) (string, error)

// Channel is an integration adapter. Start blocks until ctx is cancelled or an
// unrecoverable error occurs.
type Channel interface {
	Name() string
	Start(ctx context.Context, handler Handler) error
}
