package media

import (
	"image"
	"image/color"
	"image/png"
	"bytes"
	"testing"
)

func createTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}
	return buf.Bytes()
}

func TestComputeImageSize_ValidImage(t *testing.T) {
	pngData := createTestPNG(t, 96, 96) // 1x1 inch at 96 DPI
	cx, cy, err := ComputeImageSize(pngData, 6, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cx <= 0 || cy <= 0 {
		t.Errorf("expected positive dimensions, got cx=%d cy=%d", cx, cy)
	}
}

func TestComputeImageSize_ScalesToFit(t *testing.T) {
	pngData := createTestPNG(t, 192, 96) // 2x1 inch at 96 DPI
	cx, cy, err := ComputeImageSize(pngData, 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should scale down to fit 1x1 inch box
	if cx > int(1*emusPerInch) || cy > int(1*emusPerInch) {
		t.Errorf("expected dimensions within 1x1 inch, got cx=%d cy=%d", cx, cy)
	}
}

func TestComputeImageSize_NoUpscale(t *testing.T) {
	pngData := createTestPNG(t, 48, 48) // 0.5x0.5 inch at 96 DPI
	cx, cy, err := ComputeImageSize(pngData, 6, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	small := int(0.5 * emusPerInch)
	// Should NOT upscale (scale > 1 → scale = 1)
	if cx != small || cy != small {
		t.Errorf("expected no upscale to %d, got cx=%d cy=%d", small, cx, cy)
	}
}

func TestComputeImageSize_InvalidData(t *testing.T) {
	_, _, err := ComputeImageSize([]byte("not-an-image"), 6, 4)
	if err == nil {
		t.Fatal("expected error for invalid image data")
	}
}

func TestComputeImageSize_EmptyData(t *testing.T) {
	_, _, err := ComputeImageSize([]byte{}, 6, 4)
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestMediaMap(t *testing.T) {
	m := make(MediaMap)
	m["img.png"] = &Media{Data: []byte("data"), WordFilename: "image/image.png"}
	if len(m) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m))
	}
	if m["img.png"].WordFilename != "image/image.png" {
		t.Errorf("got %q, want %q", m["img.png"].WordFilename, "image/image.png")
	}
}
