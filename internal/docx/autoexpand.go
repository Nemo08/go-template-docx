package docx

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
	"github.com/JJJJJJack/go-template-docx/xml"
)

var reExpandRow = regexp.MustCompile(
	`\{\{[^}]*\.(\w+)\.0(?:\.(\w+))?[^}]*\}\}` +
		`|\{\{[^}]*index\s+\.(\w+)\s+0[^}]*\}\}`,
)

var reAnyIndex = regexp.MustCompile(`(\.)(\d+)(\.)`)

// AutoExpandRowsPreProcessor returns an xml.HandlersMap that rewrites
// expandable table rows in word/document.xml before the Go template engine
// runs. It needs the template data to determine actual slice lengths.
func AutoExpandRowsPreProcessor(data any) xml.HandlersMap {
	dataMap := reflectToMap(data)

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
		return rowXML, nil
	}

	arrayName := m[1]
	if arrayName == "" {
		arrayName = m[3]
	}
	if arrayName == "" {
		return rowXML, nil
	}

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

func rewriteIndex(rowXML string, i int) string {
	result := rewriteTemplateBlocks(rowXML, func(block string) string {
		out := reAnyIndex.ReplaceAllStringFunc(block, func(_ string) string {
			return fmt.Sprintf(".%d.", i)
		})
		out = regexp.MustCompile(`(\bindex\s+\.\w+\s+)\d+`).
			ReplaceAllString(out, fmt.Sprintf("${1}%d", i))
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

func reflectToMap(data any) map[string]any {
	if data == nil {
		return nil
	}
	switch v := data.(type) {
	case map[string]any:
		return v
	case []byte:
		var m map[string]any
		if err := json.Unmarshal(v, &m); err != nil {
			return nil
		}
		return m
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil
		}
		return m
	}
}
