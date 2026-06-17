// Package mcp connects an agent to Model Context Protocol (MCP) servers and
// exposes their tools through the SDK's tools.Tool interface. Servers are
// declared in configuration (never chosen by the model) and connected at load
// time over either the stdio or streamable-HTTP transport, using the official
// github.com/modelcontextprotocol/go-sdk client.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ryanhill4L/agents-sdk/pkg/tools"
	"github.com/ryanhill4L/agents-sdk/pkg/tracing"
)

// Transport names.
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

// connectTimeout bounds the initialization handshake when the caller's context
// has no deadline, so a misbehaving server can't hang loading forever.
const connectTimeout = 30 * time.Second

// ServerConfig declares a single MCP server.
type ServerConfig struct {
	// Name namespaces the server's tools (exposed as "<name>_<tool>").
	Name string `yaml:"name"`
	// Transport is "stdio" or "http".
	Transport string `yaml:"transport"`

	// Command is the subprocess to run for the stdio transport, e.g.
	// ["npx", "-y", "@modelcontextprotocol/server-github"].
	Command []string `yaml:"command"`
	// Env sets extra environment variables for a stdio server (in addition to
	// the inherited process environment). Values are expanded with $VAR syntax.
	Env map[string]string `yaml:"env"`

	// URL is the endpoint for the http (streamable-HTTP) transport.
	URL string `yaml:"url"`
	// Headers are sent on every HTTP request. Values are expanded with $VAR.
	Headers map[string]string `yaml:"headers"`

	// Tools optionally restricts which of the server's tools are exposed
	// (by raw tool name). Empty means expose all.
	Tools []string `yaml:"tools"`
}

// Client is a connected MCP session and its adapted tools.
type Client struct {
	cfg     ServerConfig
	session *mcpsdk.ClientSession
	tools   []tools.Tool
	logger  *slog.Logger
}

// Connect dials the server described by cfg and lists its tools. A nil logger
// disables logging.
func Connect(ctx context.Context, cfg ServerConfig, logger *slog.Logger) (*Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.Name == "" {
		return nil, fmt.Errorf("mcp server is missing a name")
	}

	transport, err := cfg.transport()
	if err != nil {
		return nil, fmt.Errorf("mcp server %q: %w", cfg.Name, err)
	}
	return connectTransport(ctx, cfg, transport, logger)
}

// connectTransport connects a client over an arbitrary transport. It is the
// shared core of Connect and is used by tests with in-memory transports.
func connectTransport(ctx context.Context, cfg ServerConfig, transport mcpsdk.Transport, logger *slog.Logger) (*Client, error) {
	log := tracing.OrDiscard(logger)

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, connectTimeout)
		defer cancel()
	}

	log.Debug("mcp connecting", "server", cfg.Name, "transport", cfg.Transport)
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "agents-sdk", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp server %q: connect: %w", cfg.Name, err)
	}

	c := &Client{cfg: cfg, session: session, logger: log}
	if err := c.loadTools(ctx); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("mcp server %q: %w", cfg.Name, err)
	}
	log.Info("mcp connected", "server", cfg.Name, "tools", len(c.tools))
	return c, nil
}

// Tools returns the server's tools adapted to the tools.Tool interface.
func (c *Client) Tools() []tools.Tool { return c.tools }

// Close terminates the session (and, for stdio, the subprocess).
func (c *Client) Close() error {
	if c.session == nil {
		return nil
	}
	return c.session.Close()
}

func (c *Client) loadTools(ctx context.Context) error {
	allow := map[string]bool{}
	for _, name := range c.cfg.Tools {
		allow[name] = true
	}

	res, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("listing tools: %w", err)
	}

	for _, t := range res.Tools {
		if len(allow) > 0 && !allow[t.Name] {
			continue
		}
		name := namespaced(c.cfg.Name, t.Name)
		c.tools = append(c.tools, &mcpTool{
			session:     c.session,
			rawName:     t.Name,
			name:        name,
			description: t.Description,
			schema:      convertSchema(t.InputSchema),
		})
		if c.logger != nil {
			c.logger.Debug("mcp tool discovered", "server", c.cfg.Name, "tool", name)
		}
	}
	return nil
}

// transport builds the SDK transport for this server config.
func (cfg ServerConfig) transport() (mcpsdk.Transport, error) {
	switch strings.ToLower(cfg.Transport) {
	case TransportStdio, "":
		if len(cfg.Command) == 0 {
			return nil, fmt.Errorf("stdio transport requires a command")
		}
		cmd := exec.Command(cfg.Command[0], cfg.Command[1:]...)
		cmd.Env = os.Environ()
		for k, v := range cfg.Env {
			cmd.Env = append(cmd.Env, k+"="+os.ExpandEnv(v))
		}
		return &mcpsdk.CommandTransport{Command: cmd}, nil

	case TransportHTTP:
		if cfg.URL == "" {
			return nil, fmt.Errorf("http transport requires a url")
		}
		return &mcpsdk.StreamableClientTransport{
			Endpoint:   cfg.URL,
			HTTPClient: httpClient(cfg.Headers),
		}, nil

	default:
		return nil, fmt.Errorf("unknown transport %q (use stdio or http)", cfg.Transport)
	}
}

func httpClient(headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return nil // SDK uses its default client
	}
	expanded := make(map[string]string, len(headers))
	for k, v := range headers {
		expanded[k] = os.ExpandEnv(v)
	}
	return &http.Client{Transport: &headerRoundTripper{base: http.DefaultTransport, headers: expanded}}
}

type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	return h.base.RoundTrip(req)
}

// namespaced prefixes a tool name with its server, sanitized to characters
// accepted by provider tool-name schemas.
func namespaced(server, tool string) string {
	if server == "" {
		return tool
	}
	return sanitize(server) + "_" + tool
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// convertSchema maps an MCP JSON-schema input into our ParameterSchema.
//
// TODO: this carries only per-property type/description; nested object
// properties, array `items`, and `enum` constraints are dropped (a limitation
// shared with tools.PropertySchema). Tools with rich array/object parameters
// will be under-specified to the model until the schema type is enriched.
func convertSchema(input any) tools.ParameterSchema {
	schema := tools.ParameterSchema{
		Type:       "object",
		Properties: map[string]tools.PropertySchema{},
		Required:   []string{},
	}
	if input == nil {
		return schema
	}

	raw, err := json.Marshal(input)
	if err != nil {
		return schema
	}
	var parsed struct {
		Type       string `json:"type"`
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return schema
	}

	if parsed.Type != "" {
		schema.Type = parsed.Type
	}
	for name, p := range parsed.Properties {
		typ := p.Type
		if typ == "" {
			typ = "string"
		}
		schema.Properties[name] = tools.PropertySchema{Type: typ, Description: p.Description}
	}
	if parsed.Required != nil {
		schema.Required = parsed.Required
	}
	return schema
}
