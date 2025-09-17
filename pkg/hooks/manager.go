package hooks

import (
	"fmt"
	"sort"
	"sync"

	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

type Manager struct {
	mu    sync.RWMutex
	hooks map[types.HookEvent][]types.Hook
}

func NewManager() *Manager {
	return &Manager{
		hooks: make(map[types.HookEvent][]types.Hook),
	}
}

func (m *Manager) RegisterHook(hook types.Hook) error {
	if hook == nil {
		return fmt.Errorf("hook cannot be nil")
	}

	if hook.Name() == "" {
		return fmt.Errorf("hook must have a name")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, event := range hook.Events() {
		existing := m.hooks[event]

		for _, h := range existing {
			if h.Name() == hook.Name() {
				return fmt.Errorf("hook with name '%s' already registered for event '%s'", hook.Name(), event)
			}
		}

		m.hooks[event] = append(m.hooks[event], hook)

		sort.Slice(m.hooks[event], func(i, j int) bool {
			return m.hooks[event][i].Priority() < m.hooks[event][j].Priority()
		})
	}

	return nil
}

func (m *Manager) UnregisterHook(name string) error {
	if name == "" {
		return fmt.Errorf("hook name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	for event, hooks := range m.hooks {
		filtered := make([]types.Hook, 0, len(hooks))
		for _, h := range hooks {
			if h.Name() != name {
				filtered = append(filtered, h)
			} else {
				found = true
			}
		}
		m.hooks[event] = filtered
	}

	if !found {
		return fmt.Errorf("hook with name '%s' not found", name)
	}

	return nil
}

func (m *Manager) ExecuteHooks(event types.HookEvent, ctx *types.HookContext) error {
	m.mu.RLock()
	hooks, exists := m.hooks[event]
	if !exists || len(hooks) == 0 {
		m.mu.RUnlock()
		return nil
	}

	hooksToExecute := make([]types.Hook, len(hooks))
	copy(hooksToExecute, hooks)
	m.mu.RUnlock()

	ctx.Event = event

	for _, hook := range hooksToExecute {
		if err := hook.Execute(ctx); err != nil {
			return fmt.Errorf("hook '%s' failed: %w", hook.Name(), err)
		}
	}

	return nil
}

func (m *Manager) GetHooks(event types.HookEvent) []types.Hook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hooks, exists := m.hooks[event]
	if !exists {
		return nil
	}

	result := make([]types.Hook, len(hooks))
	copy(result, hooks)
	return result
}

func (m *Manager) ClearHooks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hooks = make(map[types.HookEvent][]types.Hook)
}

func (m *Manager) HasHooks(event types.HookEvent) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hooks, exists := m.hooks[event]
	return exists && len(hooks) > 0
}

func (m *Manager) GetAllHooks() map[types.HookEvent][]types.Hook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[types.HookEvent][]types.Hook)
	for event, hooks := range m.hooks {
		hooksCopy := make([]types.Hook, len(hooks))
		copy(hooksCopy, hooks)
		result[event] = hooksCopy
	}
	return result
}

func (m *Manager) RegisterSimpleHook(name string, events []types.HookEvent, fn types.HookFunc) error {
	hook := types.NewSimpleHook(name, events, fn)
	return m.RegisterHook(hook)
}

func (m *Manager) RegisterSimpleHookWithPriority(name string, events []types.HookEvent, priority int, fn types.HookFunc) error {
	hook := types.NewSimpleHook(name, events, fn)
	hook.SetPriority(priority)
	return m.RegisterHook(hook)
}