package tools

import (
	"context"
)

// Tool represents a capability that can be used by agents
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns the tool description
	Description() string

	// Schema returns the parameter schema for the tool
	Schema() ParameterSchema

	// Execute runs the tool with the provided arguments
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)

	// Validate checks if the tool configuration is valid
	Validate() error
}