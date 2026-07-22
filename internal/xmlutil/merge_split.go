package xmlutil

import (
	"html"
	"regexp"
	"strings"
)

var rePlaceholder = regexp.MustCompile(`\{\{[\s\S]*?\}\}`)

// MergeSplitPlaceholders находит плейсхолдеры {{...}} в DOCX XML,
// которые разорваны на несколько <w:r> элементов, и сливает их в один <w:r>
// с форматированием первого затронутого рана.
//
// Ограничения:
//   - Если разрыв плейсхолдера проходит через непарный структурный элемент
//     (<w:tab/>, <w:br/>, <w:noBreakHyphen/>, <w:cr/>), слияние не
//     выполняется — такой плейсхолдер считается невалидным.
//   - Функция обрабатывает только <w:p> параграфы. Другие контейнеры
//     (например текстовые框ки) не поддерживаются.
func MergeSplitPlaceholders(content string) (string, error) {
	if !strings.Contains(content, "{{") {
		return content, nil
	}

	var result strings.Builder
	pos := 0
	for {
		pIdx := strings.Index(content[pos:], "<w:p")
		if pIdx == -1 {
			result.WriteString(content[pos:])
			break
		}
		pIdx += pos

		pClose := findCloseTag(content, pIdx, "w:p")
		if pClose == -1 {
			result.WriteString(content[pos:])
			break
		}

		result.WriteString(content[pos:pIdx])

		paraXML := content[pIdx:pClose]
		if strings.Contains(paraXML, "{{") {
			processed, err := processParagraph(paraXML)
			if err != nil {
				return "", err
			}
			result.WriteString(processed)
		} else {
			result.WriteString(paraXML)
		}

		pos = pClose
	}
	return result.String(), nil
}

// findCloseTag ищет закрывающий тег для заданного открывающего,
// корректно обрабатывая вложенность. start должен указывать на
// начало открывающего тега (включая сам тег).
func findCloseTag(s string, start int, tag string) int {
	openPrefix := "<" + tag
	closeTag := "</" + tag + ">"
	depth := 1

	// Пропускаем сам открывающий тег (tag + closing > + возможные атрибуты)
	tagEnd := strings.IndexByte(s[start:], '>')
	if tagEnd == -1 {
		return -1
	}
	pos := start + tagEnd + 1

	for depth > 0 && pos < len(s) {
		nextOpen := strings.Index(s[pos:], openPrefix)
		nextClose := strings.Index(s[pos:], closeTag)

		if nextClose == -1 {
			return -1
		}
		nextClose += pos

		if nextOpen != -1 && nextOpen+pos < nextClose && isTagOpen(s[nextOpen+pos:], tag) {
			depth++
			pos = nextOpen + pos + 1
		} else {
			depth--
			if depth == 0 {
				return nextClose + len(closeTag)
			}
			pos = nextClose + len(closeTag)
		}
	}
	return -1
}

type runData struct {
	rPr      string
	text     string
	preserve bool
}

type segment struct {
	raw   string
	isRun bool
	run   runData
}

// processParagraph обрабатывает один <w:p>, сливая разорванные плейсхолдеры.
func processParagraph(paraXML string) (string, error) {
	segs := parseParagraph(paraXML)

	if len(segs) == 0 {
		return paraXML, nil
	}

	// Собираем информацию о run'ах
	type ri struct {
		segIdx   int
		text     string
		textLen  int
		preserve bool
		rPr      string
	}
	var runs []ri
	for i, s := range segs {
		if s.isRun {
			runs = append(runs, ri{
				segIdx:   i,
				text:     s.run.text,
				textLen:  len(s.run.text),
				preserve: s.run.preserve,
				rPr:      s.run.rPr,
			})
		}
	}
	if len(runs) == 0 {
		return paraXML, nil
	}

	// Строим общий текст и таблицу смещений
	var combined strings.Builder
	runStarts := make([]int, len(runs))
	for i, r := range runs {
		runStarts[i] = combined.Len()
		combined.WriteString(r.text)
	}
	combinedStr := combined.String()

	// Ищем плейсхолдеры
	locs := rePlaceholder.FindAllStringIndex(combinedStr, -1)
	if len(locs) == 0 {
		return paraXML, nil
	}

	// Обрабатываем справа налево, чтобы не сбивать смещения
	for i := len(locs) - 1; i >= 0; i-- {
		phStart, phEnd := locs[i][0], locs[i][1]

		firstRun := findRun(phStart, runStarts)
		lastRun := findRun(phEnd-1, runStarts)
		if firstRun < 0 || lastRun < 0 || firstRun >= len(runs) || lastRun >= len(runs) {
			continue
		}
		if firstRun == lastRun {
			continue
		}

		// Проверяем структурные элементы между run'ами:
		// <w:tab/>, <w:br/>, <w:noBreakHyphen/>, <w:cr/>
		// Непарные/структурные элементы, которые НЕ являются текстом —
		// учитываются как разрывы последовательности текстовых кусков.
		// Прочие не-run элементы (переносы строк, теги обёрток вроде
		// <w:hyperlink>, <w:bookmarkStart>) не являются разрывами.
		firstSegIdx := runs[firstRun].segIdx
		lastSegIdx := runs[lastRun].segIdx
		hasStructural := false
		for j := firstSegIdx + 1; j < lastSegIdx; j++ {
			if containsStructuralElement(segs[j].raw) {
				hasStructural = true
				break
			}
		}
		if hasStructural {
			continue
		}

		// Разбираем текст до/после/внутри плейсхолдера
		firstText := runs[firstRun].text
		lastText := runs[lastRun].text
		firstOffset := runStarts[firstRun]
		lastOffset := runStarts[lastRun]
		lastEnd := lastOffset + len(lastText)

		// Текст до плейсхолдера в первом ране
		var prefixText string
		if phStart > firstOffset {
			prefixText = firstText[:phStart-firstOffset]
		}

		// Текст после плейсхолдера в последнем ране
		var suffixText string
		if phEnd < lastEnd {
			suffixText = lastText[phEnd-lastOffset:]
		}

		// Слитый текст плейсхолдера
		mergedText := combinedStr[phStart:phEnd]

		// Форматирование: rPr первого рана; preserve если любой из затронутых имел
		mergedRPr := runs[firstRun].rPr
		mergedPreserve := false
		for j := firstRun; j <= lastRun; j++ {
			if runs[j].preserve {
				mergedPreserve = true
				break
			}
		}

		// Перестраиваем сегменты
		var newSegs []segment
		newSegs = append(newSegs, segs[:firstSegIdx]...)

		if prefixText != "" {
			newSegs = append(newSegs, segment{
				raw:   buildRunXML(runs[firstRun].rPr, prefixText, runs[firstRun].preserve),
				isRun: true,
				run:   runData{rPr: runs[firstRun].rPr, text: prefixText, preserve: runs[firstRun].preserve},
			})
		}

		newSegs = append(newSegs, segment{
			raw:   buildRunXML(mergedRPr, mergedText, mergedPreserve),
			isRun: true,
			run:   runData{rPr: mergedRPr, text: mergedText, preserve: mergedPreserve},
		})

		if suffixText != "" {
			newSegs = append(newSegs, segment{
				raw:   buildRunXML(runs[lastRun].rPr, suffixText, runs[lastRun].preserve),
				isRun: true,
				run:   runData{rPr: runs[lastRun].rPr, text: suffixText, preserve: runs[lastRun].preserve},
			})
		}

		newSegs = append(newSegs, segs[lastSegIdx+1:]...)

		// Обновляем runs и runStarts для последующих итераций
		segs = newSegs
		runs = nil
		for i, s := range segs {
			if s.isRun {
				runs = append(runs, ri{
					segIdx:   i,
					text:     s.run.text,
					textLen:  len(s.run.text),
					preserve: s.run.preserve,
					rPr:      s.run.rPr,
				})
			}
		}
		runStarts = make([]int, len(runs))
		combined.Reset()
		for i, r := range runs {
			runStarts[i] = combined.Len()
			combined.WriteString(r.text)
		}
		combinedStr = combined.String()
	}

	// Собираем результат
	var out strings.Builder
	for _, s := range segs {
		out.WriteString(s.raw)
	}
	return out.String(), nil
}

// parseParagraph разбирает XML параграфа на сегменты (run и не-run).
func parseParagraph(paraXML string) []segment {
	var segs []segment
	pos := 0
	for pos < len(paraXML) {
		rIdx := strings.Index(paraXML[pos:], "<w:r")
		if rIdx == -1 {
			if pos < len(paraXML) {
				segs = append(segs, segment{raw: paraXML[pos:], isRun: false})
			}
			break
		}
		rIdx += pos

		if rIdx > pos {
			segs = append(segs, segment{raw: paraXML[pos:rIdx], isRun: false})
		}

		rEnd := findRunEnd(paraXML, rIdx)
		if rEnd == -1 {
			segs = append(segs, segment{raw: paraXML[rIdx:], isRun: false})
			break
		}

		runXML := paraXML[rIdx:rEnd]
		rd := extractRunData(runXML)
		segs = append(segs, segment{raw: runXML, isRun: true, run: rd})

		pos = rEnd
	}
	return segs
}

// isTagOpen проверяет, является ли строка, начинающаяся с <tag, открывающим
// тегом (а не, например, <tagPr> или </tag>).
func isTagOpen(s string, tag string) bool {
	prefix := "<" + tag
	if !strings.HasPrefix(s, prefix) {
		return false
	}
	after := s[len(prefix):]
	if len(after) == 0 {
		return false
	}
	return after[0] == '>' || after[0] == ' ' || after[0] == '/'
}

// isRunOpenTag проверяет, является ли строка, начинающаяся с <w:r, открывающим
// тегом run'а (а не, например, <w:rPr>).
func isRunOpenTag(s string) bool {
	return isTagOpen(s, "w:r")
}

// findRunEnd находит конец <w:r> элемента (позицию после </w:r>).
func findRunEnd(s string, start int) int {
	closeTag := "</w:r>"
	depth := 1
	pos := start + len("<w:r")
	for depth > 0 && pos < len(s) {
		nextOpen := strings.Index(s[pos:], "<w:r")
		nextClose := strings.Index(s[pos:], closeTag)

		if nextClose == -1 {
			return -1
		}
		nextClose += pos

		if nextOpen != -1 && nextOpen+pos < nextClose && isRunOpenTag(s[nextOpen+pos:]) {
			depth++
			pos = nextOpen + pos + 1
		} else {
			depth--
			if depth == 0 {
				return nextClose + len(closeTag)
			}
			pos = nextClose + len(closeTag)
		}
	}
	return -1
}

// extractRunData извлекает rPr, текст и preserve из XML <w:r>.
func extractRunData(runXML string) runData {
	var rPr string
	var texts []string
	preserve := false

	rPrIdx := strings.Index(runXML, "<w:rPr>")
	if rPrIdx != -1 {
		rPrEnd := strings.Index(runXML[rPrIdx:], "</w:rPr>")
		if rPrEnd != -1 {
			rPrContent := runXML[rPrIdx+len("<w:rPr>") : rPrIdx+rPrEnd]
			// Проверяем, есть ли что-то содержательное в rPr
			if strings.TrimSpace(rPrContent) != "" {
				rPr = rPrContent
			}
		}
	}

	tPos := 0
	for {
		tIdx := strings.Index(runXML[tPos:], "<w:t")
		if tIdx == -1 {
			break
		}
		tIdx += tPos

		tTagEnd := strings.IndexByte(runXML[tIdx:], '>')
		if tTagEnd == -1 {
			break
		}
		tAttr := runXML[tIdx : tIdx+tTagEnd]

		// Проверяем xml:space="preserve"
		if strings.Contains(tAttr, "xml:space=\"preserve\"") ||
			strings.Contains(tAttr, "xml:space='preserve'") {
			preserve = true
		}

		tContentStart := tIdx + tTagEnd + 1
		tClose := strings.Index(runXML[tContentStart:], "</w:t>")
		if tClose == -1 {
			break
		}
		tContent := runXML[tContentStart : tContentStart+tClose]
		texts = append(texts, html.UnescapeString(tContent))

		tPos = tContentStart + tClose + len("</w:t>")
	}

	return runData{
		rPr:      rPr,
		text:     strings.Join(texts, ""),
		preserve: preserve,
	}
}

// buildRunXML собирает XML для <w:r> с заданными rPr, текстом и preserve.
func buildRunXML(rPr, text string, preserve bool) string {
	escaped := html.EscapeString(text)
	var b strings.Builder
	b.WriteString("<w:r>")
	if rPr != "" {
		b.WriteString("<w:rPr>")
		b.WriteString(rPr)
		b.WriteString("</w:rPr>")
	}
	b.WriteString("<w:t")
	if preserve {
		b.WriteString(" xml:space=\"preserve\"")
	}
	b.WriteString(">")
	b.WriteString(escaped)
	b.WriteString("</w:t>")
	b.WriteString("</w:r>")
	return b.String()
}

// findRun находит индекс run'а, содержащего заданную позицию в общем тексте.
func findRun(pos int, runStarts []int) int {
	for i := len(runStarts) - 1; i >= 0; i-- {
		if runStarts[i] <= pos {
			return i
		}
	}
	return 0
}

// structuralElements — список XML-элементов, которые считаются разрывами
// последовательности текстовых кусков. Если такой элемент находится между
// частями плейсхолдера {{...}}, слияние НЕ выполняется.
var structuralElements = []string{
	"<w:tab",
	"<w:br",
	"<w:noBreakHyphen",
	"<w:cr",
}

// containsStructuralElement проверяет, содержит ли строка
// один из структурных элементов (<w:tab/>, <w:br/>, <w:noBreakHyphen/>, <w:cr/>).
func containsStructuralElement(s string) bool {
	for _, elem := range structuralElements {
		if strings.Contains(s, elem) {
			return true
		}
	}
	return false
}
