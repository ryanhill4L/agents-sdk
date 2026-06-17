package mcp

import (
	"context"
	"fmt"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type echoIn struct {
	Message string `json:"message" jsonschema:"the message to echo"`
}

// startTestServer spins up an in-memory MCP server exposing an "echo" tool and
// returns a connected Client.
func startTestServer(t *testing.T, cfg ServerConfig) *Client {
	t.Helper()

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.0.1"}, nil)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "echo", Description: "Echoes the message"},
		func(_ context.Context, _ *mcpsdk.CallToolRequest, in echoIn) (*mcpsdk.CallToolResult, any, error) {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "echo: " + in.Message}},
			}, nil, nil
		})

	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client, err := connectTransport(context.Background(), cfg, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestMCPToolAdaptation(t *testing.T) {
	client := startTestServer(t, ServerConfig{Name: "demo"})

	tls := client.Tools()
	if len(tls) != 1 {
		t.Fatalf("got %d tools, want 1", len(tls))
	}
	tool := tls[0]

	if tool.Name() != "demo_echo" {
		t.Errorf("tool name = %q, want demo_echo (namespaced)", tool.Name())
	}
	if tool.Description() != "Echoes the message" {
		t.Errorf("description = %q", tool.Description())
	}
	if _, ok := tool.Schema().Properties["message"]; !ok {
		t.Errorf("schema missing 'message' property: %+v", tool.Schema())
	}

	out, err := tool.Execute(context.Background(), map[string]interface{}{"message": "hi"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if fmt.Sprintf("%v", out) != "echo: hi" {
		t.Errorf("output = %v, want 'echo: hi'", out)
	}
}

func TestMCPToolAllowlist(t *testing.T) {
	// Restrict to a tool that does not exist -> no tools exposed.
	client := startTestServer(t, ServerConfig{Name: "demo", Tools: []string{"nonexistent"}})
	if len(client.Tools()) != 0 {
		t.Errorf("expected allowlist to exclude all tools, got %d", len(client.Tools()))
	}
}

func TestTransportValidation(t *testing.T) {
	if _, err := (ServerConfig{Name: "x", Transport: "stdio"}).transport(); err == nil {
		t.Error("expected error for stdio with no command")
	}
	if _, err := (ServerConfig{Name: "x", Transport: "http"}).transport(); err == nil {
		t.Error("expected error for http with no url")
	}
	if _, err := (ServerConfig{Name: "x", Transport: "carrier-pigeon"}).transport(); err == nil {
		t.Error("expected error for unknown transport")
	}
}

func TestNamespaced(t *testing.T) {
	if got := namespaced("git hub", "create.issue"); got != "git_hub_create.issue" {
		t.Errorf("namespaced = %q", got)
	}
	if got := namespaced("", "tool"); got != "tool" {
		t.Errorf("namespaced empty server = %q, want 'tool'", got)
	}
}
