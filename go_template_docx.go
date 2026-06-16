// Package gotemplatedocx renders Go templates inside DOCX files, producing
// modified DOCX output with interpolated text, images, charts, and more.
package gotemplatedocx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/JJJJJJack/go-template-docx/internal/docx"
	"github.com/JJJJJJack/go-template-docx/internal/xlsx"
	docxtemplate "github.com/JJJJJJack/go-template-docx/internal/template"
	"github.com/JJJJJJack/go-template-docx/internal/xmlutil"
	"github.com/JJJJJJack/go-template-docx/internal/zio"
	"github.com/JJJJJJack/go-template-docx/xml"
)

var (
	reChartsPath      = regexp.MustCompile(`word/charts/chart\d*?\.xml`)
	reXlsxEmbedded    = regexp.MustCompile(`/embeddings/Microsoft_Excel_Worksheet\d*?\.xlsx`)
	reHeaderFooterDoc = regexp.MustCompile(`word/(header|footer|document)\d*?\.xml`)

	defaultSkipFilter = orFilter(
		matchPath(docx.DocumentRelsPath, docx.ContentTypesPath),
		matchRe(reChartsPath, reXlsxEmbedded, reHeaderFooterDoc),
	)
)

type docxTemplate struct {
	State  TemplateState
	Config TemplateConfig
}

// TemplateConfig holds all immutable configuration options.
type TemplateConfig struct {
	RemoveEmptyTableRows bool
	RemoveRangeRows      bool
	IgnoreMissingKey     bool
	DeleteMissingKey     bool
	WarnOnMissingKey     bool
	MissingKeyLogger     *slog.Logger
	Filename             string
	TemplateFuncs        template.FuncMap
	PreProcessors        []xml.HandlersMap
	PostProcessors       []xml.HandlersMap
}

// TemplateState holds mutable state during template processing.
type TemplateState struct {
	Input      bytes.Buffer
	Output     bytes.Buffer
	Rel        *docx.Relationship
	RelMedia   []docx.MediaRel
	Media      docx.MediaMap
	XlsxCharts xlsxChartsMap
}

// TemplateProcessor contains the processing logic.
type TemplateProcessor struct {
	Config *TemplateConfig
	State  *TemplateState
}

// copyTemplateFuncs creates a copy of the template.FuncMap.
func copyTemplateFuncs(src template.FuncMap) template.FuncMap {
	dst := make(template.FuncMap, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func newDocxTemplate(inputBuffer bytes.Buffer, filename string, options ...TemplateOption) *docxTemplate {
	cfg := defaultConfig()
	cfg.Filename = filename
	cfg.TemplateFuncs = copyTemplateFuncs(docx.TemplateFuncs)
	cfg.PreProcessors = append([]xml.HandlersMap{}, docx.DefaultPreProcessors...)
	cfg.PostProcessors = append([]xml.HandlersMap{}, docx.DefaultPostProcessors...)

	for _, opt := range options {
		opt(cfg)
	}

	return &docxTemplate{
		Config: *cfg,
		State: TemplateState{
			Input:      inputBuffer,
			Output:     bytes.Buffer{},
			Media:      make(docx.MediaMap),
			Rel:        &docx.Relationship{},
			RelMedia:   []docx.MediaRel{},
			XlsxCharts: make(xlsxChartsMap),
		},
	}
}

func defaultConfig() *TemplateConfig {
	return &TemplateConfig{
		RemoveEmptyTableRows: true,
		IgnoreMissingKey:     false,
		WarnOnMissingKey:     false,
		MissingKeyLogger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
}

// TemplateProcessor applies the template to the DOCX content.
func (p *TemplateProcessor) Apply(templateValues any) error {
	templateValues = p.Config.normalize(templateValues)
	templateValues = docx.EscapeTemplateValues(templateValues)

	if len(p.Config.PreProcessors) > 0 {
		if err := xml.ProcessedOutput(p.Config.PreProcessors, &p.State.Input, "pre"); err != nil {
			return fmt.Errorf("unable to pre-process output DOCX file: %w", err)
		}
	}

	if err := p.applyTemplatePipeline(templateValues); err != nil {
		return err
	}

	if len(p.Config.PostProcessors) > 0 {
		if err := xml.ProcessedOutput(p.Config.PostProcessors, &p.State.Output, "post"); err != nil {
			return fmt.Errorf("unable to post-process output DOCX file: %w", err)
		}
	}

	return nil
}

func (c *TemplateConfig) normalize(templateValues any) any {
	switch v := templateValues.(type) {
	case []byte:
		if len(v) == 0 {
			return map[string]any{}
		}
		var m map[string]any
		if err := json.Unmarshal(v, &m); err != nil {
			return v
		}
		for k, val := range m {
			if val == nil {
				m[k] = ""
			}
		}
		return m
	case map[string]string:
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[k] = val
		}
		return m
	}
	return templateValues
}

// normalizeTemplateValues нормализует данные шаблона к map[string]any.
func (dt *docxTemplate) normalizeTemplateValues(templateValues any) any {
	return dt.Config.normalize(templateValues)
}

// applyTemplatePipeline runs the core template pipeline: parse input, apply template, write output.
func (p *TemplateProcessor) applyTemplatePipeline(templateValues any) error {
	zipWriter := zio.NewZipWriter(&p.State.Output)

	src, err := zio.NewFromBytes(p.State.Input.Bytes())
	if err != nil {
		return fmt.Errorf("unable to create DOCX zip source: %w", err)
	}

	document, err := docx.ParseDocumentMeta(src, p.Config.TemplateFuncs)
	if err != nil {
		return fmt.Errorf("unable to parse document metadata: %w", err)
	}
	document.SetRemoveEmptyTableRows(p.Config.RemoveEmptyTableRows)
	document.SetRemoveRangeRows(p.Config.RemoveRangeRows)
	document.SetIgnoreMissingKey(p.Config.IgnoreMissingKey)
	document.SetDeleteMissingKey(p.Config.DeleteMissingKey)

	if err := p.writeMediaFiles(zipWriter, document.NextImageNumber); err != nil {
		return err
	}
	document.SetMediaMap(p.State.Media)

	if err := copyNonTemplateFiles(zipWriter, src, defaultSkipFilter); err != nil {
		return err
	}
	if err := p.updateContentTypes(zipWriter, src); err != nil {
		return err
	}
	if err := p.parseDocumentRels(src); err != nil {
		return err
	}

	chartRelToTargetXlsx, err := buildChartXlsxMap(src)
	if err != nil {
		return err
	}

	if err := p.processXlsxFiles(zipWriter, src, templateValues); err != nil {
		return err
	}

	var warnDataMap map[string]any
	if p.Config.WarnOnMissingKey {
		warnDataMap = toStringMap(templateValues)
	}

	if err := p.processHeadersFootersDocument(zipWriter, src, document.ApplyTemplate, templateValues, warnDataMap); err != nil {
		return err
	}
	if err := p.processChartFiles(zipWriter, src, chartRelToTargetXlsx, templateValues, warnDataMap); err != nil {
		return err
	}
	if err := p.updateDocumentRels(zipWriter, src); err != nil {
		return err
	}

	return zipWriter.Close()
}

// writeMediaFiles writes media files into the ZIP and assigns Word-compatible filenames.
func (p *TemplateProcessor) writeMediaFiles(zipWriter zio.ZipWriter, nextImageNumber func() uint64) error {
	for filename, media := range p.State.Media {
		imageN := nextImageNumber()
		wordFilename := fmt.Sprintf("image%d%s", imageN, path.Ext(filename))
		media.WordFilename = wordFilename

		filepath := path.Join("word/media", media.WordFilename)
		fw, err := zipWriter.Create(filepath)
		if err != nil {
			return fmt.Errorf("unable to create media file '%s': %w", filepath, err)
		}
		if _, err := fw.Write(media.Data); err != nil {
			return fmt.Errorf("unable to write media file '%s': %w", filepath, err)
		}
	}
	return nil
}

// updateContentTypes adds media type entries for loaded images.
func (p *TemplateProcessor) updateContentTypes(zipWriter zio.ZipWriter, src zio.FileSource) error {
	ctData, found, err := src.ReadFile(docx.ContentTypesPath)
	if err != nil {
		return fmt.Errorf("unable to read content types file '%s': %w", docx.ContentTypesPath, err)
	}
	if !found {
		return fmt.Errorf("content types file '%s' not found", docx.ContentTypesPath)
	}
	contentTypes, err := docx.ParseContentTypes(ctData)
	if err != nil {
		return fmt.Errorf("unable to parse content types file '%s': %w", docx.ContentTypesPath, err)
	}
	for filename := range p.State.Media {
		ext := path.Ext(filename)
		switch lowerExt := strings.ToLower(ext); lowerExt {
		case ".jpg", ".jpeg", ".jfif":
			contentTypes.AddDefaultUnique(lowerExt[1:], "image/jpeg")
		case ".png":
			contentTypes.AddDefaultUnique("png", "image/png")
		}
	}
	updatedCt, err := contentTypes.ToXML()
	if err != nil {
		return fmt.Errorf("unable to marshal content types to XML: %w", err)
	}
	return zio.RewriteToZip(zipWriter, src, docx.ContentTypesPath, []byte(updatedCt))
}

// parseDocumentRels parses the document relationships file.
func (p *TemplateProcessor) parseDocumentRels(src zio.FileSource) error {
	relData, found, err := src.ReadFile(docx.DocumentRelsPath)
	if err != nil {
		return fmt.Errorf("unable to read rel file '%s': %w", docx.DocumentRelsPath, err)
	}
	if !found {
		return fmt.Errorf("rel file '%s' not found", docx.DocumentRelsPath)
	}
	p.State.Rel, err = docx.ParseRelationship(relData)
	if err != nil {
		return fmt.Errorf("unable to parse rel file '%s': %w", docx.DocumentRelsPath, err)
	}
	return nil
}

// processXlsxFiles applies templates to all embedded XLSX files.
func (p *TemplateProcessor) processXlsxFiles(zipWriter zio.ZipWriter, src zio.FileSource, templateValues any) error {
	for i := 0; ; i++ {
		xlsxFilename := fmt.Sprintf(docx.XlsxPathFormat, i)
		if i == 0 {
			xlsxFilename = docx.XlsxFirstPath
		}
		xlsxData, found, err := src.ReadFile(xlsxFilename)
		if err != nil {
			return fmt.Errorf("unable to read XLSX file '%s': %w", xlsxFilename, err)
		}
		if !found {
			break
		}
		if err := p.writeXlsxIntoZip(zipWriter, src, xlsxFilename, xlsxData, templateValues, p.Config.IgnoreMissingKey, p.Config.DeleteMissingKey); err != nil {
			return fmt.Errorf("unable to apply template to XLSX file '%s': %w", xlsxFilename, err)
		}
	}
	return nil
}

// processHeadersFootersDocument applies templates to header, footer, and main document files.
// Header and footer image rels are written to their own .rels files; document rels go to State.RelMedia.
func (p *TemplateProcessor) processHeadersFootersDocument(zipWriter zio.ZipWriter, src zio.FileSource, applyTemplate func(name string, content []byte, data any) ([]byte, []docx.MediaRel, error), templateValues any, warnDataMap map[string]any) error {
	for i := 1; ; i++ {
		headerName := fmt.Sprintf(docx.HeaderPathFormat, i)
		data, found, err := src.ReadFile(headerName)
		if err != nil {
			return fmt.Errorf("unable to read header file '%s': %w", headerName, err)
		}
		if !found {
			break
		}
		p.warnMissingKeysForFile(headerName, src, warnDataMap)
		output, media, err := applyTemplate(headerName, data, templateValues)
		if err != nil {
			return fmt.Errorf("unable to apply template to header file '%s': %w", headerName, err)
		}
		if err := p.updateFileRels(zipWriter, src, fmt.Sprintf(docx.HeaderRelsPathFormat, i), media); err != nil {
			return fmt.Errorf("unable to update header rels for '%s': %w", headerName, err)
		}
		if err := zio.RewriteToZip(zipWriter, src, headerName, output); err != nil {
			return fmt.Errorf("unable to write header file '%s': %w", headerName, err)
		}
	}
	for i := 1; ; i++ {
		footerName := fmt.Sprintf(docx.FooterPathFormat, i)
		data, found, err := src.ReadFile(footerName)
		if err != nil {
			return fmt.Errorf("unable to read footer file '%s': %w", footerName, err)
		}
		if !found {
			break
		}
		p.warnMissingKeysForFile(footerName, src, warnDataMap)
		output, media, err := applyTemplate(footerName, data, templateValues)
		if err != nil {
			return fmt.Errorf("unable to apply template to footer file '%s': %w", footerName, err)
		}
		if err := p.updateFileRels(zipWriter, src, fmt.Sprintf(docx.FooterRelsPathFormat, i), media); err != nil {
			return fmt.Errorf("unable to update footer rels for '%s': %w", footerName, err)
		}
		if err := zio.RewriteToZip(zipWriter, src, footerName, output); err != nil {
			return fmt.Errorf("unable to write footer file '%s': %w", footerName, err)
		}
	}
	documentData, found, err := src.ReadFile(docx.DocumentXMLPath)
	if err != nil {
		return fmt.Errorf("unable to read document file: %w", err)
	}
	if !found {
		return fmt.Errorf("%s not found in the DOCX file", docx.DocumentXMLPath)
	}
	p.warnMissingKeysForFile(docx.DocumentXMLPath, src, warnDataMap)
	output, media, err := applyTemplate(docx.DocumentXMLPath, documentData, templateValues)
	if err != nil {
		return fmt.Errorf("unable to apply template to document file: %w", err)
	}
	p.State.RelMedia = append(p.State.RelMedia, media...)
	if err := zio.RewriteToZip(zipWriter, src, docx.DocumentXMLPath, output); err != nil {
		return fmt.Errorf("unable to write document file: %w", err)
	}
	return nil
}

// warnMissingKeysForFile warns about missing template keys in a file.
func (p *TemplateProcessor) warnMissingKeysForFile(name string, src zio.FileSource, warnDataMap map[string]any) {
	if !p.Config.WarnOnMissingKey || warnDataMap == nil {
		return
	}
	b, found, err := src.ReadFile(name)
	if err != nil || !found {
		return
	}
	tmpl, err := template.New(path.Base(name)).Funcs(p.Config.TemplateFuncs).Parse(xmlutil.PatchXML(string(b)))
	if err != nil {
		return
	}
	p.warnMissingKeysInFile(tmpl, warnDataMap)
}

// warnMissingKeysInFile compares template variables with data and warns about missing keys.
func (p *TemplateProcessor) warnMissingKeysInFile(tmpl *template.Template, data map[string]any) {
	docxName := p.Config.Filename
	if docxName == "" {
		docxName = "<bytes>"
	}
	vars := docxtemplate.ExtractAllVariables(tmpl)
	for v := range vars {
		key := strings.TrimPrefix(v, ".")
		if idx := strings.Index(key, "."); idx != -1 {
			key = key[:idx]
		}
		if strings.HasPrefix(key, "$") || key == "" {
			continue
		}
		if _, ok := data[key]; !ok {
			p.Config.MissingKeyLogger.Warn("missing key in template", "file", docxName, "placeholder", v)
		}
	}
}

// updateFileRels reads (or creates) a .rels file at relsPath, adds media relationships,
// and writes it back to the ZIP. Used for header/footer rels files.
func (p *TemplateProcessor) updateFileRels(zipWriter zio.ZipWriter, src zio.FileSource, relsPath string, media []docx.MediaRel) error {
	if len(media) == 0 {
		return nil
	}
	relData, found, err := src.ReadFile(relsPath)
	if err != nil {
		return fmt.Errorf("unable to read rel file '%s': %w", relsPath, err)
	}
	var rel *docx.Relationship
	if found {
		rel, err = docx.ParseRelationship(relData)
		if err != nil {
			return fmt.Errorf("unable to parse rel file '%s': %w", relsPath, err)
		}
	} else {
		rel = &docx.Relationship{}
	}
	rel.AddMediaToRels(media)
	relContent, err := rel.ToXML()
	if err != nil {
		return fmt.Errorf("unable to marshal rels: %w", err)
	}
	return zio.RewriteToZip(zipWriter, src, relsPath, relContent)
}

// updateDocumentRels rewrites the document relationships file with any new media references.
func (p *TemplateProcessor) updateDocumentRels(zipWriter zio.ZipWriter, src zio.FileSource) error {
	documentRelContent, found, err := src.ReadFile(docx.DocumentRelsPath)
	if err != nil {
		return fmt.Errorf("unable to read rel file '%s': %w", docx.DocumentRelsPath, err)
	}
	if !found {
		return fmt.Errorf("rel file '%s' not found", docx.DocumentRelsPath)
	}
	if len(p.State.RelMedia) != 0 {
		p.State.Rel.AddMediaToRels(p.State.RelMedia)
		documentRelContent, err = p.State.Rel.ToXML()
		if err != nil {
			return fmt.Errorf("unable to marshal rels: %w", err)
		}
	}
	return zio.RewriteToZip(zipWriter, src, docx.DocumentRelsPath, documentRelContent)
}

// buildChartXlsxMap maps chart filenames to their target XLSX embedded files.
func buildChartXlsxMap(src zio.FileSource) (map[string]string, error) {
	chartRelToTargetXlsx := make(map[string]string)
	for i := 1; ; i++ {
		relsChartFilename := fmt.Sprintf(docx.ChartRelsPathFormat, i)
		fileContent, found, err := src.ReadFile(relsChartFilename)
		if err != nil {
			return nil, fmt.Errorf("unable to read chart rel file '%s': %w", relsChartFilename, err)
		}
		if !found {
			break
		}
		chartsRelationships, _ := docx.ParseRelationship(fileContent)
		for _, relationship := range chartsRelationships.Relationships {
			if !reXlsxEmbedded.MatchString(relationship.Target) {
				continue
			}
			targetXlsxFilename := strings.Replace(relationship.Target, "../", "word/", 1)
			chartFilename, err := docx.ExtractChartFilename(relsChartFilename)
			if err != nil {
				return nil, fmt.Errorf("unable to extract chart name from file '%s': %w", relsChartFilename, err)
			}
			chartRelToTargetXlsx[chartFilename] = targetXlsxFilename
		}
	}
	return chartRelToTargetXlsx, nil
}

// writeXlsxIntoZip writes a processed XLSX file into the ZIP.
func (p *TemplateProcessor) writeXlsxIntoZip(zipWriter zio.ZipWriter, src zio.FileSource, xlsxFilename string, xlsxData []byte, templateValues any, ignoreMissingKey, deleteMissingKey bool) error {
	xlsxBytes, err := p.modifyXlsxInMemory(xlsxFilename, xlsxData, templateValues, ignoreMissingKey, deleteMissingKey)
	if err != nil {
		return fmt.Errorf("error modifying XLSX in memory: %w", err)
	}

	if err := zio.RewriteToZip(zipWriter, src, xlsxFilename, xlsxBytes); err != nil {
		return fmt.Errorf("error writing XLSX to zip: %w", err)
	}

	return nil
}

// processChartFiles applies templates and updates chart data for all chart XML files.
func (p *TemplateProcessor) processChartFiles(zipWriter zio.ZipWriter, src zio.FileSource, chartRelToTargetXlsx map[string]string, templateValues any, warnDataMap map[string]any) error {
	for i := 1; ; i++ {
		chartN := fmt.Sprintf(docx.ChartPathFormat, i)
		chartContent, found, err := src.ReadFile(chartN)
		if err != nil {
			return fmt.Errorf("unable to read chart file '%s': %w", chartN, err)
		}
		if !found {
			break
		}
		p.warnMissingKeysForFile(chartN, src, warnDataMap)

		fileContent, err := docx.ApplyTemplateToXML(chartN, chartContent, templateValues, p.Config.TemplateFuncs, p.Config.IgnoreMissingKey, p.Config.DeleteMissingKey)
		if err != nil {
			return fmt.Errorf("unable to apply template to chart file '%s': %w", chartN, err)
		}
		chartFilename, err := docx.ExtractChartFilename(chartN)
		if err != nil {
			return fmt.Errorf("unable to extract chart name from file '%s': %w", chartN, err)
		}
		xlsxFileTarget := chartRelToTargetXlsx[chartFilename]
		fileContent, err = docx.UpdateChart(fileContent, p.State.XlsxCharts[xlsxFileTarget])
		if err != nil {
			return fmt.Errorf("unable to update preview chart file '%s': %w", chartN, err)
		}
		if err := zio.RewriteToZip(zipWriter, src, chartN, fileContent); err != nil {
			return fmt.Errorf("unable to rewrite chart file '%s': %w", chartN, err)
		}
	}
	return nil
}

// modifyXlsxInMemory modifies an XLSX byte slice in memory, applying templates.
func (p *TemplateProcessor) modifyXlsxInMemory(xlsxName string, xlsxData []byte, templateValues any, ignoreMissingKey, deleteMissingKey bool) ([]byte, error) {
	var sharedStringsNumbers map[int]string
	var sharedStringsNewIndexes map[int]int

	xlsxSrc, err := zio.NewFromBytes(xlsxData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XLSX zip: %w", err)
	}

	var buf bytes.Buffer
	zipWriter := zio.NewZipWriter(&buf)

	err = xlsxSrc.Each(func(name string) error {
		if reSheetN.MatchString(name) || reSharedStringsN.MatchString(name) {
			return nil
		}
		return zio.CopyToZip(zipWriter, xlsxSrc, name)
	})
	if err != nil {
		return nil, err
	}

	sharedStringsContent, found, err := xlsxSrc.ReadFile(docx.SharedStringsPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file '%s': %w", docx.SharedStringsPath, err)
	}
	if !found {
		return nil, fmt.Errorf("shared strings file '%s' not found in embedded XLSX", docx.SharedStringsPath)
	}

	sharedStringsContent, err = xlsx.ApplyTemplateToCells(docx.SharedStringsPath, sharedStringsContent, templateValues, ignoreMissingKey, deleteMissingKey)
	if err != nil {
		return nil, fmt.Errorf("error applying template to shared strings: %w", err)
	}

	sharedStringsContent, sharedStringsNumbers, sharedStringsNewIndexes, err = xlsx.GetReferencedSharedStringsByIndexAndCleanup(sharedStringsContent)
	if err != nil {
		return nil, fmt.Errorf("error cleaning up shared strings: %w", err)
	}

	sharedStringsCount := uint(0)
	for i := 1; ; i++ {
		sheetN := fmt.Sprintf(docx.SheetPathFormat, i)

		sheetContent, found, err := xlsxSrc.ReadFile(sheetN)
		if err != nil {
			return nil, fmt.Errorf("error reading sheet file '%s': %w", sheetN, err)
		}
		if !found {
			break
		}

		var chartValues map[string]string
		sheetContent, chartValues, err = xlsx.UpdateSheet(sheetContent, sharedStringsNumbers, sharedStringsNewIndexes)
		if err != nil {
			return nil, fmt.Errorf("error processing sheet '%s': %w", sheetN, err)
		}

		p.State.XlsxCharts[xlsxName] = chartValues

		sharedStringsRefs, err := xlsx.GetCountFromXML(sheetContent)
		if err != nil {
			return nil, fmt.Errorf("error getting shared strings refs from sheet '%s': %w", sheetN, err)
		}

		sharedStringsCount += sharedStringsRefs

		if err := zio.RewriteToZip(zipWriter, xlsxSrc, sheetN, sheetContent); err != nil {
			return nil, fmt.Errorf("error writing sheet '%s': %w", sheetN, err)
		}
	}

	sharedStringsContent, err = xlsx.UpdateSharedStringsCounts(sharedStringsContent, sharedStringsCount)
	if err != nil {
		return nil, fmt.Errorf("error recounting shared strings: %w", err)
	}

	if err := zio.RewriteToZip(zipWriter, xlsxSrc, docx.SharedStringsPath, sharedStringsContent); err != nil {
		return nil, fmt.Errorf("error writing shared strings: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("error closing zip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// NewDocxTemplateFromBytes creates a new docxTemplate object from the provided DOCX file bytes.
func NewDocxTemplateFromBytes(docxBytes []byte, options ...TemplateOption) (*docxTemplate, error) {
	inputBuffer := bytes.Buffer{}

	_, err := inputBuffer.Write(docxBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to write DOCX bytes to buffer: %w", err)
	}

	return newDocxTemplate(inputBuffer, "", options...), nil
}

// NewDocxTemplateFromFilename creates a new docxTemplate object from the provided DOCX filename.
func NewDocxTemplateFromFilename(docxFilename string, options ...TemplateOption) (*docxTemplate, error) {
	// #nosec G304 — filename comes from the caller, not from user input
	docxBytes, err := os.ReadFile(docxFilename)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s: %w", docxFilename, err)
	}

	inputBuffer := bytes.Buffer{}
	_, err = inputBuffer.Write(docxBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to write DOCX bytes to buffer: %w", err)
	}

	return newDocxTemplate(inputBuffer, docxFilename, options...), nil
}

// Media adds a media file to the docxTemplate object.
func (dt *docxTemplate) Media(filename string, data []byte) {
	filename = filepath.Base(filename)

	dt.State.Media[filename] = &docx.Media{
		Data: data,
	}
}

// AddTemplateFuncs adds custom template functions.
func (dt *docxTemplate) AddTemplateFuncs(funcMap template.FuncMap) {
	for funcName, fn := range funcMap {
		dt.Config.TemplateFuncs[funcName] = fn
	}
}

// AddPreProcessors adds XML pre-processing maps.
func (dt *docxTemplate) AddPreProcessors(filesPreProcessors ...xml.HandlersMap) {
	dt.Config.PreProcessors = append(dt.Config.PreProcessors, filesPreProcessors...)
}

// AddPostProcessors adds XML post-processing maps.
func (dt *docxTemplate) AddPostProcessors(filesPostProcessors ...xml.HandlersMap) {
	dt.Config.PostProcessors = append(dt.Config.PostProcessors, filesPostProcessors...)
}

// toStringMap converts templateValues to map[string]any for key checking.
func toStringMap(templateValues any) map[string]any {
	switch v := templateValues.(type) {
	case map[string]any:
		return v
	case map[string]string:
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[k] = val
		}
		return m
	case []byte:
		if len(v) == 0 {
			return map[string]any{}
		}
		var m map[string]any
		if err := json.Unmarshal(v, &m); err != nil {
			return nil
		}
		return m
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil
		}
		return m
	}
}

// GetTemplateVariables extracts all template variables from the DOCX file.
func (dt *docxTemplate) GetTemplateVariables() (map[string]struct{}, error) {
	src, err := zio.NewFromBytes(dt.State.Input.Bytes())
	if err != nil {
		return nil, fmt.Errorf("unable to create DOCX zip source: %w", err)
	}

	vars := map[string]struct{}{}
	err = src.Each(func(name string) error {
		if strings.HasPrefix(name, "word/media/") {
			return nil
		}

		b, found, err := src.ReadFile(name)
		if err != nil {
			return fmt.Errorf("unable to read file '%s': %w", name, err)
		}
		if !found {
			return nil
		}

		tmpl, err := template.New(path.Base(name)).Funcs(dt.Config.TemplateFuncs).Parse(xmlutil.PatchXML(string(b)))
		if err != nil {
			return fmt.Errorf("unable to parse template in file '%s': %w", name, err)
		}

		x := docxtemplate.ExtractAllVariables(tmpl)
		for k := range x {
			vars[k] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return vars, nil
}

// fileFilter is a predicate that returns true when a file should be skipped.
type fileFilter func(name string) bool

// orFilter combines multiple fileFilter predicates with OR logic.
func orFilter(filters ...fileFilter) fileFilter {
	return func(name string) bool {
		for _, f := range filters {
			if f(name) {
				return true
			}
		}
		return false
	}
}

func matchPath(patterns ...string) fileFilter {
	return func(name string) bool {
		for _, p := range patterns {
			if name == p {
				return true
			}
		}
		return false
	}
}

func matchRe(re ...*regexp.Regexp) fileFilter {
	return func(name string) bool {
		for _, r := range re {
			if r.MatchString(name) {
				return true
			}
		}
		return false
	}
}

// copyNonTemplateFiles copies all unmodified files, skipping those matching the filter.
func copyNonTemplateFiles(zipWriter zio.ZipWriter, src zio.FileSource, skip fileFilter) error {
	return src.Each(func(filename string) error {
		if skip(filename) {
			return nil
		}
		return zio.CopyToZip(zipWriter, src, filename)
	})
}

// Apply applies the template with the provided values to the DOCX file.
func (dt *docxTemplate) Apply(templateValues any) error {
	processor := &TemplateProcessor{
		Config: &dt.Config,
		State:  &dt.State,
	}

	return processor.Apply(templateValues)
}

// Save saves the modified docx file to the specified filename.
func (dt *docxTemplate) Save(filename string) error {
	return os.WriteFile(filename, dt.State.Output.Bytes(), 0600)
}

// Bytes returns the output bytes of the output xlsx file bytes.
func (dt *docxTemplate) Bytes() []byte {
	return dt.State.Output.Bytes()
}

// TemplateOption configures a TemplateConfig during construction.
type TemplateOption func(*TemplateConfig)

// WithFilename sets the source filename for the template (used in logs).
func WithFilename(name string) TemplateOption {
	return func(c *TemplateConfig) {
		c.Filename = name
	}
}

// NoRemoveEmptyTableRows disables the default removal of empty table rows.
func NoRemoveEmptyTableRows() TemplateOption {
	return func(c *TemplateConfig) {
		c.RemoveEmptyTableRows = false
	}
}

// RemoveRangeRows removes range directive rows after template rendering.
func RemoveRangeRows() TemplateOption {
	return func(c *TemplateConfig) {
		c.RemoveRangeRows = true
	}
}

// IgnoreMissingKey suppresses template execution errors when a referenced key
// is not present in the data. The template expression is left untouched in the
// output so it can be resolved in a later pass (e.g. with DeleteMissingKey or
// with the actual data).
func IgnoreMissingKey() TemplateOption {
	return func(c *TemplateConfig) {
		c.IgnoreMissingKey = true
	}
}

// DeleteMissingKey replaces missing template keys with empty strings instead
// of erroring. Unlike IgnoreMissingKey, the resulting output is clean — no
// template placeholders remain. This is the option to use when the data for
// the current pass is final.
func DeleteMissingKey() TemplateOption {
	return func(c *TemplateConfig) {
		c.DeleteMissingKey = true
	}
}

// WarnOnMissingKey enables warnings for missing keys in stderr.
// Also enables DeleteMissingKey so missing keys are replaced with empty
// strings (not left as template placeholders for reprocessing).
func WarnOnMissingKey() TemplateOption {
	return func(c *TemplateConfig) {
		c.DeleteMissingKey = true
		c.WarnOnMissingKey = true
	}
}

// SetMissingKeyLogger allows setting a custom *slog.Logger.
func SetMissingKeyLogger(logger *slog.Logger) TemplateOption {
	return func(c *TemplateConfig) {
		c.MissingKeyLogger = logger
	}
}

// WithAutoExpandRows enables implicit table-row expansion: a row containing
// {{.Array.0.Field}} is automatically cloned for every element of Array.
// Disabled by default.
func WithAutoExpandRows(data any) RenderOption {
	return func(t *docxTemplate) {
		t.AddPreProcessors(docx.AutoExpandRowsPreProcessor(data))
	}
}

// AutoExpandRows is the v1-style TemplateOption equivalent.
func AutoExpandRows(data any) TemplateOption {
	return func(c *TemplateConfig) {
		c.PreProcessors = append(c.PreProcessors, docx.AutoExpandRowsPreProcessor(data))
	}
}
