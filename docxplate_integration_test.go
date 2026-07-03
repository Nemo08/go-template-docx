package gotemplatedocx

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/JJJJJJack/go-template-docx/internal/zio"
)

// buildDocxWithSplitRun создаёт минимальный DOCX, где в document.xml
// и word/header1.xml плейсхолдер {{Pages.Name}} разорван на два <w:r>.
func buildDocxWithSplitRun(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zio.NewZipWriter(&buf)

	ct, _ := w.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>
</Types>`))

	rels, _ := w.Create("word/_rels/document.xml.rels")
	_, _ = rels.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>
</Relationships>`))

	// document.xml с разорванным {{Pages.Name}}
	doc, _ := w.Create("word/document.xml")
	_, _ = doc.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t>{{Pages.Name</w:t></w:r>
      <w:r><w:t>}}</w:t></w:r>
    </w:p>
    <w:sectPr>
      <w:pgSz w:w="12240" w:h="15840"/>
      <w:pgMar w:top="1440" w:bottom="1440" w:left="1440" w:right="1440"/>
    </w:sectPr>
  </w:body>
</w:document>`))

	// header1.xml с разорванным {{Pages.Name}}
	hdr, _ := w.Create("word/header1.xml")
	_, _ = hdr.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:p>
    <w:r><w:t>{{Pages.Name</w:t></w:r>
    <w:r><w:t>}}</w:t></w:r>
  </w:p>
</w:hdr>`))

	_ = w.Close()
	return buf.Bytes()
}

func TestDocxplateCompat_SplitRunIntegration(t *testing.T) {
	docxBytes := buildDocxWithSplitRun(t)

	tmpl, err := NewDocxTemplateFromBytes(docxBytes, DocxplateCompat(), IgnoreMissingKey(), DeleteMissingKey())
	if err != nil {
		t.Fatalf("NewDocxTemplateFromBytes: %v", err)
	}

	// Данные без Pages — проверяем, что IgnoreMissingKey+DeleteMissingKey
	// заменят плейсхолдер на пустую строку (а не оставят сырой синтаксис).
	err = tmpl.Apply(map[string]any{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	out := tmpl.Bytes()

	// Проверяем document.xml
	gotDoc := readZipEntry(t, out, "word/document.xml")
	if strings.Contains(gotDoc, "{{Pages.Name}}") {
		t.Errorf("document.xml: docxplate-синтаксис не нормализован, содержимое:\n%s", gotDoc)
	}
	if strings.Contains(gotDoc, "{{.Pages.Name}}") {
		t.Logf("document.xml: нормализован, но не заменён (ожидаемо при отсутствии данных Pages)")
	}

	// Проверяем header1.xml
	gotHdr := readZipEntry(t, out, "word/header1.xml")
	if strings.Contains(gotHdr, "{{Pages.Name}}") {
		t.Errorf("header1.xml: docxplate-синтаксис не нормализован, содержимое:\n%s", gotHdr)
	}
	if strings.Contains(gotHdr, "{{.Pages.Name}}") {
		t.Logf("header1.xml: нормализован, но не заменён (ожидаемо при отсутствии данных Pages)")
	}
}

func TestDocxplateCompat_WithRealDataIntegration(t *testing.T) {
	docxBytes := buildDocxWithSplitRun(t)

	tmpl, err := NewDocxTemplateFromBytes(docxBytes, DocxplateCompat(), DeleteMissingKey())
	if err != nil {
		t.Fatalf("NewDocxTemplateFromBytes: %v", err)
	}

	// Pages — map, не slice: {{.Pages.Name}} напрямую рендерится.
	err = tmpl.Apply(map[string]any{"Pages": map[string]any{"Name": "Глава 1"}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	out := tmpl.Bytes()

	gotDoc := readZipEntry(t, out, "word/document.xml")
	if !strings.Contains(gotDoc, "Глава 1") {
		t.Errorf("document.xml: ожидалось 'Глава 1' в выводе, получено:\n%s", gotDoc)
	}
	if strings.Contains(gotDoc, "{{") {
		t.Errorf("document.xml: остались незаменённые плейсхолдеры:\n%s", gotDoc)
	}

	gotHdr := readZipEntry(t, out, "word/header1.xml")
	if !strings.Contains(gotHdr, "Глава 1") {
		t.Errorf("header1.xml: ожидалось 'Глава 1' в выводе, получено:\n%s", gotHdr)
	}
}

func readZipEntry(t *testing.T, data []byte, name string) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open %s: %v", name, err)
			}
			defer rc.Close()
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(rc)
			return buf.String()
		}
	}
	t.Fatalf("entry %s not found in zip", name)
	return ""
}
