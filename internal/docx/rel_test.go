package docx

import (
	"strings"
	"testing"
)

func TestParseRelationship(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/image1.png"/>
</Relationships>`)
	rel, err := ParseRelationship(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rel.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(rel.Relationships))
	}
	if rel.Relationships[0].ID != "rId1" {
		t.Errorf("expected rId1, got %q", rel.Relationships[0].ID)
	}
	if rel.Relationships[0].Target != "media/image1.png" {
		t.Errorf("expected media/image1.png, got %q", rel.Relationships[0].Target)
	}
}

func TestParseRelationship_InvalidXML(t *testing.T) {
	_, err := ParseRelationship([]byte(`invalid`))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestRelationship_AddMediaToRels(t *testing.T) {
	rel := &Relationship{}
	media := []MediaRel{
		{Type: ImageMediaType, Source: "media/img1.png", RefID: "rId1"},
		{Type: 0, Source: "media/doc.pdf", RefID: "rId2"},
	}
	rel.AddMediaToRels(media)
	if len(rel.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(rel.Relationships))
	}
	if rel.Relationships[0].Target != "media/img1.png" {
		t.Errorf("expected media/img1.png, got %q", rel.Relationships[0].Target)
	}
}

func TestRelationship_ToXML(t *testing.T) {
	rel := &Relationship{}
	rel.AddMediaToRels([]MediaRel{
		{Type: ImageMediaType, Source: "media/logo.png", RefID: "rId99"},
	})
	xml, err := rel.ToXML()
	if err != nil {
		t.Fatal(err)
	}
	output := string(xml)
	if !strings.Contains(output, `rId99`) {
		t.Errorf("expected rId99 in XML, got %s", output)
	}
	if !strings.Contains(output, `media/logo.png`) {
		t.Errorf("expected media/logo.png in XML, got %s", output)
	}
	if !strings.Contains(output, `<?xml version="1.0"`) {
		t.Errorf("expected XML header, got %s", output)
	}
}

func TestRelationship_ToXML_Empty(t *testing.T) {
	rel := &Relationship{}
	xml, err := rel.ToXML()
	if err != nil {
		t.Fatal(err)
	}
	output := string(xml)
	if !strings.Contains(output, "<Relationships") {
		t.Errorf("expected Relationships tag, got %s", output)
	}
}
