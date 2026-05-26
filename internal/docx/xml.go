package docx

import "github.com/JJJJJJack/go-template-docx/internal/xmlutil"

// PatchXml removes automatically inserted content between template expressions.
func PatchXml(srcXml string) string {
	return xmlutil.PatchXml(srcXml)
}
