package docx

import (
	"regexp"

	"github.com/JJJJJJack/go-template-docx/internal/xml"
)

// reNoDotDocxplate matches docxplate/JJack references {{Word.Word}} without a
// leading dot and replaces them with {{.Word.Word}} (valid Go template syntax).
// This is logically related to autoexpand.reDocxplateVar which uses the same
// pattern but with an optional leading dot (\.?) and a different replacement
// (normalization for table-row expansion). If you change one, check the other.
var reNoDotDocxplate = regexp.MustCompile(`\{\{\s*(\w+)\.(\w+)\s*\}\}`)

func normalizeDocxplateHandler(content string) (string, error) {
	return reNoDotDocxplate.ReplaceAllString(content, "{{.$1.$2}}"), nil
}

func DocxplateCompatPreProcessor() xml.HandlersMap {
	return xml.HandlersMap{
		"*": {normalizeDocxplateHandler},
	}
}
