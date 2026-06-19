package media

import (
	"bytes"
	"fmt"
	stdimage "image"
	_ "image/jpeg"
	_ "image/png"
	"math"
)

const (
	emusPerInch = 914400.0
	emusPerMM   = 36000.0 // 1 мм = 36000 EMU (914400 / 25.4)
	defaultDPI  = 96.0
)

type Media struct {
	Data         []byte
	WordFilename string
}

type MediaMap map[string]*Media

func ComputeImageSize(imageData []byte, maxWidthInches, maxHeightInches float64) (int, int, error) {
	cfg, _, err := stdimage.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return 0, 0, err
	}

	if cfg.Width == 0 || cfg.Height == 0 {
		return 0, 0, fmt.Errorf("invalid image dimensions")
	}

	widthInches := float64(cfg.Width) / defaultDPI
	heightInches := float64(cfg.Height) / defaultDPI

	scale := math.Min(maxWidthInches/widthInches, maxHeightInches/heightInches)
	if scale > 1 {
		scale = 1
	}

	newWidth := widthInches * scale
	newHeight := heightInches * scale

	cx := int(newWidth * emusPerInch)
	cy := int(newHeight * emusPerInch)
	return cx, cy, nil
}

// ComputeImageSizeWithWidth вычисляет EMU-размеры изображения, масштабируя
// его до заданной ширины widthMM в миллиметрах с сохранением пропорций.
func ComputeImageSizeWithWidth(imageData []byte, widthMM int) (cx, cy int, err error) {
	origW, origH, err := ImageDimensions(imageData)
	if err != nil {
		return 0, 0, err
	}
	cx = int(float64(widthMM) * emusPerMM)
	cy = cx * origH / origW
	return cx, cy, nil
}

// ImageDimensions возвращает ширину и высоту изображения в пикселях.
func ImageDimensions(imageData []byte) (width, height int, err error) {
	cfg, _, err := stdimage.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return 0, 0, err
	}
	if cfg.Width == 0 || cfg.Height == 0 {
		return 0, 0, fmt.Errorf("invalid image dimensions")
	}
	return cfg.Width, cfg.Height, nil
}
