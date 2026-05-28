package gotemplatedocx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"

	"github.com/JJJJJJack/go-template-docx/internal/docx"
	"github.com/JJJJJJack/go-template-docx/internal/xlsx"
	"github.com/JJJJJJack/go-template-docx/internal/zio"
)

var (
	reSheetN         = regexp.MustCompile(`xl/worksheets/sheet\d*\.xml`)
	reSharedStringsN = regexp.MustCompile(`xl/(sharedStrings\d*)\.xml`)
)

type chartCellAndValue map[string]string

type xlsxChartsMap map[string]chartCellAndValue

// modifyXlsxInMemory modifies an XLSX byte slice in memory, applying templates.
func (dt *docxTemplate) modifyXlsxInMemory(xlsxName string, xlsxData []byte, templateValues any, ignoreMissingKey, deleteMissingKey bool) ([]byte, error) {
	var sharedStringsNumbers map[int]string
	var sharedStringsNewIndexes map[int]int

	xlsxSrc, err := zio.NewFromBytes(xlsxData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XLSX zip: %w", err)
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

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

		dt.xlsxChartsMeta[xlsxName] = chartValues

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

func (dt *docxTemplate) writeXlsxIntoZip(zipWriter *zip.Writer, src zio.FileSource, xlsxFilename string, xlsxData []byte, templateValues any, ignoreMissingKey, deleteMissingKey bool) error {
	xlsxBytes, err := dt.modifyXlsxInMemory(xlsxFilename, xlsxData, templateValues, ignoreMissingKey, deleteMissingKey)
	if err != nil {
		return fmt.Errorf("error modifying XLSX in memory: %w", err)
	}

	if err := zio.RewriteToZip(zipWriter, src, xlsxFilename, xlsxBytes); err != nil {
		return fmt.Errorf("error writing XLSX to zip: %w", err)
	}

	return nil
}
