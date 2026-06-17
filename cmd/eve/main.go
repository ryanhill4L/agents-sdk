// Command eve is the CLI for the Eve-inspired, filesystem-first agent framework.
//
//	eve init <dir>        scaffold a new agent directory
//	eve validate <dir>    load an agent and report what was found
//	eve run <dir> [input] run the agent once and print its reply
//	eve dev <dir>         serve the agent over HTTP (POST /chat)
//	eve schedules <dir>   run the agent's cron schedules
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/channels"
	"github.com/ryanhill4L/agents-sdk/pkg/loader"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/schedules"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "init":
		err = cmdInit(args)
	case "validate":
		err = cmdValidate(args)
	case "run":
		err = cmdRun(args)
	case "dev", "serve":
		err = cmdDev(args)
	case "schedules":
		err = cmdSchedules(args)
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `eve — filesystem-first AI agents in Go

Usage:
  eve init <dir>          Scaffold a new agent directory
  eve validate <dir>      Load an agent and report what was found
  eve run <dir> [input]   Run the agent once (reads stdin if input omitted)
  eve dev <dir> [addr]    Serve the agent over HTTP (default :8080)
  eve schedules <dir>     Run the agent's cron schedules

Environment:
  PROVIDER                openai | anthropic | gemini | ollama (else auto-detect)
  OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY, OLLAMA_HOST
`)
}

// loadAgent loads the agent at dir using the CLI's builtin tool registry,
// connecting any declared MCP servers. Callers should defer agent.Close().
func loadAgent(dir string) (*agents.Agent, error) {
	return loader.Load(dir, builtinRegistry())
}

// newRunner builds a Runner whose provider is resolved from the agent's
// declared provider (or the environment).
func newRunner(agent *agents.Agent) (*agents.Runner, error) {
	provider, err := providers.Resolve(agent.Provider)
	if err != nil {
		return nil, err
	}
	return agents.NewRunner(agents.WithProvider(provider)), nil
}

func cmdValidate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: eve validate <dir>")
	}
	// Skip connecting to MCP servers so validate stays offline.
	agent, err := loader.LoadWithOptions(args[0], builtinRegistry(), loader.Options{SkipMCP: true})
	if err != nil {
		return err
	}
	fmt.Printf("✓ agent %q is valid\n", agent.Name)
	fmt.Printf("  model:     %s\n", orNone(agent.Model))
	fmt.Printf("  provider:  %s\n", orNone(agent.Provider))
	fmt.Printf("  tools:     %s\n", orNone(toolNames(agent)))
	fmt.Printf("  skills:    %s\n", orNone(skillNames(agent)))
	fmt.Printf("  subagents: %s\n", orNone(handoffNames(agent)))
	fmt.Printf("  mcp:       %s\n", orNone(strings.Join(agent.MCPServers, ", ")))
	return nil
}

func cmdRun(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: eve run <dir> [input]")
	}
	agent, err := loadAgent(args[0])
	if err != nil {
		return err
	}
	defer agent.Close()

	input := strings.Join(args[1:], " ")
	if input == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		input = strings.TrimSpace(string(data))
	}
	if input == "" {
		return fmt.Errorf("no input provided")
	}

	runner, err := newRunner(agent)
	if err != nil {
		return err
	}

	result, err := runner.Run(context.Background(), agent, input)
	if err != nil {
		return err
	}
	fmt.Println(result.FinalOutput)
	return nil
}

func cmdDev(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: eve dev <dir> [addr]")
	}
	addr := ":8080"
	if len(args) > 1 {
		addr = args[1]
	}

	agent, err := loadAgent(args[0])
	if err != nil {
		return err
	}
	defer agent.Close()

	runner, err := newRunner(agent)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	channel := channels.NewHTTPChannel(addr)
	fmt.Printf("eve dev: serving agent %q on %s\n", agent.Name, addr)
	fmt.Printf("  curl -s localhost%s/chat -d '{\"input\":\"hello\"}'\n", addr)

	return channel.Start(ctx, func(ctx context.Context, input string) (string, error) {
		result, err := runner.Run(ctx, agent, input)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%v", result.FinalOutput), nil
	})
}

func cmdSchedules(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: eve schedules <dir>")
	}
	agent, err := loadAgent(args[0])
	if err != nil {
		return err
	}
	defer agent.Close()

	scheds, err := schedules.LoadDir(agent.Dir + "/" + loader.SchedulesDir)
	if err != nil {
		return err
	}
	if len(scheds) == 0 {
		return fmt.Errorf("no schedules found in %s/%s", agent.Dir, loader.SchedulesDir)
	}
	runner, err := newRunner(agent)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("eve schedules: running %d schedule(s) for agent %q\n", len(scheds), agent.Name)
	for _, s := range scheds {
		fmt.Printf("  - %s: %q\n", s.Name, s.Cron)
	}

	err = schedules.Run(ctx, scheds, func(ctx context.Context, s schedules.Schedule) error {
		fmt.Printf("[%s] triggering: %s\n", s.Name, s.Input)
		result, runErr := runner.Run(ctx, agent, s.Input)
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "[%s] error: %v\n", s.Name, runErr)
			return nil // keep the scheduler alive
		}
		fmt.Printf("[%s] %v\n", s.Name, result.FinalOutput)
		return nil
	})
	if err == context.Canceled {
		return nil
	}
	return err
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func toolNames(a *agents.Agent) string {
	var names []string
	for _, t := range a.Tools {
		names = append(names, t.Name())
	}
	return strings.Join(names, ", ")
}

func skillNames(a *agents.Agent) string {
	var names []string
	for _, s := range a.Skills {
		names = append(names, s.Name)
	}
	return strings.Join(names, ", ")
}

func handoffNames(a *agents.Agent) string {
	var names []string
	for _, h := range a.Handoffs {
		names = append(names, fmt.Sprintf("%s (%s)", h.Name, a.GetHandoffMode(h.Name)))
	}
	return strings.Join(names, ", ")
}
