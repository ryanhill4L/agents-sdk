package agents

import (
	"strings"
	"testing"

	"github.com/ryanhill4L/agents-sdk/pkg/skills"
)

func TestGetInstructionsIncludesSkillCatalog(t *testing.T) {
	agent := NewAgent("a",
		WithInstructions("Base instructions."),
		WithSkills(skills.Skill{Name: "refunds", Description: "Refund flow"}),
	)

	got := agent.GetInstructions()
	if !strings.Contains(got, "Base instructions.") {
		t.Error("expected base instructions to be present")
	}
	if !strings.Contains(got, "refunds") {
		t.Errorf("expected skill catalog in instructions, got:\n%s", got)
	}
}

func TestGetInstructionsNoSkills(t *testing.T) {
	agent := NewAgent("a", WithInstructions("Just this."))
	if agent.GetInstructions() != "Just this." {
		t.Errorf("instructions = %q", agent.GetInstructions())
	}
}

func TestEffectiveToolsAddsLoadSkill(t *testing.T) {
	agent := NewAgent("a", WithSkills(skills.Skill{Name: "x", Description: "d", Content: "c"}))

	tools := agent.EffectiveTools()
	var found bool
	for _, tl := range tools {
		if tl.Name() == skills.LoadSkillToolName {
			found = true
		}
	}
	if !found {
		t.Error("expected load_skill tool in EffectiveTools")
	}

	// Without skills, no builtin is added.
	plain := NewAgent("b")
	if len(plain.EffectiveTools()) != 0 {
		t.Errorf("expected no tools, got %d", len(plain.EffectiveTools()))
	}
}
