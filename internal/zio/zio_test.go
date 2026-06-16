package zio

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestNewFromBytes_ValidZip(t *testing.T) {
	data := createTestZip(t, map[string]string{"a.txt": "hello"})
	src, err := NewFromBytes(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src == nil {
		t.Fatal("expected non-nil source")
	}
}

func TestNewFromBytes_InvalidZip(t *testing.T) {
	_, err := NewFromBytes([]byte("not-a-zip"))
	if err == nil {
		t.Fatal("expected error for invalid zip")
	}
}

func TestNewFromBytes_EmptyData(t *testing.T) {
	_, err := NewFromBytes([]byte{})
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestFileSource_ReadFile(t *testing.T) {
	src := mustSrc(t, map[string]string{"x.txt": "content"})
	data, found, err := src.ReadFile("x.txt")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if string(data) != "content" {
		t.Errorf("got %q, want %q", string(data), "content")
	}
}

func TestFileSource_ReadFile_NotFound(t *testing.T) {
	src := mustSrc(t, map[string]string{"a.txt": "data"})
	_, found, err := src.ReadFile("nonexistent.txt")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if found {
		t.Fatal("expected found=false")
	}
}

func TestFileSource_Each(t *testing.T) {
	src := mustSrc(t, map[string]string{"a.txt": "1", "b.txt": "2"})
	var names []string
	err := src.Each(func(name string) error {
		names = append(names, name)
		return nil
	})
	if err != nil {
		t.Fatalf("Each error: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 files, got %d", len(names))
	}
}

func TestFileSource_Each_StopsOnError(t *testing.T) {
	src := mustSrc(t, map[string]string{"a.txt": "1", "b.txt": "2"})
	err := src.Each(func(name string) error {
		return assertAnError{}
	})
	if err == nil {
		t.Fatal("expected error from Each")
	}
}

func TestNewZipWriter_WritesToBuffer(t *testing.T) {
	var buf bytes.Buffer
	zw := NewZipWriter(&buf)
	fw, err := zw.Create("test.txt")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	_, _ = fw.Write([]byte("data"))
	err = zw.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected non-empty buffer after close")
	}
}

func TestCopyToZip(t *testing.T) {
	var outBuf bytes.Buffer
	dst := NewZipWriter(&outBuf)
	src := mustSrc(t, map[string]string{"f.txt": "preserve"})

	err := CopyToZip(dst, src, "f.txt")
	if err != nil {
		t.Fatalf("CopyToZip error: %v", err)
	}
	_ = dst.Close()

	got := readZipEntry(t, outBuf.Bytes(), "f.txt")
	if got != "preserve" {
		t.Errorf("got %q, want %q", got, "preserve")
	}
}

func TestCopyToZip_SourceNotFound(t *testing.T) {
	var outBuf bytes.Buffer
	dst := NewZipWriter(&outBuf)
	src := mustSrc(t, map[string]string{})

	err := CopyToZip(dst, src, "missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestRewriteToZip(t *testing.T) {
	var outBuf bytes.Buffer
	dst := NewZipWriter(&outBuf)

	err := RewriteToZip(dst, nil, "new.txt", []byte("replaced"))
	if err != nil {
		t.Fatalf("RewriteToZip error: %v", err)
	}
	_ = dst.Close()

	got := readZipEntry(t, outBuf.Bytes(), "new.txt")
	if got != "replaced" {
		t.Errorf("got %q, want %q", got, "replaced")
	}
}

type assertAnError struct{}

func (assertAnError) Error() string { return "mock error" }

func createTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		fw, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create %q in test zip: %v", name, err)
		}
		_, err = fw.Write([]byte(content))
		if err != nil {
			t.Fatalf("failed to write %q in test zip: %v", name, err)
		}
	}
	_ = zw.Close()
	return buf.Bytes()
}

func mustSrc(t *testing.T, files map[string]string) FileSource {
	t.Helper()
	data := createTestZip(t, files)
	src, err := NewFromBytes(data)
	if err != nil {
		t.Fatalf("NewFromBytes error: %v", err)
	}
	return src
}

func readZipEntry(t *testing.T, data []byte, name string) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	for _, f := range r.File {
		if f.Name == name {
			rc, _ := f.Open()
			defer func() { _ = rc.Close() }()
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(rc)
			return buf.String()
		}
	}
	t.Fatalf("entry %q not found in zip", name)
	return ""
}
