package tools

import (
	"context"
	"testing"
)

func newTestTool(name string) Tool {
	return NewTool(name, "desc", func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
		return name, nil
	})
}

func TestRegistryRegisterAndResolve(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(newTestTool("a"), newTestTool("b")); err != nil {
		t.Fatalf("Register: %v", err)
	}

	resolved, err := r.Resolve([]string{"b", "a"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolved) != 2 || resolved[0].Name() != "b" || resolved[1].Name() != "a" {
		t.Fatalf("resolve order wrong: %v", resolved)
	}
}

func TestRegistryDuplicate(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(newTestTool("a"))
	if err := r.Register(newTestTool("a")); err == nil {
		t.Error("expected duplicate registration error")
	}
}

func TestRegistryResolveMissing(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(newTestTool("a"))
	if _, err := r.Resolve([]string{"a", "x", "y"}); err == nil {
		t.Error("expected error for unknown tools")
	}
}

func TestSimpleToolExecute(t *testing.T) {
	tool := NewTool("echo", "echoes msg", func(_ context.Context, args map[string]interface{}) (interface{}, error) {
		return args["msg"], nil
	}, Param{Name: "msg", Type: "string", Required: true})

	if got := tool.Schema().Required; len(got) != 1 || got[0] != "msg" {
		t.Errorf("required = %v", got)
	}
	out, err := tool.Execute(context.Background(), map[string]interface{}{"msg": "hi"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "hi" {
		t.Errorf("out = %v", out)
	}
}
