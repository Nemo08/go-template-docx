package xmlutil

import (
	"strings"
	"testing"
)

func TestMergeSplitPlaceholders_MidWord(t *testing.T) {
	input := `<w:p>
<w:r><w:t>before </w:t></w:r>
<w:r><w:t>{{Pa</w:t></w:r>
<w:r><w:t>ges.Name}}</w:t></w:r>
<w:r><w:t> after</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected merged {{Pages.Name}} in output, got:\n%s", got)
	}
	if strings.Contains(got, "{{Pa</w:t>") {
		t.Errorf("expected split '{{Pa' to be merged, got:\n%s", got)
	}
	if strings.Contains(got, "<w:r><w:t>ges.Name}}") {
		t.Errorf("expected split run 'ges.Name}}' to be merged, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_ThreeOrMoreRuns(t *testing.T) {
	input := `<w:p>
<w:r><w:t>{{Pages.</w:t></w:r>
<w:r><w:t>Name</w:t></w:r>
<w:r><w:t>}}</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected merged {{Pages.Name}} across 3 runs, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_PreservesFormattingOfFirstRun(t *testing.T) {
	input := `<w:p>
<w:r><w:rPr><w:b/></w:rPr><w:t>{{Pa</w:t></w:r>
<w:r><w:rPr><w:i/></w:rPr><w:t>ges.Name}}</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Должен быть <w:b/> (из первого рана), а не <w:i/>
	if !strings.Contains(got, "<w:b/>") {
		t.Errorf("expected <w:b/> from first run in merged output, got:\n%s", got)
	}
	if strings.Contains(got, "<w:i/>") {
		t.Errorf("expected NO <w:i/> from second run in merged output, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_PreservesSurroundingText(t *testing.T) {
	input := `<w:p>
<w:r><w:rPr><w:b/></w:rPr><w:t>before </w:t></w:r>
<w:r><w:rPr><w:i/></w:rPr><w:t>{{Pa</w:t></w:r>
<w:r><w:rPr><w:u/></w:rPr><w:t>ges.Name}}</w:t></w:r>
<w:r><w:rPr><w:b/></w:rPr><w:t> after</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "before ") {
		t.Errorf("expected 'before ' text to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, " after") {
		t.Errorf("expected ' after' text to be preserved, got:\n%s", got)
	}
	// Форматирование окружающего текста должно сохраниться
	if !strings.Contains(got, "<w:b/>") {
		t.Errorf("expected <w:b/> formatting in output, got:\n%s", got)
	}
	if strings.Count(got, "<w:b/>") < 2 {
		t.Errorf("expected <w:b/> on both surrounding runs, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_SkipsTabBreak(t *testing.T) {
	input := `<w:p>
<w:r><w:t>{{Pages.</w:t></w:r>
<w:r><w:tab/></w:r>
<w:r><w:t>Name}}</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Не должен сливать, так как <w:tab/> между частями
	if strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected NO merge when <w:tab/> between runs, got merged:\n%s", got)
	}
	// Исходный текст должен остаться без изменений
	if !strings.Contains(got, "<w:tab/>") {
		t.Errorf("expected <w:tab/> to be preserved, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_Hyperlink(t *testing.T) {
	input := `<w:p>
<w:r><w:t>before </w:t></w:r>
<w:hyperlink r:id="rId1">
<w:r><w:t>{{Pa</w:t></w:r>
<w:r><w:t>ges.Name}}</w:t></w:r>
</w:hyperlink>
<w:r><w:t> after</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected merged {{Pages.Name}} inside hyperlink, got:\n%s", got)
	}
	if !strings.Contains(got, "<w:hyperlink") {
		t.Errorf("expected hyperlink wrapper to be preserved, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_Bookmark(t *testing.T) {
	input := `<w:p>
<w:bookmarkStart w:id="0" w:name="myBM"/>
<w:r><w:t>{{Pa</w:t></w:r>
<w:r><w:t>ges.Name}}</w:t></w:r>
<w:bookmarkEnd w:id="0"/>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected merged {{Pages.Name}} with bookmark, got:\n%s", got)
	}
	if !strings.Contains(got, "<w:bookmarkStart") {
		t.Errorf("expected bookmarkStart to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "<w:bookmarkEnd") {
		t.Errorf("expected bookmarkEnd to be preserved, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_MultiplePlaceholdersInParagraph(t *testing.T) {
	input := `<w:p>
<w:r><w:t>{{Pa</w:t></w:r>
<w:r><w:t>ges.Name}}</w:t></w:r>
<w:r><w:t> </w:t></w:r>
<w:r><w:t>{{Ite</w:t></w:r>
<w:r><w:t>ms.Title}}</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected first placeholder merged, got:\n%s", got)
	}
	if !strings.Contains(got, "{{Items.Title}}") {
		t.Errorf("expected second placeholder merged, got:\n%s", got)
	}
	if strings.Count(got, "{{Pages.Name}}") != 1 {
		t.Errorf("expected exactly one {{Pages.Name}}, got:\n%s", got)
	}
	if strings.Count(got, "{{Items.Title}}") != 1 {
		t.Errorf("expected exactly one {{Items.Title}}, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_XmlSpacePreserve(t *testing.T) {
	// Проверяем, что xml:space="preserve" сохраняется на run'ах,
	// которые не сливаются (окружающий текст).
	input := `<w:p>
<w:r><w:t xml:space="preserve">before </w:t></w:r>
<w:r><w:t>{{Pa</w:t></w:r>
<w:r><w:t>ges.Name}}</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected merged {{Pages.Name}}, got:\n%s", got)
	}
	// xml:space="preserve" на окружающем run'е "before " должен сохраниться
	if !strings.Contains(got, "xml:space=\"preserve\"") {
		t.Errorf("expected xml:space=\"preserve\" to be preserved on surrounding run, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_XmlSpacePreserveOnMerged(t *testing.T) {
	// Если один из сливаемых run'ов имел xml:space="preserve",
	// результирующий слитый run тоже должен его иметь.
	input := `<w:p>
<w:r><w:t xml:space="preserve">{{Pa</w:t></w:r>
<w:r><w:t>ges.Name}}</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected merged {{Pages.Name}}, got:\n%s", got)
	}
	// Слитый run должен получить xml:space="preserve" от первого run'а
	if !strings.Contains(got, "xml:space=\"preserve\"") {
		t.Errorf("expected xml:space=\"preserve\" on merged run, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_Idempotent(t *testing.T) {
	inputs := []string{
		// Разрыв посреди слова
		`<w:p><w:r><w:t>{{Pa</w:t></w:r><w:r><w:t>ges.Name}}</w:t></w:r></w:p>`,
		// Три рана
		`<w:p><w:r><w:t>{{Pages.</w:t></w:r><w:r><w:t>Name</w:t></w:r><w:r><w:t>}}</w:t></w:r></w:p>`,
		// Без разрыва
		`<w:p><w:r><w:t>{{Pages.Name}}</w:t></w:r></w:p>`,
		// Несколько плейсхолдеров
		`<w:p><w:r><w:t>{{Pa</w:t></w:r><w:r><w:t>ges.Name}}</w:t></w:r><w:r><w:t> </w:t></w:r><w:r><w:t>{{Ite</w:t></w:r><w:r><w:t>ms.Title}}</w:t></w:r></w:p>`,
		// С форматированием
		`<w:p><w:r><w:rPr><w:b/></w:rPr><w:t>{{Pa</w:t></w:r><w:r><w:rPr><w:i/></w:rPr><w:t>ges.Name}}</w:t></w:r></w:p>`,
		// Нет {{, не должно меняться
		`<w:p><w:r><w:t>plain text</w:t></w:r></w:p>`,
		// Уже слитый
		`<w:p><w:r><w:t>{{Pages.Name}}</w:t></w:r></w:p>`,
	}

	for _, input := range inputs {
		once, err := MergeSplitPlaceholders(input)
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", input, err)
		}
		twice, err := MergeSplitPlaceholders(once)
		if err != nil {
			t.Fatalf("unexpected error on second call for input %q: %v", input, err)
		}
		if once != twice {
			t.Errorf("MergeSplitPlaceholders not idempotent for input %q:\nonce:  %q\ntwice: %q", input, once, twice)
		}
	}
}

func TestMergeSplitPlaceholders_NoPlaceholder(t *testing.T) {
	input := `<w:p>
<w:r><w:t>plain text without placeholders</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("expected no changes for input without placeholders, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_MultipleParagraphs(t *testing.T) {
	input := `<w:body>
<w:p>
<w:r><w:t>first paragraph</w:t></w:r>
</w:p>
<w:p>
<w:r><w:t>{{Pa</w:t></w:r>
<w:r><w:t>ges.Name}}</w:t></w:r>
</w:p>
<w:p>
<w:r><w:t>third paragraph</w:t></w:r>
</w:p>
</w:body>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected merged placeholder in second paragraph, got:\n%s", got)
	}
	if !strings.Contains(got, "first paragraph") {
		t.Errorf("expected first paragraph preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "third paragraph") {
		t.Errorf("expected third paragraph preserved, got:\n%s", got)
	}
}

func TestMergeSplitPlaceholders_PreservesOrderOfAttributeStyle(t *testing.T) {
	// Плейсхолдер, где первый ран содержит текст до и часть плейсхолдера
	input := `<w:p>
<w:r><w:t>text before {{Pa</w:t></w:r>
<w:r><w:t>ges.Name}} text after</w:t></w:r>
</w:p>`

	got, err := MergeSplitPlaceholders(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "text before ") {
		t.Errorf("expected 'text before ' preserved, got:\n%s", got)
	}
	if !strings.Contains(got, " text after") {
		t.Errorf("expected ' text after' preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "{{Pages.Name}}") {
		t.Errorf("expected merged {{Pages.Name}}, got:\n%s", got)
	}
}
