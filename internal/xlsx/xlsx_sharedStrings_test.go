package xlsx

import (
	"strings"
	"testing"
)

func TestGetUniqueCountFromXml(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="3" uniqueCount="3"><si><t>a</t></si><si><t>b</t></si></sst>`)
	count, err := GetUniqueCountFromXml(data)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 unique strings, got %d", count)
	}
}

func TestGetUniqueCountFromXml_Empty(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="0" uniqueCount="0"></sst>`)
	count, err := GetUniqueCountFromXml(data)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestUpdateSharedStringsCounts(t *testing.T) {
	input := []byte(`<?xml version="1.0"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="5" uniqueCount="5"><si><t>a</t></si></sst>`)
	got, err := UpdateSharedStringsCounts(input, 3)
	if err != nil {
		t.Fatal(err)
	}
	output := string(got)
	if !strings.Contains(output, `count="3"`) {
		t.Errorf("expected count=3, got %s", output)
	}
	if !strings.Contains(output, `uniqueCount="1"`) {
		t.Errorf("expected uniqueCount=1, got %s", output)
	}
}

func TestGetReferencedSharedStringsByIndexAndCleanup_NoNumberCells(t *testing.T) {
	input := []byte(`<sst><si><t>hello</t></si><si><t>world</t></si></sst>`)
	output, numCells, _, err := GetReferencedSharedStringsByIndexAndCleanup(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(numCells) != 0 {
		t.Errorf("expected no number cells, got %v", numCells)
	}
	if !strings.Contains(string(output), "hello") {
		t.Errorf("expected hello preserved, got %s", string(output))
	}
}

func TestGetReferencedSharedStringsByIndexAndCleanup_WithNumberCells(t *testing.T) {
	input := []byte(`<sst><si><t>[[NUMBER:42]]</t></si><si><t>keep</t></si></sst>`)
	output, numCells, _, err := GetReferencedSharedStringsByIndexAndCleanup(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(numCells) != 1 {
		t.Errorf("expected 1 number cell, got %d", len(numCells))
	}
	if numCells[0] != "42" {
		t.Errorf("expected number value 42 at index 0, got %v", numCells)
	}
	if !strings.Contains(string(output), "keep") {
		t.Errorf("expected keep preserved, got %s", string(output))
	}
	// The NUMBER entry should be removed
	if strings.Contains(string(output), "[[NUMBER:42]]") {
		t.Errorf("expected NUMBER placeholder removed, got %s", string(output))
	}
}
