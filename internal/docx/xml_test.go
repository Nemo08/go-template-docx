package docx

import "testing"

func TestPatchXML_FixSeparatedBraces(t *testing.T) {
	input := `before{ {.Name}} after`
	expected := `before{{.Name}} after`
	got := PatchXML(input)
	if got != expected {
		t.Errorf("PatchXML(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXML_FixSeparatedCloseBraces(t *testing.T) {
	input := `before{{.Name} } after`
	expected := `before{{.Name}} after`
	got := PatchXML(input)
	if got != expected {
		t.Errorf("PatchXML(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXML_RemoveXmlTags(t *testing.T) {
	input := `before{{ .Name <w:rPr><w:b/></w:rPr>}} after`
	expected := `before{{ .Name }} after`
	got := PatchXML(input)
	if got != expected {
		t.Errorf("PatchXML(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXML_UnescapeEntities(t *testing.T) {
	input := `{{shapeBgFillColor (index .Map &quot;Color2&quot;)}}`
	expected := `{{shapeBgFillColor (index .Map "Color2")}}`
	got := PatchXML(input)
	if got != expected {
		t.Errorf("PatchXML(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXML_WrapBareHexShapeFunc(t *testing.T) {
	input := `{{shapeBgFillColor 00FF00}}`
	expected := `{{shapeBgFillColor "00FF00"}}`
	got := PatchXML(input)
	if got != expected {
		t.Errorf("PatchXML(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXML_WrapBareHexTableCellFunc(t *testing.T) {
	input := `{{tableCellBgColor 00FF00}}`
	expected := `{{tableCellBgColor "00FF00"}}`
	got := PatchXML(input)
	if got != expected {
		t.Errorf("PatchXML(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXML_NoChangeWhenClean(t *testing.T) {
	input := `hello {{.Name}} world`
	got := PatchXML(input)
	if got != input {
		t.Errorf("PatchXML(%q) = %q, want unchanged", input, got)
	}
}

func TestPatchXML_AmpersandLast(t *testing.T) {
	// &amp; is unescaped LAST, so &amp;lt; → &lt; (not <)
	// This prevents double-unescaping of &amp;amp; → &amp;
	input := `{{&amp;lt;}}`
	got := PatchXML(input)
	expected := `{{&lt;}}`
	if got != expected {
		t.Errorf("PatchXML ampersand order: got %q, want %q", got, expected)
	}
}

func TestPatchXML_DoubleAmpersand(t *testing.T) {
	// &amp;amp; should become &amp; (not & or empty)
	input := `{{&amp;amp;}}`
	got := PatchXML(input)
	expected := `{{&amp;}}`
	if got != expected {
		t.Errorf("PatchXML double ampersand: got %q, want %q", got, expected)
	}
}
