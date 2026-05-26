package docx

import (
	"testing"
)

func TestBold(t *testing.T) {
	got := bold("hello")
	expected := `<w:rPr><w:b /><w:bCs /></w:rPr><w:t>hello</w:t>`
	if got != expected {
		t.Errorf("bold() = %q, want %q", got, expected)
	}
}

func TestItalic(t *testing.T) {
	got := italic("text")
	expected := `<w:rPr><w:i /><w:iCs /></w:rPr><w:t>text</w:t>`
	if got != expected {
		t.Errorf("italic() = %q, want %q", got, expected)
	}
}

func TestUnderline(t *testing.T) {
	got := underline("text")
	expected := `<w:rPr><w:u w:val="single"/></w:rPr><w:t>text</w:t>`
	if got != expected {
		t.Errorf("underline() = %q, want %q", got, expected)
	}
}

func TestStrike(t *testing.T) {
	got := strike("text")
	expected := `<w:rPr><w:strike /></w:rPr><w:t>text</w:t>`
	if got != expected {
		t.Errorf("strike() = %q, want %q", got, expected)
	}
}

func TestFontSize(t *testing.T) {
	got := fontSize("text", 12)
	expected := `<w:rPr><w:sz w:val="24" /><w:szCs w:val="24" /></w:rPr><w:t>text</w:t>`
	if got != expected {
		t.Errorf("fontSize(12) = %q, want %q", got, expected)
	}
}

func TestFontSize_Zero(t *testing.T) {
	got := fontSize("text", 0)
	// Should default to 1 half-point
	expected := `<w:rPr><w:sz w:val="2" /><w:szCs w:val="2" /></w:rPr><w:t>text</w:t>`
	if got != expected {
		t.Errorf("fontSize(0) = %q, want %q", got, expected)
	}
}

func TestColor(t *testing.T) {
	got, err := color("text", "FF0000")
	if err != nil {
		t.Fatal(err)
	}
	expected := `<w:rPr><w:color w:val="FF0000" /></w:rPr><w:t>text</w:t>`
	if got != expected {
		t.Errorf("color() = %q, want %q", got, expected)
	}
}

func TestColor_WithHash(t *testing.T) {
	got, err := color("text", "#FF0000")
	if err != nil {
		t.Fatal(err)
	}
	expected := `<w:rPr><w:color w:val="FF0000" /></w:rPr><w:t>text</w:t>`
	if got != expected {
		t.Errorf("color(#FF0000) = %q, want %q", got, expected)
	}
}

func TestColor_Invalid(t *testing.T) {
	_, err := color("text", "FF")
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

func TestHighlight(t *testing.T) {
	got, err := highlight("text", "yellow")
	if err != nil {
		t.Fatal(err)
	}
	expected := `<w:rPr><w:highlight w:val="yellow" /></w:rPr><w:t>text</w:t>`
	if got != expected {
		t.Errorf("highlight() = %q, want %q", got, expected)
	}
}

func TestHighlight_InvalidColor(t *testing.T) {
	_, err := highlight("text", "invalid")
	if err == nil {
		t.Error("expected error for invalid highlight color")
	}
}

func TestList(t *testing.T) {
	got := list("a", "b", "c")
	if len(got) != 3 {
		t.Errorf("list() length = %d, want 3", len(got))
	}
}

func TestInlineStyledText(t *testing.T) {
	got, err := inlineStyledText("text", "b", "i")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, "<w:b />") {
		t.Errorf("expected bold tag, got %q", got)
	}
	if !contains(got, "<w:i />") {
		t.Errorf("expected italic tag, got %q", got)
	}
}

func TestInlineStyledText_DuplicateBold(t *testing.T) {
	_, err := inlineStyledText("text", "b", "bold")
	if err == nil {
		t.Error("expected error for duplicate bold styles")
	}
}

func TestImage(t *testing.T) {
	got := image("photo.png")
	expected := "[[IMAGE:photo.png]]"
	if got != expected {
		t.Errorf("image() = %q, want %q", got, expected)
	}
}

func TestReplaceImage(t *testing.T) {
	got := replaceImage("photo.png")
	expected := "[[REPLACE_IMAGE:photo.png]]"
	if got != expected {
		t.Errorf("replaceImage() = %q, want %q", got, expected)
	}
}

func TestPreserveNewline(t *testing.T) {
	got := preserveNewline("a\nb")
	expected := `a</w:t><w:br/><w:t>b`
	if got != expected {
		t.Errorf("preserveNewline() = %q, want %q", got, expected)
	}
}

func TestBreakParagraph(t *testing.T) {
	got := breakParagraph("a\nb")
	expected := `a</w:t></w:r></w:p><w:p><w:r><w:t>b`
	if got != expected {
		t.Errorf("breakParagraph() = %q, want %q", got, expected)
	}
}

func TestShadeTextBg(t *testing.T) {
	got, err := shadeTextBg("text", "C0FFEE")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, `w:fill="C0FFEE"`) {
		t.Errorf("expected fill color, got %q", got)
	}
}

func TestShadeTextBg_InvalidHex(t *testing.T) {
	_, err := shadeTextBg("text", "FF")
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

func TestShapeBgFillColor(t *testing.T) {
	got, err := shapeBgFillColor("FF0000")
	if err != nil {
		t.Fatal(err)
	}
	expected := "[[SHAPE_BG_FILL_COLOR:FF0000]]"
	if got != expected {
		t.Errorf("shapeBgFillColor() = %q, want %q", got, expected)
	}
}

func TestTableCellBgColor(t *testing.T) {
	got, err := tableCellBgColor("FF0000")
	if err != nil {
		t.Fatal(err)
	}
	expected := "[[TABLE_CELL_BG_COLOR:FF0000]]"
	if got != expected {
		t.Errorf("tableCellBgColor() = %q, want %q", got, expected)
	}
}

func TestFormatStylesTags_Bold(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"b"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, "<w:b />") {
		t.Errorf("expected bold tag, got %q", got)
	}
}

func TestFormatStylesTags_Italic(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"i"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, "<w:i />") {
		t.Errorf("expected italic tag, got %q", got)
	}
}

func TestFormatStylesTags_FontSize(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"fs:14"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, `w:val="28"`) {
		t.Errorf("expected sz val=28, got %q", got)
	}
}

func TestFormatStylesTags_Color(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"#FF0000"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, `w:val="FF0000"`) {
		t.Errorf("expected color FF0000, got %q", got)
	}
}

func TestFormatStylesTags_Shading(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"bg:C0FFEE"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, `w:fill="C0FFEE"`) {
		t.Errorf("expected shading C0FFEE, got %q", got)
	}
}

func TestFormatStylesTags_UnknownStyle(t *testing.T) {
	_, err := formatStylesTags([]interface{}{"unknown"}, "test")
	if err == nil {
		t.Error("expected error for unknown style")
	}
}

func TestFormatStylesTags_NonStringParam(t *testing.T) {
	_, err := formatStylesTags([]interface{}{42}, "test")
	if err == nil {
		t.Error("expected error for non-string param")
	}
}

func TestFormatStylesTags_HighlightColor(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"yellow"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, `w:val="yellow"`) {
		t.Errorf("expected highlight yellow, got %q", got)
	}
}

func TestStyledText(t *testing.T) {
	got, err := styledText("text", []interface{}{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, "<w:b />") || !contains(got, "text") {
		t.Errorf("styledText() = %q, want bold text wrapper", got)
	}
}
