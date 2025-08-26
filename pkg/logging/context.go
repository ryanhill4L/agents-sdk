package logging

import (
	"context"
	"github.com/google/uuid"
)

// Context keys for logging
type contextKey string

const (
	loggerContextKey   contextKey = "logger"
	traceIDContextKey  contextKey = "trace_id"
	requestIDContextKey contextKey = "request_id"
)

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// FromContext extracts a logger from the context, returns a default logger if none exists
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerContextKey).(Logger); ok {
		return logger
	}
	return NewConsoleLogger(InfoLevel, false)
}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDContextKey, traceID)
}

// GetTraceID extracts the trace ID from the context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDContextKey).(string); ok {
		return traceID
	}
	return ""
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// GetRequestID extracts the request ID from the context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDContextKey).(string); ok {
		return requestID
	}
	return ""
}

// NewRequestID generates a new request ID
func NewRequestID() string {
	return uuid.New().String()
}

// WithNewRequestID adds a new request ID to the context
func WithNewRequestID(ctx context.Context) context.Context {
	return WithRequestID(ctx, NewRequestID())
}