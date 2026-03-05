package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"

	goziputils "github.com/JJJJJJack/go-zip-utils"
)

type documentMeta struct {
	docPrIdsBijectiveIndex uint32
	docPrIds               []uint32
	// greaterCNvPrId         uint64
	greaterRId uint64
	// greaterWP14DocId       uint64
	greaterPictureNumber uint64
	// greaterChartNumber     uint64
	greaterImageNumber uint64
	maxWidthInches     float64
	maxHeightInches    float64
	templateFuncs      template.FuncMap
	mediaMap           MediaMap
	//Options
	RemoveEmptyTableRows bool
	IgnoreMissingKey     bool
}

const DOC_PR_ID_ROOF = 2_147_483_647 // docx id attributes are 32-bit signed integers

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

func (d *documentMeta) RandUniqueDocPrId() (uint32, error) {
	if d.docPrIdsBijectiveIndex == 0 {
		d.docPrIdsBijectiveIndex = 1
	}

	nextDocPrId := uint32(0)
findNextPrId:
	for i := 0; ; i++ {
		if i >= DOC_PR_ID_ROOF {
			return 0, fmt.Errorf("this should not happen, surpassed %d attempts to create a unique id for a wp:docPr tag", DOC_PR_ID_ROOF)
		}

		nextDocPrId = bijective32(d.docPrIdsBijectiveIndex) % DOC_PR_ID_ROOF
		d.docPrIdsBijectiveIndex++

		for _, docPrId := range d.docPrIds {
			if nextDocPrId == docPrId {
				continue findNextPrId
			}
		}

		if nextDocPrId != 0 {
			break
		}
	}

	d.docPrIds = append(d.docPrIds, nextDocPrId)

	return nextDocPrId, nil
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

// wrapMissingKeys конвертирует данные в map[string]any и заменяет nil → "".
// tmpl используется чтобы найти все переменные шаблона и добавить
// отсутствующие ключи со значением "" — иначе missingkey=zero вернёт
// нулевой interface{} который рендерится как "<no value>" и ломает XML.
func wrapMissingKeys(data any, tmpl *template.Template) map[string]any {
	var m map[string]any

	switch v := data.(type) {
	case map[string]any:
		m = make(map[string]any, len(v))
		for k, val := range v {
			if val == nil {
				m[k] = ""
			} else {
				m[k] = val
			}
		}
	case map[string]string:
		m = make(map[string]any, len(v))
		for k, val := range v {
			m[k] = val
		}
	default:
		return nil
	}

	// Добавляем все переменные шаблона со значением "" если их нет в данных.
	// Это единственный способ избежать "<no value>" при missingkey=zero.
	if tmpl != nil {
		for _, t := range tmpl.Templates() {
			if t.Tree == nil || t.Tree.Root == nil {
				continue
			}
			for varName := range extractFieldNames(t.Tree.Root) {
				if _, exists := m[varName]; !exists {
					m[varName] = ""
				}
			}
		}
	}

	return m
}

// extractFieldNames обходит дерево шаблона и возвращает имена полей верхнего
// уровня (например ".Name" → "Name"). Нужно чтобы заполнить отсутствующие ключи.
func extractFieldNames(node parse.Node) map[string]struct{} {
	result := map[string]struct{}{}
	extractFieldNamesRec(node, result)
	return result
}

func extractFieldNamesRec(node parse.Node, out map[string]struct{}) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *parse.ListNode:
		for _, elem := range n.Nodes {
			extractFieldNamesRec(elem, out)
		}
	case *parse.ActionNode:
		extractFieldNamesRec(n.Pipe, out)
	case *parse.IfNode:
		extractFieldNamesRec(n.Pipe, out)
		extractFieldNamesRec(n.List, out)
		extractFieldNamesRec(n.ElseList, out)
	case *parse.RangeNode:
		extractFieldNamesRec(n.Pipe, out)
		extractFieldNamesRec(n.List, out)
		extractFieldNamesRec(n.ElseList, out)
	case *parse.WithNode:
		extractFieldNamesRec(n.Pipe, out)
		extractFieldNamesRec(n.List, out)
		extractFieldNamesRec(n.ElseList, out)
	case *parse.PipeNode:
		for _, cmd := range n.Cmds {
			extractFieldNamesRec(cmd, out)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			extractFieldNamesRec(arg, out)
		}
	case *parse.FieldNode:
		// .Name или .Name.Sub — берём только первый уровень
		if len(n.Ident) > 0 {
			out[n.Ident[0]] = struct{}{}
		}
	}
}

// TODO: use xml parsing instead of regex
func ParseDocumentMeta(zm goziputils.ZipMap, tf template.FuncMap) (*documentMeta, error) {
	d := documentMeta{
		templateFuncs: tf,
	}

	// work on word/document.xml

	documentFile := zm["word/document.xml"]
	if documentFile == nil {
		return nil, fmt.Errorf("word/document.xml not found in docx")
	}

	documentContent, err := goziputils.ReadZipFileContent(documentFile)
	if err != nil {
		return nil, fmt.Errorf("error reading zip file content: %w", err)
	}

	d.maxWidthInches, d.maxHeightInches, err = parseDocumentSettings(documentContent)
	if err != nil {
		return nil, fmt.Errorf("could not parse document settings: %w", err)
	}

	idAndPictureNRegEx := regexp.MustCompile(`<wp:docPr\s+id="(\d+)"\s+name="Picture\s+(\d+)"\s*/>`)

	docPrAttrsMatches := idAndPictureNRegEx.FindAllStringSubmatch(string(documentContent), -1)
	d.docPrIds = make([]uint32, 0, len(docPrAttrsMatches))
	for _, m := range docPrAttrsMatches {
		docPrId, err := strconv.ParseUint(m[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("could not parse DocPr ID '%s': %w", m[1], err)
		}

		d.docPrIds = append(d.docPrIds, uint32(docPrId))

		pictureNumber, err := strconv.ParseUint(m[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse Picture Number '%s': %w", m[2], err)
		}

		if pictureNumber > d.greaterPictureNumber {
			d.greaterPictureNumber = pictureNumber
		}
	}

	// work on word/_rels/document.xml.rels

	wordDocumentRelsFile := zm["word/_rels/document.xml.rels"]
	if wordDocumentRelsFile == nil {
		return nil, fmt.Errorf("word/_rels/document.xml.rels not found in zip")
	}

	wordDocumentRelsContent, err := goziputils.ReadZipFileContent(wordDocumentRelsFile)
	if err != nil {
		return nil, fmt.Errorf("could not read zip file content: %w", err)
	}

	rIdNRegEx := regexp.MustCompile(`"rId(\d+)"`)

	rIdMatches := rIdNRegEx.FindAllStringSubmatch(string(wordDocumentRelsContent), -1)
	for _, match := range rIdMatches {
		num, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse rId '%s': %w", match[1], err)
		}

		if num > d.greaterRId {
			d.greaterRId = num
		}
	}

	// work on word/media/images
	for filename := range zm {
		if !strings.HasPrefix(filename, "word/media/image") {
			continue
		}

		imageNumberStr := strings.TrimPrefix(filename, "word/media/image")
		imageNumberStr = strings.TrimSuffix(imageNumberStr, path.Ext(filename))

		imageNumber, err := strconv.ParseUint(imageNumberStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse image number from filename '%s': %w", filename, err)
		}

		if imageNumber > d.greaterImageNumber {
			d.greaterImageNumber = imageNumber
		}
	}

	return &d, nil
}

func (d *documentMeta) ApplyTemplate(f *zip.File, zipWriter *zip.Writer, data any) ([]MediaRel, error) {
	documentXml, err := goziputils.ReadZipFileContent(f)
	if err != nil {
		return nil, fmt.Errorf("unable to read document file '%s': %w", f.Name, err)
	}

	documentXml = []byte(PatchXml(string(documentXml)))

	tplOption := "missingkey=error"
	if d.IgnoreMissingKey {
		tplOption = "missingkey=zero"
	}

	tmpl, err := template.New(f.Name).
		Option(tplOption).
		Funcs(d.templateFuncs).
		Parse(string(documentXml))

	if err != nil {
		return nil, fmt.Errorf("unable to parse template in file '%s': %w", f.Name, err)
	}

	// Заполняем отсутствующие ключи после парсинга — теперь знаем все переменные шаблона.
	if d.IgnoreMissingKey {
		if wrapped := wrapMissingKeys(data, tmpl); wrapped != nil {
			data = wrapped
		}
	}

	appliedTemplate := bytes.Buffer{}
	err = tmpl.Execute(&appliedTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("unable to execute template in file '%s': %w", f.Name, err)
	}

	output, media, err := d.applyImages(appliedTemplate.String())
	if err != nil {
		return nil, fmt.Errorf("unable to apply images in file '%s': %w", f.Name, err)
	}

	output, replaceMedia := d.replaceImages(output)

	media = append(media, replaceMedia...)

	output = d.applyShapesBgFillColor(output)

	output = d.replaceTableCellBgColors(output)

	output = flattenNestedTextRuns(output)

	output = ensureXmlSpacePreserve(output)

	if d.RemoveEmptyTableRows {
		output = removeEmptyTableRows(output)
	}

	err = goziputils.RewriteFileIntoZipWriter(zipWriter, f, []byte(output))
	if err != nil {
		return nil, fmt.Errorf("unable to rewrite file '%s' in zip: %w", f.Name, err)
	}

	return media, nil
}
