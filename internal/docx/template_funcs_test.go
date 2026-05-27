package docx

import (
	"strings"
	"testing"
	"time"
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
	if !strings.Contains(got, "<w:b />") {
		t.Errorf("expected bold tag, got %q", got)
	}
	if !strings.Contains(got, "<w:i />") {
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
	if !strings.Contains(got, `w:fill="C0FFEE"`) {
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
	if !strings.Contains(got, "<w:b />") {
		t.Errorf("expected bold tag, got %q", got)
	}
}

func TestFormatStylesTags_Italic(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"i"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<w:i />") {
		t.Errorf("expected italic tag, got %q", got)
	}
}

func TestFormatStylesTags_FontSize(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"fs:14"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `w:val="28"`) {
		t.Errorf("expected sz val=28, got %q", got)
	}
}

func TestFormatStylesTags_Color(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"#FF0000"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `w:val="FF0000"`) {
		t.Errorf("expected color FF0000, got %q", got)
	}
}

func TestFormatStylesTags_Shading(t *testing.T) {
	got, err := formatStylesTags([]interface{}{"bg:C0FFEE"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `w:fill="C0FFEE"`) {
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
	if !strings.Contains(got, `w:val="yellow"`) {
		t.Errorf("expected highlight yellow, got %q", got)
	}
}

func TestStyledText(t *testing.T) {
	got, err := styledText("text", []interface{}{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<w:b />") || !strings.Contains(got, "text") {
		t.Errorf("styledText() = %q, want bold text wrapper", got)
	}
}

// ─── toFloat64 ────────────────────────────────────────────────────────────────

func TestToFloat64_Types(t *testing.T) {
	cases := []struct {
		in   any
		want float64
	}{
		{int(3), 3},
		{int32(3), 3},
		{int64(3), 3},
		{float32(3.5), 3.5},
		{float64(3.14), 3.14},
	}
	for _, c := range cases {
		got, err := toFloat64(c.in)
		if err != nil {
			t.Errorf("toFloat64(%T): unexpected error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("toFloat64(%T): got %v, want %v", c.in, got, c.want)
		}
	}
}

func TestToFloat64_Unsupported(t *testing.T) {
	_, err := toFloat64("string")
	if err == nil {
		t.Error("expected error for string")
	}
}

// ─── formatThousands ──────────────────────────────────────────────────────────

func TestFormatThousands_Small(t *testing.T) {
	cases := map[int64]string{0: "0", 999: "999", 1: "1"}
	for in, want := range cases {
		if got := formatThousands(in); got != want {
			t.Errorf("formatThousands(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatThousands_Grouped(t *testing.T) {
	cases := map[int64]string{
		1000:    "1\u00A0000",
		1234567: "1\u00A0234\u00A0567",
		100000:  "100\u00A0000",
	}
	for in, want := range cases {
		if got := formatThousands(in); got != want {
			t.Errorf("formatThousands(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatThousands_Negative(t *testing.T) {
	got := formatThousands(-1000)
	if got != "-1\u00A0000" {
		t.Errorf("got %q", got)
	}
}

// ─── formatNum ────────────────────────────────────────────────────────────────

func TestFormatNum_Basic(t *testing.T) {
	cases := []struct {
		val  any
		dec  int
		sep  string
		want string
	}{
		{float64(1234), 0, ",", "1\u00A0234"},
		{float64(1234.567), 2, ",", "1\u00A0234,57"},
		{float64(1234.5), 2, ".", "1\u00A0234.50"},
		{float64(-1500.5), 1, ",", "-1\u00A0500,5"},
		{float64(0), 2, ",", "0,00"},
		{int(42), 0, ".", "42"},
		{int32(1000), 0, ",", "1\u00A0000"},
		{int64(1000000), 0, ",", "1\u00A0000\u00A0000"},
	}
	for _, c := range cases {
		got, err := formatNum(c.val, c.dec, c.sep)
		if err != nil {
			t.Errorf("formatNum(%v,%d,%q): %v", c.val, c.dec, c.sep, err)
			continue
		}
		if got != c.want {
			t.Errorf("formatNum(%v,%d,%q) = %q, want %q", c.val, c.dec, c.sep, got, c.want)
		}
	}
}

func TestFormatNum_UnsupportedType(t *testing.T) {
	_, err := formatNum("nope", 2, ",")
	if err == nil {
		t.Error("expected error")
	}
}

// ─── formatDate ───────────────────────────────────────────────────────────────

func TestFormatDate(t *testing.T) {
	d := time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC)
	if got := formatDate(d, "02.01.2006"); got != "15.04.2025" {
		t.Errorf("got %q", got)
	}
}

// ─── formatDateRU ─────────────────────────────────────────────────────────────

func TestFormatDateRU_AllMonths(t *testing.T) {
	want := []string{
		"января", "февраля", "марта", "апреля",
		"мая", "июня", "июля", "августа",
		"сентября", "октября", "ноября", "декабря",
	}
	for i, w := range want {
		d := time.Date(2025, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC)
		got := formatDateRU(d, "January")
		if got != w {
			t.Errorf("month %d: got %q, want %q", i+1, got, w)
		}
	}
}

func TestFormatDateRU_FullLayout(t *testing.T) {
	d := time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC)
	got := formatDateRU(d, "02 January 2006 г.")
	if got != "15 апреля 2025 г." {
		t.Errorf("got %q", got)
	}
}

func TestFormatDateRU_NoMonthToken(t *testing.T) {
	d := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)
	got := formatDateRU(d, "02.01.2006")
	if got != "05.03.2025" {
		t.Errorf("got %q", got)
	}
}

// ─── hideRowFn / pageBreakFn ──────────────────────────────────────────────────

func TestHideRowFn(t *testing.T) {
	if hideRowFn(true) != HideRowSentinel {
		t.Error("expected sentinel for true")
	}
	if hideRowFn(false) != "" {
		t.Error("expected empty for false")
	}
}

func TestPageBreakFn(t *testing.T) {
	if pageBreakFn() != PageBreakPlaceholder {
		t.Error("expected placeholder")
	}
}

// ─── defaultVal ───────────────────────────────────────────────────────────────

func TestDefaultVal(t *testing.T) {
	dash := "—"
	cases := []struct {
		in   any
		want any
	}{
		{"", dash},
		{"hello", "hello"},
		{0, dash},
		{42, 42},
		{0.0, dash},
		{3.14, 3.14},
		{false, dash},
		{true, true},
		{nil, dash},
		{[]int(nil), dash},
	}
	for _, c := range cases {
		got := defaultVal(c.in, dash)
		if got != c.want {
			t.Errorf("defaultVal(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

// ─── sumCol / avgCol ──────────────────────────────────────────────────────────

func TestSumCol_StructSlice(t *testing.T) {
	type Row struct{ Amount float64 }
	rows := []Row{{1.5}, {2.5}, {3.0}}
	got, err := sumCol(rows, "Amount")
	if err != nil {
		t.Fatal(err)
	}
	if got != 7.0 {
		t.Errorf("got %v", got)
	}
}

func TestSumCol_MapSlice(t *testing.T) {
	rows := []map[string]any{
		{"Amount": float64(10)},
		{"Amount": float64(20)},
		{"Amount": float64(30)},
	}
	got, err := sumCol(rows, "Amount")
	if err != nil {
		t.Fatal(err)
	}
	if got != 60.0 {
		t.Errorf("got %v", got)
	}
}

func TestSumCol_EmptySlice(t *testing.T) {
	got, err := sumCol([]map[string]any{}, "Amount")
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Errorf("got %v", got)
	}
}

func TestSumCol_NotSlice(t *testing.T) {
	_, err := sumCol("not a slice", "Field")
	if err == nil {
		t.Error("expected error")
	}
}

func TestSumCol_MissingField(t *testing.T) {
	rows := []map[string]any{{"Other": float64(1)}}
	_, err := sumCol(rows, "Amount")
	if err == nil {
		t.Error("expected error for missing field")
	}
}

func TestSumCol_IntField(t *testing.T) {
	type Row struct{ Count int }
	rows := []Row{{3}, {7}}
	got, err := sumCol(rows, "Count")
	if err != nil {
		t.Fatal(err)
	}
	if got != 10 {
		t.Errorf("got %v", got)
	}
}

func TestAvgCol_Basic(t *testing.T) {
	type Row struct{ Score float64 }
	rows := []Row{{10}, {20}, {30}}
	got, err := avgCol(rows, "Score")
	if err != nil {
		t.Fatal(err)
	}
	if got != 20.0 {
		t.Errorf("got %v", got)
	}
}

func TestAvgCol_EmptySlice(t *testing.T) {
	got, err := avgCol([]map[string]any{}, "Score")
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Errorf("got %v", got)
	}
}

// ─── truncate ─────────────────────────────────────────────────────────────────

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hell…"},
		{"привет мир", 7, "привет…"},
		{"hi", 0, ""},
		{"hi", 1, "…"},
	}
	for _, c := range cases {
		got := truncate(c.in, c.max)
		if got != c.want {
			t.Errorf("truncate(%q,%d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

// ─── romanNum ─────────────────────────────────────────────────────────────────

func TestRomanNum(t *testing.T) {
	cases := map[int]string{
		1: "I", 4: "IV", 5: "V", 9: "IX",
		14: "XIV", 40: "XL", 58: "LVIII",
		90: "XC", 399: "CCCXCIX", 1994: "MCMXCIV",
		3999: "MMMCMXCIX",
	}
	for in, want := range cases {
		got, err := romanNum(in)
		if err != nil {
			t.Errorf("romanNum(%d): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("romanNum(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestRomanNum_OutOfRange(t *testing.T) {
	for _, n := range []int{0, -1, 4000} {
		if _, err := romanNum(n); err == nil {
			t.Errorf("romanNum(%d): expected error", n)
		}
	}
}

// ─── padRight ─────────────────────────────────────────────────────────────────

func TestPadRight(t *testing.T) {
	cases := []struct {
		in   string
		min  int
		want string
	}{
		{"hi", 5, "hi   "},
		{"hello", 5, "hello"},
		{"hello", 3, "hello"},
		{"да", 5, "да   "},
		{"", 3, "   "},
	}
	for _, c := range cases {
		got := padRight(c.in, c.min)
		if got != c.want {
			t.Errorf("padRight(%q,%d) = %q, want %q", c.in, c.min, got, c.want)
		}
	}
}

// ─── extraFuncMap completeness ────────────────────────────────────────────────

func TestExtraFuncMap_ContainsAll(t *testing.T) {
	required := []string{
		"formatNum", "formatDate", "formatDateRU",
		"hideRow", "pageBreak",
		"default", "sumCol", "avgCol",
		"truncate", "romanNum", "padRight",
	}
	for _, name := range required {
		if _, ok := extraFuncMap[name]; !ok {
			t.Errorf("extraFuncMap missing %q", name)
		}
	}
}
