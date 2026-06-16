package docx

import (
	"strings"

	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
	"github.com/JJJJJJack/go-template-docx/internal/xml"
)

// DefaultPostProcessors returns the default XML post-processors that are
// applied to every rendered document (hideRow, pageBreak).
func DefaultPostProcessors() []xml.HandlersMap {
	return []xml.HandlersMap{
		{
			"word/document.xml": {func(s string) (string, error) {
				return removeHiddenRows(s), nil
			}},
		},
		{
			"word/document.xml": {func(s string) (string, error) {
				return strings.ReplaceAll(s, PageBreakPlaceholder, PageBreakReplacement), nil
			}},
		},
	}
}

func removeHiddenRows(content string) string {
	const rowClose = "</w:tr>"
	var sb strings.Builder
	sb.Grow(len(content))
	pos := 0
	for pos < len(content) {
		start := xmlutil.FindNext(content, pos, "<w:tr ", "<w:tr>")
		if start == -1 {
			sb.WriteString(content[pos:])
			break
		}
		end := strings.Index(content[start:], rowClose)
		if end == -1 {
			sb.WriteString(content[pos:])
			break
		}
		end += start + len(rowClose)
		rowXML := content[start:end]
		if strings.Contains(rowXML, HideRowSentinel) {
			sb.WriteString(content[pos:start])
		} else {
			sb.WriteString(content[pos:end])
		}
		pos = end
	}
	return sb.String()
}
