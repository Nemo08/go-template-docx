package rels

import (
	"encoding/xml"
)

var XMLStdHeader = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)

const (
	imageRelationship = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
)

type relationshipDetail struct {
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
	ID     string `xml:"Id,attr"`
}

// Relationship holds parsed relationships from a .rels XML file.
type Relationship struct {
	XMLName       xml.Name             `xml:"http://schemas.openxmlformats.org/package/2006/relationships Relationships"`
	Relationships []relationshipDetail `xml:"Relationship"`
}

// AddMediaToRels appends media relationship entries to the relationship list.
func (r *Relationship) AddMediaToRels(media []MediaRel) {
	for _, m := range media {
		if m.Type == ImageMediaType {
			r.addRelationship(
				imageRelationship,
				m.Source,
				m.RefID,
			)
		}
	}
}

func (r *Relationship) addRelationship(relType, target, id string) {
	newRel := relationshipDetail{
		Type:   relType,
		Target: target,
		ID:     id,
	}

	r.Relationships = append(r.Relationships, newRel)
}

// WithXMLHeader prepends the standard XML header to marshalled XML bytes.
func WithXMLHeader(xmlBytes []byte) []byte {
	b := make([]byte, 0, len(XMLStdHeader)+len(xmlBytes))
	b = append(b, XMLStdHeader...)
	b = append(b, xmlBytes...)
	return b
}

// ToXML serializes the relationship list back to XML bytes.
func (r *Relationship) ToXML() ([]byte, error) {
	output, err := xml.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}

	return WithXMLHeader(output), nil
}

// ParseRelationship parses a .rels XML byte slice into a Relationship struct.
func ParseRelationship(data []byte) (*Relationship, error) {
	var relationships Relationship
	err := xml.Unmarshal(data, &relationships)
	if err != nil {
		return nil, err
	}

	return &relationships, nil
}
