package xmlutil

import "testing"

func TestFindNext_Found(t *testing.T) {
	s := "abc<w:tr>xyz"
	idx := FindNext(s, 0, "<w:tr>", "<w:tr ")
	if idx != 3 {
		t.Errorf("expected 3, got %d", idx)
	}
}

func TestFindNext_NotFound(t *testing.T) {
	s := "abc"
	idx := FindNext(s, 0, "<w:tr>")
	if idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}

func TestFindNext_PicksEarliest(t *testing.T) {
	s := "<w:tr>...<w:tr "
	idx := FindNext(s, 0, "<w:tr>", "<w:tr ")
	if idx != 0 {
		t.Errorf("expected 0, got %d", idx)
	}
}

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
	input := `{{&amp;lt;}}`
	got := PatchXML(input)
	expected := `{{&lt;}}`
	if got != expected {
		t.Errorf("PatchXML ampersand order: got %q, want %q", got, expected)
	}
}

func TestPatchXML_DoubleAmpersand(t *testing.T) {
	input := `{{&amp;amp;}}`
	got := PatchXML(input)
	expected := `{{&amp;}}`
	if got != expected {
		t.Errorf("PatchXML double ampersand: got %q, want %q", got, expected)
	}
}
