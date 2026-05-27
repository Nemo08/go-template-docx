package xmlutil

import "testing"

func TestFindNext_Found(t *testing.T) {
	s := "abc<w:tr>xyz"
	idx := FindNext(s, 0, "<w:tr>", "<w:tr ")
	if idx != 3 {
		t.Errorf("expected 3, got %d", idx)
	}
}

func TestFindNext_NotFound(t *testing.T) {
	s := "abc"
	idx := FindNext(s, 0, "<w:tr>")
	if idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}

func TestFindNext_PicksEarliest(t *testing.T) {
	s := "<w:tr>...<w:tr "
	idx := FindNext(s, 0, "<w:tr>", "<w:tr ")
	if idx != 0 {
		t.Errorf("expected 0, got %d", idx)
	}
}
