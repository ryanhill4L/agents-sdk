package tools

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ryanhill4L/agents-sdk/pkg/types"
)

type FunctionTool struct {
	name               string
	description        string
	fn                 reflect.Value
	fnType             reflect.Type
	schema             types.ToolSchema
	requiresPermission bool
}

func NewFunctionTool(name, description string, fn interface{}) (*FunctionTool, error) {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided value is not a function")
	}

	tool := &FunctionTool{
		name:               name,
		description:        description,
		fn:                 fnValue,
		fnType:             fnType,
		requiresPermission: true,
	}

	if err := tool.buildSchema(); err != nil {
		return nil, err
	}

	return tool, nil
}

func (f *FunctionTool) buildSchema() error {
	f.schema = types.ToolSchema{
		Type:        "object",
		Properties:  make(map[string]types.Property),
		Description: f.description,
	}

	paramIndex := 0
	for i := 0; i < f.fnType.NumIn(); i++ {
		param := f.fnType.In(i)

		if param.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			continue
		}

		paramName := fmt.Sprintf("arg%d", paramIndex)
		f.schema.Properties[paramName] = f.typeToProperty(param)
		f.schema.Required = append(f.schema.Required, paramName)
		paramIndex++
	}

	return nil
}

func (f *FunctionTool) typeToProperty(t reflect.Type) types.Property {
	prop := types.Property{}

	switch t.Kind() {
	case reflect.String:
		prop.Type = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		prop.Type = "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		prop.Type = "integer"
	case reflect.Float32, reflect.Float64:
		prop.Type = "number"
	case reflect.Bool:
		prop.Type = "boolean"
	case reflect.Slice, reflect.Array:
		prop.Type = "array"
		if t.Elem() != nil {
			itemProp := f.typeToProperty(t.Elem())
			prop.Items = &itemProp
		}
	case reflect.Map:
		prop.Type = "object"
	case reflect.Struct:
		prop.Type = "object"
		prop.Properties = make(map[string]types.Property)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}
			fieldName := field.Tag.Get("json")
			if fieldName == "" {
				fieldName = field.Name
			}
			prop.Properties[fieldName] = f.typeToProperty(field.Type)
		}
	case reflect.Ptr:
		if t.Elem() != nil {
			return f.typeToProperty(t.Elem())
		}
		prop.Type = "object"
	default:
		prop.Type = "string"
	}

	return prop
}

func (f *FunctionTool) Name() string {
	return f.name
}

func (f *FunctionTool) Description() string {
	return f.description
}

func (f *FunctionTool) Schema() types.ToolSchema {
	return f.schema
}

func (f *FunctionTool) RequiresPermission() bool {
	return f.requiresPermission
}

func (f *FunctionTool) SetRequiresPermission(requires bool) {
	f.requiresPermission = requires
}

func (f *FunctionTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	fnArgs := make([]reflect.Value, 0, f.fnType.NumIn())

	paramIndex := 0
	for i := 0; i < f.fnType.NumIn(); i++ {
		param := f.fnType.In(i)

		if param.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			fnArgs = append(fnArgs, reflect.ValueOf(ctx))
			continue
		}

		argName := fmt.Sprintf("arg%d", paramIndex)
		if val, ok := args[argName]; ok {
			convertedVal, err := f.convertToType(val, param)
			if err != nil {
				return nil, fmt.Errorf("failed to convert argument %s: %w", argName, err)
			}
			fnArgs = append(fnArgs, convertedVal)
		} else {
			fnArgs = append(fnArgs, reflect.Zero(param))
		}
		paramIndex++
	}

	results := f.fn.Call(fnArgs)

	if len(results) == 0 {
		return nil, nil
	}

	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return nil, lastResult.Interface().(error)
			}
		}
	}

	if results[0].IsValid() && !results[0].IsNil() {
		return results[0].Interface(), nil
	}

	return nil, nil
}

func (f *FunctionTool) convertToType(val interface{}, targetType reflect.Type) (reflect.Value, error) {
	if val == nil {
		return reflect.Zero(targetType), nil
	}

	valType := reflect.TypeOf(val)

	if valType == targetType {
		return reflect.ValueOf(val), nil
	}

	if valType.ConvertibleTo(targetType) {
		return reflect.ValueOf(val).Convert(targetType), nil
	}

	switch targetType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(targetType.Kind()).Convert(targetType), nil
		case float32:
			return reflect.ValueOf(int64(v)).Convert(targetType), nil
		default:
			if valType.Kind() >= reflect.Int && valType.Kind() <= reflect.Int64 {
				return reflect.ValueOf(val).Convert(targetType), nil
			}
		}

	case reflect.Float32, reflect.Float64:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(v).Convert(targetType), nil
		case float32:
			return reflect.ValueOf(float64(v)).Convert(targetType), nil
		case int:
			return reflect.ValueOf(float64(v)).Convert(targetType), nil
		}

	case reflect.String:
		return reflect.ValueOf(fmt.Sprintf("%v", val)), nil

	case reflect.Bool:
		if b, ok := val.(bool); ok {
			return reflect.ValueOf(b), nil
		}
	}

	return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to %s", val, targetType.String())
}

func Function(name, description string, fn interface{}) types.Tool {
	tool, err := NewFunctionTool(name, description, fn)
	if err != nil {
		panic(err)
	}
	return tool
}