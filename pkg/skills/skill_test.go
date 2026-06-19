package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseWithFrontMatter(t *testing.T) {
	data := []byte("---\nname: refunds\ndescription: How to process a refund\n---\nStep 1. Verify the order.\nStep 2. Issue the refund.")
	s, err := Parse(data, "fallback")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.Name != "refunds" {
		t.Errorf("name = %q, want refunds", s.Name)
	}
	if s.Description != "How to process a refund" {
		t.Errorf("description = %q", s.Description)
	}
	if s.Content == "" || s.Content[:6] != "Step 1" {
		t.Errorf("content = %q", s.Content)
	}
}

func TestParseWithoutFrontMatter(t *testing.T) {
	s, err := Parse([]byte("# Title line\nbody"), "myskill")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.Name != "myskill" {
		t.Errorf("name = %q, want myskill", s.Name)
	}
	if s.Description != "Title line" {
		t.Errorf("description = %q, want 'Title line'", s.Description)
	}
}

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.md"), "---\nname: alpha\ndescription: A\n---\nbody a")
	write(t, filepath.Join(dir, "b.md"), "---\nname: beta\ndescription: B\n---\nbody b")
	write(t, filepath.Join(dir, "ignore.txt"), "not a skill")

	skills, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("got %d skills, want 2", len(skills))
	}
	if skills[0].Name != "alpha" || skills[1].Name != "beta" {
		t.Errorf("unexpected skill order: %q, %q", skills[0].Name, skills[1].Name)
	}
}

func TestLoadDirMissing(t *testing.T) {
	skills, err := LoadDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("LoadDir missing: %v", err)
	}
	if skills != nil {
		t.Errorf("expected nil skills, got %v", skills)
	}
}

func TestLoadSkillTool(t *testing.T) {
	skills := []Skill{{Name: "refunds", Description: "d", Content: "the body"}}
	tool := NewLoadSkillTool(skills)
	if tool.Name() != LoadSkillToolName {
		t.Fatalf("tool name = %q", tool.Name())
	}

	out, err := tool.Execute(context.Background(), map[string]interface{}{"name": "refunds"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "the body" {
		t.Errorf("output = %v, want 'the body'", out)
	}

	if _, err := tool.Execute(context.Background(), map[string]interface{}{"name": "missing"}); err == nil {
		t.Error("expected error for unknown skill")
	}
}

func TestCatalog(t *testing.T) {
	if Catalog(nil) != "" {
		t.Error("empty catalog should be empty string")
	}
	c := Catalog([]Skill{{Name: "x", Description: "does x"}})
	if c == "" || !contains(c, "x: does x") {
		t.Errorf("catalog = %q", c)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
