package docx

import "github.com/JJJJJJack/go-template-docx/internal/xmlutil"

// PatchXML removes automatically inserted content between template expressions.
func PatchXML(srcXML string) string {
	return xmlutil.PatchXML(srcXML)
}
