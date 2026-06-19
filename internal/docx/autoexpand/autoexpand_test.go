package autoexpand

import (
	"strings"
	"testing"

	"github.com/JJJJJJack/go-template-docx/internal/util"
)

func TestReflectToMap_MapStringAny(t *testing.T) {
	input := map[string]any{"Rows": []any{1, 2, 3}}
	got := util.ToStringMap(input)
	if got == nil {
		t.Fatal("expected non-nil map")
	}
	if _, ok := got["Rows"]; !ok {
		t.Error("expected Rows key")
	}
}

func TestReflectToMap_Struct(t *testing.T) {
	type Row struct{ Name string }
	type Data struct{ Items []Row }
	input := Data{Items: []Row{{Name: "A"}, {Name: "B"}}}
	got := util.ToStringMap(input)
	if got == nil {
		t.Fatal("expected non-nil")
	}
}

func TestReflectToMap_JSONBytes(t *testing.T) {
	input := []byte(`{"Rows":[1,2]}`)
	got := util.ToStringMap(input)
	if got == nil {
		t.Fatal("expected non-nil")
	}
}

func TestReflectToMap_Nil(t *testing.T) {
	got := util.ToStringMap(nil)
	if got != nil {
		t.Error("expected nil for nil input")
	}
}

func TestReflectToMap_InvalidJSON(t *testing.T) {
	got := util.ToStringMap([]byte("not json"))
	if got != nil {
		t.Error("expected nil for invalid json bytes")
	}
}

func TestSliceLen_Found(t *testing.T) {
	m := map[string]any{"Items": []any{"a", "b", "c"}}
	n, ok := sliceLen(m, "Items")
	if !ok {
		t.Fatal("expected ok")
	}
	if n != 3 {
		t.Errorf("expected 3, got %d", n)
	}
}

func TestSliceLen_NotFound(t *testing.T) {
	m := map[string]any{}
	_, ok := sliceLen(m, "Missing")
	if ok {
		t.Error("expected not ok for missing key")
	}
}

func TestSliceLen_NotSlice(t *testing.T) {
	m := map[string]any{"Key": "string"}
	_, ok := sliceLen(m, "Key")
	if ok {
		t.Error("expected not ok for non-slice value")
	}
}

func TestSliceLen_Empty(t *testing.T) {
	m := map[string]any{"Items": []any{}}
	n, ok := sliceLen(m, "Items")
	if !ok {
		t.Fatal("expected ok for empty slice")
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestRewriteIndex_DotStyle(t *testing.T) {
	row := `<w:t>{{.Items.0.Name}}</w:t>`
	got := rewriteIndex(row, 2)
	if !strings.Contains(got, ".2.") {
		t.Errorf("expected .2. in output, got %s", got)
	}
	if strings.Contains(got, ".0.") {
		t.Errorf("expected no .0. in output, got %s", got)
	}
}

func TestRewriteIndex_IndexStyle(t *testing.T) {
	row := `<w:t>{{(index .Items 0).Name}}</w:t>`
	got := rewriteIndex(row, 3)
	if !strings.Contains(got, "3") {
		t.Errorf("expected index 3 in output, got %s", got)
	}
}

func TestRewriteIndex_MultipleFields(t *testing.T) {
	row := `<w:t>{{.Rows.0.Name}}</w:t><w:t>{{.Rows.0.Value}}</w:t>`
	got := rewriteIndex(row, 1)
	count := strings.Count(got, ".1.")
	if count != 2 {
		t.Errorf("expected 2 occurrences of .1., got %d in: %s", count, got)
	}
}

func TestRewriteIndex_NoTemplate(t *testing.T) {
	row := `<w:t>plain text</w:t>`
	got := rewriteIndex(row, 5)
	if got != row {
		t.Errorf("expected unchanged, got %s", got)
	}
}

func TestTryExpandRow_NotExpandable(t *testing.T) {
	row := `<w:tr><w:t>{{.Title}}</w:t></w:tr>`
	dm := map[string]any{"Title": "hello"}
	got, err := tryExpandRow(row, dm)
	if err != nil {
		t.Fatal(err)
	}
	if got != row {
		t.Errorf("expected unchanged row, got %s", got)
	}
}

func TestTryExpandRow_ExpandsStructSlice(t *testing.T) {
	row := `<w:tr><w:t>{{.Items.0.Name}}</w:t></w:tr>`
	dm := map[string]any{
		"Items": []any{
			map[string]any{"Name": "A"},
			map[string]any{"Name": "B"},
			map[string]any{"Name": "C"},
		},
	}
	got, err := tryExpandRow(row, dm)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got, "<w:tr>") != 3 {
		t.Errorf("expected 3 rows, got: %s", got)
	}
	if !strings.Contains(got, ".0.") {
		t.Error("expected original .0. row")
	}
	if !strings.Contains(got, ".1.") {
		t.Error("expected cloned .1. row")
	}
	if !strings.Contains(got, ".2.") {
		t.Error("expected cloned .2. row")
	}
}

func TestTryExpandRow_ExpandsScalarSlice(t *testing.T) {
	row := `<w:tr><w:t>{{.Vals.0}}</w:t></w:tr>`
	dm := map[string]any{"Vals": []any{"x", "y"}}
	got, err := tryExpandRow(row, dm)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got, "<w:tr>") != 2 {
		t.Errorf("expected 2 rows, got: %s", got)
	}
}

func TestTryExpandRow_MissingKey(t *testing.T) {
	row := `<w:tr><w:t>{{.NoSuch.0.Name}}</w:t></w:tr>`
	dm := map[string]any{}
	got, err := tryExpandRow(row, dm)
	if err != nil {
		t.Fatal(err)
	}
	if got != row {
		t.Errorf("expected unchanged row for missing key, got %s", got)
	}
}

func TestTryExpandRow_EmptySlice(t *testing.T) {
	row := `<w:tr><w:t>{{.Items.0.Name}}</w:t></w:tr>`
	dm := map[string]any{"Items": []any{}}
	got, err := tryExpandRow(row, dm)
	if err != nil {
		t.Fatal(err)
	}
	if got != row {
		t.Errorf("expected unchanged row for empty slice, got %s", got)
	}
}

func TestExpandRows_MultipleTablesAndRows(t *testing.T) {
	doc := `<w:tbl>` +
		`<w:tr><w:t>header</w:t></w:tr>` +
		`<w:tr><w:t>{{.Rows.0.Val}}</w:t></w:tr>` +
		`</w:tbl>`
	dm := map[string]any{
		"Rows": []any{
			map[string]any{"Val": "one"},
			map[string]any{"Val": "two"},
		},
	}
	got, err := expandRows(doc, dm)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "header") {
		t.Error("header row should be preserved")
	}
	if strings.Count(got, "<w:tr>") != 3 {
		t.Errorf("expected 3 rows total, got doc: %s", got)
	}
}

func TestExpandRows_NoRows(t *testing.T) {
	doc := `<w:body><w:p><w:t>{{.Title}}</w:t></w:p></w:body>`
	dm := map[string]any{"Title": "hi"}
	got, err := expandRows(doc, dm)
	if err != nil {
		t.Fatal(err)
	}
	if got != doc {
		t.Errorf("expected unchanged doc, got %s", got)
	}
}

func TestAutoExpandRowsPreProcessor_ReturnsHandlerForDocument(t *testing.T) {
	data := map[string]any{"Items": []any{"a"}}
	m := AutoExpandRowsPreProcessor(data)
	if _, ok := m["word/document.xml"]; !ok {
		t.Error("expected handler for word/document.xml")
	}
}

func TestTryExpandRow_DocxplateFormat(t *testing.T) {
	row := `<w:tr><w:t>{{Pages.Name}}</w:t></w:tr>`
	dm := map[string]any{
		"Pages": []any{
			map[string]any{"Name": "A"},
			map[string]any{"Name": "B"},
			map[string]any{"Name": "C"},
		},
	}
	got, err := tryExpandRow(row, dm)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got, "<w:tr>") != 3 {
		t.Errorf("expected 3 rows, got: %s", got)
	}
	if !strings.Contains(got, "index .Pages 0") {
		t.Error("expected original (index .Pages 0) row")
	}
	if !strings.Contains(got, "index .Pages 1") {
		t.Error("expected cloned (index .Pages 1) row")
	}
	if !strings.Contains(got, "index .Pages 2") {
		t.Error("expected cloned (index .Pages 2) row")
	}
}

func TestTryExpandRow_DocxplateWithLeadingDot(t *testing.T) {
	row := `<w:tr><w:t>{{.Items.Name}}</w:t></w:tr>`
	dm := map[string]any{
		"Items": []any{
			map[string]any{"Name": "X"},
			map[string]any{"Name": "Y"},
		},
	}
	got, err := tryExpandRow(row, dm)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got, "<w:tr>") != 2 {
		t.Errorf("expected 2 rows, got: %s", got)
	}
}

func TestTryExpandRow_DocxplateNonSlice(t *testing.T) {
	row := `<w:tr><w:t>{{Title.Name}}</w:t></w:tr>`
	dm := map[string]any{"Title": map[string]any{"Name": "hello"}}
	got, err := tryExpandRow(row, dm)
	if err != nil {
		t.Fatal(err)
	}
	if got != row {
		t.Errorf("expected unchanged for non-slice, got %s", got)
	}
}

func TestTryExpandRow_DocxplateMissingKey(t *testing.T) {
	row := `<w:tr><w:t>{{NoSuch.Name}}</w:t></w:tr>`
	dm := map[string]any{}
	got, err := tryExpandRow(row, dm)
	if err != nil {
		t.Fatal(err)
	}
	if got != row {
		t.Errorf("expected unchanged for missing key, got %s", got)
	}
}

func TestNormalizeDocxplateRow(t *testing.T) {
	row := `<w:t>{{Pages.Name}}</w:t><w:t>{{Pages.Page}}</w:t>`
	got := normalizeDocxplateRow(row, "Pages")
	expected := `<w:t>{{(index .Pages 0).Name}}</w:t><w:t>{{(index .Pages 0).Page}}</w:t>`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestNormalizeDocxplateRow_WithLeadingDot(t *testing.T) {
	row := `<w:t>{{.Pages.Name}}</w:t>`
	got := normalizeDocxplateRow(row, "Pages")
	expected := `<w:t>{{(index .Pages 0).Name}}</w:t>`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestNormalizeDocxplateRow_OnlyMatchingArray(t *testing.T) {
	row := `<w:t>{{Pages.Name}}</w:t><w:t>{{Other.Title}}</w:t>`
	got := normalizeDocxplateRow(row, "Pages")
	expected := `<w:t>{{(index .Pages 0).Name}}</w:t><w:t>{{Other.Title}}</w:t>`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestAutoExpandRowsPreProcessor_HandlerExpands(t *testing.T) {
	data := map[string]any{
		"Items": []any{
			map[string]any{"Name": "Alice"},
			map[string]any{"Name": "Bob"},
		},
	}
	m := AutoExpandRowsPreProcessor(data)
	handlers := m["word/document.xml"]
	if len(handlers) == 0 {
		t.Fatal("expected at least one handler")
	}
	input := `<w:tr><w:t>{{.Items.0.Name}}</w:t></w:tr>`
	got, err := handlers[0](input)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got, "<w:tr>") != 2 {
		t.Errorf("expected 2 rows, got: %s", got)
	}
}
