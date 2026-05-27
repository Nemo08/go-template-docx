package docx

import (
	"strings"
	"testing"
)

func TestRemoveEmptyTableRows_EmptyRow(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t></w:t></w:r></w:p></w:tc></w:tr>`
	got := removeEmptyTableRows(input)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestRemoveEmptyTableRows_NonEmptyRow(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t>hello</w:t></w:r></w:p></w:tc></w:tr>`
	got := removeEmptyTableRows(input)
	if got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestRemoveEmptyTableRows_RowWithDrawing(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:drawing>...</w:drawing></w:r></w:p></w:tc></w:tr>`
	got := removeEmptyTableRows(input)
	if got != input {
		t.Errorf("expected row with drawing preserved, got %q", got)
	}
}

func TestRemoveEmptyTableRows_MixedRows(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t></w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:p><w:r><w:t>text</w:t></w:r></w:p></w:tc></w:tr>`
	expected := `<w:tr><w:tc><w:p><w:r><w:t>text</w:t></w:r></w:p></w:tc></w:tr>`
	got := removeEmptyTableRows(input)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestIsRowEmpty_Empty(t *testing.T) {
	if !isRowEmpty("<w:tr><w:t></w:t></w:tr>") {
		t.Error("expected row with empty text to be empty")
	}
}

func TestIsRowEmpty_WithText(t *testing.T) {
	if isRowEmpty("<w:tr><w:t>hello</w:t></w:tr>") {
		t.Error("expected row with text to be non-empty")
	}
}

func TestIsRowEmpty_WithVisual(t *testing.T) {
	if isRowEmpty("<w:tr><w:drawing>...</w:drawing></w:tr>") {
		t.Error("expected row with drawing to be non-empty")
	}
}

func TestMarkRangeDirectiveRows_WithRange(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t>{{range .Items}}</w:t></w:r></w:p></w:tc></w:tr>`
	got := markRangeDirectiveRows(input)
	if !strings.Contains(got, rangeRowMarker) {
		t.Errorf("expected marker in output, got %q", got)
	}
}

func TestMarkRangeDirectiveRows_NoDirective(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t>plain text</w:t></w:r></w:p></w:tc></w:tr>`
	got := markRangeDirectiveRows(input)
	if got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestRemoveMarkedEmptyRows_MarkedAndEmpty(t *testing.T) {
	// marker is placed BEFORE closing </w:tr> by markRangeDirectiveRows
	input := `<w:tr><w:tc><w:p><w:r><w:t></w:t></w:r></w:p></w:tc>` + rangeRowMarker + `</w:tr>`
	got := removeMarkedEmptyRows(input)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestRemoveMarkedEmptyRows_MarkedButNotEmpty(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t>text</w:t></w:r></w:p></w:tc></w:tr>` + rangeRowMarker
	got := removeMarkedEmptyRows(input)
	if got != input {
		t.Errorf("expected unchanged for non-empty marked row, got %q", got)
	}
}

func TestRemoveMarkedEmptyRows_UnmarkedEmpty(t *testing.T) {
	input := `<w:tr><w:tc><w:p><w:r><w:t></w:t></w:r></w:p></w:tc></w:tr>`
	got := removeMarkedEmptyRows(input)
	if got != input {
		t.Errorf("expected unmarked empty row preserved, got %q", got)
	}
}

func TestEnsureXmlSpacePreserve_AddsAttribute(t *testing.T) {
	input := `<w:t> hello</w:t>`
	expected := `<w:t xml:space="preserve"> hello</w:t>`
	got := ensureXMLSpacePreserve(input)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestEnsureXmlSpacePreserve_AlreadyHasAttribute(t *testing.T) {
	input := `<w:t xml:space="preserve"> hello</w:t>`
	got := ensureXMLSpacePreserve(input)
	if got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestEnsureXmlSpacePreserve_NoWhitespace(t *testing.T) {
	input := `<w:t>hello</w:t>`
	got := ensureXMLSpacePreserve(input)
	if got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestFlattenNestedTextRuns_Simple(t *testing.T) {
	input := `<w:t><w:rPr><w:b/></w:rPr><w:t>text</w:t></w:t>`
	expected := `<w:rPr><w:b/></w:rPr><w:t>text</w:t>`
	got := flattenNestedTextRuns(input)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestFlattenNestedTextRuns_PreserveSpace(t *testing.T) {
	input := `<w:t xml:space="preserve"><w:rPr><w:b/></w:rPr><w:t> text</w:t></w:t>`
	expected := `<w:rPr><w:b/></w:rPr><w:t xml:space="preserve"> text</w:t>`
	got := flattenNestedTextRuns(input)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestPropagateRunPropsAfterBreak_NoBreak(t *testing.T) {
	input := `<w:r><w:rPr><w:b/></w:rPr><w:t>hello</w:t></w:r>`
	got := propagateRunPropsAfterBreak(input)
	if got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestPropagateRunPropsAfterBreak_WithBreak(t *testing.T) {
	input := `<w:r><w:rPr><w:b/></w:rPr><w:t>a</w:t></w:r></w:p><w:p><w:r><w:t>b</w:t></w:r></w:p>`
	got := propagateRunPropsAfterBreak(input)
	expected := `<w:r><w:rPr><w:b/></w:rPr><w:t>a</w:t></w:r></w:p><w:p><w:r><w:rPr><w:b/></w:rPr><w:t>b</w:t></w:r></w:p>`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestPropagateRunPropsAfterBreak_Chain(t *testing.T) {
	input := `<w:r><w:rPr><w:b/></w:rPr><w:t>a</w:t></w:r></w:p><w:p><w:r><w:t>b</w:t></w:r></w:p><w:p><w:r><w:t>c</w:t></w:r></w:p>`
	got := propagateRunPropsAfterBreak(input)
	if !strings.Contains(got, `<w:rPr><w:b/></w:rPr><w:t>b`) {
		t.Errorf("expected rPr on second paragraph, got %q", got)
	}
	if !strings.Contains(got, `<w:rPr><w:b/></w:rPr><w:t>c`) {
		t.Errorf("expected rPr on third paragraph, got %q", got)
	}
}

func TestPropagateRunPropsAfterBreak_EmptyRun(t *testing.T) {
	input := `<w:r><w:rPr><w:b/></w:rPr><w:t></w:t></w:r></w:p><w:p><w:r><w:t>b</w:t></w:r>`
	got := propagateRunPropsAfterBreak(input)
	expected := `<w:r><w:rPr><w:b/></w:rPr><w:t></w:t></w:r></w:p><w:p><w:r><w:rPr><w:b/></w:rPr><w:t>b</w:t></w:r>`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestFlattenNestedTextRuns_NoNesting(t *testing.T) {
	input := `<w:r><w:t>hello</w:t></w:r>`
	got := flattenNestedTextRuns(input)
	if got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}


