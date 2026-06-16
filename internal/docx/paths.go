package docx

// Standard OOXML paths used in DOCX processing.
const (
	DocumentXMLPath  = "word/document.xml"
	DocumentRelsPath = "word/_rels/document.xml.rels"
	ContentTypesPath = "[Content_Types].xml"
	ImagePrefix      = "word/media/image"

	HeaderPathFormat      = "word/header%d.xml"
	HeaderRelsPathFormat  = "word/_rels/header%d.xml.rels"
	FooterPathFormat      = "word/footer%d.xml"
	FooterRelsPathFormat  = "word/_rels/footer%d.xml.rels"
	ChartPathFormat     = "word/charts/chart%d.xml"
	ChartRelsPathFormat = "word/charts/_rels/chart%d.xml.rels"
	XlsxPathFormat      = "word/embeddings/Microsoft_Excel_Worksheet%d.xlsx"
	XlsxFirstPath       = "word/embeddings/Microsoft_Excel_Worksheet.xlsx"

	// XLSX internal paths
	SharedStringsPath = "xl/sharedStrings.xml"
	SheetPathFormat   = "xl/worksheets/sheet%d.xml"
)
