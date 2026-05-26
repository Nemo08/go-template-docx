package docx

import (
	"fmt"
	"regexp"
	"strings"
)

// Скомпилированные регэкспы уровня пакета — компилируются один раз при старте.
var (
	reTableRow      = regexp.MustCompile(`(?s)<w:tr\b[^>]*>.*?</w:tr>`)
	reTableRowParts = regexp.MustCompile(`(?s)(<w:tr\b[^>]*>)(.*?)(</w:tr>)`)
	reTextContent   = regexp.MustCompile(`(?is)<w:t\b[^>]*>(.*?)</w:t>`)
	reVisualContent = regexp.MustCompile(`(?is)<w:drawing\b|<w:pict\b|<mc:AlternateContent\b|<v:shape\b|<wps:spPr\b`)
	reXmlSpace      = regexp.MustCompile(`xml:space="preserve"`)
	reTextElement   = regexp.MustCompile(`<w:t\b([^>]*)>([\s\S]*?)</w:t>`)
	reNestedRun     = regexp.MustCompile(`(?is)<w:t\b([^>]*)>\s*(<w:rPr>[\s\S]*?</w:rPr>)\s*<w:t\b([^>]*)>([\s\S]*?)</w:t>\s*</w:t>`)
	// Директивы Go-шаблона, строки с которыми должны удаляться после рендеринга.
	// Проверяем текстовое содержимое ячеек (внутри <w:t>), а не весь XML строки —
	// так надёжнее после PatchXml, который уже убрал XML-теги из выражений {{ }}.
	reDirectiveText = regexp.MustCompile(`\{\{\s*(?:range|if|else|with|end)\b`)
)

const rangeRowMarker = `<!--ppdftb_range_row-->`

// removeEmptyTableRows removes empty table rows from the provided XML string.
func removeEmptyTableRows(srcXML string) string {
	return reTableRow.ReplaceAllStringFunc(srcXML, func(row string) string {
		if isRowEmpty(row) {
			return ""
		}
		return row
	})
}

func isRowEmpty(row string) bool {
	if reVisualContent.MatchString(row) {
		return false
	}
	texts := reTextContent.FindAllStringSubmatch(row, -1)
	if len(texts) == 0 {
		return true
	}
	for _, m := range texts {
		if strings.TrimSpace(m[1]) != "" {
			return false
		}
	}
	return true
}

// markRangeDirectiveRows добавляет маркер к строкам таблицы, текстовое
// содержимое которых содержит директивы Go-шаблона (range/if/else/with/end).
// Вызывать ПОСЛЕ PatchXml — иначе {{ }} могут быть разбиты по нескольким
// XML-тегам и регэксп их не найдёт.
func markRangeDirectiveRows(srcXML string) string {
	return reTableRowParts.ReplaceAllStringFunc(srcXML, func(row string) string {
		m := reTableRowParts.FindStringSubmatch(row)
		if m == nil {
			return row
		}
		// Проверяем только текстовое содержимое ячеек, а не весь XML строки.
		// Это исключает ложные срабатывания на XML-атрибуты или комментарии.
		hasDirective := false
		for _, tm := range reTextContent.FindAllStringSubmatch(m[2], -1) {
			if reDirectiveText.MatchString(tm[1]) {
				hasDirective = true
				break
			}
		}
		if !hasDirective {
			return row
		}
		return m[1] + m[2] + rangeRowMarker + m[3]
	})
}

// removeMarkedEmptyRows удаляет только те строки таблицы, которые:
// 1. были помечены как директивные (содержат rangeRowMarker);
// 2. после выполнения шаблона оказались пустыми.
// Обычные пустые строки без маркера остаются нетронутыми.
func removeMarkedEmptyRows(srcXML string) string {
	return reTableRow.ReplaceAllStringFunc(srcXML, func(row string) string {
		if !strings.Contains(row, rangeRowMarker) {
			return row
		}
		if isRowEmpty(row) {
			return ""
		}
		return row
	})
}

// propagateRunPropsAfterBreak ensures that <w:r> elements created by breakParagraph
// inherit the <w:rPr> from the originating run.
func propagateRunPropsAfterBreak(srcXML string) string {
	re := regexp.MustCompile(`(<w:rPr>[^<]*(?:<[^>]+/>[^<]*)*</w:rPr>)(<w:t[^>]*>[^<]*</w:t></w:r></w:p><w:p><w:r>)<w:t`)
	for {
		next := re.ReplaceAllString(srcXML, `${1}${2}${1}<w:t`)
		if next == srcXML {
			break
		}
		srcXML = next
	}
	return srcXML
}

// ensureXmlSpacePreserve ensures all <w:t> elements with leading/trailing
// whitespace get xml:space="preserve".
func ensureXmlSpacePreserve(srcXML string) string {
	return reTextElement.ReplaceAllStringFunc(srcXML, func(match string) string {
		submatches := reTextElement.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		attrs := submatches[1]
		text := submatches[2]

		hasAttribute := reXmlSpace.MatchString(attrs)
		needsAttribute := text != "" && text != strings.TrimSpace(text)

		if hasAttribute || !needsAttribute {
			return match
		}
		return fmt.Sprintf(`<w:t xml:space="preserve">%s</w:t>`, text)
	})
}

// flattenNestedTextRuns fixes cases where a template function that returns
// `<w:rPr>..</w:rPr><w:t>..</w:t>` got injected inside an existing `<w:t>`.
func flattenNestedTextRuns(srcXML string) string {
	for reNestedRun.MatchString(srcXML) {
		srcXML = reNestedRun.ReplaceAllStringFunc(srcXML, func(match string) string {
			submatches := reNestedRun.FindStringSubmatch(match)
			if len(submatches) < 5 {
				return match
			}
			outerAttrs := submatches[1]
			rPr := submatches[2]
			innerAttrs := submatches[3]
			text := submatches[4]

			if reXmlSpace.MatchString(outerAttrs) || reXmlSpace.MatchString(innerAttrs) {
				return fmt.Sprintf(`%s<w:t xml:space="preserve">%s</w:t>`, rPr, text)
			}
			return fmt.Sprintf(`%s<w:t>%s</w:t>`, rPr, text)
		})
	}
	return srcXML
}
