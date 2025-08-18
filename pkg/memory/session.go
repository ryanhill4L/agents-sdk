package memory

import (
	"context"
	"time"
)

// Message represents a conversation message
// This needs to be defined here to avoid circular imports
type Message struct {
	ID        int64                  `json:"id,omitempty"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Session manages conversation history and state
type Session interface {
	// GetItems retrieves messages from the session
	GetItems(ctx context.Context, limit int) ([]Message, error)

	// AddItems adds messages to the session
	AddItems(ctx context.Context, items []Message) error

	// PopItem removes and returns the most recent message
	PopItem(ctx context.Context) (*Message, error)

	// Clear removes all messages from the session
	Clear(ctx context.Context) error

	// Close closes the session and cleans up resources
	Close() error
}