package docx

import (
	"html"
	"reflect"
	"text/template"
	"text/template/parse"
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
			for varName := range extractFieldNames(t.Root) {
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

func extractFieldNames(node parse.Node) map[string]struct{} {
	result := map[string]struct{}{}
	extractFieldNamesRec(node, result)
	return result
}

func extractFieldNamesRec(node parse.Node, out map[string]struct{}) {
	if node == nil {
		return
	}
	if v := reflect.ValueOf(node); v.Kind() == reflect.Pointer && v.IsNil() {
		return
	}
	switch n := node.(type) {
	case *parse.ListNode:
		for _, elem := range n.Nodes {
			extractFieldNamesRec(elem, out)
		}
	case *parse.ActionNode:
		extractFieldNamesRec(n.Pipe, out)
	case *parse.IfNode:
		extractFieldNamesRec(n.Pipe, out)
		extractFieldNamesRec(n.List, out)
		extractFieldNamesRec(n.ElseList, out)
	case *parse.RangeNode:
		extractFieldNamesRec(n.Pipe, out)
		extractFieldNamesRec(n.List, out)
		extractFieldNamesRec(n.ElseList, out)
	case *parse.WithNode:
		extractFieldNamesRec(n.Pipe, out)
		extractFieldNamesRec(n.List, out)
		extractFieldNamesRec(n.ElseList, out)
	case *parse.PipeNode:
		for _, cmd := range n.Cmds {
			extractFieldNamesRec(cmd, out)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			extractFieldNamesRec(arg, out)
		}
	case *parse.FieldNode:
		if len(n.Ident) > 0 {
			out[n.Ident[0]] = struct{}{}
		}
	}
}
