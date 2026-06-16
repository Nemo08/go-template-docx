package gotemplatedocx

import "regexp"

var (
	reSheetN         = regexp.MustCompile(`xl/worksheets/sheet\d*\.xml`)
	reSharedStringsN = regexp.MustCompile(`xl/(sharedStrings\d*)\.xml`)
)

type chartCellAndValue map[string]string

type xlsxChartsMap map[string]chartCellAndValue
