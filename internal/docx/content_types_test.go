package docx

import (
	"testing"
)

func TestParseContentTypes(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="png" ContentType="image/png"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`)
	ct, err := ParseContentTypes(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(ct.Defaults) != 1 {
		t.Errorf("expected 1 Default, got %d", len(ct.Defaults))
	}
	if ct.Defaults[0].Extension != "png" {
		t.Errorf("expected extension png, got %s", ct.Defaults[0].Extension)
	}
}

func TestAddDefaultUnique_New(t *testing.T) {
	ct := &contentTypes{}
	ct.AddDefaultUnique("jpg", "image/jpeg")
	if len(ct.Defaults) != 1 {
		t.Errorf("expected 1 default after add, got %d", len(ct.Defaults))
	}
}

func TestAddDefaultUnique_Duplicate(t *testing.T) {
	ct := &contentTypes{}
	ct.AddDefaultUnique("png", "image/png")
	ct.AddDefaultUnique("png", "image/png")
	if len(ct.Defaults) != 1 {
		t.Errorf("expected 1 default after duplicate add, got %d", len(ct.Defaults))
	}
}

func TestAddDefaultUnique_DifferentType(t *testing.T) {
	ct := &contentTypes{}
	ct.AddDefaultUnique("png", "image/png")
	ct.AddDefaultUnique("jpg", "image/jpeg")
	if len(ct.Defaults) != 2 {
		t.Errorf("expected 2 defaults, got %d", len(ct.Defaults))
	}
}

func TestToXml(t *testing.T) {
	ct := &contentTypes{
		Defaults: []tagDefault{
			{Extension: "png", ContentType: "image/png"},
		},
	}
	xmlBytes, err := ct.ToXml()
	if err != nil {
		t.Fatal(err)
	}
	output := string(xmlBytes)
	if !contains(output, `Extension="png"`) {
		t.Errorf("expected Extension=\"png\" in output, got %s", output)
	}
	if !contains(output, `?>`) {
		t.Errorf("expected XML header in output")
	}
}

func TestReplaceEmptyTags(t *testing.T) {
	input := []byte(`></Default>`)
	got := replaceEmptyTags(input)
	if string(got) != ` />` {
		t.Errorf("expected ' />', got %q", string(got))
	}
}
