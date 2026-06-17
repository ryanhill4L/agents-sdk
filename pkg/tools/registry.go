package tools

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a name-indexed collection of tools.
//
// Because Go tools are compiled functions rather than interpreted files, the
// filesystem-first agent definition references tools by name (in agent.yaml)
// and resolves them against a Registry at load time. Register your tools once
// at program startup, then point the loader at the registry.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds one or more tools to the registry. It returns an error if a
// tool with the same name was already registered or if a tool is invalid.
func (r *Registry) Register(ts ...Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, t := range ts {
		if t == nil {
			return fmt.Errorf("cannot register nil tool")
		}
		if err := t.Validate(); err != nil {
			return fmt.Errorf("invalid tool %q: %w", t.Name(), err)
		}
		if _, exists := r.tools[t.Name()]; exists {
			return fmt.Errorf("tool %q already registered", t.Name())
		}
		r.tools[t.Name()] = t
	}
	return nil
}

// MustRegister is like Register but panics on error. Useful for package-level
// registration where a duplicate name is a programming error.
func (r *Registry) MustRegister(ts ...Tool) {
	if err := r.Register(ts...); err != nil {
		panic(err)
	}
}

// Get returns the tool registered under name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tools[name]
	return t, ok
}

// Resolve looks up every requested name and returns the matching tools in the
// same order. It returns an error listing any names that are not registered.
func (r *Registry) Resolve(names []string) ([]Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resolved := make([]Tool, 0, len(names))
	var missing []string
	for _, name := range names {
		t, ok := r.tools[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		resolved = append(resolved, t)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("unknown tool(s): %v", missing)
	}
	return resolved, nil
}

// Names returns the sorted names of all registered tools.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
