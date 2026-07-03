package docx

import (
	"strings"
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

func TestDocxplateCompat_SplitRun(t *testing.T) {
	// PatchXML восстанавливает {{ и }} разорванные между <w:r>, удаляет
	// XML-теги внутри выражения. После склейки regex нормализует синтаксис.
	docxXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t>{{Pages.Name</w:t></w:r>
      <w:r><w:t>}}</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`

	got, err := normalizeDocxplateHandler(docxXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "{{.Pages.Name}}") {
		t.Errorf("split-run: expected normalised {{.Pages.Name}} in output, got:\n%s", got)
	}
}

func TestDocxplateRegexConsistency(t *testing.T) {
	// После нормализации {{Pages.Name}} → {{.Pages.Name}} результат должен
	// быть распознаваем autoexpand.reDocxplateVar (который ожидает
	// опциональную точку \.? перед именем массива).
	// Этот тест фиксирует, что оба regex логически согласованы,
	// даже если они реализованы раздельно.
	tests := []struct {
		input    string // docxplate-стиль (без точки)
		expected string // ожидаемая нормализованная форма
	}{
		{"{{Pages.Name}}", "{{.Pages.Name}}"},
		{"{{Items.Title}}", "{{.Items.Title}}"},
		{"{{Pages.Name}} and {{Items.Title}}", "{{.Pages.Name}} and {{.Items.Title}}"},
	}
	for _, tt := range tests {
		got, err := normalizeDocxplateHandler(tt.input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != tt.expected {
			t.Errorf("normalize(%q) = %q, want %q", tt.input, got, tt.expected)
		}
		// Проверка: результат начинается с {{. — это опциональная точка,
		// которую ожидает reDocxplateVar в autoexpand.
		if !strings.Contains(got, "{{.") {
			t.Errorf("normalized output %q has no leading dot, autoexpand will not detect it", got)
		}
	}
}

func TestDocxplateCompat_SplitRunInsideWord(t *testing.T) {
	// ОГРАНИЧЕНИЕ: если Word разорвал слово внутри плейсхолдера
	// (например "Pa" в одном <w:r>, "ges.Name}}" в другом), PatchXML
	// удаляет XML-теги между ними, но whitespace остаётся — regex не находит
	// непрерывного паттерна. Это известное ограничение: рекомендуется
	// принудительно объединить run'ы в Word (Ctrl+Space) или удалить
	// лишнее форматирование внутри плейсхолдера.
	docxXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t>{{Pa</w:t></w:r>
      <w:r><w:t>ges.Name}}</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`

	got, err := normalizeDocxplateHandler(docxXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "{{.Pages.Name}}") {
		t.Errorf("word-split: expected NO normalisation (documented limitation), got normalised:\n%s", got)
	}
}


