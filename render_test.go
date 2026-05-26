package gotemplatedocx

import (
	"archive/zip"
	"bytes"
	"testing"
	"text/template"

	"github.com/JJJJJJack/go-template-docx/internal/docx"
	"github.com/JJJJJJack/go-template-docx/xml"
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
	tpl := &docxTemplate{templateFuncs: copyTemplateFuncs(make(template.FuncMap))}
	fn := func() string { return "result" }
	WithFuncs(template.FuncMap{"myfn": fn})(tpl)
	if _, ok := tpl.templateFuncs["myfn"]; !ok {
		t.Error("expected myfn to be added")
	}
}

func TestWithImage_SetsMedia(t *testing.T) {
	tpl := &docxTemplate{media: make(docx.MediaMap)}
	WithImage("test.png", []byte("data"))(tpl)
	if len(tpl.media) != 1 {
		t.Errorf("expected 1 media, got %d", len(tpl.media))
	}
	if tpl.media["test.png"] == nil {
		t.Error("expected media entry")
	}
}

func TestWithPreProcessors_Appends(t *testing.T) {
	tpl := &docxTemplate{
		filesPreProcessors: []xml.HandlersMap{},
	}
	m := xml.HandlersMap{"test.xml": {}}
	WithPreProcessors(m)(tpl)
	if len(tpl.filesPreProcessors) != 1 {
		t.Errorf("expected 1 pre-processor map, got %d", len(tpl.filesPreProcessors))
	}
}

func TestWithPostProcessors_Appends(t *testing.T) {
	tpl := &docxTemplate{
		filesPostProcessors: []xml.HandlersMap{},
	}
	m := xml.HandlersMap{"test.xml": {}}
	WithPostProcessors(m)(tpl)
	if len(tpl.filesPostProcessors) != 1 {
		t.Errorf("expected 1 post-processor map, got %d", len(tpl.filesPostProcessors))
	}
}

func TestWithRemoveEmptyTableRows_SetsFalse(t *testing.T) {
	tpl := &docxTemplate{}
	WithRemoveEmptyTableRows(false)(tpl)
	if tpl.removeEmptyTableRows {
		t.Error("expected removeEmptyTableRows false")
	}
}

func TestWithRemoveEmptyTableRows_SetsTrue(t *testing.T) {
	tpl := &docxTemplate{}
	WithRemoveEmptyTableRows(true)(tpl)
	if !tpl.removeEmptyTableRows {
		t.Error("expected removeEmptyTableRows true")
	}
}

func TestWithRemoveRangeRows_SetsTrue(t *testing.T) {
	tpl := &docxTemplate{}
	WithRemoveRangeRows(true)(tpl)
	if !tpl.removeRangeRows {
		t.Error("expected removeRangeRows true")
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
	if !tpl.ignoreMissingKey {
		t.Error("expected ignoreMissingKey true")
	}
}

// buildMinimalDocx creates a minimal valid DOCX file for testing.
func buildMinimalDocx(t *testing.T, documentBody string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

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
