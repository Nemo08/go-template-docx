package docx

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

// PatchXml removes automatically insert content between template expressions
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
		// Unescape common XML/HTML entities that Word may inject inside attributes
		// (e.g., {{shapeBgFillColor (index .Map &quot;Color2&quot;)}})
		templateText = strings.NewReplacer(
			"&quot;", "\"",
			"&#34;", "\"",
			"&apos;", "'",
			"&#39;", "'",
			"&lt;", "<",
			"&#60;", "<",
			"&gt;", ">",
			"&#62;", ">",
			// &amp; MUST be last to avoid double-unescaping
			"&amp;", "&",
			"&#38;", "&",
		).Replace(templateText)

		srcXml = strings.ReplaceAll(srcXml, match, templateText)
	}

	// Word may strip quotes inside certain attribute values (e.g., alt/descr of shapes).
	// That leads to invalid Go template syntax like: {{shapeBgFillColor 00FF00}}.
	// To make templating robust, wrap bare hex arguments in quotes for known funcs.
	srcXml = reBareHexArg1.ReplaceAllString(srcXml, `{{shapeBgFillColor "$1"}}`)
	srcXml = reBareHexArg2.ReplaceAllString(srcXml, `{{tableCellBgColor "$1"}}`)

	return srcXml
}
