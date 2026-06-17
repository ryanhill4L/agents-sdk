package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryanhill4L/agents-sdk/pkg/agents"
	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testRegistry() *tools.Registry {
	r := tools.NewRegistry()
	r.MustRegister(tools.NewTool("ping", "pings", func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
		return "pong", nil
	}))
	return r
}

func TestLoadFullAgent(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, ConfigFile), `name: support
provider: anthropic
model: claude-sonnet-4-6
temperature: 0.3
max_tokens: 512
tools:
  - ping
`)
	writeFile(t, filepath.Join(dir, InstructionsFile), "You are support.")
	writeFile(t, filepath.Join(dir, SkillsDir, "refunds.md"), "---\nname: refunds\ndescription: Refund flow\n---\nbody")

	// A subagent.
	sub := filepath.Join(dir, SubagentsDir, "billing")
	writeFile(t, filepath.Join(sub, ConfigFile), "name: billing\nmodel: claude-sonnet-4-6\n")
	writeFile(t, filepath.Join(sub, InstructionsFile), "You handle billing.")

	agent, err := Load(dir, testRegistry())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if agent.Name != "support" {
		t.Errorf("name = %q", agent.Name)
	}
	if agent.Provider != "anthropic" {
		t.Errorf("provider = %q", agent.Provider)
	}
	if agent.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q", agent.Model)
	}
	if agent.Temperature != 0.3 {
		t.Errorf("temperature = %v", agent.Temperature)
	}
	if len(agent.Tools) != 1 || agent.Tools[0].Name() != "ping" {
		t.Errorf("tools = %v", agent.Tools)
	}
	if len(agent.Skills) != 1 || agent.Skills[0].Name != "refunds" {
		t.Errorf("skills = %v", agent.Skills)
	}
	if _, ok := agent.GetHandoff("billing"); !ok {
		t.Error("expected billing subagent as handoff")
	}

	// The load_skill builtin should be exposed because the agent has skills.
	var hasLoadSkill bool
	for _, tool := range agent.EffectiveTools() {
		if tool.Name() == "load_skill" {
			hasLoadSkill = true
		}
	}
	if !hasLoadSkill {
		t.Error("expected load_skill builtin in EffectiveTools")
	}
}

func TestLoadSubagentModes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ConfigFile), `name: parent
model: m
subagents:
  worker: forked
`)
	sub := filepath.Join(dir, SubagentsDir, "worker")
	writeFile(t, filepath.Join(sub, ConfigFile), "name: worker\nmodel: m\n")

	agent, err := Load(dir, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := agent.GetHandoffMode("worker"); got != agents.ContextForked {
		t.Errorf("worker mode = %v, want forked", got)
	}
}

func TestLoadInvalidSubagentMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ConfigFile), "name: parent\nmodel: m\nsubagents:\n  worker: sideways\n")
	writeFile(t, filepath.Join(dir, SubagentsDir, "worker", ConfigFile), "name: worker\nmodel: m\n")
	if _, err := Load(dir, nil); err == nil {
		t.Error("expected error for invalid subagent mode")
	}
}

func TestLoadRemoteSkill(t *testing.T) {
	body := "---\nname: remote\ndescription: A remote skill\n---\nbody"
	sum := sha256.Sum256([]byte(body))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ConfigFile), "name: x\nmodel: m\nskills:\n  - source: "+srv.URL+"/s.md\n    sha256: "+hex.EncodeToString(sum[:])+"\n")

	agent, err := LoadWithOptions(dir, nil, Options{
		SkillCache:   t.TempDir(),
		AllowedHosts: []string{"127.0.0.1"},
	})
	if err != nil {
		t.Fatalf("LoadWithOptions: %v", err)
	}
	if len(agent.Skills) != 1 || agent.Skills[0].Name != "remote" {
		t.Errorf("skills = %+v, want one 'remote' skill", agent.Skills)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	if _, err := Load(t.TempDir(), nil); err == nil {
		t.Error("expected error for missing agent.yaml")
	}
}

func TestLoadUnknownTool(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ConfigFile), "name: x\nmodel: m\ntools:\n  - nope\n")
	if _, err := Load(dir, testRegistry()); err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestLoadToolsWithoutRegistry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ConfigFile), "name: x\nmodel: m\ntools:\n  - ping\n")
	if _, err := Load(dir, nil); err == nil {
		t.Error("expected error when tools declared but no registry")
	}
}
