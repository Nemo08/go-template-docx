package template

import (
	"testing"
	"text/template"
)

func TestExtractAllVariables_FieldNodes(t *testing.T) {
	tmpl, err := template.New("test").Parse(`{{.Name}} {{.Age}}`)
	if err != nil {
		t.Fatal(err)
	}
	vars := ExtractAllVariables(tmpl)
	if _, ok := vars[".Name"]; !ok {
		t.Errorf("expected .Name in vars, got %v", vars)
	}
	if _, ok := vars[".Age"]; !ok {
		t.Errorf("expected .Age in vars, got %v", vars)
	}
}

func TestExtractAllVariables_NestedField(t *testing.T) {
	tmpl, err := template.New("test").Parse(`{{.User.Name}}`)
	if err != nil {
		t.Fatal(err)
	}
	vars := ExtractAllVariables(tmpl)
	if _, ok := vars[".User.Name"]; !ok {
		t.Errorf("expected .User.Name in vars, got %v", vars)
	}
}

func TestExtractAllVariables_VariableNode(t *testing.T) {
	tmpl, err := template.New("test").Parse(`{{range $i, $v := .Items}}{{$i}}{{$v}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	vars := ExtractAllVariables(tmpl)
	if _, ok := vars["$i"]; !ok {
		t.Errorf("expected $i in vars, got %v", vars)
	}
	if _, ok := vars["$v"]; !ok {
		t.Errorf("expected $v in vars, got %v", vars)
	}
}

func TestExtractAllVariables_Range(t *testing.T) {
	tmpl, err := template.New("test").Parse(`{{range .Items}}{{.Name}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	vars := ExtractAllVariables(tmpl)
	if _, ok := vars[".Items"]; !ok {
		t.Errorf("expected .Items in vars, got %v", vars)
	}
	if _, ok := vars[".Name"]; !ok {
		t.Errorf("expected .Name in vars, got %v", vars)
	}
}

func TestExtractAllVariables_IfElse(t *testing.T) {
	tmpl, err := template.New("test").Parse(`{{if .Flag}}yes{{else}}no{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	vars := ExtractAllVariables(tmpl)
	if _, ok := vars[".Flag"]; !ok {
		t.Errorf("expected .Flag in vars, got %v", vars)
	}
}

func TestExtractAllVariables_With(t *testing.T) {
	tmpl, err := template.New("test").Parse(`{{with .X}}{{.Y}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	vars := ExtractAllVariables(tmpl)
	if _, ok := vars[".X"]; !ok {
		t.Errorf("expected .X in vars, got %v", vars)
	}
	if _, ok := vars[".Y"]; !ok {
		t.Errorf("expected .Y in vars, got %v", vars)
	}
}

func TestExtractAllVariables_Empty(t *testing.T) {
	tmpl, err := template.New("test").Parse(`static text`)
	if err != nil {
		t.Fatal(err)
	}
	vars := ExtractAllVariables(tmpl)
	if len(vars) != 0 {
		t.Errorf("expected no vars, got %v", vars)
	}
}

func TestExtractAllVariables_NilTree(t *testing.T) {
	tmpl := template.New("empty")
	vars := ExtractAllVariables(tmpl)
	if len(vars) != 0 {
		t.Errorf("expected no vars for nil tree, got %v", vars)
	}
}
