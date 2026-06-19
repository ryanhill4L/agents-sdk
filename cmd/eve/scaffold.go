package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ryanhill4L/agents-sdk/pkg/loader"
)

// scaffoldFiles maps a relative path within the new agent directory to its
// initial contents.
var scaffoldFiles = map[string]string{
	loader.ConfigFile: `name: assistant
provider: openai          # openai | anthropic | gemini | ollama
model: gpt-4o
temperature: 0.7
max_tokens: 1024
tools:
  - current_time
  - add
`,
	loader.InstructionsFile: `You are a helpful assistant.

Be concise and friendly. Use the available tools when they help answer the
user's request, and consult your skills before acting on their topics.
`,
	filepath.Join(loader.SkillsDir, "greeting.md"): `---
name: greeting
description: How to greet users warmly and set expectations.
---
When greeting a user for the first time:

1. Greet them by name if you know it.
2. Briefly state what you can help with.
3. Ask an open question to get started.
`,
	filepath.Join(loader.SchedulesDir, "daily-standup.yaml"): `name: daily-standup
cron: "0 9 * * 1-5"   # 09:00, Monday–Friday
input: "Give me a short motivational message to start the workday."
`,
}

func cmdInit(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: eve init <dir>")
	}
	dir := args[0]

	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		if _, err := os.Stat(filepath.Join(dir, loader.ConfigFile)); err == nil {
			return fmt.Errorf("%s already contains an %s", dir, loader.ConfigFile)
		}
	}

	for rel, content := range scaffoldFiles {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}

	fmt.Printf("Created agent in %s\n", dir)
	fmt.Println("Next steps:")
	fmt.Printf("  export OPENAI_API_KEY=...   # or set another provider's key\n")
	fmt.Printf("  eve run %s \"hello\"\n", dir)
	fmt.Printf("  eve dev %s\n", dir)
	return nil
}
