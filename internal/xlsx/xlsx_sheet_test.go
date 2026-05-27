package xlsx

import (
	"testing"
)

func TestGetCountFromXML(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row><c t="s"><v>0</v></c><c t="n"><v>1</v></c></row></sheetData></worksheet>`)
	count, err := GetCountFromXML(data)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 shared string cell, got %d", count)
	}
}

func TestGetCountFromXML_NoSharedStrings(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row><c t="n"><v>1</v></c></row></sheetData></worksheet>`)
	count, err := GetCountFromXML(data)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestGetCountFromXML_EmptySheet(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData></sheetData></worksheet>`)
	count, err := GetCountFromXML(data)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestUpdateSheet_NumberCells(t *testing.T) {
	input := []byte(`<worksheet><sheetData><row><c r="A1" t="s"><v>0</v></c></row></sheetData></worksheet>`)
	numberVals := map[int]string{0: "42"}
	newIndexes := map[int]int{}
	output, chartVals, err := UpdateSheet(input, numberVals, newIndexes)
	if err != nil {
		t.Fatal(err)
	}
	outputStr := string(output)
	if !contains(outputStr, ">42<") {
		t.Errorf("expected number 42 in cell, got %s", outputStr)
	}
	if !contains(outputStr, `r="A1"`) {
		t.Errorf("expected cell A1 preserved, got %s", outputStr)
	}
	if chartVals["A1"] != "42" {
		t.Errorf("expected chart value A1=42, got %v", chartVals)
	}
}

func TestUpdateSheet_IndexReindex(t *testing.T) {
	input := []byte(`<worksheet><sheetData><row><c r="A1" t="s"><v>0</v></c></row></sheetData></worksheet>`)
	numberVals := map[int]string{}
	newIndexes := map[int]int{0: 5}
	output, _, err := UpdateSheet(input, numberVals, newIndexes)
	if err != nil {
		t.Fatal(err)
	}
	outputStr := string(output)
	if !contains(outputStr, ">5<") {
		t.Errorf("expected new index 5, got %s", outputStr)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
