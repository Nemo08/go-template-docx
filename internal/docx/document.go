package docx

import (
	"encoding/xml"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/JJJJJJack/go-template-docx/internal/zio"
)

// DocumentProcessor defines the interface for applying templates to DOCX parts.
type DocumentProcessor interface {
	ApplyTemplate(name string, content []byte, data any) (output []byte, media []MediaRel, err error)
	NextImageNumber() uint64
	SetMediaMap(mm MediaMap)
	SetRemoveEmptyTableRows(v bool)
	SetRemoveRangeRows(v bool)
	SetIgnoreMissingKey(v bool)
	SetDeleteMissingKey(v bool)
}

// documentConfig holds immutable configuration for document processing.
type documentConfig struct {
	maxWidthInches    float64
	maxHeightInches   float64
	templateFuncs     template.FuncMap
	mediaMap          MediaMap
	RemoveEmptyTableRows bool
	RemoveRangeRows      bool
	IgnoreMissingKey     bool
	DeleteMissingKey     bool
}

// documentState holds mutable state during document processing.
type documentState struct {
	docPrIDsBijectiveIndex uint32
	docPrIDs               []uint32
	greaterRId             uint64
	greaterPictureNumber   uint64
	greaterImageNumber     uint64
}

type documentMeta struct {
	documentConfig
	documentState
}

// DocPrIDRoof is the maximum allowed docPr ID value.
const DocPrIDRoof = 2_147_483_647 // docx id attributes are 32-bit signed integers

// rotl32 rotates a 32-bit integer left by k bits.
func rotl32(x uint32, k uint) uint32 {
	return (x << k) | (x >> (32 - k))
}

// bijective32 is a fast bijective permutation on 32-bit integers.
func bijective32(x uint32) uint32 {
	x *= 0x9E3779B1
	x = rotl32(x, 16)
	x ^= 0x85EBCA6B
	return x
}

func (d *documentMeta) RandUniqueDocPrID() (uint32, error) {
	if d.docPrIDsBijectiveIndex == 0 {
		d.docPrIDsBijectiveIndex = 1
	}

	nextDocPrID := uint32(0)
findNextPrId:
	for i := 0; ; i++ {
		if i >= DocPrIDRoof {
			return 0, fmt.Errorf("this should not happen, surpassed %d attempts to create a unique id for a wp:docPr tag", DocPrIDRoof)
		}

		nextDocPrID = bijective32(d.docPrIDsBijectiveIndex) % DocPrIDRoof
		d.docPrIDsBijectiveIndex++

		for _, docPrID := range d.docPrIDs {
			if nextDocPrID == docPrID {
				continue findNextPrId
			}
		}

		if nextDocPrID != 0 {
			break
		}
	}

	d.docPrIDs = append(d.docPrIDs, nextDocPrID)

	return nextDocPrID, nil
}

func (d *documentMeta) NextPictureNumber() uint64 {
	d.greaterPictureNumber++
	return d.greaterPictureNumber
}

func (d *documentMeta) NextImageNumber() uint64 {
	d.greaterImageNumber++
	return d.greaterImageNumber
}

func (d *documentMeta) NextRId() uint64 {
	d.greaterRId++
	return d.greaterRId
}

func (d *documentMeta) SetMediaMap(mm MediaMap) {
	d.mediaMap = mm
}

func (d *documentMeta) SetRemoveEmptyTableRows(v bool) { d.RemoveEmptyTableRows = v }

func (d *documentMeta) SetRemoveRangeRows(v bool) { d.RemoveRangeRows = v }

func (d *documentMeta) SetIgnoreMissingKey(v bool) { d.IgnoreMissingKey = v }

func (d *documentMeta) SetDeleteMissingKey(v bool) { d.DeleteMissingKey = v }

type sectPr struct {
	PgSz struct {
		W int `xml:"w,attr"`
		H int `xml:"h,attr"`
	} `xml:"pgSz"`
	PgMar struct {
		Top    int `xml:"top,attr"`
		Bottom int `xml:"bottom,attr"`
		Left   int `xml:"left,attr"`
		Right  int `xml:"right,attr"`
	} `xml:"pgMar"`
}

type document struct {
	SectPr sectPr `xml:"body>sectPr"`
}

const (
	twipsPerInch = 1440.0
)

func parseDocumentSettings(docXML []byte) (usableWidthInches, usableHeightInches float64, err error) {
	var doc document
	if err = xml.Unmarshal(docXML, &doc); err != nil {
		return 0, 0, fmt.Errorf("failed to parse document.xml: %w", err)
	}

	usableWidthInches = float64(doc.SectPr.PgSz.W-doc.SectPr.PgMar.Left-doc.SectPr.PgMar.Right) / twipsPerInch
	usableHeightInches = float64(doc.SectPr.PgSz.H-doc.SectPr.PgMar.Top-doc.SectPr.PgMar.Bottom) / twipsPerInch

	return usableWidthInches, usableHeightInches, nil
}

var (
	reDocPr       = regexp.MustCompile(`<wp:docPr\s+id="(\d+)"\s+name="Picture\s+(\d+)"\s*/>`)
	reRId         = regexp.MustCompile(`"rId(\d+)"`)
	reImagePrefix = regexp.MustCompile(`^` + ImagePrefix)
)

// ParseDocumentMeta extracts document metadata and builds a DocumentProcessor from a DOCX archive.
func ParseDocumentMeta(source zio.FileSource, tf template.FuncMap) (DocumentProcessor, error) {
	d := documentMeta{
		documentConfig: documentConfig{
			templateFuncs: tf,
		},
	}

	// work on word/document.xml

	documentContent, found, err := source.ReadFile(DocumentXMLPath)
	if err != nil {
		return nil, fmt.Errorf("error reading zip file content: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("%s not found in docx", DocumentXMLPath)
	}

	d.maxWidthInches, d.maxHeightInches, err = parseDocumentSettings(documentContent)
	if err != nil {
		return nil, fmt.Errorf("could not parse document settings: %w", err)
	}

	docPrAttrsMatches := reDocPr.FindAllStringSubmatch(string(documentContent), -1)
	d.docPrIDs = make([]uint32, 0, len(docPrAttrsMatches))
	for _, m := range docPrAttrsMatches {
		docPrID, err := strconv.ParseUint(m[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("could not parse DocPr ID '%s': %w", m[1], err)
		}

		d.docPrIDs = append(d.docPrIDs, uint32(docPrID))

		pictureNumber, err := strconv.ParseUint(m[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse Picture Number '%s': %w", m[2], err)
		}

		if pictureNumber > d.greaterPictureNumber {
			d.greaterPictureNumber = pictureNumber
		}
	}

	// work on word/_rels/document.xml.rels

	wordDocumentRelsContent, found, err := source.ReadFile(DocumentRelsPath)
	if err != nil {
		return nil, fmt.Errorf("could not read zip file content: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("%s not found in zip", DocumentRelsPath)
	}

	rIDMatches := reRId.FindAllStringSubmatch(string(wordDocumentRelsContent), -1)
	for _, match := range rIDMatches {
		num, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse rId '%s': %w", match[1], err)
		}

		if num > d.greaterRId {
			d.greaterRId = num
		}
	}

	// work on word/media/images
	err = source.Each(func(filename string) error {
		if !reImagePrefix.MatchString(filename) {
			return nil
		}

		imageNumberStr := strings.TrimPrefix(filename, ImagePrefix)
		imageNumberStr = strings.TrimSuffix(imageNumberStr, path.Ext(filename))

		imageNumber, err := strconv.ParseUint(imageNumberStr, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse image number from filename '%s': %w", filename, err)
		}

		if imageNumber > d.greaterImageNumber {
			d.greaterImageNumber = imageNumber
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &d, nil
}
