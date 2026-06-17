# Filesystem-first agent example

This directory shows the Eve-inspired, filesystem-first layout. The agent lives
entirely in `assistant/` as plain files:

```
assistant/
  agent.yaml                     # model, provider, temperature, tools
  instructions.md                # system prompt
  skills/
    weather.md                   # on-demand knowledge (front-matter + body)
  subagents/
    researcher/                  # a delegated agent (handoff)
      agent.yaml
      instructions.md
  schedules/
    morning-brief.yaml           # a cron trigger
```

## Run it with the `eve` CLI

Build the CLI from the repo root:

```bash
go build -o eve ./cmd/eve
```

Then, from the repo root:

```bash
export OPENAI_API_KEY=...        # or ANTHROPIC_API_KEY / GEMINI_API_KEY / OLLAMA_HOST

# Inspect what was loaded (no API key required):
./eve validate examples/filesystem-agent/assistant

# Run a single turn:
./eve run examples/filesystem-agent/assistant "What time is it, and what is 21 + 21?"

# Serve it over HTTP:
./eve dev examples/filesystem-agent/assistant
#   curl -s localhost:8080/chat -d '{"input":"hello"}'

# Run its cron schedules:
./eve schedules examples/filesystem-agent/assistant
```

## How tools work in Go

Because Go tools are compiled functions, `agent.yaml` references them by name and
the CLI resolves them against its builtin registry (`current_time`, `add`,
`http_get`). To use your own Go tools, load the agent from your own program with
a custom registry:

```go
reg := tools.NewRegistry()
reg.MustRegister(myTool)
agent, _ := loader.Load("examples/filesystem-agent/assistant", reg)
runner := agents.NewRunner(agents.WithProvider(p))
result, _ := runner.Run(ctx, agent, "hello")
```
