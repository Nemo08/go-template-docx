package docx

import (
	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
	"github.com/JJJJJJack/go-template-docx/internal/xml"
)

// DefaultPreProcessors returns the default XML pre-processors that are
// applied before user-defined pre-processors (e.g., AutoExpandRows).
// PatchXML must run first so template expressions like {{.Name}} are
// reassembled from DOCX-fragmented <w:r> runs before other processors
// try to parse them.
func DefaultPreProcessors() []xml.HandlersMap {
	return []xml.HandlersMap{
		{
			DocumentXMLPath: {func(s string) (string, error) {
				return xmlutil.PatchXML(s), nil
			}},
		},
	}
}
