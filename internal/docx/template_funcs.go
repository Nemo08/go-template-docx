package docx

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"
)

// XMLImageData holds rendering parameters for an embedded image in the document.
type XMLImageData struct {
	DocPrID uint32
	Name    string
	RefID   string
	Cx      int
	Cy      int
}

const imageTemplateXML = `<w:drawing>
  <wp:inline distT="0" distB="0" distL="0" distR="0"
    xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
    xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
    xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"
    xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
    <wp:extent cx="{{.Cx}}" cy="{{.Cy}}" />
    <wp:docPr id="{{.DocPrID}}" name="{{.Name}}" />
    <a:graphic>
      <a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">
        <pic:pic>
          <pic:nvPicPr>
            <pic:cNvPr id="{{.DocPrID}}" name="{{.Name}}" />
            <pic:cNvPicPr />
          </pic:nvPicPr>
          <pic:blipFill>
            <a:blip r:embed="{{.RefID}}" />
            <a:stretch>
              <a:fillRect />
            </a:stretch>
          </pic:blipFill>
          <pic:spPr>
            <a:xfrm>
              <a:off x="0" y="0" />
              <a:ext cx="{{.Cx}}" cy="{{.Cy}}" />
            </a:xfrm>
            <a:prstGeom prst="rect">
              <a:avLst />
            </a:prstGeom>
          </pic:spPr>
        </pic:pic>
      </a:graphicData>
    </a:graphic>
  </wp:inline>
</w:drawing>`

const (
	docxNewlineInject        = `</w:t><w:br/><w:t>`
	docxBreakParagraphInject = `</w:t></w:r></w:p><w:p><w:r><w:t>`

	styleWrapperF     = `<w:rPr>%s</w:rPr><w:t>%s</w:t>`
	boldWTag          = `<w:b /><w:bCs />`
	italicWTag        = `<w:i /><w:iCs />`
	underlineWTag     = `<w:u w:val="single"/>`
	strikethroughWTag = `<w:strike />`
	fontSizeWTagF     = `<w:sz w:val="%d" /><w:szCs w:val="%d" />`
	colorWTagF        = `<w:color w:val="%s" />`
	highlightWTagF    = `<w:highlight w:val="%s" />`
	// HIGHLIGHT all values: https://learn.microsoft.com/en-us/dotnet/api/documentformat.openxml.wordprocessing.highlightcolor?view=openxml-2.8.1
	shadingWTagF = `<w:shd w:val="clear" w:color="auto" w:fill="%s"/>`
)

var (
	boldWrapperF          = fmt.Sprintf(styleWrapperF, boldWTag, "%s")
	italicWrapperF        = fmt.Sprintf(styleWrapperF, italicWTag, "%s")
	underlineWrapperF     = fmt.Sprintf(styleWrapperF, underlineWTag, "%s")
	strikethroughWrapperF = fmt.Sprintf(styleWrapperF, strikethroughWTag, "%s")
	colorWrapperF         = fmt.Sprintf(styleWrapperF, colorWTagF, "%s")
	highlightWrapperF     = fmt.Sprintf(styleWrapperF, highlightWTagF, "%s")
	shadingWrapperF       = fmt.Sprintf(styleWrapperF, shadingWTagF, "%s")
)

func fontSizeWrapperf(sizeHalfPoints int) string {
	if sizeHalfPoints <= 0 {
		sizeHalfPoints = 1
	}

	return fmt.Sprintf(fontSizeWTagF, sizeHalfPoints*2, sizeHalfPoints*2)
}

const (
	fontSizeStylePrefix      = "fontSize:"
	fontSizeStylePrefixShort = "fs:"
	textShadingStylePrefix   = "bg:"
)

type styleEntry struct {
	tag    string
	dupKey string
}

var namedStyles = func() map[string]styleEntry {
	m := map[string]styleEntry{
		"b":             {boldWTag, boldWTag},
		"bold":          {boldWTag, boldWTag},
		"i":             {italicWTag, italicWTag},
		"italic":        {italicWTag, italicWTag},
		"u":             {underlineWTag, underlineWTag},
		"underline":     {underlineWTag, underlineWTag},
		"s":             {strikethroughWTag, strikethroughWTag},
		"strike":        {strikethroughWTag, strikethroughWTag},
		"strikethrough": {strikethroughWTag, strikethroughWTag},
	}
	for _, c := range highlightColorNames {
		m[c] = styleEntry{
			tag:    fmt.Sprintf(highlightWTagF, c),
			dupKey: "<w:highlight w:val=",
		}
	}
	return m
}()

var validHighlightColor = func() map[string]struct{} {
	m := make(map[string]struct{}, len(highlightColorNames))
	for _, c := range highlightColorNames {
		m[c] = struct{}{}
	}
	return m
}()

var highlightColorNames = []string{
	"black", "blue", "cyan", "green",
	"magenta", "red", "yellow", "white",
	"darkBlue", "darkCyan", "darkGreen",
	"darkMagenta", "darkRed", "darkYellow",
	"darkGray", "lightGray", "none",
}

// list enables you to take a variadic number of arguments and
// returns them as a slice of interface{} to another function
// directly from the template expressions.
func list(args ...interface{}) []interface{} {
	return args
}

// formatStylesTags takes a slice of styles and returns the corresponding XML tags.
func formatStylesTags(stylesList []interface{}, funcName string) (string, error) {
	var styles strings.Builder
	for _, arg := range stylesList {
		styleParam, ok := arg.(string)
		if !ok {
			return "", fmt.Errorf("%s got non-string style parameter: %v", funcName, arg)
		}

		tag, err := dispatchStyleTag(styleParam, funcName, styles.String())
		if err != nil {
			return "", err
		}
		styles.WriteString(tag)
	}

	return styles.String(), nil
}

func dispatchStyleTag(styleParam, funcName, existing string) (string, error) {
	if strings.HasPrefix(styleParam, fontSizeStylePrefix) || strings.HasPrefix(styleParam, fontSizeStylePrefixShort) {
		return parseFontSizeStyle(styleParam, funcName, existing)
	}
	if strings.HasPrefix(styleParam, "#") {
		return parseColorStyle(styleParam, funcName, existing)
	}
	if strings.HasPrefix(styleParam, textShadingStylePrefix) {
		return parseShadingStyle(styleParam, funcName, existing)
	}
	return parseNamedStyle(styleParam, funcName, existing)
}

func parseFontSizeStyle(styleParam, funcName, existing string) (string, error) {
	if strings.Contains(existing, "<w:sz w:val=") {
		return "", fmt.Errorf("%s got multiple font size styles", funcName)
	}

	sizeStr := strings.TrimPrefix(styleParam, fontSizeStylePrefix)
	sizeStr = strings.TrimPrefix(sizeStr, fontSizeStylePrefixShort)

	ptSize, err := strconv.Atoi(sizeStr)
	if err != nil {
		return "", fmt.Errorf("%s got invalid size: %s", funcName, sizeStr)
	}

	return fontSizeWrapperf(ptSize), nil
}

func parseColorStyle(styleParam, funcName, existing string) (string, error) {
	if strings.Contains(existing, "<w:color w:val=") {
		return "", fmt.Errorf("%s got multiple color styles", funcName)
	}

	hex := strings.ToUpper(strings.TrimPrefix(styleParam, "#"))
	return fmt.Sprintf(colorWTagF, hex), nil
}

func parseShadingStyle(styleParam, funcName, existing string) (string, error) {
	if strings.Contains(existing, "<w:shd w:val=") {
		return "", fmt.Errorf("%s got multiple background shading styles", funcName)
	}

	hex := strings.ToUpper(strings.TrimPrefix(styleParam, textShadingStylePrefix))
	hex = strings.TrimPrefix(hex, "#")

	return fmt.Sprintf(shadingWTagF, hex), nil
}

func parseNamedStyle(styleParam, funcName, existing string) (string, error) {
	entry, ok := namedStyles[styleParam]
	if !ok {
		return "", fmt.Errorf("%s got unknown style: %s", funcName, styleParam)
	}
	if strings.Contains(existing, entry.dupKey) {
		return "", fmt.Errorf("%s got multiple %s styles", funcName, styleParam)
	}
	return entry.tag, nil
}

// styledText takes a strings and a slice of styles to apply to the text.
// You can use this function to style text with a set variable containing
// a reusable style in your code.
func styledText(text any, styles []interface{}) (string, error) {
	stylesTags, err := formatStylesTags(styles, "styledText")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(styleWrapperF, stylesTags, fmt.Sprint(text)), nil
}

// inlineStyledText applies multiple styles to the given text.
// The first argument is the text, the following arguments are styles.
// You can use this function to apply multiple styles to a text without
// having to wrap them in a list.
func inlineStyledText(text any, styles ...interface{}) (string, error) {
	stylesTags, err := formatStylesTags(styles, "inlineStyledText")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(styleWrapperF, stylesTags, fmt.Sprint(text)), nil
}

// bold makes the text bold
func bold(s any) string {
	return fmt.Sprintf(boldWrapperF, fmt.Sprint(s))
}

// italic makes the text italic
func italic(s any) string {
	return fmt.Sprintf(italicWrapperF, fmt.Sprint(s))
}

// underline underlines the text
func underline(s any) string {
	return fmt.Sprintf(underlineWrapperF, fmt.Sprint(s))
}

// strike applies strikethrough to the text
func strike(s any) string {
	return fmt.Sprintf(strikethroughWrapperF, fmt.Sprint(s))
}

// fontSize sets the font size of the text
func fontSize(s any, size int) string {
	return fmt.Sprintf(styleWrapperF, fontSizeWrapperf(size), fmt.Sprint(s))
}

// color sets the font color of the text
func color(s any, hex string) (string, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "", fmt.Errorf("func 'color': invalid hex color value: %s (must be 6 characters like '0077FF')", hex)
	}

	return fmt.Sprintf(colorWrapperF, strings.ToUpper(hex), fmt.Sprint(s)), nil
}

// highlight applies a highlight color to the text
func highlight(s any, color string) (string, error) {
	if _, ok := validHighlightColor[color]; !ok {
		return "", fmt.Errorf("func 'highlight': invalid highlight color value: %s", color)
	}

	return fmt.Sprintf(highlightWrapperF, color, fmt.Sprint(s)), nil
}

// shadeTextBg applies a background color to the given text
func shadeTextBg(s any, hex string) (string, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "", fmt.Errorf("func 'shadeTextBg': invalid hex color value: %s (must be 6 characters like '0077FF')", hex)
	}

	return fmt.Sprintf(shadingWrapperF, strings.ToUpper(hex), fmt.Sprint(s)), nil
}

// image wraps a placeholder around the given filename for image insertion in the document.
// If widthMM > 0 is provided, the image is scaled to that width in mm preserving aspect ratio.
func image(filename any, width ...int) string {
	s := fmt.Sprint(filename)
	if s == "" || s == "<nil>" {
		return ""
	}
	if len(width) > 0 && width[0] > 0 {
		return fmt.Sprintf("[[IMAGE:%s|W:%d]]", s, width[0])
	}
	return fmt.Sprintf("[[IMAGE:%s]]", s)
}

// replaceImage insert a placeholder around the given filename for image replacement in the document.
func replaceImage(filename any) string {
	s := fmt.Sprint(filename)
	if s == "" || s == "<nil>" {
		return ""
	}
	return fmt.Sprintf("[[REPLACE_IMAGE:%s]]", s)
}

// preserveNewline newlines are treated as `SHIFT + ENTER` input,
// thus keeping the text in the same paragraph.
func preserveNewline(text any) string {
	return strings.ReplaceAll(fmt.Sprint(text), "\n", docxNewlineInject)
}

// breakParagraph newlines are treated as `ENTER` input,
// thus creating a new paragraph for the sequent line.
func breakParagraph(text any) string {
	return strings.ReplaceAll(fmt.Sprint(text), "\n", docxBreakParagraphInject)
}

// shapeBgFillColor replace fillcolor to shapes
func shapeBgFillColor(hex string) (string, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "", fmt.Errorf("func 'shapeBgFillColor': invalid hex color value: %s  (must be 6 characters like '0077FF')", hex)
	}

	return fmt.Sprintf("[[SHAPE_BG_FILL_COLOR:%s]]", strings.ToUpper(hex)), nil
}

// tableCellBgColor replace background color of table cells
func tableCellBgColor(hex string) (string, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "", fmt.Errorf("func 'tableCellBgColor': invalid hex color value: %s  (must be 6 characters like '0077FF')", hex)
	}

	return fmt.Sprintf("[[TABLE_CELL_BG_COLOR:%s]]", strings.ToUpper(hex)), nil
}

// ─── Extra template functions ───────────────────────────────────────────────────

const (
	// HideRowSentinel is emitted by the hideRow template function; the
	// post-processor removes any <w:tr> containing this sentinel.
	HideRowSentinel = "\x00HIDEROW\x00"
	// PageBreakPlaceholder is emitted by the pageBreak template function;
	// the post-processor replaces it with the real OOXML page-break fragment.
	PageBreakPlaceholder = "\x00PAGEBREAK\x00"
	// PageBreakReplacement is the OOXML fragment inserted by the
	// pageBreak post-processor.
	PageBreakReplacement = `<w:r><w:br w:type="page"/></w:r>`
)

// formatNum formats a numeric value with thousands separator (non-breaking
// space, U+00A0) and a configurable decimal separator.
func formatNum(value any, decimals int, decSep string) (string, error) {
	f, err := toFloat64(value)
	if err != nil {
		return "", fmt.Errorf("formatNum: %w", err)
	}
	negative := f < 0
	if negative {
		f = -f
	}
	pow := math.Pow(10, float64(decimals))
	rounded := math.Round(f*pow) / pow
	intPart := int64(rounded)
	fracPart := rounded - float64(intPart)
	intStr := formatThousands(intPart)
	var result string
	if decimals > 0 {
		fracStr := fmt.Sprintf("%.*f", decimals, fracPart)[1:]
		result = intStr + strings.ReplaceAll(fracStr, ".", decSep)
	} else {
		result = intStr
	}
	if negative {
		result = "-" + result
	}
	return result, nil
}

func formatThousands(n int64) string {
	if n < 0 {
		return "-" + formatThousands(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		if i > 0 || rem > 0 {
			b.WriteRune('\u00A0')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func toFloat64(v any) (float64, error) {
	switch x := v.(type) {
	case int:
		return float64(x), nil
	case int32:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case float32:
		return float64(x), nil
	case float64:
		return x, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}

// formatDate formats a time.Time using a standard Go layout string.
func formatDate(t time.Time, layout string) string {
	return t.Format(layout)
}

// formatDateRU formats a time.Time replacing the English month name with
// the Russian genitive form.
func formatDateRU(t time.Time, layout string) string {
	result := t.Format(layout)
	month := t.Format("January")
	if ru, ok := ruMonthsGenitive[month]; ok {
		result = strings.ReplaceAll(result, month, ru)
	}
	return result
}

var ruMonthsGenitive = map[string]string{
	"January": "января", "February": "февраля", "March": "марта",
	"April": "апреля", "May": "мая", "June": "июня",
	"July": "июля", "August": "августа", "September": "сентября",
	"October": "октября", "November": "ноября", "December": "декабря",
}

// hideRowFn emits a sentinel when hide==true for post-processor removal.
func hideRowFn(hide bool) string {
	if hide {
		return HideRowSentinel
	}
	return ""
}

// pageBreakFn returns a placeholder that the post-processor replaces with
// the real OOXML page-break fragment.
func pageBreakFn() string { return PageBreakPlaceholder }

// defaultVal returns fallback when value is the zero value for its type.
func defaultVal(value any, fallback any) any {
	if value == nil {
		return fallback
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.String:
		if rv.Len() == 0 {
			return fallback
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if rv.Int() == 0 {
			return fallback
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if rv.Uint() == 0 {
			return fallback
		}
	case reflect.Float32, reflect.Float64:
		if rv.Float() == 0 {
			return fallback
		}
	case reflect.Bool:
		if !rv.Bool() {
			return fallback
		}
	case reflect.Slice, reflect.Map, reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return fallback
		}
	}
	return value
}

// sumCol sums a named numeric field across a slice.
func sumCol(slice any, field string) (float64, error) {
	return aggregateCol(slice, field, false)
}

// avgCol returns the arithmetic mean of a named numeric field.
func avgCol(slice any, field string) (float64, error) {
	return aggregateCol(slice, field, true)
}

func aggregateCol(slice any, field string, avg bool) (float64, error) {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return 0, fmt.Errorf("sumCol/avgCol: expected slice, got %T", slice)
	}
	n := rv.Len()
	if n == 0 {
		return 0, nil
	}
	var sum float64
	for i := 0; i < n; i++ {
		elem := rv.Index(i)
		for elem.Kind() == reflect.Pointer {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		var fval float64
		switch elem.Kind() {
		case reflect.Map:
			v := elem.MapIndex(reflect.ValueOf(field))
			if !v.IsValid() {
				return 0, fmt.Errorf("sumCol/avgCol: field %q not found in map at index %d", field, i)
			}
			f, err := toFloat64(v.Interface())
			if err != nil {
				return 0, fmt.Errorf("sumCol/avgCol: index %d field %q: %w", i, field, err)
			}
			fval = f
		case reflect.Struct:
			f := elem.FieldByName(field)
			if !f.IsValid() {
				return 0, fmt.Errorf("sumCol/avgCol: field %q not found in struct at index %d", field, i)
			}
			v, err := toFloat64(f.Interface())
			if err != nil {
				return 0, fmt.Errorf("sumCol/avgCol: index %d field %q: %w", i, field, err)
			}
			fval = v
		default:
			return 0, fmt.Errorf("sumCol/avgCol: unsupported element type %s", elem.Kind())
		}
		sum += fval
	}
	if avg {
		return sum / float64(n), nil
	}
	return sum, nil
}

// truncate cuts s to at most maxRunes runes, appending "…" if truncated.
func truncate(s any, maxRunes int) string {
	ss := fmt.Sprint(s)
	runes := []rune(ss)
	if len(runes) <= maxRunes {
		return ss
	}
	if maxRunes <= 0 {
		return ""
	}
	return string(runes[:maxRunes-1]) + "…"
}

// romanNum converts a positive integer to an uppercase Roman numeral.
func romanNum(n int) (string, error) {
	if n < 0 || n > 3999 {
		return "", fmt.Errorf("romanNum: value %d out of range [1, 3999]", n)
	}
	if n == 0 {
		return "", nil
	}
	type pair struct {
		val int
		sym string
	}
	table := []pair{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}
	var sb strings.Builder
	for _, p := range table {
		for n >= p.val {
			sb.WriteString(p.sym)
			n -= p.val
		}
	}
	return sb.String(), nil
}

// padRight pads s with spaces on the right to minLen runes.
func padRight(s any, minLen int) string {
	ss := fmt.Sprint(s)
	n := utf8.RuneCountInString(ss)
	if n >= minLen {
		return ss
	}
	return ss + strings.Repeat(" ", minLen-n)
}

// TemplateFuncs is the global registry of template functions available in all DOCX templates.
var TemplateFuncs = template.FuncMap{
	"list":             list,
	"bold":             bold,
	"italic":           italic,
	"underline":        underline,
	"strike":           strike,
	"fontSize":         fontSize,
	"inlineStyledText": inlineStyledText,
	"styledText":       styledText,
	"color":            color,
	"highlight":        highlight,
	"preserveNewline":  preserveNewline,
	"breakParagraph":   breakParagraph,
	"shadeTextBg":      shadeTextBg,
	"image":            image,
	"replaceImage":     replaceImage,
	"shapeBgFillColor": shapeBgFillColor,
	"tableCellBgColor": tableCellBgColor,
	"formatNum":        formatNum,
	"formatDate":       formatDate,
	"formatDateRU":     formatDateRU,
	"hideRow":          hideRowFn,
	"pageBreak":        pageBreakFn,
	"default":          defaultVal,
	"sumCol":           sumCol,
	"avgCol":           avgCol,
	"truncate":         truncate,
	"romanNum":         romanNum,
	"padRight":         padRight,
}
