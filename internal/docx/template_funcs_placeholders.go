package docx

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/JJJJJJack/go-template-docx/internal/docx/media"
	"github.com/JJJJJJack/go-template-docx/internal/docx/rels"
)

var (
	reImagePlaceholder = regexp.MustCompile(`\[\[IMAGE:.*?\]\]`)
	imageTemplate      = template.Must(template.New("image-template").Parse(imageTemplateXML))
)

func (d *documentMeta) applyImages(srcXML string) (string, []rels.MediaRel, error) {
	mediaRels := []rels.MediaRel{}

	xmlBlocks := reImagePlaceholder.FindAllString(srcXML, -1)
	for _, xmlBlock := range xmlBlocks {
		filename := strings.TrimPrefix(xmlBlock, "[[IMAGE:")
		filename = strings.TrimSuffix(filename, "]]")

		buffer := bytes.Buffer{}
		docPrID, err := d.RandUniqueDocPrID()
		if err != nil {
			return srcXML, mediaRels, fmt.Errorf("unable to get unique docPrID: %w", err)
		}

		rid := d.NextRId()
		rID := fmt.Sprintf("rID%d", rid)

		v, ok := d.mediaMap[filename]
		if !ok {
			srcXML = strings.ReplaceAll(srcXML, xmlBlock, "")
			continue
		}

		cx, cy, err := media.ComputeImageSize(v.Data, d.maxWidthInches, d.maxHeightInches)
		if err != nil {
			return srcXML, mediaRels, fmt.Errorf("unable to compute image size for '%s': %w", filename, err)
		}

		pictureN := fmt.Sprintf("Picture %d", d.NextPictureNumber())
		err = imageTemplate.Execute(&buffer, XMLImageData{
			DocPrID: docPrID,
			Name:    pictureN,
			RefID:   rID,
			Cx:      cx,
			Cy:      cy,
		})
		if err != nil {
			return srcXML, mediaRels, fmt.Errorf("unable to execute image template: %w", err)
		}

		mediaRels = append(mediaRels, rels.MediaRel{
			Type:   rels.ImageMediaType,
			RefID:  rID,
			Source: path.Join("media", v.WordFilename),
		})

		srcXML = strings.ReplaceAll(srcXML, xmlBlock, buffer.String())
	}

	return srcXML, mediaRels, nil
}

var (
	reDrawingBlock = regexp.MustCompile(`(?s)<w:drawing>.*?</w:drawing>`)
	reReplaceImage = regexp.MustCompile(`\[\[REPLACE_IMAGE:([^\]]+)\]\]`)
	reBlipEmbed    = regexp.MustCompile(`(<a:blip\s+r:embed=")[^"]*(")`)
	reHexColor     = regexp.MustCompile(`\[\[TABLE_CELL_BG_COLOR:#?([0-9A-Fa-f]{6})\]\]`)
	reShdTag       = regexp.MustCompile(`(?i)<w:shd[^>]*?/>`)
	reFillAttr     = regexp.MustCompile(`(?i)(<w:shd[^>]*? w:fill=")[^"]*(")`)
	reShdOpenTag   = regexp.MustCompile(`(?i)(<w:shd)`)
	reTcBlock      = regexp.MustCompile(`(?s)<w:tc>.*?</w:tc>`)
)

// replaceImages looks for [[REPLACE_IMAGE:filename.ext]] placeholders inside <w:drawing>...</w:drawing> blocks
// remove the placeholder and replaces the image reference inside the block with the given image's rID.
func (d *documentMeta) replaceImages(srcXML string) (string, []rels.MediaRel) {

	mediaRels := []rels.MediaRel{}
	result := srcXML
	if d.mediaMap != nil {
		result = reDrawingBlock.ReplaceAllStringFunc(srcXML, func(block string) string {
			pm := reReplaceImage.FindStringSubmatch(block)
			if len(pm) < 2 {
				return block
			}
			filename := pm[1]

			block = reReplaceImage.ReplaceAllString(block, "")

			rid := d.NextRId()
			rID := fmt.Sprintf("rID%d", rid)

			// Нормализуем путь перед поиском в mediaMap:
			// replaceImage(.field) передаёт значение поля как есть (может быть UNC-путь
			// с обратными слэшами), а mediaMap хранит ключи по filepath.Base().
			// Пробуем сначала как есть, затем по Base() нормализованного пути.
			media, ok := d.mediaMap[filename]
			if !ok {
				normFilename := path.Base(strings.ReplaceAll(filename, `\`, `/`))
				media, ok = d.mediaMap[normFilename]
			}
			if !ok {
				return block
			}

			mediaRels = append(mediaRels, rels.MediaRel{
				Type:   rels.ImageMediaType,
				RefID:  rID,
				Source: path.Join("media", media.WordFilename),
			})

			block = reBlipEmbed.ReplaceAllString(block, "${1}"+rID+"${2}")

			return block
		})

	}

	return result, mediaRels
}

// adjustBrightnessHex lightens or darkens a hex color by factor (0..1).
// If lighten is true, moves towards 255; else towards 0.
func adjustBrightnessHex(hex string, factor float64, lighten bool) string {
	if len(hex) != 6 {
		return hex
	}
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	rf, gf, bf := float64(r), float64(g), float64(b)
	if lighten {
		rf += (255.0 - rf) * factor
		gf += (255.0 - gf) * factor
		bf += (255.0 - bf) * factor
	} else {
		rf *= (1.0 - factor)
		gf *= (1.0 - factor)
		bf *= (1.0 - factor)
	}
	clamp := func(x float64) uint8 {
		if x < 0 {
			x = 0
		}
		if x > 255 {
			x = 255
		}
		return uint8(x + 0.5)
	}
	return fmt.Sprintf("%02x%02x%02x", clamp(rf), clamp(gf), clamp(bf))
}

const withoutSolidFill = `</a:prstGeom></wps:spPr>`

var (
	wpsSpPrRe            = regexp.MustCompile(`(?is)<wps:spPr>.*?</wps:spPr>`)
	placeholderRe        = regexp.MustCompile(`\[\[SHAPE_BG_FILL_COLOR:#?([0-9A-Fa-f]{6})\]\]`)
	mcAlternateContentRe = regexp.MustCompile(`(?s)<mc:AlternateContent>.*?</mc:AlternateContent>`)
	aGradFillRe          = regexp.MustCompile(`(?is)<a:gradFill\b[\s\S]*?</a:gradFill>`)
	aSchemeToSrgb        = regexp.MustCompile(`(?is)<a:schemeClr\b[^>]*>([\s\S]*?)</a:schemeClr>`)
	aSrgbValAttrRe       = regexp.MustCompile(`(?is)(<a:srgbClr\b[^>]*\bval=")[^"]*(")`)
	aSolidFillRe         = regexp.MustCompile(`(?is)<a:solidFill\b[\s\S]*?</a:solidFill>`)
	aPrstGeomCloseRe     = regexp.MustCompile(`(?is)</a:prstGeom>\s*`)
	aShadingRe           = regexp.MustCompile(`(?is)<a:(?:lumMod|lumOff|tint|shade|satMod)\b[^>]*/>`)
	wpsStyleFixRe        = regexp.MustCompile(`(?is)<wps:style>.*?</wps:style>`)
	fillColorRe          = regexp.MustCompile(`(?i)\bfillcolor="[^"]*"`)
	aSrgbClrRe           = regexp.MustCompile(`(?is)<a:srgbClr([^>]*)>\s*(</a:(?:fillRef|effectRef|fontRef)>)`)
	vFillRe              = regexp.MustCompile(`(?is)<v:fill\b[^>]*/>`)
)

// ReplaceAllShapeBgColors finds shapes that contain the [[SHAPE_BG_COLOR:RRGGBB]]/[[SHAPE_BG_COLOR:#RRGGBB]]
// placeholder and uses its value to replace the fillcolor attribute of the shape
// TODO: replace with proper XML parsing
func (d *documentMeta) applyShapesBgFillColor(srcXML string) string {
	return mcAlternateContentRe.ReplaceAllStringFunc(srcXML, func(block string) string {
		placeholders := placeholderRe.FindAllStringSubmatch(block, -1)
		if len(placeholders) == 0 {
			return block
		}

		for _, pm := range placeholders {
			hex := strings.ToLower(pm[1])

			block = wpsSpPrRe.ReplaceAllStringFunc(block, func(sppr string) string {
				sppr = aGradFillRe.ReplaceAllStringFunc(sppr, func(grad string) string {
					grad = aSchemeToSrgb.ReplaceAllString(grad, `<a:srgbClr val="`+hex+`">$1</a:srgbClr>`)
					grad = aSrgbValAttrRe.ReplaceAllString(grad, `${1}`+hex+`${2}`)
					return grad
				})

				if !strings.Contains(sppr, "<a:gradFill") {
					sppr = applySolidFill(sppr, hex)
				}

				return sppr
			})

			block = wpsStyleFixRe.ReplaceAllStringFunc(block, func(style string) string {
				style = aSrgbClrRe.ReplaceAllString(style, `<a:srgbClr$1/>$2`)
				return style
			})

			block = fillColorRe.ReplaceAllStringFunc(block, func(_ string) string {
				return `fillcolor="#` + hex + `"`
			})

			block = vFillRe.ReplaceAllStringFunc(block, func(tag string) string {
				return applyVmlGradientFill(tag, hex)
			})
		}

		block = placeholderRe.ReplaceAllString(block, "")

		return block
	})
}

const withoutShading = `></w:tcPr>`
const withShading = `><w:shd w:val="clear" w:color="auto" w:fill="FFFFFF" /></w:tcPr>`

// replaceTableCellBgColors is used to apply the hex color found in the
// [[TABLE_CELL_BG_COLOR:RRGGBB]]/[[TABLE_CELL_BG_COLOR:#RRGGBB]] as the background color of the table cell
func (d *documentMeta) replaceTableCellBgColors(srcXML string) string {
	output := reTcBlock.ReplaceAllStringFunc(srcXML, func(block string) string {
		hexMatch := reHexColor.FindStringSubmatch(block)
		if len(hexMatch) < 2 {
			return block
		}
		hex := hexMatch[1]

		if !reShdTag.MatchString(block) {
			block = strings.Replace(block, withoutShading, withShading, 1)
		}

		if reFillAttr.MatchString(block) {
			block = reFillAttr.ReplaceAllString(block, `${1}`+hex+`${2}`)
		} else {
			block = reShdOpenTag.ReplaceAllString(block, `${1} w:fill="`+hex+`"`)
		}

		block = reHexColor.ReplaceAllString(block, ``)
		block = strings.ReplaceAll(block, "<w:r><w:t></w:t></w:r>", "")

		return block
	})

	return output
}

func applySolidFill(sppr, hex string) string {
	replacement := `<a:solidFill><a:srgbClr val="` + hex + `"/></a:solidFill>`
	sppr = aSolidFillRe.ReplaceAllString(sppr, replacement)
	if !strings.Contains(sppr, `<a:solidFill>`) {
		if loc := aPrstGeomCloseRe.FindStringIndex(sppr); loc != nil {
			sppr = sppr[:loc[1]] + replacement + sppr[loc[1]:]
		} else {
			sppr = strings.Replace(sppr, withoutSolidFill, `</a:prstGeom>`+replacement+`</wps:spPr>`, 1)
		}
	}
	sppr = aShadingRe.ReplaceAllString(sppr, "")
	return sppr
}

func applyVmlGradientFill(tag, hex string) string {
	extractAttr := func(name string) string {
		marker := name + `="`
		idx := strings.Index(tag, marker)
		if idx < 0 {
			return ""
		}
		start := idx + len(marker)
		end := strings.IndexByte(tag[start:], '"')
		if end < 0 {
			return ""
		}
		return tag[start : start+end]
	}
	rotate := extractAttr("rotate")
	angle := extractAttr("angle")
	focus := extractAttr("focus")

	base := strings.ToLower(strings.TrimPrefix(hex, "#"))
	darker := adjustBrightnessHex(base, 0.25, false)
	lighter := adjustBrightnessHex(base, 0.35, true)

	attrs := []string{`type="gradient"`, `colors="0 #` + darker + `;0.5 #` + base + `;1 #` + lighter + `"`, `color2="#` + lighter + `"`}
	if rotate != "" {
		attrs = append(attrs, `rotate="`+rotate+`"`)
	}
	if angle != "" {
		attrs = append(attrs, `angle="`+angle+`"`)
	}
	if focus != "" {
		attrs = append(attrs, `focus="`+focus+`"`)
	}
	return `<v:fill ` + strings.Join(attrs, " ") + `/>`
}
