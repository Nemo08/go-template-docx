package gotemplatedocx

import (
	"archive/zip"
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
	input    bytes.Buffer
	output   bytes.Buffer
	rel      *docx.Relationship
	relMedia []docx.MediaRel
	// filename : { data, wordFilename }
	media               docx.MediaMap
	xlsxChartsMeta      xlsxChartsMap
	templateFuncs       template.FuncMap
	filesPreProcessors  []xml.HandlersMap
	filesPostProcessors []xml.HandlersMap
	//Options
	removeEmptyTableRows bool //remove empty table rows in template
	removeRangeRows      bool //remove empty rows left by {{range}}/{{end}} directives
	ignoreMissingKey     bool
	// warnOnMissingKey — если true, при обработке шаблона выводит предупреждение
	// в консоль для каждого плейсхолдера, которого нет в переданных данных.
	warnOnMissingKey bool
	missingKeyLogger *slog.Logger
	// filename — имя исходного .docx файла, используется в логах.
	filename string
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
	tpl := &docxTemplate{
		input:                inputBuffer,
		output:               bytes.Buffer{},
		media:                make(docx.MediaMap),
		rel:                  &docx.Relationship{},
		relMedia:             []docx.MediaRel{},
		xlsxChartsMeta:       make(xlsxChartsMap),
		templateFuncs:        copyTemplateFuncs(docx.TemplateFuncs),
		filesPreProcessors:   []xml.HandlersMap{},
		filesPostProcessors:  []xml.HandlersMap{},
		removeEmptyTableRows: true,
		ignoreMissingKey:     false,
		warnOnMissingKey:     false,
		missingKeyLogger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
		filename:             filename,
	}

	for _, opt := range options {
		opt(tpl)
	}

	return tpl
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

	dt.media[filename] = &docx.Media{
		Data: data,
	}
}

// AddTemplateFuncs adds custom template functions.
func (dt *docxTemplate) AddTemplateFuncs(funcMap template.FuncMap) {
	for funcName, fn := range funcMap {
		dt.templateFuncs[funcName] = fn
	}
}

// AddPreProcessors adds XML pre-processing maps.
func (dt *docxTemplate) AddPreProcessors(filesPreProcessors ...xml.HandlersMap) {
	dt.filesPreProcessors = append(dt.filesPreProcessors, filesPreProcessors...)
}

// AddPostProcessors adds XML post-processing maps.
func (dt *docxTemplate) AddPostProcessors(filesPostProcessors ...xml.HandlersMap) {
	dt.filesPostProcessors = append(dt.filesPostProcessors, filesPostProcessors...)
}

// warnMissingKeysInFile compares template variables with data and warns about missing keys.
func (dt *docxTemplate) warnMissingKeysInFile(tmpl *template.Template, data map[string]any) {
	docxName := dt.filename
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
			dt.missingKeyLogger.Warn("missing key in template", "file", docxName, "placeholder", v)
		}
	}
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
	src, err := zio.NewFromBytes(dt.input.Bytes())
	if err != nil {
		return nil, fmt.Errorf("unable to create DOCX zip source: %w", err)
	}

	vars := map[string]struct{}{}
	err = src.Each(func(name string) error {
		b, found, err := src.ReadFile(name)
		if err != nil {
			return fmt.Errorf("unable to read file '%s': %w", name, err)
		}
		if !found {
			return nil
		}

		tmpl, err := template.New(path.Base(name)).Funcs(dt.templateFuncs).Parse(xmlutil.PatchXml(string(b)))
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

// normalizeTemplateValues converts templateValues to a consistent map[string]any form.
func (dt *docxTemplate) normalizeTemplateValues(templateValues any) any {
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

func matchRe(re... *regexp.Regexp) fileFilter {
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
func copyNonTemplateFiles(zipWriter *zip.Writer, src zio.FileSource, skip fileFilter) error {
	return src.Each(func(filename string) error {
		if skip(filename) {
			return nil
		}
		return zio.CopyToZip(zipWriter, src, filename)
	})
}

// updateContentTypes adds media type entries for loaded images.
func (dt *docxTemplate) updateContentTypes(zipWriter *zip.Writer, src zio.FileSource) error {
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
	for filename := range dt.media {
		ext := path.Ext(filename)
		switch lowerExt := strings.ToLower(ext); lowerExt {
		case ".jpg", ".jpeg", ".jfif":
			contentTypes.AddDefaultUnique(lowerExt[1:], "image/jpeg")
		case ".png":
			contentTypes.AddDefaultUnique("png", "image/png")
		}
	}
	updatedCt, err := contentTypes.ToXml()
	if err != nil {
		return fmt.Errorf("unable to marshal content types to XML: %w", err)
	}
	return zio.RewriteToZip(zipWriter, src, docx.ContentTypesPath, []byte(updatedCt))
}

// parseDocumentRels parses the document relationships file.
func (dt *docxTemplate) parseDocumentRels(src zio.FileSource) error {
	relData, found, err := src.ReadFile(docx.DocumentRelsPath)
	if err != nil {
		return fmt.Errorf("unable to read rel file '%s': %w", docx.DocumentRelsPath, err)
	}
	if !found {
		return fmt.Errorf("rel file '%s' not found", docx.DocumentRelsPath)
	}
	dt.rel, err = docx.ParseRelationship(relData)
	if err != nil {
		return fmt.Errorf("unable to parse rel file '%s': %w", docx.DocumentRelsPath, err)
	}
	return nil
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

// processXlsxFiles applies templates to all embedded XLSX files.
func (dt *docxTemplate) processXlsxFiles(zipWriter *zip.Writer, src zio.FileSource, templateValues any) error {
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
		if err := dt.writeXlsxIntoZip(zipWriter, src, xlsxFilename, xlsxData, templateValues, dt.ignoreMissingKey); err != nil {
			return fmt.Errorf("unable to apply template to XLSX file '%s': %w", xlsxFilename, err)
		}
	}
	return nil
}

// warnMissingKeysForFile warns about missing template keys in a file.
func (dt *docxTemplate) warnMissingKeysForFile(name string, src zio.FileSource, warnDataMap map[string]any) {
	if !dt.warnOnMissingKey || warnDataMap == nil {
		return
	}
	b, found, err := src.ReadFile(name)
	if err != nil || !found {
		return
	}
	tmpl, err := template.New(path.Base(name)).Funcs(dt.templateFuncs).Parse(xmlutil.PatchXml(string(b)))
	if err != nil {
		return
	}
	dt.warnMissingKeysInFile(tmpl, warnDataMap)
}

// processHeadersFootersDocument applies templates to header, footer, and main document files.
func (dt *docxTemplate) processHeadersFootersDocument(zipWriter *zip.Writer, src zio.FileSource, applyTemplate func(name string, content []byte, data any) ([]byte, []docx.MediaRel, error), templateValues any, warnDataMap map[string]any) error {
	for i := 1; ; i++ {
		headerName := fmt.Sprintf(docx.HeaderPathFormat, i)
		data, found, err := src.ReadFile(headerName)
		if err != nil {
			return fmt.Errorf("unable to read header file '%s': %w", headerName, err)
		}
		if !found {
			break
		}
		dt.warnMissingKeysForFile(headerName, src, warnDataMap)
		output, media, err := applyTemplate(headerName, data, templateValues)
		if err != nil {
			return fmt.Errorf("unable to apply template to header file '%s': %w", headerName, err)
		}
		dt.relMedia = append(dt.relMedia, media...)
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
		dt.warnMissingKeysForFile(footerName, src, warnDataMap)
		output, media, err := applyTemplate(footerName, data, templateValues)
		if err != nil {
			return fmt.Errorf("unable to apply template to footer file '%s': %w", footerName, err)
		}
		dt.relMedia = append(dt.relMedia, media...)
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
	dt.warnMissingKeysForFile(docx.DocumentXMLPath, src, warnDataMap)
	output, media, err := applyTemplate(docx.DocumentXMLPath, documentData, templateValues)
	if err != nil {
		return fmt.Errorf("unable to apply template to document file: %w", err)
	}
	dt.relMedia = append(dt.relMedia, media...)
	if err := zio.RewriteToZip(zipWriter, src, docx.DocumentXMLPath, output); err != nil {
		return fmt.Errorf("unable to write document file: %w", err)
	}
	return nil
}

// processChartFiles applies templates and updates chart data for all chart XML files.
func (dt *docxTemplate) processChartFiles(zipWriter *zip.Writer, src zio.FileSource, chartRelToTargetXlsx map[string]string, templateValues any, warnDataMap map[string]any) error {
	for i := 1; ; i++ {
		chartN := fmt.Sprintf(docx.ChartPathFormat, i)
		chartContent, found, err := src.ReadFile(chartN)
		if err != nil {
			return fmt.Errorf("unable to read chart file '%s': %w", chartN, err)
		}
		if !found {
			break
		}
		dt.warnMissingKeysForFile(chartN, src, warnDataMap)

		fileContent, err := docx.ApplyTemplateToXml(chartN, chartContent, templateValues, dt.templateFuncs, dt.ignoreMissingKey)
		if err != nil {
			return fmt.Errorf("unable to apply template to chart file '%s': %w", chartN, err)
		}
		chartFilename, err := docx.ExtractChartFilename(chartN)
		if err != nil {
			return fmt.Errorf("unable to extract chart name from file '%s': %w", chartN, err)
		}
		xlsxFileTarget := chartRelToTargetXlsx[chartFilename]
		fileContent, err = docx.UpdateChart(fileContent, dt.xlsxChartsMeta[xlsxFileTarget])
		if err != nil {
			return fmt.Errorf("unable to update preview chart file '%s': %w", chartN, err)
		}
		if err := zio.RewriteToZip(zipWriter, src, chartN, fileContent); err != nil {
			return fmt.Errorf("unable to rewrite chart file '%s': %w", chartN, err)
		}
	}
	return nil
}

// updateDocumentRels rewrites the document relationships file with any new media references.
func (dt *docxTemplate) updateDocumentRels(zipWriter *zip.Writer, src zio.FileSource) error {
	documentRelContent, found, err := src.ReadFile(docx.DocumentRelsPath)
	if err != nil {
		return fmt.Errorf("unable to read rel file '%s': %w", docx.DocumentRelsPath, err)
	}
	if !found {
		return fmt.Errorf("rel file '%s' not found", docx.DocumentRelsPath)
	}
	if len(dt.relMedia) != 0 {
		dt.rel.AddMediaToRels(dt.relMedia)
		documentRelContent, err = dt.rel.ToXml()
		if err != nil {
			return fmt.Errorf("unable to marshal rels: %w", err)
		}
	}
	return zio.RewriteToZip(zipWriter, src, docx.DocumentRelsPath, documentRelContent)
}

// applyTemplatePipeline runs the core template pipeline: parse input, apply template, write output.
func (dt *docxTemplate) applyTemplatePipeline(templateValues any) error {
	zipWriter := zip.NewWriter(&dt.output)

	src, err := zio.NewFromBytes(dt.input.Bytes())
	if err != nil {
		return fmt.Errorf("unable to create DOCX zip source: %w", err)
	}

	document, err := docx.ParseDocumentMeta(src, dt.templateFuncs)
	if err != nil {
		return fmt.Errorf("unable to parse document metadata: %w", err)
	}
	document.SetRemoveEmptyTableRows(dt.removeEmptyTableRows)
	document.SetRemoveRangeRows(dt.removeRangeRows)
	document.SetIgnoreMissingKey(dt.ignoreMissingKey)

	if err := dt.writeMediaFiles(zipWriter, document.NextImageNumber); err != nil {
		return err
	}
	document.SetMediaMap(dt.media)

	if err := copyNonTemplateFiles(zipWriter, src, defaultSkipFilter); err != nil {
		return err
	}
	if err := dt.updateContentTypes(zipWriter, src); err != nil {
		return err
	}
	if err := dt.parseDocumentRels(src); err != nil {
		return err
	}

	chartRelToTargetXlsx, err := buildChartXlsxMap(src)
	if err != nil {
		return err
	}

	if err := dt.processXlsxFiles(zipWriter, src, templateValues); err != nil {
		return err
	}

	var warnDataMap map[string]any
	if dt.warnOnMissingKey {
		warnDataMap = toStringMap(templateValues)
	}

	if err := dt.processHeadersFootersDocument(zipWriter, src, document.ApplyTemplate, templateValues, warnDataMap); err != nil {
		return err
	}
	if err := dt.processChartFiles(zipWriter, src, chartRelToTargetXlsx, templateValues, warnDataMap); err != nil {
		return err
	}
	if err := dt.updateDocumentRels(zipWriter, src); err != nil {
		return err
	}

	return zipWriter.Close()
}

// Apply applies the template with the provided values to the DOCX file.
func (dt *docxTemplate) Apply(templateValues any) error {
	templateValues = dt.normalizeTemplateValues(templateValues)

	if len(dt.filesPreProcessors) > 0 {
		if err := xml.ProcessedOutput(dt.filesPreProcessors, &dt.input, "pre"); err != nil {
			return fmt.Errorf("unable to pre-process output DOCX file: %w", err)
		}
	}

	if err := dt.applyTemplatePipeline(templateValues); err != nil {
		return err
	}

	if len(dt.filesPostProcessors) > 0 {
		if err := xml.ProcessedOutput(dt.filesPostProcessors, &dt.output, "post"); err != nil {
			return fmt.Errorf("unable to post-process output DOCX file: %w", err)
		}
	}

	return nil
}

// writeMediaFiles writes media files into the ZIP and assigns Word-compatible filenames.
func (dt *docxTemplate) writeMediaFiles(zipWriter *zip.Writer, nextImageNumber func() uint64) error {
	for filename, media := range dt.media {
		imageN := nextImageNumber()
		wordFilename := fmt.Sprintf("image%d%s", imageN, path.Ext(filename))
		dt.media[filename].WordFilename = wordFilename

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

// Save saves the modified docx file to the specified filename.
func (dt *docxTemplate) Save(filename string) error {
	return os.WriteFile(filename, dt.output.Bytes(), 0644)
}

// Bytes returns the output bytes of the output xlsx file bytes.
func (dt *docxTemplate) Bytes() []byte {
	return dt.output.Bytes()
}

type TemplateOption func(*docxTemplate)

func WithFilename(name string) TemplateOption {
	return func(t *docxTemplate) {
		t.filename = name
	}
}

func NoRemoveEmptyTableRows() TemplateOption {
	return func(t *docxTemplate) {
		t.removeEmptyTableRows = false
	}
}

// RemoveRangeRows removes range directive rows after template rendering.
func RemoveRangeRows() TemplateOption {
	return func(t *docxTemplate) {
		t.removeRangeRows = true
	}
}

func IgnoreMissingKey() TemplateOption {
	return func(t *docxTemplate) {
		t.ignoreMissingKey = true
	}
}

// WarnOnMissingKey enables warnings for missing keys in stderr.
func WarnOnMissingKey() TemplateOption {
	return func(t *docxTemplate) {
		t.ignoreMissingKey = true
		t.warnOnMissingKey = true
	}
}

// SetMissingKeyLogger allows setting a custom *slog.Logger.
func SetMissingKeyLogger(logger *slog.Logger) TemplateOption {
	return func(t *docxTemplate) {
		t.missingKeyLogger = logger
	}
}
