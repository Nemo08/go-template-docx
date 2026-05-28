// Package docx implements DOCX template processing: parsing, media
// handling, relationship management, and template application.
package docx

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
)

func (d *documentMeta) prepareContent(content []byte) []byte {
	content = []byte(xmlutil.PatchXML(string(content)))
	if d.RemoveRangeRows {
		content = []byte(markRangeDirectiveRows(string(content)))
	}
	return content
}

func (d *documentMeta) parseTemplate(name string, content string) (*template.Template, error) {
	tplOption := "missingkey=error"
	if d.DeleteMissingKey {
		tplOption = "missingkey=zero"
	}

	return template.New(name).
		Option(tplOption).
		Funcs(d.templateFuncs).
		Parse(content)
}

func (d *documentMeta) wrapData(data any, tmpl *template.Template) any {
	if !d.DeleteMissingKey {
		return data
	}
	if wrapped := wrapMissingKeys(data, tmpl); wrapped != nil {
		return wrapped
	}
	return data
}

func (d *documentMeta) executeTemplate(tmpl *template.Template, name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("unable to execute template in file '%s': %w", name, err)
	}
	return buf.String(), nil
}

func (d *documentMeta) postProcessContent(output string) string {
	output = d.applyShapesBgFillColor(output)
	output = d.replaceTableCellBgColors(output)
	output = flattenNestedTextRuns(output)
	output = propagateRunPropsAfterBreak(output)
	output = ensureXMLSpacePreserve(output)
	if d.RemoveEmptyTableRows {
		output = removeEmptyTableRows(output)
	}
	if d.RemoveRangeRows {
		output = removeMarkedEmptyRows(output)
	}
	return output
}

func (d *documentMeta) ApplyTemplate(name string, content []byte, data any) ([]byte, []MediaRel, error) {
	content = d.prepareContent(content)

	tmpl, err := d.parseTemplate(name, string(content))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse template in file '%s': %w", name, err)
	}

	data = d.wrapData(data, tmpl)

	output, err := d.executeTemplate(tmpl, name, data)
	if err != nil {
		if d.IgnoreMissingKey {
			// Return the prepared (patched) content as-is so template
			// placeholders like {{.MissingKey}} remain for a later pass.
			return content, nil, nil
		}
		return nil, nil, err
	}

	output, media, err := d.applyImages(output)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to apply images in file '%s': %w", name, err)
	}

	output, replaceMedia := d.replaceImages(output)
	media = append(media, replaceMedia...)

	output = d.postProcessContent(output)

	return []byte(output), media, nil
}
