package docx

import (
	"html"
	"text/template"

	docxtemplate "github.com/JJJJJJack/go-template-docx/internal/template"
)

// WrapMissingKeys converts data to map[string]any and replaces nil with "".
// It finds all template variables and adds missing keys with "" value
// to avoid "<no value>" rendering with missingkey=zero.
func WrapMissingKeys(data any, tmpl *template.Template) map[string]any {
	var m map[string]any

	switch v := data.(type) {
	case map[string]any:
		m = make(map[string]any, len(v))
		for k, val := range v {
			if val == nil {
				m[k] = ""
			} else {
				m[k] = val
			}
		}
	case map[string]string:
		m = make(map[string]any, len(v))
		for k, val := range v {
			m[k] = val
		}
	default:
		return nil
	}

	if tmpl != nil {
		for _, t := range tmpl.Templates() {
			if t.Tree == nil || t.Root == nil {
				continue
			}
			for varName := range docxtemplate.ExtractFieldNames(t.Root) {
				if _, exists := m[varName]; !exists {
					m[varName] = ""
				}
			}
		}
	}

	return m
}

// EscapeTemplateValues recursively escapes XML special characters (& < > " ')
// in all string values within maps and slices to prevent XML parsing errors.
func EscapeTemplateValues(data any) any {
	switch v := data.(type) {
	case string:
		return html.EscapeString(v)
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, value := range v {
			result[key] = EscapeTemplateValues(value)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, value := range v {
			result[i] = EscapeTemplateValues(value)
		}
		return result
	default:
		return v
	}
}


