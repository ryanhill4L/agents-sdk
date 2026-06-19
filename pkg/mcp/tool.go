package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

// mcpTool adapts a single MCP server tool to the tools.Tool interface.
type mcpTool struct {
	session     *mcpsdk.ClientSession
	rawName     string // name on the server, used for tools/call
	name        string // namespaced name exposed to the model
	description string
	schema      tools.ParameterSchema
}

func (t *mcpTool) Name() string                  { return t.name }
func (t *mcpTool) Description() string           { return t.description }
func (t *mcpTool) Schema() tools.ParameterSchema { return t.schema }

func (t *mcpTool) Validate() error {
	if t.session == nil {
		return fmt.Errorf("mcp tool %q has no session", t.name)
	}
	if t.rawName == "" {
		return fmt.Errorf("mcp tool is missing a name")
	}
	return nil
}

// Execute invokes the tool on the server and returns its text content.
func (t *mcpTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	res, err := t.session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      t.rawName,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("mcp tool %q: %w", t.name, err)
	}

	text := textContent(res)
	if res.IsError {
		if text == "" {
			text = "tool call failed"
		}
		return nil, fmt.Errorf("mcp tool %q: %s", t.name, text)
	}

	// Prefer structured content when present; otherwise return the text.
	if res.StructuredContent != nil {
		return res.StructuredContent, nil
	}
	return text, nil
}

func textContent(res *mcpsdk.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// Manager holds connections to multiple MCP servers and aggregates their tools.
type Manager struct {
	clients []*Client
}

// ConnectAll connects to every server in cfgs. If any connection fails, all
// already-opened connections are closed and the error is returned. A nil logger
// disables logging.
func ConnectAll(ctx context.Context, cfgs []ServerConfig, logger *slog.Logger) (*Manager, error) {
	m := &Manager{}
	for _, cfg := range cfgs {
		client, err := Connect(ctx, cfg, logger)
		if err != nil {
			_ = m.Close()
			return nil, err
		}
		m.clients = append(m.clients, client)
	}
	return m, nil
}

// Tools returns the combined tools from every connected server.
func (m *Manager) Tools() []tools.Tool {
	var all []tools.Tool
	for _, c := range m.clients {
		all = append(all, c.Tools()...)
	}
	return all
}

// Close closes all server connections, returning the first error encountered.
func (m *Manager) Close() error {
	var firstErr error
	for _, c := range m.clients {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
