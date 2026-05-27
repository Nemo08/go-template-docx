package docx

import (
	"strings"
	"testing"
	"text/template"
)

func TestExtractChartFilename_Valid(t *testing.T) {
	got, err := ExtractChartFilename("word/charts/chart1.xml")
	if err != nil {
		t.Fatal(err)
	}
	if got != "chart1" {
		t.Errorf("expected chart1, got %s", got)
	}
}

func TestExtractChartFilename_NoMatch(t *testing.T) {
	_, err := ExtractChartFilename("word/document.xml")
	if err == nil {
		t.Error("expected error for non-chart path")
	}
}

func TestExtractChartFilename_Numbered(t *testing.T) {
	got, err := ExtractChartFilename("word/charts/chart42.xml")
	if err != nil {
		t.Fatal(err)
	}
	if got != "chart42" {
		t.Errorf("expected chart42, got %s", got)
	}
}

func TestUpdateChart_ReplacesValues(t *testing.T) {
	input := []byte(`<c:chart><c:f>Sheet1!$A$2:$A$3</c:f><c:strCache><c:pt idx="0"><c:v>old1</c:v></c:pt><c:pt idx="1"><c:v>old2</c:v></c:pt></c:strCache></c:chart>`)
	cellValues := map[string]string{"A2": "new1", "A3": "new2"}
	got, err := UpdateChart(input, cellValues)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "new1") {
		t.Errorf("expected new1 in output, got %s", string(got))
	}
	if !strings.Contains(string(got), "new2") {
		t.Errorf("expected new2 in output, got %s", string(got))
	}
	if strings.Contains(string(got), "old1") {
		t.Errorf("unexpected old1 in output")
	}
}

func TestUpdateChart_NoMatch(t *testing.T) {
	input := []byte(`<c:chart><c:f>Sheet1!$A$2:$A$3</c:f><c:strCache><c:pt idx="0"><c:v>keep</c:v></c:pt></c:strCache></c:chart>`)
	cellValues := map[string]string{"B1": "new"}
	got, err := UpdateChart(input, cellValues)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "keep") {
		t.Errorf("expected original value preserved, got %s", string(got))
	}
}

func TestApplyTemplateToXML_Basic(t *testing.T) {
	content := `<?xml version="1.0"?><c:chart><c:v>{{.Value}}</c:v></c:chart>`

	funcs := template.FuncMap{}
	got, err := ApplyTemplateToXML("chart1.xml", []byte(content), map[string]any{"Value": "42"}, funcs, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "42") {
		t.Errorf("expected 42 in output, got %s", string(got))
	}
}

func TestApplyTemplateToXML_IgnoreMissingKey(t *testing.T) {
	content := `<?xml version="1.0"?><c:chart><c:v>{{.Missing}}</c:v></c:chart>`

	funcs := template.FuncMap{}
	// With ignoreMissingKey=false, this should error
	_, err := ApplyTemplateToXML("chart2.xml", []byte(content), map[string]any{}, funcs, false)
	if err == nil {
		t.Error("expected error for missing key when ignoreMissingKey=false")
	}

	// With ignoreMissingKey=true, this should succeed
	got, err := ApplyTemplateToXML("chart2.xml", []byte(content), map[string]any{}, funcs, true)
	if err != nil {
		t.Fatalf("unexpected error with ignoreMissingKey=true: %v", err)
	}
	if !strings.Contains(string(got), "<c:v>") {
		t.Errorf("expected valid XML output, got %s", string(got))
	}
}
