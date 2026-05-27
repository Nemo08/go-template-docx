package docx

import (
	"bytes"
	"fmt"
	stdimage "image"
	_ "image/jpeg" // register JPEG decoder for image.Decode
	_ "image/png"  // register PNG decoder for image.Decode
	"math"
)

// ImageMediaType is a media type constant for images.
const (
	ImageMediaType = iota + 1
)

const (
	emusPerInch = 914400.0
	defaultDPI  = 96.0
)

// Media holds binary data and the assigned Word-compatible filename for an embedded image.
type Media struct {
	Data         []byte
	WordFilename string
}

// MediaMap maps original filenames to their Media entries.
type MediaMap map[string]*Media

// MediaRel describes a single media file relationship.
type MediaRel struct {
	Type   uint
	RefID  string
	Source string
}

func (d *documentMeta) computeDocxImageSize(imageData []byte) (int, int, error) {
	cfg, _, err := stdimage.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return 0, 0, err
	}

	if cfg.Width == 0 || cfg.Height == 0 {
		return 0, 0, fmt.Errorf("invalid image dimensions")
	}

	widthInches := float64(cfg.Width) / defaultDPI
	heightInches := float64(cfg.Height) / defaultDPI

	scale := math.Min(d.maxWidthInches/widthInches, d.maxHeightInches/heightInches)
	if scale > 1 {
		scale = 1
	}

	newWidth := widthInches * scale
	newHeight := heightInches * scale

	cx := int(newWidth * emusPerInch)
	cy := int(newHeight * emusPerInch)
	return cx, cy, nil
}
