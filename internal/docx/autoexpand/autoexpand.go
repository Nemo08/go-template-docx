package autoexpand

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/JJJJJJack/go-template-docx/internal/util"
	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
	"github.com/JJJJJJack/go-template-docx/internal/xml"
)

var (
	reExpandRow = regexp.MustCompile(
		`\{\{[^}]*\.(\w+)\.0(?:\.(\w+))?[^}]*\}\}` +
			`|\{\{[^}]*index\s+\.(\w+)\s+0[^}]*\}\}`,
	)

	reAnyIndex = regexp.MustCompile(`(\.)(\d+)(\.)`)

	// reDocxplateVar matches docxplate-like references: {{ArrayName.Field}} or {{.ArrayName.Field}}.
	// Logically related to docx.reNoDotDocxplate which normalises the same syntax
	// to {{.Word.Word}} across all XML parts. If you change one, check the other.
	reDocxplateVar = regexp.MustCompile(`\{\{\s*\.?(\w+)\.(\w+)\s*\}\}`)

	// reIndexPattern matches "index .X 0" function call to replace 0.
	reIndexPattern = regexp.MustCompile(`(\bindex\s+\.\w+\s+)\d+`)

	// reNormalizeDocxplateRow is a reusable regex builder for normalizeDocxplateRow.
	// Use getNormalizeDocxplateRE to obtain a compiled regex for a given arrayName.
	reNormalizeDocxplateCache sync.Map
)

func getNormalizeDocxplateRE(arrayName string) *regexp.Regexp {
	v, ok := reNormalizeDocxplateCache.Load(arrayName)
	if ok {
		return v.(*regexp.Regexp)
	}
	quoted := regexp.QuoteMeta(arrayName)
	re := regexp.MustCompile(`\{\{\s*\.?` + quoted + `\.(\w+)\s*\}\}`)
	reNormalizeDocxplateCache.Store(arrayName, re)
	return re
}

// AutoExpandRowsPreProcessor returns an xml.HandlersMap that rewrites
// expandable table rows in word/document.xml before the Go template engine
// runs. It needs the template data to determine actual slice lengths.
func AutoExpandRowsPreProcessor(data any) xml.HandlersMap {
	dataMap := util.ToStringMap(data)

	handler := func(content string) (string, error) {
		return expandRows(content, dataMap)
	}

	return xml.HandlersMap{
		"word/document.xml": {handler},
	}
}

func expandRows(content string, dataMap map[string]any) (string, error) {
	const (
		rowOpen  = "<w:tr "
		rowOpen2 = "<w:tr>"
		rowClose = "</w:tr>"
	)

	var sb strings.Builder
	sb.Grow(len(content) * 2)

	pos := 0
	for pos < len(content) {
		start := xmlutil.FindNext(content, pos, rowOpen, rowOpen2)
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

		sb.WriteString(content[pos:start])

		expanded, err := tryExpandRow(rowXML, dataMap)
		if err != nil {
			return "", err
		}
		sb.WriteString(expanded)

		pos = end
	}

	return sb.String(), nil
}

func tryExpandRow(rowXML string, dataMap map[string]any) (string, error) {
	m := reExpandRow.FindStringSubmatch(rowXML)
	if m == nil {
		return tryExpandDocxplateRow(rowXML, dataMap)
	}

	arrayName := m[1]
	if arrayName == "" {
		arrayName = m[3]
	}
	if arrayName == "" {
		return rowXML, nil
	}

	return expandClones(rowXML, arrayName, dataMap)
}

// tryExpandDocxplateRow checks for docxplate-like references ({{ArrayName.Field}})
// and expands the row if the named key is a known slice in dataMap.
func tryExpandDocxplateRow(rowXML string, dataMap map[string]any) (string, error) {
	m := reDocxplateVar.FindStringSubmatch(rowXML)
	if m == nil {
		return rowXML, nil
	}

	arrayName := m[1]
	if _, ok := sliceLen(dataMap, arrayName); !ok {
		return rowXML, nil
	}

	normalized := normalizeDocxplateRow(rowXML, arrayName)
	if normalized == rowXML {
		return rowXML, nil
	}

	return expandClones(normalized, arrayName, dataMap)
}

// expandClones clones the rowXML for each element of the named slice in dataMap,
// rewriting template block indices (.0. → .1. etc.) for each clone.
func expandClones(rowXML string, arrayName string, dataMap map[string]any) (string, error) {
	length, ok := sliceLen(dataMap, arrayName)
	if !ok || length == 0 {
		return rowXML, nil
	}

	var sb strings.Builder
	sb.Grow(len(rowXML) * length)
	sb.WriteString(rowXML)

	for i := 1; i < length; i++ {
		sb.WriteString(rewriteIndex(rowXML, i))
	}

	return sb.String(), nil
}

// normalizeDocxplateRow rewrites {{ArrayName.Field}} to {{(index .ArrayName 0).Field}}
// within all template blocks of rowXML, which is valid Go template syntax.
func normalizeDocxplateRow(rowXML string, arrayName string) string {
	re := getNormalizeDocxplateRE(arrayName)
	return re.ReplaceAllStringFunc(rowXML, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		return fmt.Sprintf("{{(index .%s 0).%s}}", arrayName, sub[1])
	})
}

func rewriteIndex(rowXML string, i int) string {
	result := rewriteTemplateBlocks(rowXML, func(block string) string {
		out := reAnyIndex.ReplaceAllStringFunc(block, func(_ string) string {
			return fmt.Sprintf(".%d.", i)
		})
		out = reIndexPattern.ReplaceAllString(out, fmt.Sprintf("${1}%d", i))
		return out
	})
	return result
}

func rewriteTemplateBlocks(s string, fn func(string) string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	pos := 0
	for pos < len(s) {
		open := strings.Index(s[pos:], "{{")
		if open == -1 {
			sb.WriteString(s[pos:])
			break
		}
		open += pos
		end := strings.Index(s[open:], "}}")
		if end == -1 {
			sb.WriteString(s[pos:])
			break
		}
		end += open + 2

		sb.WriteString(s[pos:open])
		sb.WriteString(fn(s[open:end]))
		pos = end
	}
	return sb.String()
}

func sliceLen(dataMap map[string]any, key string) (int, bool) {
	v, ok := dataMap[key]
	if !ok {
		return 0, false
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		return rv.Len(), true
	}
	return 0, false
}


