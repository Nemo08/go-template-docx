package docx

import (
	"reflect"
	"text/template"
	"text/template/parse"
)

// wrapMissingKeys converts data to map[string]any and replaces nil with "".
// It finds all template variables and adds missing keys with "" value
// to avoid "<no value>" rendering with missingkey=zero.
func wrapMissingKeys(data any, tmpl *template.Template) map[string]any {
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

// extractFieldNames walks the template AST and returns top-level field names
// (e.g. ".Name" -> "Name"). Used to fill in missing keys.
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
