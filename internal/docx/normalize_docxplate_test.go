package docx

import (
	"testing"
)

func TestNormalizeDocxplateHandler_DocxplateSyntax(t *testing.T) {
	input := `{{Pages.Name}}`
	expected := `{{.Pages.Name}}`
	got, err := normalizeDocxplateHandler(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestNormalizeDocxplateHandler_DocxplateWithWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{{ Pages.Name }}`, `{{.Pages.Name}}`},
		{`{{Pages.Name }}`, `{{.Pages.Name}}`},
		{`{{ Pages.Name}}`, `{{.Pages.Name}}`},
	}
	for _, tt := range tests {
		got, err := normalizeDocxplateHandler(tt.input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != tt.expected {
			t.Errorf("normalize(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeDocxplateHandler_StandardGoSyntaxUntouched(t *testing.T) {
	inputs := []string{
		`{{.VarName}}`,
		`{{VarName}}`,
		`{{range .Items}}`,
		`{{if .Cond}}`,
		`{{add 1 2}}`,
		`{{index .Arr 0}}`,
		`{{.Nested.Field}}`,
		`{{template "name" .}}`,
		`{{/* comment */}}`,
	}
	for _, input := range inputs {
		got, err := normalizeDocxplateHandler(input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", input, err)
		}
		if got != input {
			t.Errorf("should not modify %q, got %q", input, got)
		}
	}
}

func TestNormalizeDocxplateHandler_AlreadyHasLeadingDot(t *testing.T) {
	input := `{{.Pages.Name}}`
	got, err := normalizeDocxplateHandler(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("should not modify %q, got %q", input, got)
	}
}

func TestDocxplateCompatPreProcessor_ReturnsWildcardKey(t *testing.T) {
	hm := DocxplateCompatPreProcessor()
	handlers, ok := hm["*"]
	if !ok {
		t.Fatal("expected wildcard key \"*\" in HandlersMap")
	}
	if len(handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(handlers))
	}
}
