package gotemplatedocx

import (
	"archive/zip"
	"bytes"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/JJJJJJack/go-template-docx/internal/zio"
)

// minimalPNG creates a valid 1x1 red pixel PNG.
func minimalPNG() []byte {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Pix[0] = 255
	img.Pix[1] = 0
	img.Pix[2] = 0
	img.Pix[3] = 255
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// buildDocxWithHeader creates a minimal DOCX with a header containing documentBody
// and a header containing headerBody.
func buildDocxWithHeader(t *testing.T, headerBody string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zio.NewZipWriter(&buf)

	// [Content_Types].xml
	ct, _ := w.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>
</Types>`))

	// word/_rels/document.xml.rels
	rels, _ := w.Create("word/_rels/document.xml.rels")
	_, _ = rels.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>
</Relationships>`))

	// word/document.xml
	doc, _ := w.Create("word/document.xml")
	_, _ = doc.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>Body text</w:t>
      </w:r>
    </w:p>
    <w:sectPr>
      <w:pgSz w:w="12240" w:h="15840"/>
      <w:pgMar w:top="1440" w:bottom="1440" w:left="1440" w:right="1440"/>
      <w:headerReference w:type="default" r:id="rId2"/>
    </w:sectPr>
  </w:body>
</w:document>`))


	// word/header1.xml
	hdr, _ := w.Create("word/header1.xml")
	_, _ = hdr.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
       xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
       xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:p>
    <w:r>
      <w:t>` + headerBody + `</w:t>
    </w:r>
  </w:p>
</w:hdr>`))

	_ = w.Close()
	return buf.Bytes()
}

// extractFromZip reads a file from a DOCX zip byte slice.
func extractFromZip(t *testing.T, docxBytes []byte, name string) []byte {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open %s: %v", f.Name, err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("failed to read %s: %v", f.Name, err)
			}
			return data
		}
	}
	t.Fatalf("%s not found in DOCX", name)
	return nil
}

// zipContains checks if a file exists in a DOCX zip blob.
func zipContains(t *testing.T, docxBytes []byte, name string) bool {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	for _, f := range r.File {
		if f.Name == name {
			return true
		}
	}
	return false
}

func TestRender_ImageInHeader(t *testing.T) {
	pngData := minimalPNG()
	docxBytes := buildDocxWithHeader(t, `{{image .Logo}}`)

	result, err := Render(docxBytes, map[string]any{"Logo": "logo.png"},
		WithImage("logo.png", pngData),
	)
	if err != nil {
		t.Fatal(err)
	}

	// 1. header1.xml должен содержать <w:drawing>
	headerXML := string(extractFromZip(t, result, "word/header1.xml"))
	if !strings.Contains(headerXML, "<w:drawing>") {
		t.Errorf("header1.xml missing <w:drawing>, got:\n%s", headerXML)
	}
	if !strings.Contains(headerXML, `r:embed="`) {
		t.Errorf("header1.xml missing r:embed, got:\n%s", headerXML)
	}

	// 2. header1.xml.rels должен содержать image relationship
	headerRels := string(extractFromZip(t, result, "word/_rels/header1.xml.rels"))
	if !strings.Contains(headerRels, `media/image`) {
		t.Errorf("header1.xml.rels missing media reference, got:\n%s", headerRels)
	}
	if !strings.Contains(headerRels, `http://schemas.openxmlformats.org/officeDocument/2006/relationships/image`) {
		t.Errorf("header1.xml.rels missing image relationship type, got:\n%s", headerRels)
	}

	// 3. word/media/image1.png должен существовать
	if !zipContains(t, result, "word/media/image1.png") {
		t.Error("word/media/image1.png not found in output")
	}

	// 4. [Content_Types].xml должен иметь png MIME
	ctXML := string(extractFromZip(t, result, "[Content_Types].xml"))
	if !strings.Contains(ctXML, `Extension="png"`) {
		t.Errorf("[Content_Types].xml missing png extension, got:\n%s", ctXML)
	}
}
