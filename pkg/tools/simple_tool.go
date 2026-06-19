package tools

import (
	"context"
	"fmt"
)

// HandlerFunc executes a tool given decoded arguments.
type HandlerFunc func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// SimpleTool is a Tool defined by an explicit schema and a handler function.
//
// Unlike FunctionTool (which derives a positional schema via reflection),
// SimpleTool lets you declare named, described parameters — which produces far
// better tool-use behaviour from models. It is the preferred way to author
// tools and is used internally for builtins such as load_skill.
type SimpleTool struct {
	name        string
	description string
	schema      ParameterSchema
	handler     HandlerFunc
}

// Param describes a single named parameter for NewTool.
type Param struct {
	Name        string
	Type        string // JSON schema type: string, integer, number, boolean, array, object
	Description string
	Required    bool
}

// NewTool builds a SimpleTool from a name, description, handler and parameters.
func NewTool(name, description string, handler HandlerFunc, params ...Param) *SimpleTool {
	schema := ParameterSchema{
		Type:       "object",
		Properties: make(map[string]PropertySchema, len(params)),
		Required:   make([]string, 0, len(params)),
	}
	for _, p := range params {
		typ := p.Type
		if typ == "" {
			typ = "string"
		}
		schema.Properties[p.Name] = PropertySchema{Type: typ, Description: p.Description}
		if p.Required {
			schema.Required = append(schema.Required, p.Name)
		}
	}
	return &SimpleTool{
		name:        name,
		description: description,
		schema:      schema,
		handler:     handler,
	}
}

// Name returns the tool name.
func (t *SimpleTool) Name() string { return t.name }

// Description returns the tool description.
func (t *SimpleTool) Description() string { return t.description }

// Schema returns the parameter schema.
func (t *SimpleTool) Schema() ParameterSchema { return t.schema }

// Execute runs the handler with the provided arguments.
func (t *SimpleTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if t.handler == nil {
		return nil, fmt.Errorf("tool %q has no handler", t.name)
	}
	return t.handler(ctx, args)
}

// Validate checks that the tool is usable.
func (t *SimpleTool) Validate() error {
	if t.name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if t.handler == nil {
		return fmt.Errorf("tool %q has no handler", t.name)
	}
	return nil
}
