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
	"github.com/JJJJJJack/go-template-docx/xml"
	goziputils "github.com/JJJJJJack/go-zip-utils"
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
	ignoreMissingKey     bool
	// warnOnMissingKey — если true, при обработке шаблона выводит предупреждение
	// в консоль для каждого плейсхолдера, которого нет в переданных данных.
	warnOnMissingKey bool
	missingKeyLogger *slog.Logger
	// filename — имя исходного .docx файла, используется в логах.
	filename string
}

// copyTemplateFuncs создаёт независимую копию FuncMap.
// Необходимо чтобы каждый экземпляр docxTemplate имел свою map
// и не было гонки при параллельном вызове AddTemplateFuncs из горутин.
func copyTemplateFuncs(src template.FuncMap) template.FuncMap {
	dst := make(template.FuncMap, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// NewDocxTemplateFromBytes creates a new docxTemplate object from the provided DOCX file bytes.
// The docxTemplate object can be used through the exposed high-level APIs.
// FIX: added options variadic parameter (was missing, unlike NewDocxTemplateFromFilename)
func NewDocxTemplateFromBytes(docxBytes []byte, options ...TemplateOption) (*docxTemplate, error) {
	inputBuffer := bytes.Buffer{}

	_, err := inputBuffer.Write(docxBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to write DOCX bytes to buffer: %w", err)
	}

	// FIX: initialise defaults consistently with NewDocxTemplateFromFilename
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
	}

	for _, opt := range options {
		opt(tpl)
	}

	return tpl, nil
}

// NewDocxTemplateFromFilename creates a new docxTemplate object from the provided DOCX filename (reading from disk).
// The docxTemplate object can be used through the exposed high-level APIs.
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

	tpl := &docxTemplate{
		input:               inputBuffer,
		output:              bytes.Buffer{},
		media:               docx.MediaMap{},
		rel:                 &docx.Relationship{},
		relMedia:            []docx.MediaRel{},
		xlsxChartsMeta:      make(xlsxChartsMap),
		templateFuncs:       copyTemplateFuncs(docx.TemplateFuncs),
		filesPreProcessors:  []xml.HandlersMap{},
		filesPostProcessors: []xml.HandlersMap{},

		removeEmptyTableRows: true,
		ignoreMissingKey:     false,
		warnOnMissingKey:     false,
		missingKeyLogger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
		filename:             docxFilename,
	}

	for _, opt := range options {
		opt(tpl)
	}

	return tpl, nil
}

// Media adds a media file to the docxTemplate object.
// Supported media types are currently limited to JPEG and PNG images.
// The filename match the string you pass in the template expression using the image function.
// For example {{ image "computer.png" }} will load the docx.Media that have "computer.png" as its filename.
// The data should be the byte content of the media file.
func (dt *docxTemplate) Media(filename string, data []byte) {
	filename = filepath.Base(filename)

	dt.media[filename] = &docx.Media{
		Data: data,
		// Word media folder name (e.g., "image1.png") will be assigned after parsing the document metadata
	}
}

// AddTemplateFuncs adds your custom template functions to evaluate when applying the template.
// Existing functions will be shadowed if the same name is used.
func (dt *docxTemplate) AddTemplateFuncs(funcMap template.FuncMap) {
	for funcName, fn := range funcMap {
		dt.templateFuncs[funcName] = fn
	}
}

// AddPreProcessors adds XML pre-processing maps in which the key is the XML file path
// (e.g., "word/document.xml") and the value is a list of functions that overwrite it sequentially,
// before the template is applied.
func (dt *docxTemplate) AddPreProcessors(filesPreProcessors ...xml.HandlersMap) {
	dt.filesPreProcessors = filesPreProcessors
}

// AddPostProcessors adds XML post-processing maps in which the key is the XML file path
// (e.g., "word/document.xml") and the value is a list of functions that overwrite it sequentially,
// after the template is applied.
func (dt *docxTemplate) AddPostProcessors(filesPostProcessors ...xml.HandlersMap) {
	dt.filesPostProcessors = filesPostProcessors
}

// warnMissingKeysInFile сравнивает переменные шаблона из конкретного XML-файла
// с плоским представлением данных и выводит предупреждение для каждого
// отсутствующего ключа. В лог пишется имя исходного .docx файла (dt.filename).
// Вызывается только при warnOnMissingKey == true.
func (dt *docxTemplate) warnMissingKeysInFile(tmpl *template.Template, data map[string]any) {
	docxName := dt.filename
	if docxName == "" {
		docxName = "<bytes>"
	}
	vars := docxtemplate.ExtractAllVariables(tmpl)
	for v := range vars {
		// Переменные вида ".Field" или ".Field.Sub" — проверяем верхний уровень.
		key := strings.TrimPrefix(v, ".")
		if idx := strings.Index(key, "."); idx != -1 {
			key = key[:idx]
		}
		// Пропускаем служебные переменные Go-шаблонов ($var)
		if strings.HasPrefix(key, "$") || key == "" {
			continue
		}
		if _, ok := data[key]; !ok {
			dt.missingKeyLogger.Warn("missing key in template", "file", docxName, "placeholder", v)
		}
	}
}

// toStringMap пытается привести templateValues к map[string]any для проверки ключей.
// Если значение не является map — возвращает nil (предупреждения не выводятся).
func toStringMap(templateValues any) map[string]any {
	switch v := templateValues.(type) {
	case map[string]any:
		return v
	default:
		// Пробуем через JSON round-trip (например, если передана struct)
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

// GetTemplateVariables extracts and returns all template variables used in the DOCX file
// as a map.
func (dt *docxTemplate) GetTemplateVariables() (map[string]struct{}, error) {
	zipMap, err := goziputils.NewZipMapFromBytes(dt.input.Bytes())
	if err != nil {
		return nil, fmt.Errorf("unable to create DOCX zip map: %w", err)
	}

	vars := map[string]struct{}{}
	for _, f := range zipMap {
		b, err := goziputils.ReadZipFileContent(f)
		if err != nil {
			return nil, fmt.Errorf("unable to read file '%s': %w", f.Name, err)
		}

		tmpl, err := template.New(path.Base(f.Name)).Funcs(dt.templateFuncs).Parse(docx.PatchXml(string(b)))
		if err != nil {
			return nil, fmt.Errorf("unable to parse template in file '%s': %w", f.Name, err)
		}

		x := docxtemplate.ExtractAllVariables(tmpl)
		for k := range x {
			vars[k] = struct{}{}
		}
	}

	return vars, nil
}

// Apply applies the template with the provided values to the DOCX file.
// The templateValues parameter can be any type that can be marshalled to JSON.
func (dt *docxTemplate) Apply(templateValues any) error {
	switch v := templateValues.(type) {
	case []byte:
		if len(v) == 0 {
			templateValues = map[string]any{}
			break
		}
		err := json.Unmarshal(v, &templateValues)
		if err != nil {
			return fmt.Errorf("error unmarshalling templateValues: %w", err)
		}
		// После анмаршала nil-значения рендерятся как "<no value>" что ломает XML.
		// Заменяем nil → "" на верхнем уровне map.
		if m, ok := templateValues.(map[string]any); ok {
			for k, val := range m {
				if val == nil {
					m[k] = ""
				}
			}
		}
	case map[string]string:
		// Конвертируем map[string]string → map[string]any чтобы при missingkey=zero
		// отсутствующий ключ рендерился как "" а не как "<no value>", что ломает XML.
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[k] = val
		}
		templateValues = m
	}

	// custom user pre processing
	if len(dt.filesPreProcessors) > 0 {
		err := xml.ProcessedOutput(dt.filesPreProcessors, &dt.input, "pre")
		if err != nil {
			return fmt.Errorf("unable to pre-process output DOCX file: %w", err)
		}
	}

	zipWriter := zip.NewWriter(&dt.output)

	docxZipMap, err := goziputils.NewZipMapFromBytes(dt.input.Bytes())
	if err != nil {
		return fmt.Errorf("unable to create DOCX zip map: %w", err)
	}

	document, err := docx.ParseDocumentMeta(docxZipMap, dt.templateFuncs)
	if err != nil {
		return fmt.Errorf("unable to parse document metadata: %w", err)
	}

	//set options
	document.RemoveEmptyTableRows = dt.removeEmptyTableRows
	document.IgnoreMissingKey = dt.ignoreMissingKey

	// При ignoreMissingKey=true Go-шаблон рендерит отсутствующий ключ map[string]any
	// как "<no value>" (нулевое значение interface{}), что ломает XML.
	// Решение: заранее добавить все переменные шаблона в map со значением "",
	// чтобы ключ всегда существовал и возвращал пустую строку.
	if dt.ignoreMissingKey {
		if m, ok := templateValues.(map[string]any); ok {
			allVars, _ := dt.GetTemplateVariables()
			for v := range allVars {
				key := strings.TrimPrefix(v, ".")
				if idx := strings.Index(key, "."); idx != -1 {
					key = key[:idx]
				}
				if key == "" || strings.HasPrefix(key, "$") {
					continue
				}
				if _, exists := m[key]; !exists {
					m[key] = ""
				}
			}
		}
	}

	// put loaded medias into the new docx file, following docx naming convention with sequential numbers
	for filename, media := range dt.media {
		// assign each filename to its word convention equivalent path "word/media/imageN.ext"
		imageN := document.NextImageNumber()
		wordFilename := fmt.Sprintf("image%d%s", imageN, path.Ext(filename))

		dt.media[filename].WordFilename = wordFilename

		filepath := path.Join("word/media", media.WordFilename)
		err := goziputils.WriteFile(zipWriter, filepath, media.Data)
		if err != nil {
			return fmt.Errorf("unable to write media file '%s': %w", filepath, err)
		}
	}
	document.SetMediaMap(dt.media)

	// Copy all files except the ones that will be processed
	documentRelsFilename := "word/_rels/document.xml.rels"
	contentTypesFilename := "[Content_Types].xml"
	chartsMatcher := regexp.MustCompile(`word/charts/chart\d*?\.xml`)
	xlsxMatcher := regexp.MustCompile(`/embeddings/Microsoft_Excel_Worksheet\d*?\.xlsx`)
	headerFooterDocumentMatcher := regexp.MustCompile(`word/(header|footer|document)\d*?\.xml`)
	for filename, f := range docxZipMap {
		switch {
		case
			filename == documentRelsFilename,
			filename == contentTypesFilename,
			chartsMatcher.MatchString(filename),
			xlsxMatcher.MatchString(filename),
			headerFooterDocumentMatcher.MatchString(filename):
			continue
		}

		err := goziputils.CopyFile(zipWriter, f)
		if err != nil {
			return fmt.Errorf("unable to copy original file '%s': %w", f.Name, err)
		}
	}

	// Edit [Content_Types].xml if media files are provided
	ctFile := docxZipMap[contentTypesFilename]
	ctData, err := goziputils.ReadZipFileContent(ctFile)
	if err != nil {
		return fmt.Errorf("unable to read content types file '%s': %w", ctFile.Name, err)
	}

	contentTypes, err := docx.ParseContentTypes(ctData)
	if err != nil {
		return fmt.Errorf("unable to parse content types file '%s': %w", ctFile.Name, err)
	}

	for filename := range dt.media {
		ext := path.Ext(filename)

		switch lowerExt := strings.ToLower(ext); lowerExt {
		case ".jpg", ".jpeg", ".jfif":
			contentTypes.AddDefaultUnique(lowerExt[1:], "image/jpeg")
		case ".png":
			contentTypes.AddDefaultUnique("png", "image/png")
		default:
			fmt.Println("Unsupported media file type (only accepting jpg/png for now):", filename)
			continue
		}
	}

	updatedCt, err := contentTypes.ToXml()
	if err != nil {
		return fmt.Errorf("unable to marshal content types to XML: %w", err)
	}

	err = goziputils.RewriteFileIntoZipWriter(zipWriter, ctFile, []byte(updatedCt))
	if err != nil {
		return fmt.Errorf("unable to replace content types file '%s': %w", ctFile.Name, err)
	}

	relData, err := goziputils.ReadZipFileContent(docxZipMap[documentRelsFilename])
	if err != nil {
		return fmt.Errorf("unable to read rel file '%s': %w", documentRelsFilename, err)
	}

	dt.rel, err = docx.ParseRelationship(relData)
	if err != nil {
		return fmt.Errorf("unable to parse rel file '%s': %w", documentRelsFilename, err)
	}

	// Map chart files to their target XLSX files
	chartRelToTargetXlsx := make(map[string]string)
	for i := 1; ; i++ {
		relsChartFilename := fmt.Sprintf("word/charts/_rels/chart%d.xml.rels", i)
		f := docxZipMap[relsChartFilename]
		if f == nil {
			break
		}

		fileContent, err := goziputils.ReadZipFileContent(f)
		if err != nil {
			return fmt.Errorf("unable to read chart rel file '%s': %w", f.Name, err)
		}

		chartsRelationships, _ := docx.ParseRelationship(fileContent)
		for _, relationship := range chartsRelationships.Relationships {
			if !xlsxMatcher.MatchString(relationship.Target) {
				continue
			}

			targetXlsxFilename := strings.Replace(relationship.Target, "../", "word/", 1)
			chartFilename, err := docx.ExtractChartFilename(f.Name)
			if err != nil {
				return fmt.Errorf("unable to extract chart name from file '%s': %w", f.Name, err)
			}
			chartRelToTargetXlsx[chartFilename] = targetXlsxFilename
		}
	}

	// Apply template to the XLSX files
	for i := 0; ; i++ {
		xlsxFilename := fmt.Sprintf("word/embeddings/Microsoft_Excel_Worksheet%d.xlsx", i)
		if i == 0 {
			xlsxFilename = "word/embeddings/Microsoft_Excel_Worksheet.xlsx"
		}
		f := docxZipMap[xlsxFilename]
		if f == nil {
			break
		}

		err := dt.writeXlsxIntoZip(f, zipWriter, templateValues)
		if err != nil {
			return fmt.Errorf("unable to apply template to XLSX file '%s': %w", f.Name, err)
		}
	}

	// Если включены предупреждения о пропущенных ключах — один раз конвертируем
	// данные в map для последующих проверок по каждому файлу.
	var warnDataMap map[string]any
	if dt.warnOnMissingKey {
		warnDataMap = toStringMap(templateValues)
	}

	// Apply template to the header files
	for i := 1; ; i++ {
		headerFilename := fmt.Sprintf("word/header%d.xml", i)
		f := docxZipMap[headerFilename]
		if f == nil {
			break
		}

		if dt.warnOnMissingKey && warnDataMap != nil {
			if b, err2 := goziputils.ReadZipFileContent(f); err2 == nil {
				if tmpl, err2 := template.New(path.Base(f.Name)).Funcs(dt.templateFuncs).Parse(docx.PatchXml(string(b))); err2 == nil {
					dt.warnMissingKeysInFile(tmpl, warnDataMap)
				}
			}
		}

		media, err := document.ApplyTemplate(f, zipWriter, templateValues)
		if err != nil {
			return fmt.Errorf("unable to apply template to header file '%s': %w", f.Name, err)
		}

		dt.relMedia = append(dt.relMedia, media...)
	}

	// Apply template to the footer files
	for i := 1; ; i++ {
		footerFilename := fmt.Sprintf("word/footer%d.xml", i)
		f := docxZipMap[footerFilename]
		if f == nil {
			break
		}

		if dt.warnOnMissingKey && warnDataMap != nil {
			if b, err2 := goziputils.ReadZipFileContent(f); err2 == nil {
				if tmpl, err2 := template.New(path.Base(f.Name)).Funcs(dt.templateFuncs).Parse(docx.PatchXml(string(b))); err2 == nil {
					dt.warnMissingKeysInFile(tmpl, warnDataMap)
				}
			}
		}

		media, err := document.ApplyTemplate(f, zipWriter, templateValues)
		if err != nil {
			return fmt.Errorf("unable to apply template to footer file '%s': %w", f.Name, err)
		}

		dt.relMedia = append(dt.relMedia, media...)
	}

	// Apply template to the main document file
	documentFile := docxZipMap["word/document.xml"]
	if documentFile == nil {
		return fmt.Errorf("word/document.xml not found in the DOCX file")
	}

	if dt.warnOnMissingKey && warnDataMap != nil {
		if b, err2 := goziputils.ReadZipFileContent(documentFile); err2 == nil {
			if tmpl, err2 := template.New(path.Base(documentFile.Name)).Funcs(dt.templateFuncs).Parse(docx.PatchXml(string(b))); err2 == nil {
				dt.warnMissingKeysInFile(tmpl, warnDataMap)
			}
		}
	}

	media, err := document.ApplyTemplate(documentFile, zipWriter, templateValues)
	if err != nil {
		return fmt.Errorf("unable to apply template to document file: %w", err)
	}

	dt.relMedia = append(dt.relMedia, media...)

	// Apply template to the chart files
	for i := 1; ; i++ {
		chartN := fmt.Sprintf("word/charts/chart%d.xml", i)

		f := docxZipMap[chartN]
		if f == nil {
			break
		}

		// FIX: pass ignoreMissingKey so chart templates respect the same option
		// as document/header/footer templates. Previously this flag was ignored
		// for chart files, causing a panic/error when a placeholder was absent
		// from the supplied data.
		if dt.warnOnMissingKey && warnDataMap != nil {
			if b, err2 := goziputils.ReadZipFileContent(f); err2 == nil {
				if tmpl, err2 := template.New(path.Base(f.Name)).Funcs(dt.templateFuncs).Parse(docx.PatchXml(string(b))); err2 == nil {
					dt.warnMissingKeysInFile(tmpl, warnDataMap)
				}
			}
		}

		fileContent, err := docx.ApplyTemplateToXml(f, templateValues, dt.templateFuncs, dt.ignoreMissingKey)
		if err != nil {
			return fmt.Errorf("unable to apply template to chart file '%s': %w", f.Name, err)
		}

		chartFilename, err := docx.ExtractChartFilename(f.Name)
		if err != nil {
			return fmt.Errorf("unable to extract chart name from file '%s': %w", f.Name, err)
		}

		xlsxFileTarget := chartRelToTargetXlsx[chartFilename]
		fileContent, err = docx.UpdateChart(fileContent, dt.xlsxChartsMeta[xlsxFileTarget])
		if err != nil {
			return fmt.Errorf("unable to update preview chart file '%s': %w", f.Name, err)
		}

		err = goziputils.RewriteFileIntoZipWriter(zipWriter, f, fileContent)
		if err != nil {
			return fmt.Errorf("unable to rewrite chart file '%s': %w", f.Name, err)
		}
	}

	documentRelFile := docxZipMap[documentRelsFilename]
	documentRelContent, err := goziputils.ReadZipFileContent(documentRelFile)
	if err != nil {
		return fmt.Errorf("unable to read rel file '%s': %w", documentRelsFilename, err)
	}

	if len(dt.relMedia) != 0 {
		dt.rel.AddMediaToRels(dt.relMedia)

		documentRelContent, err = dt.rel.ToXml()
		if err != nil {
			return fmt.Errorf("unable to marshal rels: %w", err)
		}
	}

	err = goziputils.RewriteFileIntoZipWriter(zipWriter, documentRelFile, documentRelContent)
	if err != nil {
		return fmt.Errorf("unable to replace rel file '%s': %w", documentRelsFilename, err)
	}

	err = zipWriter.Close()
	if err != nil {
		return fmt.Errorf("unable to close zip writer: %w", err)
	}

	// custom user post processing
	if len(dt.filesPostProcessors) > 0 {
		err := xml.ProcessedOutput(dt.filesPostProcessors, &dt.output, "post")
		if err != nil {
			return fmt.Errorf("unable to post-process output DOCX file: %w", err)
		}
	}

	return nil
}

// Save saves the modified docx file to the specified filename.
func (dt *docxTemplate) Save(filename string) error {
	return os.WriteFile(filename, dt.output.Bytes(), 0644)
}

// Bytes returns the output bytes of the output xlsx file bytes
// (empty if Apply was not used).
func (dt *docxTemplate) Bytes() []byte {
	return dt.output.Bytes()
}

type TemplateOption func(*docxTemplate)

// WithFilename задаёт имя файла для вывода в предупреждениях о пропущенных ключах.
// Полезно при использовании NewDocxTemplateFromBytes, когда имя файла неизвестно автоматически.
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

func IgnoreMissingKey() TemplateOption {
	return func(t *docxTemplate) {
		t.ignoreMissingKey = true
	}
}

// WarnOnMissingKey включает вывод предупреждений в stderr для каждого
// плейсхолдера, которого нет в переданных данных. Автоматически включает
// IgnoreMissingKey — иначе шаблон вернёт ошибку раньше, чем дойдёт до лога.
func WarnOnMissingKey() TemplateOption {
	return func(t *docxTemplate) {
		t.ignoreMissingKey = true
		t.warnOnMissingKey = true
	}
}

// SetMissingKeyLogger позволяет задать свой *slog.Logger вместо стандартного stderr.
// Пример: gotemplatedocx.SetMissingKeyLogger(slog.Default())
func SetMissingKeyLogger(logger *slog.Logger) TemplateOption {
	return func(t *docxTemplate) {
		t.missingKeyLogger = logger
	}
}
