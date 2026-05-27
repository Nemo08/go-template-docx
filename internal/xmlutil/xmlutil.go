// Package xmlutil provides XML utility functions for patching template
// expressions within DOCX XML files.
package xmlutil

import (
	"regexp"
	"strings"
)

var (
	reOpenBrace    = regexp.MustCompile(`\{([^\}]*?)\{`)
	reCloseBrace   = regexp.MustCompile(`\}([^\{]*?)\}`)
	reTemplateExpr = regexp.MustCompile(`\{\{[\s\S]*?\}\}`)
	reXMLTag       = regexp.MustCompile(`(<\s*\/?[\w-:.]+(\s+[^>]*?)?[\s\/]*>)`)
	reBareHexArg1  = regexp.MustCompile(`(?i)\{\{\s*shapeBgFillColor\s+(#?[0-9A-Fa-f]{6})\s*\}\}`)
	reBareHexArg2  = regexp.MustCompile(`(?i)\{\{\s*tableCellBgColor\s+(#?[0-9A-Fa-f]{6})\s*\}\}`)
)

// FindNext returns the position of the first occurrence of any of the given
// substrings in s starting from pos, or -1 if none found.
func FindNext(s string, pos int, subs ...string) int {
	best := -1
	for _, sub := range subs {
		idx := strings.Index(s[pos:], sub)
		if idx == -1 {
			continue
		}
		idx += pos
		if best == -1 || idx < best {
			best = idx
		}
	}
	return best
}

// PatchXML removes automatically inserted content between template expressions
// (EG: "{{ .Text }}" could have correctors highlights tags separating the expressions tokens).
func PatchXML(srcXML string) string {
	// Fix separated {{
	srcXML = reOpenBrace.ReplaceAllString(srcXML, "{{")

	// Fix separated }}
	srcXML = reCloseBrace.ReplaceAllString(srcXML, "}}")

	// Remove unnecessary XML tags inside template expressions and unescape XML entities
	matches := reTemplateExpr.FindAllString(srcXML, -1)
	for _, match := range matches {
		templateText := reXMLTag.ReplaceAllString(match, "")
		templateText = strings.NewReplacer(
			"&quot;", "\"",
			"&#34;", "\"",
			"&apos;", "'",
			"&#39;", "'",
			"&lt;", "<",
			"&#60;", "<",
			"&gt;", ">",
			"&#62;", ">",
			"&amp;", "&",
			"&#38;", "&",
		).Replace(templateText)

		srcXML = strings.ReplaceAll(srcXML, match, templateText)
	}

	srcXML = reBareHexArg1.ReplaceAllString(srcXML, `{{shapeBgFillColor "$1"}}`)
	srcXML = reBareHexArg2.ReplaceAllString(srcXML, `{{tableCellBgColor "$1"}}`)

	return srcXML
}
