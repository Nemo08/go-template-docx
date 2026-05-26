package docx

// xmlStdHeader is the standard XML declaration used for OOXML files.
var xmlStdHeader = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)

// Standard OOXML paths used in DOCX processing.
const (
	DocumentXMLPath         = "word/document.xml"
	DocumentRelsPath        = "word/_rels/document.xml.rels"
	ContentTypesPath        = "[Content_Types].xml"
	ImagePrefix             = "word/media/image"

	HeaderPathFormat        = "word/header%d.xml"
	FooterPathFormat        = "word/footer%d.xml"
	ChartPathFormat         = "word/charts/chart%d.xml"
	ChartRelsPathFormat     = "word/charts/_rels/chart%d.xml.rels"
	XlsxPathFormat          = "word/embeddings/Microsoft_Excel_Worksheet%d.xlsx"
	XlsxFirstPath           = "word/embeddings/Microsoft_Excel_Worksheet.xlsx"

	// XLSX internal paths
	SharedStringsPath = "xl/sharedStrings.xml"
	SheetPathFormat   = "xl/worksheets/sheet%d.xml"
)
