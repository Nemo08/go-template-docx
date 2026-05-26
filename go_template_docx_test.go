package gotemplatedocx

import (
	"log/slog"
	"os"
	"text/template"
	"testing"

	"github.com/JJJJJJack/go-template-docx/internal/docx"
)

func TestCopyTemplateFuncs(t *testing.T) {
	src := template.FuncMap{
		"fn1": func() string { return "hello" },
		"fn2": func() string { return "world" },
	}
	dst := copyTemplateFuncs(src)
	if len(dst) != 2 {
		t.Errorf("expected 2 funcs, got %d", len(dst))
	}
	// Modify original should not affect copy
	src["fn3"] = func() string { return "new" }
	if len(dst) != 2 {
		t.Errorf("copy should have 2 funcs after original modified, got %d", len(dst))
	}
}

func TestCopyTemplateFuncs_Empty(t *testing.T) {
	src := template.FuncMap{}
	dst := copyTemplateFuncs(src)
	if len(dst) != 0 {
		t.Errorf("expected empty copy, got %d", len(dst))
	}
}

func TestToStringMap_MapStringAny(t *testing.T) {
	input := map[string]any{"key": "val"}
	got := toStringMap(input)
	if got["key"] != "val" {
		t.Errorf("expected val, got %v", got["key"])
	}
}

func TestToStringMap_MapStringString(t *testing.T) {
	input := map[string]string{"key": "val"}
	got := toStringMap(input)
	if got["key"] != "val" {
		t.Errorf("expected val, got %v", got["key"])
	}
}

func TestToStringMap_JSONBytes(t *testing.T) {
	input := []byte(`{"key":"val"}`)
	got := toStringMap(input)
	if got["key"] != "val" {
		t.Errorf("expected val, got %v", got["key"])
	}
}

func TestToStringMap_EmptyBytes(t *testing.T) {
	input := []byte{}
	got := toStringMap(input)
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestToStringMap_InvalidBytes(t *testing.T) {
	input := []byte(`not json`)
	got := toStringMap(input)
	if got != nil {
		t.Errorf("expected nil for invalid json, got %v", got)
	}
}

func TestToStringMap_Struct(t *testing.T) {
	input := struct {
		Name string `json:"name"`
	}{Name: "test"}
	got := toStringMap(input)
	if got["name"] != "test" {
		t.Errorf("expected test, got %v", got["name"])
	}
}

func TestToStringMap_NonMapType(t *testing.T) {
	got := toStringMap(42)
	if got != nil {
		t.Errorf("expected nil for int, got %v", got)
	}
}

func TestNormalizeTemplateValues_MapStringAny(t *testing.T) {
	dt := &docxTemplate{}
	input := map[string]any{"key": "val"}
	got := dt.normalizeTemplateValues(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["key"] != "val" {
		t.Errorf("expected val, got %v", m["key"])
	}
}

func TestNormalizeTemplateValues_NilValues(t *testing.T) {
	// normalizeTemplateValues doesn't handle nil replacement itself -
	// this is done in Apply. For map[string]any, it returns as-is.
	dt := &docxTemplate{}
	input := map[string]any{"key": nil}
	got := dt.normalizeTemplateValues(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	// Pass through - nil stays nil
	if m["key"] != nil {
		t.Errorf("expected nil preserved, got %v", m["key"])
	}
}

func TestNormalizeTemplateValues_MapStringString(t *testing.T) {
	dt := &docxTemplate{}
	input := map[string]string{"key": "val"}
	got := dt.normalizeTemplateValues(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["key"] != "val" {
		t.Errorf("expected val, got %v", m["key"])
	}
}

func TestNormalizeTemplateValues_JSONBytes(t *testing.T) {
	dt := &docxTemplate{}
	input := []byte(`{"key":"val"}`)
	got := dt.normalizeTemplateValues(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["key"] != "val" {
		t.Errorf("expected val, got %v", m["key"])
	}
}

func TestNormalizeTemplateValues_EmptyJSONBytes(t *testing.T) {
	dt := &docxTemplate{}
	input := []byte(`{}`)
	got := dt.normalizeTemplateValues(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestNormalizeTemplateValues_Passthrough(t *testing.T) {
	dt := &docxTemplate{}
	input := 42
	got := dt.normalizeTemplateValues(input)
	if got != 42 {
		t.Errorf("expected passthrough, got %v", got)
	}
}

func TestWarnMissingKeysInFile_NoMissing(t *testing.T) {
	dt := &docxTemplate{filename: "test.docx", missingKeyLogger: slog.New(slog.NewTextHandler(os.Stderr, nil))}
	tmpl := template.Must(template.New("test").Parse(`{{.Name}}`))
	data := map[string]any{"Name": "test"}
	// Should not panic or error
	dt.warnMissingKeysInFile(tmpl, data)
}

func TestWarnMissingKeysInFile_SkipsVars(t *testing.T) {
	dt := &docxTemplate{filename: "test.docx", missingKeyLogger: slog.New(slog.NewTextHandler(os.Stderr, nil))}
	// $i and $v are loop variables, should be skipped by warnMissingKeysInFile
	tmpl := template.Must(template.New("test").Parse(`{{range $i, $v := .Items}}{{$i}}{{$v}}{{end}}`))
	data := map[string]any{}
	// $vars should be skipped, not warned - should not panic
	dt.warnMissingKeysInFile(tmpl, data)
}

func TestNewDocxTemplateFromBytes_Defaults(t *testing.T) {
	docxBytes := []byte("not a real docx")
	tpl, err := NewDocxTemplateFromBytes(docxBytes)
	if err != nil {
		t.Fatal(err)
	}
	if tpl == nil {
		t.Fatal("expected non-nil template")
	}
	if !tpl.removeEmptyTableRows {
		t.Error("expected removeEmptyTableRows default to true")
	}
	if tpl.ignoreMissingKey {
		t.Error("expected ignoreMissingKey default to false")
	}
	if tpl.warnOnMissingKey {
		t.Error("expected warnOnMissingKey default to false")
	}
}

func TestNewDocxTemplateFromBytes_WithOptions(t *testing.T) {
	docxBytes := []byte("not a real docx")
	tpl, err := NewDocxTemplateFromBytes(docxBytes, IgnoreMissingKey(), NoRemoveEmptyTableRows())
	if err != nil {
		t.Fatal(err)
	}
	if !tpl.ignoreMissingKey {
		t.Error("expected ignoreMissingKey true after option")
	}
	if tpl.removeEmptyTableRows {
		t.Error("expected removeEmptyTableRows false after NoRemoveEmptyTableRows")
	}
}

func TestMedia(t *testing.T) {
	dt := &docxTemplate{media: make(docx.MediaMap)}
	dt.Media("test.png", []byte("image data"))
	if len(dt.media) != 1 {
		t.Errorf("expected 1 media, got %d", len(dt.media))
	}
	if dt.media["test.png"] == nil {
		t.Error("expected media entry")
	}
}

func TestAddTemplateFuncs(t *testing.T) {
	dt := &docxTemplate{templateFuncs: copyTemplateFuncs(docx.TemplateFuncs)}
	dt.AddTemplateFuncs(template.FuncMap{
		"custom": func() string { return "custom" },
	})
	if _, ok := dt.templateFuncs["custom"]; !ok {
		t.Error("expected custom func to be added")
	}
}

func TestSaveAndBytes(t *testing.T) {
	dt := &docxTemplate{}
	// Bytes should be empty before Apply
	if len(dt.Bytes()) != 0 {
		t.Error("expected empty bytes before Apply")
	}
}
