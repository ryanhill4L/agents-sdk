package memory

import (
	"errors"

	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

var ErrSessionNotFound = errors.New("session not found")

type Session interface {
	Load() ([]types.Message, error)
	Save(messages []types.Message) error
	Clear() error
	Exists() bool
	Close() error
	GetID() string
}