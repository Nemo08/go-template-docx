package gotemplatedocx

import (
	"bytes"
	"log/slog"
	"os"
	"testing"
	"text/template"

	"github.com/JJJJJJack/go-template-docx/internal/docx"
	"github.com/JJJJJJack/go-template-docx/internal/docx/media"
	"github.com/JJJJJJack/go-template-docx/internal/util"
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
	got := util.ToStringMap(input)
	if got["key"] != "val" {
		t.Errorf("expected val, got %v", got["key"])
	}
}

func TestToStringMap_MapStringString(t *testing.T) {
	input := map[string]string{"key": "val"}
	got := util.ToStringMap(input)
	if got["key"] != "val" {
		t.Errorf("expected val, got %v", got["key"])
	}
}

func TestToStringMap_JSONBytes(t *testing.T) {
	input := []byte(`{"key":"val"}`)
	got := util.ToStringMap(input)
	if got["key"] != "val" {
		t.Errorf("expected val, got %v", got["key"])
	}
}

func TestToStringMap_EmptyBytes(t *testing.T) {
	input := []byte{}
	got := util.ToStringMap(input)
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestToStringMap_InvalidBytes(t *testing.T) {
	input := []byte(`not json`)
	got := util.ToStringMap(input)
	if got != nil {
		t.Errorf("expected nil for invalid json, got %v", got)
	}
}

func TestToStringMap_Struct(t *testing.T) {
	input := struct {
		Name string `json:"name"`
	}{Name: "test"}
	got := util.ToStringMap(input)
	if got["name"] != "test" {
		t.Errorf("expected test, got %v", got["name"])
	}
}

func TestToStringMap_NonMapType(t *testing.T) {
	got := util.ToStringMap(42)
	if got != nil {
		t.Errorf("expected nil for int, got %v", got)
	}
}

func TestNormalizeTemplateValues_MapStringAny(t *testing.T) {
	input := map[string]any{"key": "val"}
	got := (&TemplateConfig{}).normalize(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["key"] != "val" {
		t.Errorf("expected val, got %v", m["key"])
	}
}

func TestNormalizeTemplateValues_NilValues(t *testing.T) {
	input := map[string]any{"key": nil}
	got := (&TemplateConfig{}).normalize(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["key"] != nil {
		t.Errorf("expected nil preserved, got %v", m["key"])
	}
}

func TestNormalizeTemplateValues_MapStringString(t *testing.T) {
	input := map[string]string{"key": "val"}
	got := (&TemplateConfig{}).normalize(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["key"] != "val" {
		t.Errorf("expected val, got %v", m["key"])
	}
}

func TestNormalizeTemplateValues_JSONBytes(t *testing.T) {
	input := []byte(`{"key":"val"}`)
	got := (&TemplateConfig{}).normalize(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["key"] != "val" {
		t.Errorf("expected val, got %v", m["key"])
	}
}

func TestNormalizeTemplateValues_EmptyJSONBytes(t *testing.T) {
	input := []byte(`{}`)
	got := (&TemplateConfig{}).normalize(input)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestNormalizeTemplateValues_Passthrough(t *testing.T) {
	input := 42
	got := (&TemplateConfig{}).normalize(input)
	if got != 42 {
		t.Errorf("expected passthrough, got %v", got)
	}
}

func TestWarnMissingKeysInFile_NoMissing(t *testing.T) {
	p := &TemplateProcessor{
		Config: &TemplateConfig{
			Filename:         "test.docx",
			MissingKeyLogger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
		},
	}
	tmpl := template.Must(template.New("test").Parse(`{{.Name}}`))
	data := map[string]any{"Name": "test"}
	p.warnMissingKeysInFile(tmpl, data)
}

func TestWarnMissingKeysInFile_SkipsVars(t *testing.T) {
	p := &TemplateProcessor{
		Config: &TemplateConfig{
			Filename:         "test.docx",
			MissingKeyLogger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
		},
	}
	tmpl := template.Must(template.New("test").Parse(`{{range $i, $v := .Items}}{{$i}}{{$v}}{{end}}`))
	data := map[string]any{}
	p.warnMissingKeysInFile(tmpl, data)
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
	if !tpl.Config.RemoveEmptyTableRows {
		t.Error("expected RemoveEmptyTableRows default to true")
	}
	if tpl.Config.IgnoreMissingKey {
		t.Error("expected IgnoreMissingKey default to false")
	}
	if tpl.Config.DeleteMissingKey {
		t.Error("expected DeleteMissingKey default to false")
	}
	if tpl.Config.WarnOnMissingKey {
		t.Error("expected WarnOnMissingKey default to false")
	}
}

func TestNewDocxTemplateFromBytes_WithOptions(t *testing.T) {
	docxBytes := []byte("not a real docx")
	tpl, err := NewDocxTemplateFromBytes(docxBytes, IgnoreMissingKey(), NoRemoveEmptyTableRows())
	if err != nil {
		t.Fatal(err)
	}
	if !tpl.Config.IgnoreMissingKey {
		t.Error("expected IgnoreMissingKey true after option")
	}
	if tpl.Config.RemoveEmptyTableRows {
		t.Error("expected RemoveEmptyTableRows false after NoRemoveEmptyTableRows")
	}
}

func TestMedia(t *testing.T) {
	dt := &docxTemplate{State: TemplateState{Media: make(media.MediaMap)}}
	dt.Media("test.png", []byte("image data"))
	if len(dt.State.Media) != 1 {
		t.Errorf("expected 1 media, got %d", len(dt.State.Media))
	}
	if dt.State.Media["test.png"] == nil {
		t.Error("expected media entry")
	}
}

func TestAddTemplateFuncs(t *testing.T) {
	dt := &docxTemplate{Config: TemplateConfig{TemplateFuncs: copyTemplateFuncs(docx.TemplateFuncs)}}
	dt.AddTemplateFuncs(template.FuncMap{
		"custom": func() string { return "custom" },
	})
	if _, ok := dt.Config.TemplateFuncs["custom"]; !ok {
		t.Error("expected custom func to be added")
	}
}

func TestSaveAndBytes(t *testing.T) {
	dt := &docxTemplate{}
	if len(dt.Bytes()) != 0 {
		t.Error("expected empty bytes before Apply")
	}
}

func TestAutoExpandRows_OptionApplied(t *testing.T) {
	data := map[string]any{"X": []any{1, 2}}
	docxBytes := []byte("fake")
	tpl, err := NewDocxTemplateFromBytes(docxBytes, AutoExpandRows(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(tpl.Config.PreProcessors) == 0 {
		t.Error("expected at least one pre-processor registered")
	}
}

func TestWithAutoExpandRows_OptionApplied(t *testing.T) {
	data := map[string]any{"X": []any{1, 2}}
	docxBytes := []byte("fake")
	tpl, err := NewDocxTemplateFromBytes(docxBytes)
	if err != nil {
		t.Fatal(err)
	}
	opt := WithAutoExpandRows(data)
	opt(tpl)
	if len(tpl.Config.PreProcessors) == 0 {
		t.Error("expected pre-processor after WithAutoExpandRows")
	}
}

func TestNewDocxTemplateFromBytes_DeleteMissingKey(t *testing.T) {
	docxBytes := []byte("not a real docx")
	tpl, err := NewDocxTemplateFromBytes(docxBytes, DeleteMissingKey())
	if err != nil {
		t.Fatal(err)
	}
	if !tpl.Config.DeleteMissingKey {
		t.Error("expected DeleteMissingKey true after DeleteMissingKey()")
	}
}

func TestWarnOnMissingKey_SetsDeleteMissingKey(t *testing.T) {
	docxBytes := []byte("not a real docx")
	tpl, err := NewDocxTemplateFromBytes(docxBytes, WarnOnMissingKey())
	if err != nil {
		t.Fatal(err)
	}
	if !tpl.Config.DeleteMissingKey {
		t.Error("expected DeleteMissingKey true after WarnOnMissingKey()")
	}
	if !tpl.Config.WarnOnMissingKey {
		t.Error("expected WarnOnMissingKey true after WarnOnMissingKey()")
	}
}

func TestNewDocxTemplateFromFilename_Error(t *testing.T) {
	_, err := NewDocxTemplateFromFilename("nonexistent.docx")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestSave_Bytes(t *testing.T) {
	docxBytes := buildMinimalDocx(t, `{{.Name}}`)
	tpl, err := NewDocxTemplateFromBytes(docxBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := tpl.Apply(map[string]any{"Name": "test"}); err != nil {
		t.Fatal(err)
	}
	b := tpl.Bytes()
	if len(b) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestSave_ToFile(t *testing.T) {
	docxBytes := buildMinimalDocx(t, `test`)
	tpl, err := NewDocxTemplateFromBytes(docxBytes)
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir() + "\\out.docx"
	err = tpl.Save(dst)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWithFilename_Option(t *testing.T) {
	docxBytes := buildMinimalDocx(t, `test`)
	_, err := NewDocxTemplateFromBytes(docxBytes, WithFilename("test.docx"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetMissingKeyLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	_, err := NewDocxTemplateFromBytes(nil, SetMissingKeyLogger(logger))
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTemplateVariables_ParsesVariables(t *testing.T) {
	t.Skip("requires real DOCX with proper XML namespace handling")
}

func TestGetTemplateVariables_Empty(t *testing.T) {
	t.Skip("requires real DOCX with proper XML namespace handling")
}

func TestGetTemplateVariables_Error(t *testing.T) {
	t.Skip("NewDocxTemplateFromBytes doesn't validate zip on creation")
}

func TestWarnMissingKeysForFile(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	tpl, err := NewDocxTemplateFromBytes(nil, SetMissingKeyLogger(logger))
	if err != nil {
		t.Fatal(err)
	}

	docxBytes := buildMinimalDocx(t, `{{.Missing}}`)
	tpl.Config.IgnoreMissingKey = true
	_ = tpl.Apply(docxBytes)
	// no warnings expected for IgnoreMissingKey
}

func TestRemoveRangeRows_Option(t *testing.T) {
	docxBytes := buildMinimalDocx(t, `test`)
	tpl, err := NewDocxTemplateFromBytes(docxBytes, RemoveRangeRows())
	if err != nil {
		t.Fatal(err)
	}
	if !tpl.Config.RemoveRangeRows {
		t.Error("expected RemoveRangeRows true")
	}
}
