package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/logging"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// Example tools for demonstration
func add(a, b int) int {
	return a + b
}

func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func main() {
	fmt.Println("🔍 Agents SDK - Comprehensive Observability Example")
	fmt.Println("==================================================")

	// Initialize logging
	logger := logging.NewStructuredLogger(logging.DebugLevel)
	logger.Info(context.Background(), "Starting observability example")

	// Initialize tracing
	tracerConfig := &tracing.OTelConfig{
		ServiceName:     "agents-sdk-example",
		ServiceVersion:  "1.0.0",
		Environment:     "demo",
		ExporterType:    tracing.ExporterConsole, // Change to ExporterJaeger if Jaeger is running
		Endpoint:        "http://localhost:14268/api/traces",
		SamplingRatio:   1.0,
		BatchTimeout:    5 * time.Second,
		MaxExportBatch:  512,
		MaxQueueSize:    2048,
	}

	// Check if Jaeger endpoint is available
	if jaegerEndpoint := os.Getenv("JAEGER_ENDPOINT"); jaegerEndpoint != "" {
		tracerConfig.ExporterType = tracing.ExporterJaeger
		tracerConfig.Endpoint = jaegerEndpoint
	}

	// Initialize global tracer
	err := tracing.InitGlobalTracerWithConfig(tracerConfig)
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	}

	// Ensure tracer shutdown at the end
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := tracing.ShutdownGlobalTracer(ctx); err != nil {
			log.Printf("Error shutting down tracer: %v", err)
		}
	}()

	// Create tools with logging
	logger.Info(context.Background(), "Creating tools")
	addTool, err := tools.NewFunctionTool("add", 
		"Performs addition of two integer numbers. Use this for mathematical calculations requiring sum of two values.", 
		add)
	if err != nil {
		log.Fatal("Failed to create add tool:", err)
	}

	timeTool, err := tools.NewFunctionTool("get_current_time", 
		"Returns the current date and time. Use this when the user asks for the current time or date.", 
		getCurrentTime)
	if err != nil {
		log.Fatal("Failed to create time tool:", err)
	}

	// Create agent with observability
	logger.Info(context.Background(), "Creating agent")
	agent := agents.NewAgent("Observability Demo Agent",
		agents.WithInstructions(`You are a helpful assistant that can perform mathematical calculations and provide current time information.

Available Tools:
- add: Use for adding two numbers together
- get_current_time: Use when asked about current time or date

Process:
1. Analyze the user's request
2. Use appropriate tools if needed
3. Provide clear, helpful responses`),
		agents.WithModel("gpt-3.5-turbo"),
		agents.WithTools(addTool, timeTool),
		agents.WithTemperature(0.7),
	)

	if err := agent.Validate(); err != nil {
		log.Fatal("Agent validation failed:", err)
	}

	// Demo different logging levels and verbose mode
	fmt.Println("\n📊 Testing Different Logging and Tracing Configurations")
	fmt.Println("========================================================")

	configurations := []struct {
		name    string
		level   logging.LogLevel
		verbose bool
		config  *providers.OpenAIConfig
	}{
		{
			name:    "Info Level - Non-Verbose",
			level:   logging.InfoLevel,
			verbose: false,
			config:  createProviderConfig(logging.InfoLevel, false, logger),
		},
		{
			name:    "Debug Level - Verbose",
			level:   logging.DebugLevel,
			verbose: true,
			config:  createProviderConfig(logging.DebugLevel, true, logger),
		},
	}

	testInput := "Hello! Can you add 15 and 27 for me, and also tell me what time it is?"

	for i, cfg := range configurations {
		fmt.Printf("\n🧪 Test %d: %s\n", i+1, cfg.name)
		fmt.Println("----------------------------------------")

		// Create provider with specific configuration
		provider, err := providers.NewOpenAIProvider(cfg.config)
		if err != nil {
			log.Printf("Failed to create provider for %s: %v", cfg.name, err)
			continue
		}

		// Create runner with tracing
		runner := agents.NewRunner(
			agents.WithProvider(provider),
			agents.WithTracer(tracing.GetGlobalTracer()),
			agents.WithMaxTurns(3),
		)

		// Execute with distributed tracing
		ctx := context.Background()
		ctx = logging.WithNewRequestID(ctx)
		
		result, err := tracing.TraceAgentOperation(ctx, agent.GetName(), "complete_request", func(ctx context.Context) (*agents.RunResult, error) {
			return runner.Run(ctx, agent, testInput)
		})

		if err != nil {
			logger.Error(ctx, "Agent execution failed", 
				logging.Error(err),
				logging.String("config", cfg.name))
			continue
		}

		// Display results
		fmt.Printf("📋 Agent: %s\n", agent.GetName())
		fmt.Printf("📊 Configuration: %s\n", cfg.name)
		fmt.Printf("💬 Response: %s\n", result.FinalOutput)
		fmt.Printf("📈 Metrics:\n")
		fmt.Printf("   - Total Turns: %d\n", result.Metrics.TotalTurns)
		fmt.Printf("   - Total Tokens: %d\n", result.Metrics.TotalTokens)
		fmt.Printf("   - Duration: %v\n", result.Metrics.Duration)
		fmt.Printf("   - Tool Calls: %d\n", result.Metrics.ToolCalls)

		// Small delay between tests
		time.Sleep(1 * time.Second)
	}

	// Demonstrate error logging and tracing
	fmt.Println("\n🚨 Error Handling and Logging Demo")
	fmt.Println("===================================")

	ctx := context.Background()
	ctx = logging.WithNewRequestID(ctx)

	err = tracing.TraceOperation(ctx, "error_demo", func(ctx context.Context) error {
		logger.Error(ctx, "Simulated error for demonstration", 
			logging.String("error_type", "demonstration"),
			logging.String("component", "example"))
		return fmt.Errorf("this is a demonstration error")
	})

	if err != nil {
		logger.Warn(ctx, "Expected error occurred in demo", logging.Error(err))
	}

	// Environment variable configuration demo
	fmt.Println("\n🌍 Environment Variable Configuration")
	fmt.Println("====================================")

	envVars := []string{
		"AGENTS_LOG_LEVEL",
		"AGENTS_VERBOSE", 
		"AGENTS_LOG_FORMAT",
		"AGENTS_TRACE_ENABLED",
		"AGENTS_TRACE_EXPORTER",
		"AGENTS_TRACE_ENDPOINT",
		"AGENTS_SERVICE_NAME",
	}

	fmt.Println("Current environment configuration:")
	for _, envVar := range envVars {
		value := os.Getenv(envVar)
		if value == "" {
			value = "(not set)"
		}
		fmt.Printf("  %s=%s\n", envVar, value)
	}

	fmt.Println("\n📝 Available Environment Variables:")
	fmt.Println("  AGENTS_LOG_LEVEL=debug|info|warn|error")
	fmt.Println("  AGENTS_VERBOSE=true|false")
	fmt.Println("  AGENTS_LOG_FORMAT=json|console")
	fmt.Println("  AGENTS_TRACE_ENABLED=true|false")
	fmt.Println("  AGENTS_TRACE_EXPORTER=console|jaeger|zipkin|otlp")
	fmt.Println("  AGENTS_TRACE_ENDPOINT=<trace_collector_endpoint>")
	fmt.Println("  AGENTS_SERVICE_NAME=<your_service_name>")

	// Jaeger setup instructions
	fmt.Println("\n🐳 Jaeger Setup Instructions")
	fmt.Println("============================")
	fmt.Println("To run with Jaeger tracing:")
	fmt.Println("1. Start Jaeger with Docker:")
	fmt.Println("   docker run -d --name jaeger \\")
	fmt.Println("     -p 14268:14268 \\")
	fmt.Println("     -p 16686:16686 \\")
	fmt.Println("     jaegertracing/all-in-one:latest")
	fmt.Println("")
	fmt.Println("2. Set environment variables:")
	fmt.Println("   export AGENTS_TRACE_ENABLED=true")
	fmt.Println("   export AGENTS_TRACE_EXPORTER=jaeger")
	fmt.Println("   export JAEGER_ENDPOINT=http://localhost:14268/api/traces")
	fmt.Println("")
	fmt.Println("3. Access Jaeger UI at: http://localhost:16686")
	
	logger.Info(context.Background(), "Observability example completed successfully")
	fmt.Println("\n✅ Observability demonstration completed!")
	fmt.Println("💡 Check the logs above to see different logging levels and formats in action.")
	fmt.Println("🔍 If Jaeger is running, check the UI for distributed traces.")
}

// createProviderConfig creates a provider configuration with logging settings
func createProviderConfig(level logging.LogLevel, verbose bool, logger logging.Logger) *providers.OpenAIConfig {
	config := providers.NewOpenAIConfig("")
	config.LogLevel = level
	config.Verbose = verbose
	config.Logger = logger.WithLevel(level)
	
	// Use placeholder API key for demo
	if config.APIKey == "" {
		config.APIKey = "sk-demo-key-for-observability-example"
	}
	
	return config
}