package xmlutil

import (
	"regexp"
	"strings"
)

var (
	reOpenBrace    = regexp.MustCompile(`\{([^\}]*?)\{`)
	reCloseBrace   = regexp.MustCompile(`\}([^\{]*?)\}`)
	reTemplateExpr = regexp.MustCompile(`\{\{[\s\S]*?\}\}`)
	reXmlTag       = regexp.MustCompile(`(<\s*\/?[\w-:.]+(\s+[^>]*?)?[\s\/]*>)`)
	reBareHexArg1  = regexp.MustCompile(`(?i)\{\{\s*shapeBgFillColor\s+(#?[0-9A-Fa-f]{6})\s*\}\}`)
	reBareHexArg2  = regexp.MustCompile(`(?i)\{\{\s*tableCellBgColor\s+(#?[0-9A-Fa-f]{6})\s*\}\}`)
)

// PatchXml removes automatically inserted content between template expressions
// (EG: "{{ .Text }}" could have correctors highlights tags separating the expressions tokens).
func PatchXml(srcXml string) string {
	// Fix separated {{
	srcXml = reOpenBrace.ReplaceAllString(srcXml, "{{")

	// Fix separated }}
	srcXml = reCloseBrace.ReplaceAllString(srcXml, "}}")

	// Remove unnecessary XML tags inside template expressions and unescape XML entities
	matches := reTemplateExpr.FindAllString(srcXml, -1)
	for _, match := range matches {
		templateText := reXmlTag.ReplaceAllString(match, "")
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

		srcXml = strings.ReplaceAll(srcXml, match, templateText)
	}

	srcXml = reBareHexArg1.ReplaceAllString(srcXml, `{{shapeBgFillColor "$1"}}`)
	srcXml = reBareHexArg2.ReplaceAllString(srcXml, `{{tableCellBgColor "$1"}}`)

	return srcXml
}
