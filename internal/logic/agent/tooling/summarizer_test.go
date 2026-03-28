package tooling

import "testing"

func TestDefaultToolSummarizer_Object(t *testing.T) {
	s := NewDefaultToolSummarizer()

	out := s.Summarize("demo", map[string]any{
		"student_id": "2301",
		"count":      12,
		"nested": map[string]any{
			"latest_rating": 1700,
			"max_rating":    1850,
		},
	})

	if out["tool"] != "demo" {
		t.Fatalf("unexpected tool: %v", out["tool"])
	}

	identity, ok := out["identity"].(map[string]any)
	if !ok {
		t.Fatalf("expected identity, got: %#v", out["identity"])
	}
	if identity["student_id"] != "2301" {
		t.Fatalf("unexpected student_id: %v", identity["student_id"])
	}

	objects, ok := out["objects"].(map[string]any)
	if !ok {
		t.Fatalf("expected objects, got: %#v", out["objects"])
	}

	nested, ok := objects["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested object summary, got: %#v", objects["nested"])
	}

	if nested["key_count"] != 2 {
		t.Fatalf("unexpected nested key_count: %v", nested["key_count"])
	}
}

func TestDefaultToolSummarizer_Array(t *testing.T) {
	s := NewDefaultToolSummarizer()

	out := s.Summarize("list_tool", []map[string]any{
		{"student_id": "1", "count": 10},
		{"student_id": "2", "count": 8},
	})

	arr, ok := out["array"].(map[string]any)
	if !ok {
		t.Fatalf("expected array summary, got: %#v", out["array"])
	}

	if arr["count"] != 2 {
		t.Fatalf("unexpected count: %v", arr["count"])
	}

	preview, ok := arr["preview"].([]any)
	if !ok || len(preview) == 0 {
		t.Fatalf("expected preview, got: %#v", arr["preview"])
	}
}
