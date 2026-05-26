package xlsx

import (
	"testing"
)

func TestToNumberCell_Int(t *testing.T) {
	got, err := ToNumberCell(42)
	if err != nil {
		t.Fatal(err)
	}
	expected := "[[NUMBER:42]]"
	if got != expected {
		t.Errorf("ToNumberCell(42) = %q, want %q", got, expected)
	}
}

func TestToNumberCell_Float64(t *testing.T) {
	got, err := ToNumberCell(3.14)
	if err != nil {
		t.Fatal(err)
	}
	expected := "[[NUMBER:3.14]]"
	if got != expected {
		t.Errorf("ToNumberCell(3.14) = %q, want %q", got, expected)
	}
}

func TestToNumberCell_Int64(t *testing.T) {
	got, err := ToNumberCell(int64(100))
	if err != nil {
		t.Fatal(err)
	}
	expected := "[[NUMBER:100]]"
	if got != expected {
		t.Errorf("ToNumberCell(int64(100)) = %q, want %q", got, expected)
	}
}

func TestToNumberCell_String(t *testing.T) {
	_, err := ToNumberCell("not a number")
	if err == nil {
		t.Error("expected error for string input")
	}
}

func TestToNumberCell_Bool(t *testing.T) {
	_, err := ToNumberCell(true)
	if err == nil {
		t.Error("expected error for bool input")
	}
}
