// Package template provides utilities for extracting variable names from
// parsed Go templates.
package template

import (
	"reflect"
	"text/template"
	"text/template/parse"
)

// collectVariables walks the parse tree and collects variables
func collectVariables(node parse.Node, vars map[string]struct{}) {
	switch n := node.(type) {
	case *parse.ListNode:
		if n != nil {
			for _, elem := range n.Nodes {
				collectVariables(elem, vars)
			}
		}
	case *parse.ActionNode:
		collectVariables(n.Pipe, vars)
	case *parse.RangeNode:
		collectVariables(n.Pipe, vars)
		collectVariables(n.List, vars)
		collectVariables(n.ElseList, vars)
	case *parse.IfNode:
		collectVariables(n.Pipe, vars)
		collectVariables(n.List, vars)
		collectVariables(n.ElseList, vars)
	case *parse.WithNode:
		collectVariables(n.Pipe, vars)
		collectVariables(n.List, vars)
		collectVariables(n.ElseList, vars)
	case *parse.TemplateNode:
		// reference to another template, but variables come from that template's Tree
	case *parse.PipeNode:
		for _, cmd := range n.Cmds {
			collectVariables(cmd, vars)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			collectVariables(arg, vars)
		}
	case *parse.FieldNode:
		// Example: {{.User.Name}}
		vars[n.String()] = struct{}{}
	case *parse.VariableNode:
		// Example: {{$var}} or {{$var.Field}}
		vars[n.String()] = struct{}{}
	}
}

// ExtractFieldNames извлекает имена полей первого уровня (например, .Name из .Name.First)
// из переданного узла AST. Идентично extractFieldNamesRec в docx/wrap_missing_keys.go.
func ExtractFieldNames(node parse.Node) map[string]struct{} {
	result := map[string]struct{}{}
	collectFieldNamesRec(node, result)
	return result
}

func collectFieldNamesRec(node parse.Node, out map[string]struct{}) {
	if node == nil {
		return
	}
	if v := reflect.ValueOf(node); v.Kind() == reflect.Pointer && v.IsNil() {
		return
	}
	switch n := node.(type) {
	case *parse.ListNode:
		for _, elem := range n.Nodes {
			collectFieldNamesRec(elem, out)
		}
	case *parse.ActionNode:
		collectFieldNamesRec(n.Pipe, out)
	case *parse.IfNode:
		collectFieldNamesRec(n.Pipe, out)
		collectFieldNamesRec(n.List, out)
		collectFieldNamesRec(n.ElseList, out)
	case *parse.RangeNode:
		collectFieldNamesRec(n.Pipe, out)
		collectFieldNamesRec(n.List, out)
		collectFieldNamesRec(n.ElseList, out)
	case *parse.WithNode:
		collectFieldNamesRec(n.Pipe, out)
		collectFieldNamesRec(n.List, out)
		collectFieldNamesRec(n.ElseList, out)
	case *parse.PipeNode:
		for _, cmd := range n.Cmds {
			collectFieldNamesRec(cmd, out)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			collectFieldNamesRec(arg, out)
		}
	case *parse.FieldNode:
		if len(n.Ident) > 0 {
			out[n.Ident[0]] = struct{}{}
		}
	}
}

// ExtractAllVariables extract variables from ALL templates in a set
func ExtractAllVariables(t *template.Template) map[string]struct{} {
	vars := make(map[string]struct{})
	for _, tpl := range t.Templates() {
		if tpl.Tree != nil && tpl.Root != nil {
			collectVariables(tpl.Root, vars)
		}
	}

	return vars
}
