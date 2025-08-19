package tools

import (
	"context"
	"fmt"
	"reflect"
)

// FunctionTool wraps a Go function as a tool
type FunctionTool struct {
	name        string
	description string
	fn          reflect.Value
	fnType      reflect.Type
	schema      ParameterSchema
}

// ParameterSchema describes function parameters
type ParameterSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required"`
}

// PropertySchema describes a single parameter
type PropertySchema struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// NewFunctionTool creates a tool from a function
func NewFunctionTool(name, description string, fn interface{}) (*FunctionTool, error) {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided value is not a function")
	}

	tool := &FunctionTool{
		name:        name,
		description: description,
		fn:          fnValue,
		fnType:      fnType,
	}

	// Build parameter schema
	if err := tool.buildSchema(); err != nil {
		return nil, err
	}

	return tool, nil
}

// buildSchema creates the parameter schema from function signature
func (f *FunctionTool) buildSchema() error {
	f.schema = ParameterSchema{
		Type:       "object",
		Properties: make(map[string]PropertySchema),
		Required:   make([]string, 0),
	}

	// Parse function parameters
	for i := 0; i < f.fnType.NumIn(); i++ {
		param := f.fnType.In(i)

		// Skip context.Context
		if param.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			continue
		}

		// Add to schema (simplified - real implementation would be more complex)
		paramName := fmt.Sprintf("arg%d", i)
		f.schema.Properties[paramName] = PropertySchema{
			Type: f.goTypeToJSONType(param),
		}
		f.schema.Required = append(f.schema.Required, paramName)
	}

	return nil
}

// goTypeToJSONType converts Go types to JSON schema types
func (f *FunctionTool) goTypeToJSONType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "string"
	}
}

// convertToType converts a value from JSON types to the expected Go type
func (f *FunctionTool) convertToType(val interface{}, targetType reflect.Type) (reflect.Value, error) {
	if val == nil {
		return reflect.Zero(targetType), nil
	}

	valType := reflect.TypeOf(val)
	
	// If types already match, return as-is
	if valType == targetType {
		return reflect.ValueOf(val), nil
	}

	// Handle common JSON->Go type conversions
	switch targetType.Kind() {
	case reflect.Int:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(int(v)), nil
		case int:
			return reflect.ValueOf(v), nil
		case int64:
			return reflect.ValueOf(int(v)), nil
		default:
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to int", val)
		}
	case reflect.Int32:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(int32(v)), nil
		case int:
			return reflect.ValueOf(int32(v)), nil
		case int32:
			return reflect.ValueOf(v), nil
		default:
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to int32", val)
		}
	case reflect.Int64:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(int64(v)), nil
		case int:
			return reflect.ValueOf(int64(v)), nil
		case int64:
			return reflect.ValueOf(v), nil
		default:
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to int64", val)
		}
	case reflect.Float32:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(float32(v)), nil
		case float32:
			return reflect.ValueOf(v), nil
		case int:
			return reflect.ValueOf(float32(v)), nil
		default:
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to float32", val)
		}
	case reflect.Float64:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(v), nil
		case float32:
			return reflect.ValueOf(float64(v)), nil
		case int:
			return reflect.ValueOf(float64(v)), nil
		default:
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to float64", val)
		}
	case reflect.String:
		switch v := val.(type) {
		case string:
			return reflect.ValueOf(v), nil
		default:
			return reflect.ValueOf(fmt.Sprintf("%v", val)), nil
		}
	case reflect.Bool:
		switch v := val.(type) {
		case bool:
			return reflect.ValueOf(v), nil
		default:
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to bool", val)
		}
	default:
		// For other types, try direct assignment if possible
		valValue := reflect.ValueOf(val)
		if valValue.Type().ConvertibleTo(targetType) {
			return valValue.Convert(targetType), nil
		}
		return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to %s", val, targetType.String())
	}
}

// Name returns the tool name
func (f *FunctionTool) Name() string {
	return f.name
}

// Description returns the tool description
func (f *FunctionTool) Description() string {
	return f.description
}

// Schema returns the parameter schema
func (f *FunctionTool) Schema() ParameterSchema {
	return f.schema
}

// Execute runs the function with provided arguments
func (f *FunctionTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Build function arguments
	fnArgs := make([]reflect.Value, 0, f.fnType.NumIn())

	// Add context if needed
	for i := 0; i < f.fnType.NumIn(); i++ {
		param := f.fnType.In(i)

		if param.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			fnArgs = append(fnArgs, reflect.ValueOf(ctx))
		} else {
			// Get argument from map and convert type if necessary
			argName := fmt.Sprintf("arg%d", i)
			if val, ok := args[argName]; ok {
				convertedVal, err := f.convertToType(val, param)
				if err != nil {
					return nil, fmt.Errorf("failed to convert argument %s: %w", argName, err)
				}
				fnArgs = append(fnArgs, convertedVal)
			} else {
				fnArgs = append(fnArgs, reflect.Zero(param))
			}
		}
	}

	// Call function
	results := f.fn.Call(fnArgs)

	// Handle results
	if len(results) == 0 {
		return nil, nil
	}

	// Check for error return
	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return nil, lastResult.Interface().(error)
			}
		}
	}

	// Return first result
	return results[0].Interface(), nil
}

// Validate checks if the tool is valid
func (f *FunctionTool) Validate() error {
	if f.name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if !f.fn.IsValid() {
		return fmt.Errorf("invalid function")
	}

	return nil
}
