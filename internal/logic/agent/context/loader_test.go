package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	root := t.TempDir()
	rulesDir := filepath.Join(root, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "project.md"), []byte("project memory"), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}

	ruleA := "---\npaths:\n  - internal/logic/agent/**\n---\nagent rule"
	if err := os.WriteFile(filepath.Join(rulesDir, "agent.md"), []byte(ruleA), 0o644); err != nil {
		t.Fatalf("write ruleA: %v", err)
	}

	ruleB := "---\npaths:\n  - internal/logic/agent/model/**\n---\nllm rule"
	if err := os.WriteFile(filepath.Join(rulesDir, "model.md"), []byte(ruleB), 0o644); err != nil {
		t.Fatalf("write ruleB: %v", err)
	}

	loader := NewLoader(root)
	bundle, err := loader.Load([]string{"internal/logic/agent/controller.go"})
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}

	if bundle.Project != "project memory" {
		t.Fatalf("unexpected project: %q", bundle.Project)
	}
	if len(bundle.Rules) != 1 {
		t.Fatalf("unexpected rules count: %d", len(bundle.Rules))
	}
	if bundle.Rules[0].Name != "agent" {
		t.Fatalf("unexpected rule: %+v", bundle.Rules[0])
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		ok      bool
	}{
		{pattern: "internal/logic/agent/**", value: "internal/logic/agent/controller.go", ok: true},
		{pattern: "internal/*/agent/**", value: "internal/logic/agent/controller.go", ok: true},
		{pattern: "internal/logic/agent/model/**", value: "internal/logic/agent/controller.go", ok: false},
	}

	for _, tc := range tests {
		got := matchPattern(tc.pattern, tc.value)
		if got != tc.ok {
			t.Fatalf("pattern=%q value=%q got=%v want=%v", tc.pattern, tc.value, got, tc.ok)
		}
	}
}
