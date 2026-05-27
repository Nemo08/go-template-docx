// Package xlsx handles template application to embedded XLSX chart-data
// files inside DOCX archives.
package xlsx

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
)

// ApplyTemplateToCells applies the templateValues to the given file content and returns the modified content.
func ApplyTemplateToCells(name string, fileContent []byte, templateValues any, ignoreMissingKey bool) ([]byte, error) {
	opt := "missingkey=error"
	if ignoreMissingKey {
		opt = "missingkey=zero"
	}

	tmpl, err := template.New(name).
		Option(opt).
		Funcs(template.FuncMap{
			"toNumberCell": ToNumberCell,
		}).
		Parse(xmlutil.PatchXML(string(fileContent)))
	if err != nil {
		return nil, fmt.Errorf("unable to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateValues); err != nil {
		return nil, fmt.Errorf("unable to execute template: %w", err)
	}

	return buf.Bytes(), nil
}
