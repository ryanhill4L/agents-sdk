package permissions

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

type Manager struct {
	mu             sync.RWMutex
	mode           types.PermissionMode
	callback       types.PermissionCallback
	allowedPaths   []string
	blockedPaths   []string
}

func NewManager(mode types.PermissionMode) *Manager {
	return &Manager{
		mode:         mode,
		allowedPaths: make([]string, 0),
		blockedPaths: make([]string, 0),
	}
}

func (m *Manager) CheckPermission(req types.PermissionRequest) types.PermissionResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch m.mode {
	case types.PermissionBypass:
		return types.PermissionResponse{Allowed: true}

	case types.PermissionRejectAll:
		return types.PermissionResponse{
			Allowed: false,
			Reason:  "all operations are rejected in reject_all mode",
		}

	case types.PermissionAcceptAll:
		return types.PermissionResponse{Allowed: true}

	case types.PermissionAcceptEdits:
		if req.Operation == types.PermissionWrite || req.Operation == types.PermissionRead {
			return types.PermissionResponse{Allowed: true}
		}
		if req.Operation == types.PermissionDelete || req.Operation == types.PermissionExecute {
			if m.callback != nil {
				return m.callback(req)
			}
			return types.PermissionResponse{
				Allowed: false,
				Reason:  "delete and execute operations require explicit permission",
			}
		}

	case types.PermissionDefault:
		fallthrough
	default:
		if req.Path != "" {
			if !m.isPathAllowed(req.Path) {
				return types.PermissionResponse{
					Allowed: false,
					Reason:  "path is not in allowed directories or is blocked",
				}
			}
		}

		if m.callback != nil {
			return m.callback(req)
		}

		if req.Operation == types.PermissionRead {
			return types.PermissionResponse{Allowed: true}
		}

		return types.PermissionResponse{
			Allowed: false,
			Reason:  "operation requires explicit permission",
		}
	}

	return types.PermissionResponse{
		Allowed: false,
		Reason:  "permission check failed",
	}
}

func (m *Manager) isPathAllowed(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, blocked := range m.blockedPaths {
		absBlocked, err := filepath.Abs(blocked)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, absBlocked) {
			return false
		}
	}

	if len(m.allowedPaths) == 0 {
		return true
	}

	for _, allowed := range m.allowedPaths {
		absAllowed, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, absAllowed) {
			return true
		}
	}

	return false
}

func (m *Manager) SetMode(mode types.PermissionMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mode = mode
}

func (m *Manager) GetMode() types.PermissionMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mode
}

func (m *Manager) SetCallback(callback types.PermissionCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callback = callback
}

func (m *Manager) AddAllowedPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowedPaths = append(m.allowedPaths, path)
}

func (m *Manager) AddBlockedPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockedPaths = append(m.blockedPaths, path)
}

func (m *Manager) ClearPaths() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowedPaths = m.allowedPaths[:0]
	m.blockedPaths = m.blockedPaths[:0]
}

func (m *Manager) GetAllowedPaths() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.allowedPaths))
	copy(result, m.allowedPaths)
	return result
}

func (m *Manager) GetBlockedPaths() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.blockedPaths))
	copy(result, m.blockedPaths)
	return result
}