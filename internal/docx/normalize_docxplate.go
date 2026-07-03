package docx

import (
	"regexp"

	"github.com/JJJJJJack/go-template-docx/internal/xml"
	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
)

// reNoDotDocxplate matches docxplate/JJack references {{Word.Word}} without a
// leading dot and replaces them with {{.Word.Word}} (valid Go template syntax).
// This is logically related to autoexpand.reDocxplateVar which uses the same
// pattern but with an optional leading dot (\.?) and a different replacement
// (normalization for table-row expansion). If you change one, check the other.
var reNoDotDocxplate = regexp.MustCompile(`\{\{\s*(\w+)\.(\w+)\s*\}\}`)

// normalizeDocxplateHandler нормализует docxplate-синтаксис {{Word.Word}} в
// {{.Word.Word}}. Перед regex применяет PatchXML, чтобы сначала склеить
// конструкцию {{ }} из разорванных <w:r>-элементов Word — иначе препроцессор
// не увидит плейсхолдеры, физически разбитые на несколько text run'ов.
func normalizeDocxplateHandler(content string) (string, error) {
	patched := xmlutil.PatchXML(content)
	return reNoDotDocxplate.ReplaceAllString(patched, "{{.$1.$2}}"), nil
}

// DocxplateCompatPreProcessor возвращает обработчик с wildcard-ключом "*",
// который нормализует docxplate-синтаксис во всех XML и .rels частях документа.
// Бинарные части (изображения, шрифты) не затрагиваются — wildcard в
// ProcessedOutput проверяет isTextPart() перед применением обработчика.
func DocxplateCompatPreProcessor() xml.HandlersMap {
	return xml.HandlersMap{
		"*": {normalizeDocxplateHandler},
	}
}
