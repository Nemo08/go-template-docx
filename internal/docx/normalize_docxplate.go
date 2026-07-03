package docx

import (
	"regexp"

	"github.com/JJJJJJack/go-template-docx/internal/xml"
)

// reNoDotDocxplate matches docxplate/JJack references {{Word.Word}} without a
// leading dot and replaces them with {{.Word.Word}} (valid Go template syntax).
// This is logically related to autoexpand.DocxplateVar which uses the same
// pattern but with an optional leading dot (\.?) and a different replacement
// (normalization for table-row expansion). If you change one, check the other.
var reNoDotDocxplate = regexp.MustCompile(`\{\{\s*(\w+)\.(\w+)\s*\}\}`)

// normalizeDocxplateHandler нормализует docxplate-синтаксис {{Word.Word}} в
// {{.Word.Word}}. PatchXML не вызывается — DefaultPreProcessors уже применил
// его ко всем .xml/.rels частям через wildcard-ключ "*".
func normalizeDocxplateHandler(content string) (string, error) {
	return reNoDotDocxplate.ReplaceAllString(content, "{{.$1.$2}}"), nil
}

// DocxplateCompatPreProcessor возвращает обработчик с wildcard-ключом "*",
// который нормализует docxplate-синтаксис во всех XML и .rels частях документа.
// Бинарные части (изображения, шрифты) не затрагиваются — wildcard в
// ProcessedOutput проверяет isTextPart() перед применением обработчика.
// PatchXML не дублируется — DefaultPreProcessors уже применил его.
func DocxplateCompatPreProcessor() xml.HandlersMap {
	return xml.HandlersMap{
		"*": {normalizeDocxplateHandler},
	}
}
