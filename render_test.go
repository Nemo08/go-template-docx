package gotemplatedocx

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"text/template"

	"github.com/JJJJJJack/go-template-docx/internal/docx/media"
	"github.com/JJJJJJack/go-template-docx/internal/zio"
	"github.com/JJJJJJack/go-template-docx/internal/xml"
)

func TestRender_InvalidBytes(t *testing.T) {
	result, err := Render([]byte("not a zip"), nil)
	if err == nil {
		t.Fatal("expected error for invalid bytes")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %d bytes", len(result))
	}
}

func TestRender_EmptyBytes(t *testing.T) {
	result, err := Render([]byte{}, nil)
	if err == nil {
		t.Fatal("expected error for empty bytes")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %d bytes", len(result))
	}
}

func TestRender_NilOpts(t *testing.T) {
	result, err := Render([]byte("not a zip"), nil)
	if err == nil {
		t.Fatal("expected error for invalid bytes even with nil opts")
	}
	if result != nil {
		t.Errorf("expected nil result on error")
	}
}

func TestRender_WithFuncs(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Name}}`)
	result, err := Render(minimal, map[string]any{"Name": "test"},
		WithFuncs(template.FuncMap{
			"custom": func(s string) string { return "custom:" + s },
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestRender_WithIgnoreMissingKey(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Present}}`)
	// Without option - should work since Present exists
	result, err := Render(minimal, map[string]any{"Present": "hello"},
		WithIgnoreMissingKey(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestRender_WithPreProcessors(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Name}}`)
	result, err := Render(minimal, map[string]any{"Name": "value"},
		WithPreProcessors(xml.HandlersMap{
			"word/document.xml": {
				func(content string) (string, error) {
					return content, nil
				},
			},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestRender_WithPostProcessors(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Name}}`)
	result, err := Render(minimal, map[string]any{"Name": "value"},
		WithPostProcessors(xml.HandlersMap{
			"word/document.xml": {
				func(content string) (string, error) {
					return content, nil
				},
			},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestRender_WithMultipleOpts(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Name}}`)
	result, err := Render(minimal, map[string]any{"Name": "hello"},
		WithIgnoreMissingKey(true),
		WithRemoveEmptyTableRows(false),
		WithRemoveRangeRows(true),
		WithFuncs(template.FuncMap{"upper": func(s string) string { return s }}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestWithFuncs_SetsFuncMap(t *testing.T) {
	tpl := &docxTemplate{Config: TemplateConfig{TemplateFuncs: copyTemplateFuncs(make(template.FuncMap))}}
	fn := func() string { return "result" }
	WithFuncs(template.FuncMap{"myfn": fn})(tpl)
	if _, ok := tpl.Config.TemplateFuncs["myfn"]; !ok {
		t.Error("expected myfn to be added")
	}
}

func TestWithImage_SetsMedia(t *testing.T) {
	tpl := &docxTemplate{State: TemplateState{Media: make(media.MediaMap)}}
	WithImage("test.png", []byte("data"))(tpl)
	if len(tpl.State.Media) != 1 {
		t.Errorf("expected 1 media, got %d", len(tpl.State.Media))
	}
	if tpl.State.Media["test.png"] == nil {
		t.Error("expected media entry")
	}
}

func TestWithPreProcessors_Appends(t *testing.T) {
	tpl := &docxTemplate{
		Config: TemplateConfig{PreProcessors: []xml.HandlersMap{}},
	}
	m := xml.HandlersMap{"test.xml": {}}
	WithPreProcessors(m)(tpl)
	if len(tpl.Config.PreProcessors) != 1 {
		t.Errorf("expected 1 pre-processor map, got %d", len(tpl.Config.PreProcessors))
	}
}

func TestWithPostProcessors_Appends(t *testing.T) {
	tpl := &docxTemplate{
		Config: TemplateConfig{PostProcessors: []xml.HandlersMap{}},
	}
	m := xml.HandlersMap{"test.xml": {}}
	WithPostProcessors(m)(tpl)
	if len(tpl.Config.PostProcessors) != 1 {
		t.Errorf("expected 1 post-processor map, got %d", len(tpl.Config.PostProcessors))
	}
}

func TestWithRemoveEmptyTableRows_SetsFalse(t *testing.T) {
	tpl := &docxTemplate{}
	WithRemoveEmptyTableRows(false)(tpl)
	if tpl.Config.RemoveEmptyTableRows {
		t.Error("expected RemoveEmptyTableRows false")
	}
}

func TestWithRemoveEmptyTableRows_SetsTrue(t *testing.T) {
	tpl := &docxTemplate{}
	WithRemoveEmptyTableRows(true)(tpl)
	if !tpl.Config.RemoveEmptyTableRows {
		t.Error("expected RemoveEmptyTableRows true")
	}
}

func TestWithRemoveRangeRows_SetsTrue(t *testing.T) {
	tpl := &docxTemplate{}
	WithRemoveRangeRows(true)(tpl)
	if !tpl.Config.RemoveRangeRows {
		t.Error("expected RemoveRangeRows true")
	}
}

func TestRender_DynamicOpts(t *testing.T) {
	// Build opts slice dynamically (images in a loop, etc.)
	minimal := buildMinimalDocx(t, `{{.Name}}`)
	var opts []RenderOption
	opts = append(opts, WithFuncs(template.FuncMap{
		"upper": func(s string) string { return s },
	}))
	opts = append(opts, WithIgnoreMissingKey(true))
	result, err := Render(minimal, map[string]any{"Name": "hello"}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestNewRenderBuilder_Apply(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Name}}`)
	b, err := NewRenderBuilder(minimal)
	if err != nil {
		t.Fatal(err)
	}
	result, err := b.Apply(map[string]any{"Name": "builder"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestRenderBuilder_WithFuncs(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Name}}`)
	b, err := NewRenderBuilder(minimal)
	if err != nil {
		t.Fatal(err)
	}
	b.WithFuncs(template.FuncMap{"custom": func(s string) string { return s }})
	result, err := b.Apply(map[string]any{"Name": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestRenderBuilder_Chained(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Name}}`)
	b, err := NewRenderBuilder(minimal)
	if err != nil {
		t.Fatal(err)
	}
	result, err := b.
		WithIgnoreMissingKey(true).
		WithRemoveEmptyTableRows(false).
		Apply(map[string]any{"Name": "chain"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestRenderBuilder_DynamicImageLoop(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Name}}`)
	b, err := NewRenderBuilder(minimal)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate dynamic image loading in a loop
	images := map[string][]byte{
		"img1.png": {1, 2, 3},
		"img2.png": {4, 5, 6},
	}
	for name, data := range images {
		b.WithImage(name, data)
	}
	result, err := b.Apply(map[string]any{"Name": "dynamic"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestRenderBuilder_InvalidBytes(t *testing.T) {
	b, err := NewRenderBuilder([]byte("not a zip"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.Apply(nil)
	if err == nil {
		t.Fatal("expected error on Apply for invalid bytes")
	}
}

func TestWithIgnoreMissingKey_SetsTrue(t *testing.T) {
	tpl := &docxTemplate{}
	WithIgnoreMissingKey(true)(tpl)
	if !tpl.Config.IgnoreMissingKey {
		t.Error("expected IgnoreMissingKey true")
	}
}

func TestWithDeleteMissingKey_SetsTrue(t *testing.T) {
	tpl := &docxTemplate{}
	WithDeleteMissingKey()(tpl)
	if !tpl.Config.DeleteMissingKey {
		t.Error("expected DeleteMissingKey true")
	}
}

func TestRender_IgnoreMissingKeyPreservesPlaceholder(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Missing}}`)
	result, err := Render(minimal, map[string]any{},
		WithIgnoreMissingKey(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	docContent := extractDocxContent(t, result)
	if !strings.Contains(docContent, "{{.Missing}}") {
		t.Errorf("expected placeholder {{.Missing}} preserved, got:\n%s", docContent)
	}
}

func TestRender_DeleteMissingKeyReplacesWithEmpty(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Missing}}`)
	result, err := Render(minimal, map[string]any{},
		WithDeleteMissingKey(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	docContent := extractDocxContent(t, result)
	if strings.Contains(docContent, "{{.Missing}}") {
		t.Errorf("expected placeholder {{.Missing}} to be deleted, got:\n%s", docContent)
	}
}

func TestRender_DefaultErrorsOnMissingKey(t *testing.T) {
	minimal := buildMinimalDocx(t, `{{.Missing}}`)
	_, err := Render(minimal, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing key without options")
	}
}

// extractDocxContent reads word/document.xml from a DOCX zip byte slice.
func extractDocxContent(t *testing.T, docxBytes []byte) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open %s: %v", f.Name, err)
			}
			defer func() { _ = rc.Close() }()
			data, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("failed to read %s: %v", f.Name, err)
			}
			return string(data)
		}
	}
	t.Fatal("word/document.xml not found in DOCX")
	return ""
}

// buildMinimalDocx creates a minimal valid DOCX file for testing.
func buildMinimalDocx(t *testing.T, documentBody string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zio.NewZipWriter(&buf)

	// [Content_Types].xml
	ct, _ := w.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))

	// word/_rels/document.xml.rels
	rels, _ := w.Create("word/_rels/document.xml.rels")
	_, _ = rels.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`))

	// word/document.xml
	doc, _ := w.Create("word/document.xml")
	_, _ = doc.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>` + documentBody + `</w:t>
      </w:r>
    </w:p>
    <w:sectPr>
      <w:pgSz w:w="12240" w:h="15840"/>
      <w:pgMar w:top="1440" w:bottom="1440" w:left="1440" w:right="1440"/>
    </w:sectPr>
  </w:body>
</w:document>`))

	_ = w.Close()
	return buf.Bytes()
}
