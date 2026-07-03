package docx

import (
	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
	"github.com/JJJJJJack/go-template-docx/internal/xml"
)

// DefaultPreProcessors returns the default XML pre-processors.
// PatchXML is applied via wildcard "*" to all .xml/.rels parts so template
// expressions like {{.Name}} are reassembled from DOCX-fragmented <w:r>
// runs before other processors see them. This runs exactly once per file —
// downstream pre-processors (e.g. DocxplateCompat) must NOT re-apply
// PatchXML to avoid non-idempotent entity expansion (e.g. &amp;lt; → &lt; → <).
func DefaultPreProcessors() []xml.HandlersMap {
	return []xml.HandlersMap{
		{
			"*": {func(s string) (string, error) {
				return xmlutil.PatchXML(s), nil
			}},
		},
	}
}
