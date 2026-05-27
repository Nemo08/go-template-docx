package docx

import (
	"testing"
)

func TestAdjustBrightnessHex_Lighten(t *testing.T) {
	tests := []struct {
		hex    string
		factor float64
		want   string
	}{
		{"000000", 0.5, "808080"},
		{"ffffff", 0.0, "ffffff"},
		{"ff0000", 0.5, "ff8080"},
	}
	for _, tt := range tests {
		got := adjustBrightnessHex(tt.hex, tt.factor, true)
		if got != tt.want {
			t.Errorf("adjustBrightnessHex(%q, %v, true) = %q, want %q", tt.hex, tt.factor, got, tt.want)
		}
	}
}

func TestAdjustBrightnessHex_Darken(t *testing.T) {
	tests := []struct {
		hex    string
		factor float64
		want   string
	}{
		{"808080", 0.5, "404040"},
		{"ffffff", 0.0, "ffffff"},
		{"000000", 0.5, "000000"},
	}
	for _, tt := range tests {
		got := adjustBrightnessHex(tt.hex, tt.factor, false)
		if got != tt.want {
			t.Errorf("adjustBrightnessHex(%q, %v, false) = %q, want %q", tt.hex, tt.factor, got, tt.want)
		}
	}
}

func TestAdjustBrightnessHex_InvalidLength(t *testing.T) {
	got := adjustBrightnessHex("ff", 0.5, true)
	if got != "ff" {
		t.Errorf("expected unchanged 'ff', got %q", got)
	}
}

func TestApplyShapesBgFillColor_NoPlaceholder(t *testing.T) {
	input := `<mc:AlternateContent><wps:spPr><a:solidFill><a:srgbClr val="000000"/></a:solidFill></wps:spPr></mc:AlternateContent>`
	d := &documentMeta{}
	got := d.applyShapesBgFillColor(input)
	if got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestApplyShapesBgFillColor_WithPlaceholder(t *testing.T) {
	input := `<mc:AlternateContent><wps:spPr><a:solidFill><a:srgbClr val="000000"/></a:solidFill></wps:spPr>[[SHAPE_BG_FILL_COLOR:FF0000]]</mc:AlternateContent>`
	d := &documentMeta{}
	got := d.applyShapesBgFillColor(input)
	if !contains(got, `val="ff0000"`) {
		t.Errorf("expected ff0000 fill, got %q", got)
	}
	if contains(got, "[[SHAPE_BG_FILL_COLOR:FF0000]]") {
		t.Errorf("placeholder should be removed, got %q", got)
	}
}

func TestApplyShapesBgFillColor_WithHash(t *testing.T) {
	input := `<mc:AlternateContent><wps:spPr><a:solidFill><a:srgbClr val="000000"/></a:solidFill></wps:spPr>[[SHAPE_BG_FILL_COLOR:#FF0000]]</mc:AlternateContent>`
	d := &documentMeta{}
	got := d.applyShapesBgFillColor(input)
	if !contains(got, `val="ff0000"`) {
		t.Errorf("expected ff0000 fill, got %q", got)
	}
}

func TestReplaceTableCellBgColors_NoPlaceholder(t *testing.T) {
	input := `<w:tc><w:tcPr></w:tcPr><w:p><w:r><w:t>text</w:t></w:r></w:p></w:tc>`
	d := &documentMeta{}
	got := d.replaceTableCellBgColors(input)
	if got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestReplaceTableCellBgColors_WithPlaceholder(t *testing.T) {
	input := `<w:tc><w:tcPr></w:tcPr><w:p><w:r><w:t>[[TABLE_CELL_BG_COLOR:FF0000]]</w:t></w:r></w:p></w:tc>`
	d := &documentMeta{}
	got := d.replaceTableCellBgColors(input)
	if !contains(got, `w:fill="FF0000"`) {
		t.Errorf("expected FF0000 fill, got %q", got)
	}
	if contains(got, "[[TABLE_CELL_BG_COLOR:FF0000]]") {
		t.Errorf("placeholder should be removed, got %q", got)
	}
}

func TestReplaceTableCellBgColors_WithExistingShd(t *testing.T) {
	input := `<w:tc><w:tcPr><w:shd w:val="clear" w:color="auto" w:fill="FFFFFF"/></w:tcPr><w:p><w:r><w:t>[[TABLE_CELL_BG_COLOR:00FF00]]</w:t></w:r></w:p></w:tc>`
	d := &documentMeta{}
	got := d.replaceTableCellBgColors(input)
	if !contains(got, `w:fill="00FF00"`) {
		t.Errorf("expected 00FF00 fill, got %q", got)
	}
}

func TestReplaceTableCellBgColors_WithHash(t *testing.T) {
	input := `<w:tc><w:tcPr></w:tcPr><w:p><w:r><w:t>[[TABLE_CELL_BG_COLOR:#0000FF]]</w:t></w:r></w:p></w:tc>`
	d := &documentMeta{}
	got := d.replaceTableCellBgColors(input)
	if !contains(got, `w:fill="0000FF"`) {
		t.Errorf("expected 0000FF fill, got %q", got)
	}
}
