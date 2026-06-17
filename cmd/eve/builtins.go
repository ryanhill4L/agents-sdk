package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

// builtinRegistry returns the tools the eve CLI can wire into a filesystem
// agent by name. Projects that need custom Go tools should use the library
// API (loader.Load with their own *tools.Registry) instead of the CLI.
func builtinRegistry() *tools.Registry {
	r := tools.NewRegistry()
	r.MustRegister(
		currentTimeTool(),
		addTool(),
		httpGetTool(),
	)
	return r
}

func currentTimeTool() tools.Tool {
	return tools.NewTool(
		"current_time",
		"Returns the current date and time in RFC3339 format (UTC).",
		func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return time.Now().UTC().Format(time.RFC3339), nil
		},
	)
}

func addTool() tools.Tool {
	return tools.NewTool(
		"add",
		"Adds two numbers and returns their sum.",
		func(_ context.Context, args map[string]interface{}) (interface{}, error) {
			a, err := asFloat(args["a"])
			if err != nil {
				return nil, fmt.Errorf("a: %w", err)
			}
			b, err := asFloat(args["b"])
			if err != nil {
				return nil, fmt.Errorf("b: %w", err)
			}
			return a + b, nil
		},
		tools.Param{Name: "a", Type: "number", Description: "First addend.", Required: true},
		tools.Param{Name: "b", Type: "number", Description: "Second addend.", Required: true},
	)
}

func httpGetTool() tools.Tool {
	return tools.NewTool(
		"http_get",
		"Performs an HTTP GET request and returns the response body (truncated to 8KB).",
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			url, _ := args["url"].(string)
			if url == "" {
				return nil, fmt.Errorf("the 'url' argument is required")
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return nil, err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
			if err != nil {
				return nil, err
			}
			return string(body), nil
		},
		tools.Param{Name: "url", Type: "string", Description: "The URL to fetch.", Required: true},
	)
}

func asFloat(v interface{}) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("expected a number, got %T", v)
	}
}
