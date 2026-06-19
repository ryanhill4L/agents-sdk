// Package skills implements Eve-style on-demand skills: units of knowledge or
// procedure stored as markdown files. Rather than dumping every procedure into
// the system prompt, an agent advertises a catalog of skill names and
// descriptions, and the model pulls the full body only when it needs it via the
// load_skill builtin tool.
package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ryanhill4L/agents-sdk/pkg/tools"
)

// Skill is a single unit of on-demand knowledge.
type Skill struct {
	// Name is the identifier the model uses to load the skill.
	Name string `yaml:"name"`
	// Description is shown in the skill catalog so the model knows when to load it.
	Description string `yaml:"description"`
	// Content is the body of the skill (everything after the front-matter).
	Content string `yaml:"-"`
	// Path is the source file, when loaded from disk.
	Path string `yaml:"-"`
}

// frontMatter mirrors the YAML header of a skill file.
type frontMatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Parse reads a skill from raw markdown bytes. The file may begin with a YAML
// front-matter block delimited by lines containing only "---". If the
// front-matter omits a name, defaultName is used.
func Parse(data []byte, defaultName string) (Skill, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")

	skill := Skill{Name: defaultName}

	if strings.HasPrefix(text, "---\n") {
		rest := text[len("---\n"):]
		if idx := strings.Index(rest, "\n---\n"); idx >= 0 {
			header := rest[:idx]
			body := rest[idx+len("\n---\n"):]

			var fm frontMatter
			if err := yaml.Unmarshal([]byte(header), &fm); err != nil {
				return Skill{}, fmt.Errorf("invalid skill front-matter: %w", err)
			}
			if fm.Name != "" {
				skill.Name = fm.Name
			}
			skill.Description = fm.Description
			skill.Content = strings.TrimSpace(body)
			if skill.Name == "" {
				return Skill{}, fmt.Errorf("skill is missing a name")
			}
			return skill, nil
		}
	}

	// No front-matter: the whole file is the body, derive a description from the
	// first non-empty line.
	skill.Content = strings.TrimSpace(text)
	skill.Description = firstLine(skill.Content)
	if skill.Name == "" {
		return Skill{}, fmt.Errorf("skill is missing a name")
	}
	return skill, nil
}

// LoadFile reads a single skill from a markdown file. The default name is the
// file name without its extension.
func LoadFile(path string) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, err
	}
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	skill, err := Parse(data, name)
	if err != nil {
		return Skill{}, fmt.Errorf("%s: %w", path, err)
	}
	skill.Path = path
	return skill, nil
}

// LoadDir loads every *.md file in dir as a skill. A missing directory is not an
// error and yields no skills.
func LoadDir(dir string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var skills []Skill
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			continue
		}
		skill, err := LoadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}

// Catalog renders a human/model-readable list of the given skills, suitable for
// appending to a system prompt.
func Catalog(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Available skills\n")
	b.WriteString("Use the load_skill tool to read a skill's full instructions before acting on its topic.\n\n")
	for _, s := range skills {
		desc := s.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Fprintf(&b, "- %s: %s\n", s.Name, desc)
	}
	return b.String()
}

// LoadSkillToolName is the name of the builtin tool that returns skill bodies.
const LoadSkillToolName = "load_skill"

// NewLoadSkillTool builds the builtin tool that lets a model fetch a skill's
// body on demand. It closes over the provided skills.
func NewLoadSkillTool(skills []Skill) tools.Tool {
	index := make(map[string]Skill, len(skills))
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		index[s.Name] = s
		names = append(names, s.Name)
	}
	sort.Strings(names)

	description := fmt.Sprintf(
		"Load the full instructions for an available skill. Valid skill names: %s.",
		strings.Join(names, ", "),
	)

	handler := func(_ context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("the 'name' argument is required")
		}
		skill, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("unknown skill %q; valid skills: %s", name, strings.Join(names, ", "))
		}
		if skill.Content == "" {
			return fmt.Sprintf("Skill %q has no content.", name), nil
		}
		return skill.Content, nil
	}

	return tools.NewTool(LoadSkillToolName, description, handler, tools.Param{
		Name:        "name",
		Type:        "string",
		Description: "The name of the skill to load.",
		Required:    true,
	})
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(strings.TrimLeft(s, "# "))
}
