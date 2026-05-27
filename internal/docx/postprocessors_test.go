package docx

import (
	"strings"
	"testing"
)

func TestRemoveHiddenRows_RemovesMarked(t *testing.T) {
	doc := `<w:tbl>` +
		`<w:tr><w:t>keep</w:t></w:tr>` +
		`<w:tr><w:t>` + HideRowSentinel + `</w:t></w:tr>` +
		`<w:tr><w:t>keep2</w:t></w:tr>` +
		`</w:tbl>`
	got := removeHiddenRows(doc)
	if strings.Contains(got, HideRowSentinel) {
		t.Error("sentinel should be gone")
	}
	if !strings.Contains(got, "keep") || !strings.Contains(got, "keep2") {
		t.Error("other rows should survive")
	}
	if strings.Count(got, "<w:tr>") != 2 {
		t.Errorf("expected 2 rows, got: %s", got)
	}
}

func TestRemoveHiddenRows_WithAttr(t *testing.T) {
	doc := `<w:tr w:rsidR="001"><w:t>` + HideRowSentinel + `</w:t></w:tr>` +
		`<w:tr w:rsidR="002"><w:t>ok</w:t></w:tr>`
	got := removeHiddenRows(doc)
	if strings.Contains(got, HideRowSentinel) {
		t.Error("sentinel row should be removed")
	}
	if !strings.Contains(got, "ok") {
		t.Error("normal row should survive")
	}
}

func TestRemoveHiddenRows_NoneMarked(t *testing.T) {
	doc := `<w:tr><w:t>hello</w:t></w:tr>`
	if got := removeHiddenRows(doc); got != doc {
		t.Errorf("expected unchanged, got %s", got)
	}
}

func TestRemoveHiddenRows_AllMarked(t *testing.T) {
	doc := `<w:tr><w:t>` + HideRowSentinel + `</w:t></w:tr>` +
		`<w:tr><w:t>` + HideRowSentinel + `</w:t></w:tr>`
	got := removeHiddenRows(doc)
	if strings.Contains(got, "<w:tr>") {
		t.Errorf("expected no rows, got: %s", got)
	}
}

func TestPageBreakPostProcessor(t *testing.T) {
	m := pageBreakPostProcessor()
	handlers := m["word/document.xml"]
	if len(handlers) == 0 {
		t.Fatal("no handlers")
	}
	cases := []struct {
		in      string
		wantTag bool
		wantN   int
	}{
		{PageBreakPlaceholder, true, 1},
		{PageBreakPlaceholder + "<w:p/>" + PageBreakPlaceholder, true, 2},
		{"<w:p><w:t>plain</w:t></w:p>", false, 0},
	}
	for _, c := range cases {
		got, err := handlers[0](c.in)
		if err != nil {
			t.Fatal(err)
		}
		n := strings.Count(got, `w:type="page"`)
		if n != c.wantN {
			t.Errorf("input %q: got %d page breaks, want %d. output: %s", c.in, n, c.wantN, got)
		}
	}
}
