package docx

import "testing"

func TestPatchXml_FixSeparatedBraces(t *testing.T) {
	input := `before{ {.Name}} after`
	expected := `before{{.Name}} after`
	got := PatchXml(input)
	if got != expected {
		t.Errorf("PatchXml(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXml_FixSeparatedCloseBraces(t *testing.T) {
	input := `before{{.Name} } after`
	expected := `before{{.Name}} after`
	got := PatchXml(input)
	if got != expected {
		t.Errorf("PatchXml(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXml_RemoveXmlTags(t *testing.T) {
	input := `before{{ .Name <w:rPr><w:b/></w:rPr>}} after`
	expected := `before{{ .Name }} after`
	got := PatchXml(input)
	if got != expected {
		t.Errorf("PatchXml(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXml_UnescapeEntities(t *testing.T) {
	input := `{{shapeBgFillColor (index .Map &quot;Color2&quot;)}}`
	expected := `{{shapeBgFillColor (index .Map "Color2")}}`
	got := PatchXml(input)
	if got != expected {
		t.Errorf("PatchXml(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXml_WrapBareHexShapeFunc(t *testing.T) {
	input := `{{shapeBgFillColor 00FF00}}`
	expected := `{{shapeBgFillColor "00FF00"}}`
	got := PatchXml(input)
	if got != expected {
		t.Errorf("PatchXml(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXml_WrapBareHexTableCellFunc(t *testing.T) {
	input := `{{tableCellBgColor 00FF00}}`
	expected := `{{tableCellBgColor "00FF00"}}`
	got := PatchXml(input)
	if got != expected {
		t.Errorf("PatchXml(%q) = %q, want %q", input, got, expected)
	}
}

func TestPatchXml_NoChangeWhenClean(t *testing.T) {
	input := `hello {{.Name}} world`
	got := PatchXml(input)
	if got != input {
		t.Errorf("PatchXml(%q) = %q, want unchanged", input, got)
	}
}

func TestPatchXml_AmpersandLast(t *testing.T) {
	// &amp; is unescaped LAST, so &amp;lt; → &lt; (not <)
	// This prevents double-unescaping of &amp;amp; → &amp;
	input := `{{&amp;lt;}}`
	got := PatchXml(input)
	expected := `{{&lt;}}`
	if got != expected {
		t.Errorf("PatchXml ampersand order: got %q, want %q", got, expected)
	}
}

func TestPatchXml_DoubleAmpersand(t *testing.T) {
	// &amp;amp; should become &amp; (not & or empty)
	input := `{{&amp;amp;}}`
	got := PatchXml(input)
	expected := `{{&amp;}}`
	if got != expected {
		t.Errorf("PatchXml double ampersand: got %q, want %q", got, expected)
	}
}
