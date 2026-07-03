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

func TestPatchXML_Idempotent(t *testing.T) {
	// Структурные трансформации PatchXML (склейка скобок, удаление XML-тегов)
	// должны быть идемпотентными — повторный вызов не меняет результат.
	// Это гарантирует безопасность, если PatchXML применяется через wildcard
	// DefaultPreProcessors ко всем .xml/.rels частям, а downstream-обработчики
	// (normalizeDocxplateHandler) его не дублируют.
	//
	// Распаковка цепочек XML-сущностей (например &amp;lt; → &lt; → <) НЕ является
	// идемпотентной по своей природе, но PatchXML вызывается ровно один раз
	// в pipeline, поэтому такие случаи исключены из теста.
	inputs := []string{
		// разделённые скобки
		`before{ {.Name}} after`,
		`before{{.Name} } after`,
		// XML-теги внутри шаблонного выражения
		`before{{ .Name <w:rPr><w:b/></w:rPr>}} after`,
		// unescape XML-сущностей (один уровень)
		`{{shapeBgFillColor (index .Map &quot;Color2&quot;)}}`,
		// голый hex-аргумент
		`{{shapeBgFillColor 00FF00}}`,
		`{{tableCellBgColor 00FF00}}`,
		// уже чистый
		`hello {{.Name}} world`,
		// split-run: разрыв внутри слова {{Pa + ges.Name}}
		`<w:r><w:t>{{Pa</w:t></w:r><w:r><w:t>ges.Name}}</w:t></w:r>`,
		// split-run: разрыв на границе {{/}}
		`<w:r><w:t>{{Pages.Name</w:t></w:r><w:r><w:t>}}</w:t></w:r>`,
		// сущности без цепочек: &amp; → &, &lt; → < — по отдельности
		`{{&amp;}}`,
		`{{&lt;}}`,
	}

	for _, input := range inputs {
		once := PatchXML(input)
		twice := PatchXML(once)
		if once != twice {
			t.Errorf("PatchXML не идемпотентен для входной строки %q:\nonce:  %q\ntwice: %q", input, once, twice)
		}
	}
}
