package docx

import (
	"testing"
	"text/template"
)

func TestWrapMissingKeys_MapStringAny(t *testing.T) {
	data := map[string]any{"a": "hello", "b": nil}
	got := WrapMissingKeys(data, nil)
	if got["a"] != "hello" {
		t.Errorf("a = %q, want %q", got["a"], "hello")
	}
	if got["b"] != "" {
		t.Errorf("b = %q, want empty string", got["b"])
	}
}

func TestWrapMissingKeys_MapStringString(t *testing.T) {
	data := map[string]string{"a": "hello"}
	got := WrapMissingKeys(data, nil)
	if got["a"] != "hello" {
		t.Errorf("a = %q, want %q", got["a"], "hello")
	}
}

func TestWrapMissingKeys_DefaultReturnsNil(t *testing.T) {
	if got := WrapMissingKeys(42, nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
	if got := WrapMissingKeys("str", nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
	if got := WrapMissingKeys(nil, nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestWrapMissingKeys_AddsMissingFromTemplate(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(`{{.Name}} {{.Age}}`))
	data := map[string]any{"Name": "John"}
	got := WrapMissingKeys(data, tmpl)
	if got["Name"] != "John" {
		t.Errorf("Name = %q, want %q", got["Name"], "John")
	}
	if got["Age"] != "" {
		t.Errorf("Age = %q, want empty string", got["Age"])
	}
}

func TestWrapMissingKeys_DoesNotOverwriteExisting(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(`{{.Name}}`))
	data := map[string]any{"Name": "John"}
	got := WrapMissingKeys(data, tmpl)
	if got["Name"] != "John" {
		t.Errorf("Name = %q, want %q", got["Name"], "John")
	}
}

func TestWrapMissingKeys_NilTemplate(t *testing.T) {
	data := map[string]any{"a": "val"}
	got := WrapMissingKeys(data, nil)
	if len(got) != 1 {
		t.Errorf("expected 1 key, got %d", len(got))
	}
}

func TestWrapMissingKeys_TemplateWithRangeAndIf(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(
		`{{range .Items}}{{.Name}}{{end}}{{if .Visible}}yes{{end}}`,
	))
	data := map[string]any{}
	got := WrapMissingKeys(data, tmpl)
	if _, ok := got["Items"]; !ok {
		t.Error("expected Items key from range")
	}
	if _, ok := got["Visible"]; !ok {
		t.Error("expected Visible key from if")
	}
}

func TestWrapMissingKeys_EmptyTemplate(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(``))
	data := map[string]any{"a": "b"}
	got := WrapMissingKeys(data, tmpl)
	if got["a"] != "b" {
		t.Errorf("a = %q, want %q", got["a"], "b")
	}
}

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
		"name":   "foo&bar",
		"desc":   "<b>bold</b>",
		"number": 42,
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
