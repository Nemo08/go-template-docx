package docx

import (
	"regexp"

	"github.com/JJJJJJack/go-template-docx/internal/xml"
	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
)

// reNoDotDocxplate matches docxplate/JJack references {{Word.Word}} without a
// leading dot and replaces them with {{.Word.Word}} (valid Go template syntax).
// This is logically related to autoexpand.DocxplateVar which uses the same
// pattern but with an optional leading dot (\.?) and a different replacement
// (normalization for table-row expansion). If you change one, check the other.
var reNoDotDocxplate = regexp.MustCompile(`\{\{\s*(\w+)\.(\w+)\s*\}\}`)

// normalizeDocxplateHandler нормализует docxplate-синтаксис {{Word.Word}} в
// {{.Word.Word}}.
//
// Вызов xmlutil.PatchXML перед regex — defensive: в стандартном pipeline
// DefaultPreProcessors уже применил PatchXML ко всем .xml/.rels частям,
// но этот вызов гарантирует корректную работу даже если DocxplateCompat
// применяется без DefaultPreProcessors или в другом порядке. Идемпотентность
// PatchXML подтверждена тестом TestPatchXML_Idempotent — повторный вызов
// не меняет результат на уже пропатченном содержимом.
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
