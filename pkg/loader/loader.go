// Package loader builds agents from the filesystem. An agent is a directory:
//
//	myagent/
//	  agent.yaml         # model / provider / temperature / tools
//	  instructions.md    # system prompt
//	  skills/*.md        # on-demand knowledge
//	  subagents/<name>/  # delegated agents (handoffs), same layout recursively
//
// This mirrors Eve's "the filesystem is the authoring interface" idea, adapted
// to Go: because tools are compiled functions, agent.yaml references them by
// name and the loader resolves them against a tools.Registry.
package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/skills"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

// Standard filenames and subdirectories within an agent directory.
const (
	ConfigFile       = "agent.yaml"
	InstructionsFile = "instructions.md"
	SkillsDir        = "skills"
	SubagentsDir     = "subagents"
	SchedulesDir     = "schedules"
	ChannelsDir      = "channels"
)

// Config is the declarative agent.yaml schema.
type Config struct {
	Name        string   `yaml:"name"`
	Model       string   `yaml:"model"`
	Provider    string   `yaml:"provider"`
	Temperature *float32 `yaml:"temperature"`
	MaxTokens   int      `yaml:"max_tokens"`
	TopP        *float32 `yaml:"top_p"`
	Tools       []string `yaml:"tools"`
}

// Load builds an agent (and its subagents, recursively) from dir. Tools named in
// agent.yaml are resolved against registry; pass a nil registry only if no agent
// in the tree declares tools.
func Load(dir string, registry *tools.Registry) (*agents.Agent, error) {
	return load(dir, registry, map[string]bool{})
}

func load(dir string, registry *tools.Registry, seen map[string]bool) (*agents.Agent, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if seen[abs] {
		return nil, fmt.Errorf("circular subagent reference at %s", abs)
	}
	seen[abs] = true
	defer delete(seen, abs)

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("agent directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	cfg, err := readConfig(filepath.Join(abs, ConfigFile))
	if err != nil {
		return nil, err
	}

	name := cfg.Name
	if name == "" {
		name = filepath.Base(abs)
	}

	opts := []agents.AgentOption{}

	instructions, err := readInstructions(filepath.Join(abs, InstructionsFile))
	if err != nil {
		return nil, err
	}
	if instructions != "" {
		opts = append(opts, agents.WithInstructions(instructions))
	}
	if cfg.Model != "" {
		opts = append(opts, agents.WithModel(cfg.Model))
	}
	if cfg.Provider != "" {
		opts = append(opts, agents.WithProviderName(cfg.Provider))
	}
	if cfg.Temperature != nil {
		opts = append(opts, agents.WithTemperature(*cfg.Temperature))
	}
	if cfg.MaxTokens > 0 {
		opts = append(opts, agents.WithMaxTokens(cfg.MaxTokens))
	}

	// Resolve declared tools.
	if len(cfg.Tools) > 0 {
		if registry == nil {
			return nil, fmt.Errorf("agent %q declares tools %v but no tool registry was provided", name, cfg.Tools)
		}
		resolved, err := registry.Resolve(cfg.Tools)
		if err != nil {
			return nil, fmt.Errorf("agent %q: %w", name, err)
		}
		opts = append(opts, agents.WithTools(resolved...))
	}

	// Load skills.
	loadedSkills, err := skills.LoadDir(filepath.Join(abs, SkillsDir))
	if err != nil {
		return nil, fmt.Errorf("agent %q: loading skills: %w", name, err)
	}
	if len(loadedSkills) > 0 {
		opts = append(opts, agents.WithSkills(loadedSkills...))
	}

	// Load subagents as handoffs.
	subagents, err := loadSubagents(filepath.Join(abs, SubagentsDir), registry, seen)
	if err != nil {
		return nil, fmt.Errorf("agent %q: %w", name, err)
	}
	if len(subagents) > 0 {
		opts = append(opts, agents.WithHandoffs(subagents...))
	}

	agent := agents.NewAgent(name, opts...)
	agent.Dir = abs

	if err := agent.Validate(); err != nil {
		return nil, fmt.Errorf("agent %q: %w", name, err)
	}
	return agent, nil
}

func loadSubagents(dir string, registry *tools.Registry, seen map[string]bool) ([]*agents.Agent, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var subs []*agents.Agent
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sub, err := load(filepath.Join(dir, entry.Name()), registry, seen)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	sort.Slice(subs, func(i, j int) bool { return subs[i].Name < subs[j].Name })
	return subs, nil
}

func readConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, fmt.Errorf("missing %s", ConfigFile)
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("invalid %s: %w", ConfigFile, err)
	}
	return cfg, nil
}

func readInstructions(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
