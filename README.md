# Go Agents SDK

A Go SDK for building AI agents, reimagined around a **filesystem-first**
design inspired by [Vercel's Eve](https://vercel.com/blog/introducing-eve).
Core agent capabilities live in conventional files and folders, so projects are
easy to inspect, extend, and operate — while the underlying engine remains a
plain Go library with multi-provider support, tools, skills, handoffs,
guardrails, memory, and tracing.

[![Go Reference](https://pkg.go.dev/badge/github.com/ryanhill4L/agents-sdk.svg)](https://pkg.go.dev/github.com/ryanhill4L/agents-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## The filesystem is the authoring interface

An agent is a directory. Conventional files describe its behaviour:

```
myagent/
  agent.yaml            # model, provider, temperature, tools
  instructions.md       # system prompt
  skills/
    refunds.md          # on-demand knowledge (YAML front-matter + markdown body)
  subagents/
    billing/            # a delegated agent (handoff), same layout recursively
      agent.yaml
      instructions.md
  schedules/
    nightly.yaml        # cron triggers
  channels/             # integration adapters (HTTP is built in)
```

| Path             | Purpose                                                            |
|------------------|-------------------------------------------------------------------|
| `agent.yaml`     | Model, provider, sampling params, and the tools the agent may use |
| `instructions.md`| The system prompt                                                  |
| `skills/*.md`    | Procedures/knowledge surfaced to the model and loaded on demand   |
| `subagents/<n>/` | Specialized agents the parent can hand off to                     |
| `schedules/*.yaml` | Cron expressions that trigger the agent autonomously            |

### `agent.yaml`

```yaml
name: assistant
provider: openai          # openai | anthropic | gemini | ollama
model: gpt-4o
temperature: 0.7
max_tokens: 1024
tools:
  - current_time
  - add
```

### A skill (`skills/weather.md`)

```markdown
---
name: weather
description: How to answer questions about the weather using a public API.
---
1. Determine the location the user is asking about.
2. Use the `http_get` tool to fetch https://wttr.in/<location>?format=3.
3. Summarize the result in one friendly sentence.
```

Skills are not dumped into the prompt. Instead the model sees a **catalog** of
skill names and descriptions and pulls a skill's full body only when needed,
via the built-in `load_skill` tool.

## The `eve` CLI

```bash
go build -o eve ./cmd/eve

eve init myagent                     # scaffold a new agent directory
eve validate myagent                 # load and report what was found (no API key needed)
eve run myagent "what time is it?"   # run a single turn
eve dev myagent                      # serve over HTTP: POST /chat {"input": "..."}
eve schedules myagent                # run the agent's cron schedules
```

Provider credentials come from the environment:

```bash
export PROVIDER=openai            # optional; otherwise auto-detected
export OPENAI_API_KEY=...         # or ANTHROPIC_API_KEY / GEMINI_API_KEY / OLLAMA_HOST
```

> Because Go tools are compiled functions, `agent.yaml` references tools **by
> name**. The CLI ships a small builtin registry (`current_time`, `add`,
> `http_get`). To use your own Go tools, load the agent from your own program
> with a custom registry (see below).

## Library API

The CLI is a thin shell over the library. Load a filesystem agent with your own
tools and run it:

```go
package main

import (
	"context"
	"fmt"

	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/loader"
	"github.com/ryanhill4L/agents-sdk/pkg/providers"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

func main() {
	// 1. Register your Go tools by name.
	reg := tools.NewRegistry()
	reg.MustRegister(tools.NewTool("add", "Adds two numbers",
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return args["a"].(float64) + args["b"].(float64), nil
		},
		tools.Param{Name: "a", Type: "number", Required: true},
		tools.Param{Name: "b", Type: "number", Required: true},
	))

	// 2. Load the agent from the filesystem.
	agent, err := loader.Load("myagent", reg)
	if err != nil {
		panic(err)
	}

	// 3. Run it with a provider resolved from the environment.
	provider, _ := providers.Resolve(agent.Provider)
	runner := agents.NewRunner(agents.WithProvider(provider))

	result, err := runner.Run(context.Background(), agent, "What is 21 + 21?")
	if err != nil {
		panic(err)
	}
	fmt.Println(result.FinalOutput)
}
```

You can also build agents entirely in code with functional options
(`agents.NewAgent`, `agents.WithInstructions`, `agents.WithTools`,
`agents.WithSkills`, `agents.WithHandoffs`, …) — the loader simply translates
files into those calls.

## Packages

| Package           | Responsibility                                                        |
|-------------------|-----------------------------------------------------------------------|
| `pkg/agents`      | `Agent`, `Runner`, the turn loop, handoffs, skills wiring             |
| `pkg/loader`      | Builds an `Agent` from a directory (`agent.yaml`, `instructions.md`, …) |
| `pkg/skills`      | Skill parsing, the catalog, and the `load_skill` builtin tool         |
| `pkg/tools`       | `Tool` interface, `Registry`, `FunctionTool` (reflection), `SimpleTool` |
| `pkg/channels`    | Integration adapters; built-in HTTP channel (powers `eve dev`)        |
| `pkg/schedules`   | Cron parsing and a dependency-free scheduler                          |
| `pkg/providers`   | OpenAI, Anthropic, Gemini, and Ollama integrations + `Resolve`        |
| `pkg/memory`      | SQLite-backed session persistence                                     |
| `pkg/guardrails`  | Pluggable input validation                                            |
| `pkg/tracing`     | Span-based observability (no-op and console tracers)                  |

## Providers

| Provider  | Env var             | Notes                           |
|-----------|---------------------|---------------------------------|
| OpenAI    | `OPENAI_API_KEY`    | GPT-4o and friends              |
| Anthropic | `ANTHROPIC_API_KEY` | Claude models                   |
| Gemini    | `GEMINI_API_KEY`    | Google Gemini                   |
| Ollama    | `OLLAMA_HOST`       | Local models (default `:11434`) |

## Examples

- [`examples/filesystem-agent`](examples/filesystem-agent) — the filesystem-first
  layout, driven by the `eve` CLI (agent + skill + subagent + schedule).
- [`examples/basic`](examples/basic), [`examples/multi-agent`](examples/multi-agent),
  [`examples/json`](examples/json), [`examples/event-scheduler`](examples/event-scheduler)
  — code-first usage of the library API.

## Development

```bash
make build          # build all packages
make build-cli      # build the eve CLI into bin/eve
make test           # run tests
make run-fs-example # validate the filesystem-first example
make check          # fmt + vet + tidy
```

## License

MIT — see [LICENSE](LICENSE).
