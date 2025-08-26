package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ExporterType represents the type of trace exporter
type ExporterType string

const (
	ExporterOTLP   ExporterType = "otlp"
	ExporterJaeger ExporterType = "jaeger"
	ExporterZipkin ExporterType = "zipkin"
	ExporterConsole ExporterType = "console"
)

// OTelConfig holds configuration for OpenTelemetry tracing
type OTelConfig struct {
	ServiceName     string
	ServiceVersion  string
	Environment     string
	ExporterType    ExporterType
	Endpoint        string
	SamplingRatio   float64
	BatchTimeout    time.Duration
	MaxExportBatch  int
	MaxQueueSize    int
}

// DefaultOTelConfig returns a default OpenTelemetry configuration
func DefaultOTelConfig() *OTelConfig {
	return &OTelConfig{
		ServiceName:     "agents-sdk",
		ServiceVersion:  "1.0.0",
		Environment:     "development",
		ExporterType:    ExporterConsole,
		Endpoint:        "",
		SamplingRatio:   1.0, // Sample all traces in development
		BatchTimeout:    5 * time.Second,
		MaxExportBatch:  512,
		MaxQueueSize:    2048,
	}
}

// OTelTracer implements the Tracer interface using OpenTelemetry
type OTelTracer struct {
	tracer      oteltrace.Tracer
	provider    *trace.TracerProvider
	serviceName string
}

// NewOTelTracer creates a new OpenTelemetry tracer
func NewOTelTracer(config *OTelConfig) (*OTelTracer, error) {
	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create span exporter based on configuration
	var exporter trace.SpanExporter
	switch config.ExporterType {
	case ExporterOTLP:
		exporter, err = otlptracehttp.New(context.Background(),
			otlptracehttp.WithEndpoint(config.Endpoint),
		)
		if err != nil {
			return nil, err
		}
	case ExporterJaeger:
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(config.Endpoint),
		))
		if err != nil {
			return nil, err
		}
	case ExporterZipkin:
		exporter, err = zipkin.New(config.Endpoint)
		if err != nil {
			return nil, err
		}
	case ExporterConsole:
		// Use the existing console tracer as exporter
		return &OTelTracer{
			tracer:      nil, // Will use console tracer fallback
			provider:    nil,
			serviceName: config.ServiceName,
		}, nil
	default:
		return nil, ErrUnsupportedExporter
	}

	// Create tracer provider with batch span processor
	bsp := trace.NewBatchSpanProcessor(exporter,
		trace.WithBatchTimeout(config.BatchTimeout),
		trace.WithMaxExportBatchSize(config.MaxExportBatch),
		trace.WithMaxQueueSize(config.MaxQueueSize),
	)

	// Configure sampling
	sampler := trace.TraceIDRatioBased(config.SamplingRatio)

	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(sampler),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)

	// Create tracer
	tracer := tracerProvider.Tracer(config.ServiceName)

	return &OTelTracer{
		tracer:      tracer,
		provider:    tracerProvider,
		serviceName: config.ServiceName,
	}, nil
}

// StartSpan starts a new OpenTelemetry span
func (t *OTelTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	if t.tracer == nil {
		// Fallback to console tracer
		consoleTracer := NewConsoleTracer()
		return consoleTracer.StartSpan(ctx, name)
	}

	otelCtx, otelSpan := t.tracer.Start(ctx, name)
	span := &OTelSpan{
		span:    otelSpan,
		context: otelCtx,
	}

	return otelCtx, span
}

// EndSpan ends the given span
func (t *OTelTracer) EndSpan(span Span) {
	if otelSpan, ok := span.(*OTelSpan); ok {
		otelSpan.span.End()
	}
}

// Shutdown gracefully shuts down the tracer provider
func (t *OTelTracer) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// OTelSpan implements the Span interface using OpenTelemetry
type OTelSpan struct {
	span    oteltrace.Span
	context context.Context
}

// SetAttribute sets an attribute on the OpenTelemetry span
func (s *OTelSpan) SetAttribute(key string, value interface{}) {
	attr := convertToAttribute(key, value)
	s.span.SetAttributes(attr)
}

// SetError marks the span as having an error
func (s *OTelSpan) SetError(err error) {
	if err != nil {
		s.span.SetStatus(codes.Error, err.Error())
		s.span.RecordError(err)
	}
}

// End ends the span
func (s *OTelSpan) End() {
	s.span.End()
}

// Context returns the span's context
func (s *OTelSpan) Context() context.Context {
	return s.context
}

// convertToAttribute converts a Go value to an OpenTelemetry attribute
func convertToAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	case time.Duration:
		return attribute.String(key, v.String())
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

// Enhanced span creation functions

// StartProviderSpan starts a span for provider operations
func (t *OTelTracer) StartProviderSpan(ctx context.Context, provider, operation string) (context.Context, Span) {
	spanName := fmt.Sprintf("provider.%s.%s", provider, operation)
	ctx, span := t.StartSpan(ctx, spanName)
	
	span.SetAttribute("provider.name", provider)
	span.SetAttribute("provider.operation", operation)
	span.SetAttribute("component", "provider")
	
	return ctx, span
}

// StartAgentSpan starts a span for agent operations
func (t *OTelTracer) StartAgentSpan(ctx context.Context, agentName, operation string) (context.Context, Span) {
	spanName := fmt.Sprintf("agent.%s.%s", agentName, operation)
	ctx, span := t.StartSpan(ctx, spanName)
	
	span.SetAttribute("agent.name", agentName)
	span.SetAttribute("agent.operation", operation)
	span.SetAttribute("component", "agent")
	
	return ctx, span
}

// StartToolSpan starts a span for tool operations
func (t *OTelTracer) StartToolSpan(ctx context.Context, toolName string) (context.Context, Span) {
	spanName := fmt.Sprintf("tool.%s", toolName)
	ctx, span := t.StartSpan(ctx, spanName)
	
	span.SetAttribute("tool.name", toolName)
	span.SetAttribute("component", "tool")
	
	return ctx, span
}

// Helper functions for common span attributes

// AddProviderAttributes adds common provider attributes to a span
func AddProviderAttributes(span Span, provider, model string, tokenUsage map[string]int) {
	span.SetAttribute("provider.name", provider)
	span.SetAttribute("provider.model", model)
	
	if tokenUsage != nil {
		if promptTokens, ok := tokenUsage["prompt"]; ok {
			span.SetAttribute("provider.tokens.prompt", promptTokens)
		}
		if completionTokens, ok := tokenUsage["completion"]; ok {
			span.SetAttribute("provider.tokens.completion", completionTokens)
		}
		if totalTokens, ok := tokenUsage["total"]; ok {
			span.SetAttribute("provider.tokens.total", totalTokens)
		}
	}
}

// AddAgentAttributes adds common agent attributes to a span
func AddAgentAttributes(span Span, agentName string, messageCount, toolCount int) {
	span.SetAttribute("agent.name", agentName)
	span.SetAttribute("agent.messages.count", messageCount)
	span.SetAttribute("agent.tools.count", toolCount)
}

// AddRequestAttributes adds common request attributes to a span
func AddRequestAttributes(span Span, requestID string, duration time.Duration) {
	span.SetAttribute("request.id", requestID)
	span.SetAttribute("request.duration_ms", float64(duration.Nanoseconds())/1e6)
}

// Custom errors for tracing
var (
	ErrUnsupportedExporter = fmt.Errorf("unsupported trace exporter type")
	ErrTracerNotInitialized = fmt.Errorf("tracer not initialized")
)