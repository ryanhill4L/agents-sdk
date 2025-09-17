package types

import (
	"context"
)

type PermissionMode string

const (
	PermissionDefault         PermissionMode = "default"
	PermissionAcceptAll       PermissionMode = "accept_all"
	PermissionAcceptEdits     PermissionMode = "accept_edits"
	PermissionBypass          PermissionMode = "bypass"
	PermissionRejectAll       PermissionMode = "reject_all"
)

type Permission string

const (
	PermissionRead    Permission = "read"
	PermissionWrite   Permission = "write"
	PermissionExecute Permission = "execute"
	PermissionDelete  Permission = "delete"
)

type PermissionRequest struct {
	Tool       string
	Operation  Permission
	Path       string
	Arguments  map[string]any
	Context    context.Context
}

type PermissionResponse struct {
	Allowed bool
	Reason  string
	Modified map[string]any
}

type PermissionCallback func(req PermissionRequest) PermissionResponse

type PermissionManager interface {
	CheckPermission(req PermissionRequest) PermissionResponse
	SetMode(mode PermissionMode)
	GetMode() PermissionMode
	SetCallback(callback PermissionCallback)
	AddAllowedPath(path string)
	AddBlockedPath(path string)
	ClearPaths()
}