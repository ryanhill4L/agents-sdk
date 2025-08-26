# Agents SDK - Observability Example

This example demonstrates comprehensive observability features of the Agents SDK, including structured logging, distributed tracing, and monitoring integration.

## Features Demonstrated

- **Structured Logging**: JSON and console logging formats with configurable levels
- **Distributed Tracing**: OpenTelemetry integration with Jaeger, Zipkin, and OTLP exporters
- **Request Correlation**: Request IDs and trace correlation across components
- **Performance Monitoring**: Token usage, latency, and execution metrics
- **Error Tracking**: Contextual error logging and trace error propagation
- **Environment Configuration**: Runtime configuration via environment variables

## Quick Start

### 1. Basic Example (Console Logging/Tracing)

```bash
# Run with console output
go run main.go
```

### 2. With Jaeger Tracing

```bash
# Start Jaeger
docker-compose up -d jaeger

# Set environment variables
export AGENTS_TRACE_ENABLED=true
export AGENTS_TRACE_EXPORTER=jaeger
export JAEGER_ENDPOINT=http://localhost:14268/api/traces
export AGENTS_LOG_LEVEL=debug
export AGENTS_VERBOSE=true

# Run the example
go run main.go

# View traces in Jaeger UI
open http://localhost:16686
```

### 3. Full Observability Stack

```bash
# Start all services (Jaeger, OTEL Collector, Prometheus, Grafana)
docker-compose --profile advanced up -d

# Configure for OTLP export
export AGENTS_TRACE_ENABLED=true
export AGENTS_TRACE_EXPORTER=otlp
export AGENTS_TRACE_ENDPOINT=http://localhost:4318/v1/traces
export AGENTS_LOG_FORMAT=json
export AGENTS_LOG_LEVEL=debug

# Run the example
go run main.go

# Access UIs
open http://localhost:16686  # Jaeger
open http://localhost:3000   # Grafana (admin/admin)
open http://localhost:9090   # Prometheus
```

## Environment Variables

### Logging Configuration

| Variable | Description | Default | Options |
|----------|-------------|---------|---------|
| `AGENTS_LOG_LEVEL` | Minimum log level | `info` | `debug`, `info`, `warn`, `error` |
| `AGENTS_VERBOSE` | Enable verbose logging | `false` | `true`, `false` |
| `AGENTS_LOG_FORMAT` | Log output format | `console` | `console`, `json` |

### Tracing Configuration

| Variable | Description | Default | Options |
|----------|-------------|---------|---------|
| `AGENTS_TRACE_ENABLED` | Enable tracing | `false` | `true`, `false` |
| `AGENTS_TRACE_EXPORTER` | Trace exporter type | `console` | `console`, `jaeger`, `zipkin`, `otlp` |
| `AGENTS_TRACE_ENDPOINT` | Trace collector endpoint | *(varies)* | URL string |
| `AGENTS_TRACE_SAMPLING_RATIO` | Trace sampling ratio | `1.0` | `0.0` - `1.0` |
| `AGENTS_SERVICE_NAME` | Service name for traces | `agents-sdk` | Any string |
| `AGENTS_SERVICE_VERSION` | Service version | `1.0.0` | Any string |
| `AGENTS_ENVIRONMENT` | Environment name | `development` | Any string |

### Advanced Tracing Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `AGENTS_TRACE_BATCH_TIMEOUT` | Batch export timeout | `5s` |
| `AGENTS_TRACE_MAX_BATCH_SIZE` | Max spans per batch | `512` |
| `AGENTS_TRACE_MAX_QUEUE_SIZE` | Max queued spans | `2048` |

## Docker Services

### Jaeger (Port 16686)
- **Purpose**: Distributed tracing visualization
- **Access**: http://localhost:16686
- **Features**: Trace search, service map, performance analysis

### OpenTelemetry Collector (Ports 4317/4318)
- **Purpose**: Trace and metrics collection/processing
- **Features**: Protocol translation, batching, filtering
- **Config**: `otel-collector-config.yaml`

### Prometheus (Port 9090)
- **Purpose**: Metrics collection and storage
- **Access**: http://localhost:9090
- **Features**: Time series metrics, alerting

### Grafana (Port 3000)
- **Purpose**: Metrics visualization and dashboards
- **Access**: http://localhost:3000 (admin/admin)
- **Features**: Custom dashboards, alerting

## Example Output

### Console Logging
```
[2024-01-15 10:30:15.123] INFO: Starting OpenAI completion request [provider=openai request_id=abc-123]
[2024-01-15 10:30:15.456] INFO: OpenAI API request completed [duration=333ms tokens=45]
```

### JSON Logging
```json
{
  "timestamp": "2024-01-15T10:30:15.123Z",
  "level": "INFO",
  "message": "Starting OpenAI completion request",
  "provider": "openai",
  "request_id": "abc-123",
  "trace_id": "def-456"
}
```

### Trace Spans
- `agent.demo_agent.run` - Overall agent execution
- `provider.openai.complete` - LLM API call
- `tool.add` - Tool execution
- `tool.get_current_time` - Tool execution

## Monitoring Patterns

### Request Flow Tracing
```
agent.run
├── provider.openai.complete (333ms, 45 tokens)
├── tool.add (2ms)
└── tool.get_current_time (1ms)
```

### Key Metrics
- **Request Duration**: Time from start to completion
- **Token Usage**: Prompt, completion, and total tokens
- **Tool Calls**: Number and duration of tool executions
- **Error Rate**: Failed requests and error types
- **Throughput**: Requests per second

## Best Practices

### Production Configuration
```bash
# Optimized for production
export AGENTS_LOG_LEVEL=warn
export AGENTS_VERBOSE=false
export AGENTS_LOG_FORMAT=json
export AGENTS_TRACE_ENABLED=true
export AGENTS_TRACE_EXPORTER=otlp
export AGENTS_TRACE_SAMPLING_RATIO=0.1  # Sample 10%
```

### Development Configuration
```bash
# Optimized for debugging
export AGENTS_LOG_LEVEL=debug
export AGENTS_VERBOSE=true
export AGENTS_LOG_FORMAT=console
export AGENTS_TRACE_ENABLED=true
export AGENTS_TRACE_EXPORTER=console
export AGENTS_TRACE_SAMPLING_RATIO=1.0  # Sample 100%
```

## Troubleshooting

### Common Issues

1. **Jaeger Not Receiving Traces**
   - Check if Jaeger is running: `docker ps`
   - Verify endpoint configuration
   - Check firewall/network settings

2. **High Memory Usage**
   - Reduce sampling ratio
   - Increase batch timeout
   - Check for trace leaks

3. **Missing Logs**
   - Verify log level configuration
   - Check if logger is properly initialized
   - Ensure context propagation

### Debugging Commands

```bash
# Check container status
docker-compose ps

# View container logs
docker-compose logs jaeger
docker-compose logs otel-collector

# Test endpoints
curl http://localhost:16686/api/services  # Jaeger
curl http://localhost:4318/v1/traces     # OTLP endpoint
```

## Integration Examples

### Custom Logger Integration
```go
logger := logging.NewStructuredLogger(logging.InfoLevel)
config := providers.NewOpenAIConfig("your-key")
config.Logger = logger
config.Verbose = true
```

### Custom Tracer Integration
```go
tracerConfig := &tracing.OTelConfig{
    ServiceName:    "my-service",
    ExporterType:   tracing.ExporterJaeger,
    Endpoint:       "http://jaeger:14268/api/traces",
}
tracing.InitGlobalTracerWithConfig(tracerConfig)
```

## Performance Considerations

- **Sampling**: Use lower sampling ratios in production
- **Batching**: Configure appropriate batch sizes for your load
- **Retention**: Set appropriate retention policies
- **Network**: Consider local collectors for high-volume scenarios

## Security Notes

- Ensure trace endpoints are secured in production
- Avoid logging sensitive data (API keys, user data)
- Use proper authentication for observability services
- Consider data residency requirements for traces/logs