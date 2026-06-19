package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
	// Block redirects to private/loopback/link-local hosts too, since the URL is
	// model-controlled and could otherwise reach internal services or cloud
	// metadata endpoints (SSRF).
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return validatePublicURL(req.URL.String())
		},
	}
	return tools.NewTool(
		"http_get",
		"Performs an HTTP GET request to a public URL and returns the response body (truncated to 8KB).",
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			raw, _ := args["url"].(string)
			if raw == "" {
				return nil, fmt.Errorf("the 'url' argument is required")
			}
			if err := validatePublicURL(raw); err != nil {
				return nil, err
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
			if err != nil {
				return nil, err
			}
			resp, err := client.Do(req)
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
		tools.Param{Name: "url", Type: "string", Description: "The public http(s) URL to fetch.", Required: true},
	)
}

// validatePublicURL rejects non-http(s) schemes and URLs whose host resolves to
// a private, loopback, link-local, or unspecified address.
func validatePublicURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported url scheme %q (only http/https)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url has no host")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve host %q: %w", host, err)
	}
	for _, ip := range ips {
		if isDisallowedIP(ip) {
			return fmt.Errorf("host %q resolves to a disallowed address %s", host, ip)
		}
	}
	return nil
}

func isDisallowedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified()
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
