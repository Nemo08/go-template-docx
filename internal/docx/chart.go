package docx

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

var (
	reChartBlock = regexp.MustCompile(`(?s)<c:f>(Sheet\d+!\$([A-Z]+)\$(\d+)(?::\$[A-Z]+\$\d+)?)</c:f>.*?<c:(?:strCache|numCache)>(.*?)</c:(?:strCache|numCache)>`)
	reChartPt    = regexp.MustCompile(`(?s)<c:pt idx="(\d+)">.*?<c:v>(.*?)</c:v>.*?</c:pt>`)
	reChartName  = regexp.MustCompile(`(chart\d+)\.xml`)
)

// TODO: parse and unmarshal xml instead of using regex
func UpdateChart(fileContent []byte, cellAndValues map[string]string) ([]byte, error) {

	updated := reChartBlock.ReplaceAllFunc(fileContent, func(block []byte) []byte {
		m := reChartBlock.FindSubmatch(block)
		if len(m) < 5 {
			return block
		}

		col := string(m[2])                       // "A"
		startRow, _ := strconv.Atoi(string(m[3])) // 2
		cache := string(m[4])                     // contents of <c:strCache> or <c:numCache>

		// Iterate over <c:pt>
		cacheUpdated := reChartPt.ReplaceAllStringFunc(cache, func(pt string) string {
			pm := reChartPt.FindStringSubmatch(pt)
			if len(pm) < 3 {
				return pt
			}

			idx, _ := strconv.Atoi(pm[1]) // idx=0,1,2,3...
			cell := fmt.Sprintf("%s%d", col, startRow+idx)

			if number, ok := cellAndValues[cell]; ok {
				oldVal := fmt.Sprintf("<c:v>%s</c:v>", pm[2])
				newVal := fmt.Sprintf("<c:v>%s</c:v>", number)
				return strings.Replace(pt, oldVal, newVal, 1)
			}
			return pt
		})

		// Put updated cache back
		return []byte(strings.Replace(string(block), cache, cacheUpdated, 1))
	})

	return updated, nil
}

func ApplyTemplateToXml(name string, fileContent []byte, templateValues any, templateFuncs template.FuncMap, ignoreMissingKey bool) ([]byte, error) {
	missingKeyOpt := "missingkey=error"
	if ignoreMissingKey {
		missingKeyOpt = "missingkey=zero"
	}

	tmpl, err := template.New(name).
		Option(missingKeyOpt).
		Funcs(templateFuncs).
		Parse(PatchXml(string(fileContent)))
	if err != nil {
		return nil, fmt.Errorf("unable to parse template: %w", err)
	}

	if ignoreMissingKey {
		if wrapped := wrapMissingKeys(templateValues, tmpl); wrapped != nil {
			templateValues = wrapped
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateValues); err != nil {
		return nil, fmt.Errorf("unable to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// ExtractChartFilename now only works with a single submatch
func ExtractChartFilename(path string) (string, error) {
	matches := reChartName.FindStringSubmatch(path)
	if len(matches) < 2 {
		return "", fmt.Errorf("no chart name found")
	}
	return matches[1], nil
}
