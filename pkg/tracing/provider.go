package tracing

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// TracerProvider manages tracer instances and configuration
type TracerProvider struct {
	defaultTracer Tracer
	config        *OTelConfig
}

// NewTracerProvider creates a new tracer provider with environment-based configuration
func NewTracerProvider() (*TracerProvider, error) {
	config := getConfigFromEnvironment()
	
	var tracer Tracer
	var err error
	
	// Check if tracing is enabled
	if !isTracingEnabled() {
		tracer = NewNoOpTracer()
	} else {
		switch config.ExporterType {
		case ExporterConsole:
			tracer = NewConsoleTracer()
		default:
			tracer, err = NewOTelTracer(config)
			if err != nil {
				// Fallback to console tracer on error
				fmt.Printf("Warning: Failed to initialize OpenTelemetry tracer (%v), falling back to console tracer\n", err)
				tracer = NewConsoleTracer()
			}
		}
	}
	
	return &TracerProvider{
		defaultTracer: tracer,
		config:        config,
	}, nil
}

// NewTracerProviderWithConfig creates a tracer provider with explicit configuration
func NewTracerProviderWithConfig(config *OTelConfig) (*TracerProvider, error) {
	var tracer Tracer
	var err error
	
	switch config.ExporterType {
	case ExporterConsole:
		tracer = NewConsoleTracer()
	default:
		tracer, err = NewOTelTracer(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenTelemetry tracer: %w", err)
		}
	}
	
	return &TracerProvider{
		defaultTracer: tracer,
		config:        config,
	}, nil
}

// GetTracer returns the default tracer
func (tp *TracerProvider) GetTracer() Tracer {
	return tp.defaultTracer
}

// GetConfig returns the tracer configuration
func (tp *TracerProvider) GetConfig() *OTelConfig {
	return tp.config
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if otelTracer, ok := tp.defaultTracer.(*OTelTracer); ok {
		return otelTracer.Shutdown(ctx)
	}
	return nil
}

// Environment variable configuration

// isTracingEnabled checks if tracing is enabled via environment variables
func isTracingEnabled() bool {
	enabled := strings.ToLower(os.Getenv("AGENTS_TRACE_ENABLED"))
	return enabled == "true" || enabled == "1" || enabled == "yes"
}

// getConfigFromEnvironment creates an OTelConfig from environment variables
func getConfigFromEnvironment() *OTelConfig {
	config := DefaultOTelConfig()
	
	// Service name
	if serviceName := os.Getenv("AGENTS_SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}
	
	// Service version
	if serviceVersion := os.Getenv("AGENTS_SERVICE_VERSION"); serviceVersion != "" {
		config.ServiceVersion = serviceVersion
	}
	
	// Environment
	if environment := os.Getenv("AGENTS_ENVIRONMENT"); environment != "" {
		config.Environment = environment
	}
	
	// Exporter type
	if exporterType := os.Getenv("AGENTS_TRACE_EXPORTER"); exporterType != "" {
		config.ExporterType = ExporterType(strings.ToLower(exporterType))
	}
	
	// Endpoint
	if endpoint := os.Getenv("AGENTS_TRACE_ENDPOINT"); endpoint != "" {
		config.Endpoint = endpoint
	} else {
		// Set default endpoints based on exporter type
		switch config.ExporterType {
		case ExporterOTLP:
			config.Endpoint = "http://localhost:4318/v1/traces"
		case ExporterJaeger:
			config.Endpoint = "http://localhost:14268/api/traces"
		case ExporterZipkin:
			config.Endpoint = "http://localhost:9411/api/v2/spans"
		}
	}
	
	// Sampling ratio
	if samplingRatio := os.Getenv("AGENTS_TRACE_SAMPLING_RATIO"); samplingRatio != "" {
		if ratio, err := strconv.ParseFloat(samplingRatio, 64); err == nil && ratio >= 0 && ratio <= 1 {
			config.SamplingRatio = ratio
		}
	}
	
	// Batch timeout
	if batchTimeout := os.Getenv("AGENTS_TRACE_BATCH_TIMEOUT"); batchTimeout != "" {
		if timeout, err := time.ParseDuration(batchTimeout); err == nil {
			config.BatchTimeout = timeout
		}
	}
	
	// Max export batch size
	if maxBatch := os.Getenv("AGENTS_TRACE_MAX_BATCH_SIZE"); maxBatch != "" {
		if size, err := strconv.Atoi(maxBatch); err == nil && size > 0 {
			config.MaxExportBatch = size
		}
	}
	
	// Max queue size
	if maxQueue := os.Getenv("AGENTS_TRACE_MAX_QUEUE_SIZE"); maxQueue != "" {
		if size, err := strconv.Atoi(maxQueue); err == nil && size > 0 {
			config.MaxQueueSize = size
		}
	}
	
	return config
}

// Global tracer provider instance
var globalProvider *TracerProvider

// InitGlobalTracer initializes the global tracer provider
func InitGlobalTracer() error {
	provider, err := NewTracerProvider()
	if err != nil {
		return err
	}
	globalProvider = provider
	return nil
}

// InitGlobalTracerWithConfig initializes the global tracer provider with explicit configuration
func InitGlobalTracerWithConfig(config *OTelConfig) error {
	provider, err := NewTracerProviderWithConfig(config)
	if err != nil {
		return err
	}
	globalProvider = provider
	return nil
}

// GetGlobalTracer returns the global tracer instance
func GetGlobalTracer() Tracer {
	if globalProvider == nil {
		// Initialize with default configuration if not already done
		if err := InitGlobalTracer(); err != nil {
			// Fallback to no-op tracer
			return NewNoOpTracer()
		}
	}
	return globalProvider.GetTracer()
}

// ShutdownGlobalTracer shuts down the global tracer provider
func ShutdownGlobalTracer(ctx context.Context) error {
	if globalProvider != nil {
		return globalProvider.Shutdown(ctx)
	}
	return nil
}

// Convenience functions for common tracing patterns

// TraceOperation wraps an operation with a trace span
func TraceOperation(ctx context.Context, operationName string, fn func(ctx context.Context) error) error {
	tracer := GetGlobalTracer()
	ctx, span := tracer.StartSpan(ctx, operationName)
	defer tracer.EndSpan(span)
	
	err := fn(ctx)
	if err != nil {
		span.SetError(err)
	}
	
	return err
}

// TraceProviderOperation wraps a provider operation with a trace span
func TraceProviderOperation(ctx context.Context, provider, operation string, fn func(ctx context.Context) error) error {
	tracer := GetGlobalTracer()
	
	// Try to use enhanced span creation if available
	var span Span
	if otelTracer, ok := tracer.(*OTelTracer); ok {
		ctx, span = otelTracer.StartProviderSpan(ctx, provider, operation)
	} else {
		spanName := fmt.Sprintf("provider.%s.%s", provider, operation)
		ctx, span = tracer.StartSpan(ctx, spanName)
		span.SetAttribute("provider.name", provider)
		span.SetAttribute("provider.operation", operation)
	}
	
	defer tracer.EndSpan(span)
	
	err := fn(ctx)
	if err != nil {
		span.SetError(err)
	}
	
	return err
}

// TraceAgentOperation wraps an agent operation with a trace span
func TraceAgentOperation(ctx context.Context, agentName, operation string, fn func(ctx context.Context) error) error {
	tracer := GetGlobalTracer()
	
	// Try to use enhanced span creation if available
	var span Span
	if otelTracer, ok := tracer.(*OTelTracer); ok {
		ctx, span = otelTracer.StartAgentSpan(ctx, agentName, operation)
	} else {
		spanName := fmt.Sprintf("agent.%s.%s", agentName, operation)
		ctx, span = tracer.StartSpan(ctx, spanName)
		span.SetAttribute("agent.name", agentName)
		span.SetAttribute("agent.operation", operation)
	}
	
	defer tracer.EndSpan(span)
	
	err := fn(ctx)
	if err != nil {
		span.SetError(err)
	}
	
	return err
}

// TraceToolOperation wraps a tool operation with a trace span
func TraceToolOperation(ctx context.Context, toolName string, fn func(ctx context.Context) error) error {
	tracer := GetGlobalTracer()
	
	// Try to use enhanced span creation if available
	var span Span
	if otelTracer, ok := tracer.(*OTelTracer); ok {
		ctx, span = otelTracer.StartToolSpan(ctx, toolName)
	} else {
		spanName := fmt.Sprintf("tool.%s", toolName)
		ctx, span = tracer.StartSpan(ctx, spanName)
		span.SetAttribute("tool.name", toolName)
	}
	
	defer tracer.EndSpan(span)
	
	err := fn(ctx)
	if err != nil {
		span.SetError(err)
	}
	
	return err
}