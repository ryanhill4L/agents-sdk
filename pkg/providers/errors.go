package providers

import (
	"errors"
	"fmt"
)

// Provider configuration errors
var (
	ErrMissingAPIKey       = errors.New("API key is required")
	ErrInvalidTimeout      = errors.New("timeout must be greater than 0")
	ErrInvalidMaxRetries   = errors.New("max retries must be non-negative")
	ErrUnsupportedProvider = errors.New("unsupported provider type")
)

// Provider operation errors
var (
	ErrProviderUnavailable = errors.New("provider service unavailable")
	ErrRateLimited         = errors.New("rate limit exceeded")
	ErrInvalidModel        = errors.New("invalid or unsupported model")
	ErrContextTooLong      = errors.New("context length exceeds model limit")
	ErrInvalidToolCall     = errors.New("invalid tool call format")
	ErrToolNotFound        = errors.New("requested tool not found")
)

// ProviderError wraps provider-specific errors with context
type ProviderError struct {
	Provider string
	Op       string
	Err      error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s provider error in %s: %v", e.Provider, e.Op, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new provider error with context
func NewProviderError(provider, operation string, err error) error {
	return &ProviderError{
		Provider: provider,
		Op:       operation,
		Err:      err,
	}
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	var provErr *ProviderError
	if errors.As(err, &provErr) {
		return errors.Is(provErr.Err, ErrRateLimited)
	}
	return errors.Is(err, ErrRateLimited)
}

// IsTemporaryError checks if an error is temporary and retryable
func IsTemporaryError(err error) bool {
	var provErr *ProviderError
	if errors.As(err, &provErr) {
		switch provErr.Err {
		case ErrProviderUnavailable, ErrRateLimited:
			return true
		}
	}
	
	switch err {
	case ErrProviderUnavailable, ErrRateLimited:
		return true
	}
	
	return false
}