package gotemplatedocx

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/JJJJJJack/go-template-docx/internal/zio"
)

// ─── Data structures ────────────────────────────────────────────────────────────

type allFeaturesData struct {
	Simple    string
	Nested    afNestedData
	Items     []afItemData
	Logo      string
	Images    []string
	Date      time.Time
	Number    float64
	MultiLine string
	LongText  string
	RomanNum  int
	Missing   string
}

type afNestedData struct {
	Field string
}

type afItemData struct {
	Name  string
	Value float64
}

var afDefaultData = allFeaturesData{
	Simple:    "HelloWorld",
	Nested:    afNestedData{Field: "NestedValue"},
	Items:     []afItemData{{Name: "Alice", Value: 100}, {Name: "Bob", Value: 200}},
	Logo:      "logo.png",
	Images:    []string{"img1.png", "img2.png"},
	Date:      time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
	Number:    12345.67,
	MultiLine: "Line1\nLine2\nLine3",
	LongText:  "This is a very long text that should be truncated",
	RomanNum:  1984,
}

// ─── DOCX builder ───────────────────────────────────────────────────────────────

type afFile struct{ name, content string }

func afContentTypes() afFile {
	return afFile{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>
  <Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>
</Types>`}
}

func afDocRels() afFile {
	return afFile{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>
</Relationships>`}
}

func afHeaderXML() afFile {
	return afFile{"word/header1.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
       xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:p>
    <w:r>
      <w:t>Header: {{.Simple}} | {{image .Logo}}</w:t>
    </w:r>
  </w:p>
</w:hdr>`}
}

func afFooterXML() afFile {
	return afFile{"word/footer1.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:p>
    <w:r>
      <w:t>Footer: {{formatDate .Date "2006"}} | Roman {{romanNum .RomanNum}}</w:t>
    </w:r>
  </w:p>
</w:ftr>`}
}

func afDocumentXML() afFile {
	return afFile{"word/document.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
            xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
            xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"
            xmlns:wps="http://schemas.microsoft.com/office/word/2010/wordprocessingShape"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>
    <w:p><w:r><w:t>{{.Simple}} | {{.Nested.Field}}</w:t></w:r></w:p>

    <w:p>
      <w:r><w:t>{{bold .Simple}}</w:t></w:r>
      <w:r><w:t> </w:t></w:r>
      <w:r><w:t>{{italic .Simple}}</w:t></w:r>
      <w:r><w:t> </w:t></w:r>
      <w:r><w:t>{{color .Simple "#FF0000"}}</w:t></w:r>
      <w:r><w:t> </w:t></w:r>
      <w:r><w:t>{{highlight .Simple "yellow"}}</w:t></w:r>
      <w:r><w:t> </w:t></w:r>
      <w:r><w:t>{{fontSize .Simple 14}}</w:t></w:r>
    </w:p>

    <w:p>
      <w:r><w:t>{{formatNum .Number 2 ","}}</w:t></w:r>
      <w:r><w:t> | {{formatDate .Date "2006-01-02"}} | {{formatDateRU .Date "2 January 2006"}}</w:t></w:r>
    </w:p>

    <w:p><w:r><w:t>{{breakParagraph .MultiLine}}</w:t></w:r></w:p>
    <w:p><w:r><w:t>{{preserveNewline .MultiLine}}</w:t></w:r></w:p>
    <w:p><w:r><w:t>{{truncate .LongText 10}}</w:t></w:r></w:p>
    <w:p><w:r><w:t>{{padRight .Simple 20}}x</w:t></w:r></w:p>
    <w:p><w:r><w:t>{{default .Missing "fallback"}}</w:t></w:r></w:p>

    <w:p><w:r><w:t>{{inlineStyledText .Simple "b" "#FF0000" "fs:16"}}</w:t></w:r></w:p>
    <w:p><w:r><w:t>{{styledText .Simple (list "i" "green")}}</w:t></w:r></w:p>

    <w:p><w:r><w:t>{{image .Logo}}</w:t></w:r></w:p>

    <w:p><w:r><w:t>{{range .Images}}{{image .}}{{end}}</w:t></w:r></w:p>

    <w:tbl>
      <w:tr>
        <w:tc><w:p><w:r><w:t>{{.Items.Name}}</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>{{.Items.Value}}</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>

    <w:tbl>
      <w:tr><w:tc><w:p><w:r><w:t>keep row</w:t></w:r></w:p></w:tc></w:tr>
      <w:tr><w:tc><w:p><w:r><w:t>{{hideRow true}}hidden</w:t></w:r></w:p></w:tc></w:tr>
    </w:tbl>

    <w:p>
      <w:r>
        <mc:AlternateContent>
          <mc:Choice Requires="wps">
            <w:drawing>
              <wp:anchor distT="0" distB="0" distL="0" distR="0">
                <wp:docPr id="1" name="Shape"/>
                <a:graphic>
                  <a:graphicData uri="http://schemas.microsoft.com/office/word/2010/wordprocessingShape">
                    <wps:wsp>
                      <wps:spPr>
                        <a:solidFill>
                          <a:srgbClr val="000000"/>
                        </a:solidFill>
                      </wps:spPr>
                    </wps:wsp>
                  </a:graphicData>
                </a:graphic>
              </wp:anchor>
            </w:drawing>
          </mc:Choice>
          {{shapeBgFillColor "#FF0000"}}
        </mc:AlternateContent>
      </w:r>
    </w:p>

    <w:tbl>
      <w:tr>
        <w:tc>
          <w:tcPr><w:shd w:fill="FFFFFF" w:val="clear"/></w:tcPr>
          <w:p><w:r><w:t>{{tableCellBgColor "#00FF00"}}colored cell</w:t></w:r></w:p>
        </w:tc>
      </w:tr>
    </w:tbl>

    <w:p><w:r><w:t>Sum={{sumCol .Items "Value"}} Avg={{avgCol .Items "Value"}}</w:t></w:r></w:p>

    <w:p><w:r><w:t>{{pageBreak}}</w:t></w:r></w:p>

    <w:p><w:r><w:t>miss: {{.Missing}}</w:t></w:r></w:p>

    <w:sectPr>
      <w:pgSz w:w="12240" w:h="15840"/>
      <w:pgMar w:top="1440" w:bottom="1440" w:left="1440" w:right="1440"/>
      <w:headerReference w:type="default" r:id="rId2"/>
      <w:footerReference w:type="default" r:id="rId3"/>
    </w:sectPr>
  </w:body>
</w:document>`}
}

func buildAllFeaturesDocx(t *testing.T) []byte {
	t.Helper()
	files := []afFile{
		afContentTypes(),
		afDocRels(),
		afHeaderXML(),
		afFooterXML(),
		afDocumentXML(),
	}
	var zBuf bytes.Buffer
	w := zio.NewZipWriter(&zBuf)
	for _, f := range files {
		fh, _ := w.Create(f.name)
		_, _ = fh.Write([]byte(f.content))
	}
	_ = w.Close()
	return zBuf.Bytes()
}

// ─── Test helpers ───────────────────────────────────────────────────────────────

func smallPNG() []byte {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Pix[0] = 255
	img.Pix[1] = 0
	img.Pix[2] = 0
	img.Pix[3] = 255
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// ─── Tests ──────────────────────────────────────────────────────────────────────

func TestAllFeatures_RenderSuccess(t *testing.T) {
	pngData := smallPNG()
	docxBytes := buildAllFeaturesDocx(t)

	result, err := Render(docxBytes, afDefaultData,
		WithImage("logo.png", pngData),
		WithImage("img1.png", pngData),
		WithImage("img2.png", pngData),
		WithRemoveEmptyTableRows(true),
		WithAutoExpandRows(afDefaultData),
	)
	if err != nil {
		t.Fatal(err)
	}

	docContent := string(extractFromZip(t, result, "word/document.xml"))

	// 1. Variable replacement
	if !strings.Contains(docContent, "HelloWorld") {
		t.Error("missing Simple value")
	}
	if !strings.Contains(docContent, "NestedValue") {
		t.Error("missing Nested.Field value")
	}

	// 2. Formatting functions
	if !strings.Contains(docContent, "<w:b />") && !strings.Contains(docContent, "<w:b>") && !strings.Contains(docContent, "<w:bCs") {
		t.Error("missing bold <w:b")
	}
	if !strings.Contains(docContent, "<w:i") {
		t.Error("missing italic <w:i")
	}

	// 3. Date/number formatting
	if !strings.Contains(docContent, "2024-06-15") {
		t.Error("missing formatted date")
	}

	// 4. Image placeholders replaced with <w:drawing>
	drawingCount := strings.Count(docContent, "<w:drawing>")
	// logo.png (single) + img1.png (range) + img2.png (range) + shape
	if drawingCount < 2 {
		t.Errorf("expected >=2 <w:drawing>, got %d", drawingCount)
	}

	// 5. Media files in output
	if !zipContains(t, result, "word/media/image1.png") {
		t.Error("word/media/image1.png not found")
	}

	// 6. Hidden row removed
	if strings.Contains(docContent, "hidden") {
		t.Error("hidden row text should be removed")
	}
	if !strings.Contains(docContent, "keep row") {
		t.Error("visible row should remain")
	}
	if strings.Contains(docContent, "\x00HIDEROW\x00") {
		t.Error("HideRowSentinel should be removed")
	}

	// 7. Shape bg fill color placeholder removed
	if strings.Contains(docContent, "[[SHAPE_BG_FILL_COLOR:") {
		t.Error("shape bg fill color placeholder remains")
	}

	// 8. Table cell bg color placeholder removed
	if strings.Contains(docContent, "[[TABLE_CELL_BG_COLOR:") {
		t.Error("table cell bg color placeholder remains")
	}

	// 9. Sum/avg computed
	if strings.Contains(docContent, "{{sumCol") || strings.Contains(docContent, "{{avgCol") {
		t.Error("sum/avg not computed")
	}

	// 10. Page break placeholder replaced
	if strings.Contains(docContent, "\x00PAGEBREAK\x00") {
		t.Error("page break placeholder remains")
	}
	if !strings.Contains(docContent, `w:type="page"`) {
		t.Error("page break XML missing")
	}

	// 11. Default value
	if !strings.Contains(docContent, "fallback") {
		t.Error("missing fallback text")
	}

	// 12. Header
	headerXML := string(extractFromZip(t, result, "word/header1.xml"))
	if !strings.Contains(headerXML, "HelloWorld") {
		t.Error("missing Simple in header")
	}
	if !strings.Contains(headerXML, "<w:drawing>") {
		t.Error("missing image in header")
	}

	// 13. Footer
	footerXML := string(extractFromZip(t, result, "word/footer1.xml"))
	if !strings.Contains(footerXML, "2024") {
		t.Error("missing year in footer")
	}

	// 14. Content types
	ctXML := string(extractFromZip(t, result, "[Content_Types].xml"))
	if !strings.Contains(ctXML, `Extension="png"`) {
		t.Error("missing png content type")
	}

	// 15. Header rels for image
	headerRels := string(extractFromZip(t, result, "word/_rels/header1.xml.rels"))
	if !strings.Contains(headerRels, "media/image") {
		t.Error("missing image rel in header rels")
	}
}

func TestAllFeatures_IgnoreMissingKey(t *testing.T) {
	docxBytes := buildAllFeaturesDocx(t)
	data := map[string]any{
		"Simple":   "test",
		"Date":     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		"Nested":   afNestedData{Field: "ok"},
		"Items":    []afItemData{{Name: "x", Value: 1}},
		"Images":   []string{},
		"LongText": "short",
	}

	result, err := Render(docxBytes, data, WithIgnoreMissingKey(true), WithAutoExpandRows(data))
	if err != nil {
		t.Fatal(err)
	}
	docContent := string(extractFromZip(t, result, "word/document.xml"))
	if !strings.Contains(docContent, "{{.Missing}}") {
		t.Error("expected placeholder preserved with IgnoreMissingKey")
	}
}

func TestAllFeatures_DeleteMissingKey(t *testing.T) {
	docxBytes := buildAllFeaturesDocx(t)
	data := map[string]any{
		"Simple":    "test",
		"Date":      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		"Nested":    afNestedData{Field: "ok"},
		"Items":     []afItemData{{Name: "x", Value: 1}},
		"Images":    []string{},
		"LongText":  "short",
		"RomanNum":  0,
		"Number":    0.0,
		"MultiLine": "",
	}

	result, err := Render(docxBytes, data, WithDeleteMissingKey(), WithAutoExpandRows(data))
	if err != nil {
		t.Fatal(err)
	}
	docContent := string(extractFromZip(t, result, "word/document.xml"))
	if strings.Contains(docContent, "{{.Missing}}") {
		t.Error("expected placeholder removed with DeleteMissingKey")
	}
}

func TestAllFeatures_ErrorOnMissingKey(t *testing.T) {
	docxBytes := buildAllFeaturesDocx(t)
	data := map[string]any{
		"Simple":   "test",
		"Date":     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		"Nested":   afNestedData{Field: "ok"},
		"Items":    []afItemData{{Name: "x", Value: 1}},
		"Images":   []string{},
		"LongText": "short",
	}

	_, err := Render(docxBytes, data, WithAutoExpandRows(data))
	if err == nil {
		t.Fatal("expected error for missing key without options")
	}
}
