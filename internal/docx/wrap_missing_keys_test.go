package docx

import (
	"testing"
)

func TestEscapeTemplateValues_String(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"a&b", "a&amp;b"},
		{"<tag>", "&lt;tag&gt;"},
		{`"quote"`, "&#34;quote&#34;"},
		{"'single'", "&#39;single&#39;"},
		{"a&b<c>d", "a&amp;b&lt;c&gt;d"},
		{"", ""},
	}
	for _, tt := range tests {
		got := EscapeTemplateValues(tt.input)
		if got != tt.expected {
			t.Errorf("EscapeTemplateValues(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestEscapeTemplateValues_Map(t *testing.T) {
	input := map[string]any{
		"name":    "foo&bar",
		"desc":    "<b>bold</b>",
		"number":  42,
		"nested": map[string]any{
			"inner": `a"b`,
		},
	}
	got := EscapeTemplateValues(input).(map[string]any)
	if got["name"] != "foo&amp;bar" {
		t.Errorf("name = %q, want %q", got["name"], "foo&amp;bar")
	}
	if got["desc"] != "&lt;b&gt;bold&lt;/b&gt;" {
		t.Errorf("desc = %q, want %q", got["desc"], "&lt;b&gt;bold&lt;/b&gt;")
	}
	if got["number"] != 42 {
		t.Errorf("number = %v, want 42", got["number"])
	}
	nested := got["nested"].(map[string]any)
	if nested["inner"] != "a&#34;b" {
		t.Errorf("inner = %q, want %q", nested["inner"], "a&#34;b")
	}
}

func TestEscapeTemplateValues_Slice(t *testing.T) {
	input := []any{"a&b", "<c>", 7}
	got := EscapeTemplateValues(input).([]any)
	if got[0] != "a&amp;b" {
		t.Errorf("got[0] = %q, want %q", got[0], "a&amp;b")
	}
	if got[1] != "&lt;c&gt;" {
		t.Errorf("got[1] = %q, want %q", got[1], "&lt;c&gt;")
	}
	if got[2] != 7 {
		t.Errorf("got[2] = %v, want 7", got[2])
	}
}

func TestEscapeTemplateValues_NonString(t *testing.T) {
	if got := EscapeTemplateValues(42); got != 42 {
		t.Errorf("got %v, want 42", got)
	}
	if got := EscapeTemplateValues(nil); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := EscapeTemplateValues(true); got != true {
		t.Errorf("got %v, want true", got)
	}
}

func TestEscapeTemplateValues_MapStringString(t *testing.T) {
	input := map[string]string{
		"key": `a&b"c`,
	}
	got := EscapeTemplateValues(input).(map[string]string)
	if got["key"] != input["key"] {
		t.Errorf("map[string]string should pass through unchanged, got %v", got)
	}
}
