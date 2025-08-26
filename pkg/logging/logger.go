package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger defines the interface for structured logging
type Logger interface {
	// Debug logs a debug message with optional fields
	Debug(ctx context.Context, message string, fields ...Field)
	
	// Info logs an info message with optional fields
	Info(ctx context.Context, message string, fields ...Field)
	
	// Warn logs a warning message with optional fields
	Warn(ctx context.Context, message string, fields ...Field)
	
	// Error logs an error message with optional fields
	Error(ctx context.Context, message string, fields ...Field)
	
	// With creates a child logger with additional fields
	With(fields ...Field) Logger
	
	// WithLevel creates a logger that only logs at or above the given level
	WithLevel(level LogLevel) Logger
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Error creates an error field
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// ConsoleLogger implements Logger interface for human-readable console output
type ConsoleLogger struct {
	level   LogLevel
	fields  map[string]interface{}
	verbose bool
}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger(level LogLevel, verbose bool) Logger {
	return &ConsoleLogger{
		level:   level,
		fields:  make(map[string]interface{}),
		verbose: verbose,
	}
}

func (c *ConsoleLogger) shouldLog(level LogLevel) bool {
	return level >= c.level
}

func (c *ConsoleLogger) log(ctx context.Context, level LogLevel, message string, fields ...Field) {
	if !c.shouldLog(level) {
		return
	}

	// Build log entry
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := level.String()
	
	// Combine base fields with log fields
	allFields := make(map[string]interface{})
	for k, v := range c.fields {
		allFields[k] = v
	}
	
	// Add context fields
	if traceID := GetTraceID(ctx); traceID != "" {
		allFields["trace_id"] = traceID
	}
	if requestID := GetRequestID(ctx); requestID != "" {
		allFields["request_id"] = requestID
	}
	
	// Add provided fields
	for _, field := range fields {
		allFields[field.Key] = field.Value
	}

	// Format output
	if c.verbose && len(allFields) > 0 {
		fieldsJSON, _ := json.Marshal(allFields)
		fmt.Printf("[%s] %s: %s | %s\n", timestamp, levelStr, message, string(fieldsJSON))
	} else if len(allFields) > 0 {
		// Simple format with key fields
		var keyFields []string
		for k, v := range allFields {
			if k == "error" || k == "provider" || k == "model" || k == "trace_id" {
				keyFields = append(keyFields, fmt.Sprintf("%s=%v", k, v))
			}
		}
		if len(keyFields) > 0 {
			fmt.Printf("[%s] %s: %s [%s]\n", timestamp, levelStr, message, fmt.Sprintf("%v", keyFields))
		} else {
			fmt.Printf("[%s] %s: %s\n", timestamp, levelStr, message)
		}
	} else {
		fmt.Printf("[%s] %s: %s\n", timestamp, levelStr, message)
	}
}

func (c *ConsoleLogger) Debug(ctx context.Context, message string, fields ...Field) {
	c.log(ctx, DebugLevel, message, fields...)
}

func (c *ConsoleLogger) Info(ctx context.Context, message string, fields ...Field) {
	c.log(ctx, InfoLevel, message, fields...)
}

func (c *ConsoleLogger) Warn(ctx context.Context, message string, fields ...Field) {
	c.log(ctx, WarnLevel, message, fields...)
}

func (c *ConsoleLogger) Error(ctx context.Context, message string, fields ...Field) {
	c.log(ctx, ErrorLevel, message, fields...)
}

func (c *ConsoleLogger) With(fields ...Field) Logger {
	newFields := make(map[string]interface{})
	for k, v := range c.fields {
		newFields[k] = v
	}
	for _, field := range fields {
		newFields[field.Key] = field.Value
	}
	
	return &ConsoleLogger{
		level:   c.level,
		fields:  newFields,
		verbose: c.verbose,
	}
}

func (c *ConsoleLogger) WithLevel(level LogLevel) Logger {
	return &ConsoleLogger{
		level:   level,
		fields:  c.fields,
		verbose: c.verbose,
	}
}

// StructuredLogger implements Logger interface for JSON-structured output
type StructuredLogger struct {
	level  LogLevel
	fields map[string]interface{}
	logger *log.Logger
}

// NewStructuredLogger creates a new structured JSON logger
func NewStructuredLogger(level LogLevel) Logger {
	return &StructuredLogger{
		level:  level,
		fields: make(map[string]interface{}),
		logger: log.New(os.Stdout, "", 0),
	}
}

func (s *StructuredLogger) shouldLog(level LogLevel) bool {
	return level >= s.level
}

func (s *StructuredLogger) log(ctx context.Context, level LogLevel, message string, fields ...Field) {
	if !s.shouldLog(level) {
		return
	}

	// Build log entry
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     level.String(),
		"message":   message,
	}

	// Add base fields
	for k, v := range s.fields {
		entry[k] = v
	}

	// Add context fields
	if traceID := GetTraceID(ctx); traceID != "" {
		entry["trace_id"] = traceID
	}
	if requestID := GetRequestID(ctx); requestID != "" {
		entry["request_id"] = requestID
	}

	// Add provided fields
	for _, field := range fields {
		entry[field.Key] = field.Value
	}

	// Marshal and output
	if jsonBytes, err := json.Marshal(entry); err == nil {
		s.logger.Println(string(jsonBytes))
	} else {
		// Fallback to simple format
		s.logger.Printf("[%s] %s: %s (JSON marshal error: %v)", 
			entry["timestamp"], entry["level"], message, err)
	}
}

func (s *StructuredLogger) Debug(ctx context.Context, message string, fields ...Field) {
	s.log(ctx, DebugLevel, message, fields...)
}

func (s *StructuredLogger) Info(ctx context.Context, message string, fields ...Field) {
	s.log(ctx, InfoLevel, message, fields...)
}

func (s *StructuredLogger) Warn(ctx context.Context, message string, fields ...Field) {
	s.log(ctx, WarnLevel, message, fields...)
}

func (s *StructuredLogger) Error(ctx context.Context, message string, fields ...Field) {
	s.log(ctx, ErrorLevel, message, fields...)
}

func (s *StructuredLogger) With(fields ...Field) Logger {
	newFields := make(map[string]interface{})
	for k, v := range s.fields {
		newFields[k] = v
	}
	for _, field := range fields {
		newFields[field.Key] = field.Value
	}
	
	return &StructuredLogger{
		level:  s.level,
		fields: newFields,
		logger: s.logger,
	}
}

func (s *StructuredLogger) WithLevel(level LogLevel) Logger {
	return &StructuredLogger{
		level:  level,
		fields: s.fields,
		logger: s.logger,
	}
}

// NoOpLogger implements Logger interface but does nothing
type NoOpLogger struct{}

// NewNoOpLogger creates a no-operation logger
func NewNoOpLogger() Logger {
	return &NoOpLogger{}
}

func (n *NoOpLogger) Debug(ctx context.Context, message string, fields ...Field) {}
func (n *NoOpLogger) Info(ctx context.Context, message string, fields ...Field)  {}
func (n *NoOpLogger) Warn(ctx context.Context, message string, fields ...Field)  {}
func (n *NoOpLogger) Error(ctx context.Context, message string, fields ...Field) {}
func (n *NoOpLogger) With(fields ...Field) Logger                                { return n }
func (n *NoOpLogger) WithLevel(level LogLevel) Logger                            { return n }

// ParseLogLevel parses a log level from string
func ParseLogLevel(level string) LogLevel {
	switch level {
	case "DEBUG", "debug":
		return DebugLevel
	case "INFO", "info":
		return InfoLevel
	case "WARN", "warn", "WARNING", "warning":
		return WarnLevel
	case "ERROR", "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// NewLogger creates a logger based on environment configuration
func NewLogger() Logger {
	logLevel := ParseLogLevel(os.Getenv("AGENTS_LOG_LEVEL"))
	verbose := os.Getenv("AGENTS_VERBOSE") == "true"
	format := os.Getenv("AGENTS_LOG_FORMAT")

	if format == "json" {
		return NewStructuredLogger(logLevel)
	}
	return NewConsoleLogger(logLevel, verbose)
}