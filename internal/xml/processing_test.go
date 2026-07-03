package xml

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestProcessFileWithHandlers_CopiesWhenNoHandlers(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	src := newTestSrc(map[string]string{"a.txt": "hello"})

	err := processFileWithHandlers(zw, src, "a.txt", nil, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = zw.Close()

	got := readZipEntry(t, buf.Bytes(), "a.txt")
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestProcessFileWithHandlers_AppliesHandler(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	src := newTestSrc(map[string]string{"b.txt": "original"})

	handler := func(s string) (string, error) {
		return strings.ToUpper(s), nil
	}

	err := processFileWithHandlers(zw, src, "b.txt", []Handler{handler}, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = zw.Close()

	got := readZipEntry(t, buf.Bytes(), "b.txt")
	if got != "ORIGINAL" {
		t.Errorf("got %q, want %q", got, "ORIGINAL")
	}
}

func TestProcessFileWithHandlers_ErrorOnMissingFile(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	src := newTestSrc(map[string]string{})

	err := processFileWithHandlers(zw, src, "missing.txt", nil, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestProcessFileWithHandlers_ErrorOnReadFailure(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	src := &errFileSource{}

	err := processFileWithHandlers(zw, src, "any.txt", nil, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProcessedOutput_NoProcessors(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("x.txt")
	_, _ = fw.Write([]byte("data"))
	_ = zw.Close()

	err := ProcessedOutput(nil, &buf, "pre")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := readZipEntry(t, buf.Bytes(), "x.txt")
	if got != "data" {
		t.Errorf("got %q, want %q", got, "data")
	}
}

func TestProcessedOutput_AppliesProcessor(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("a.xml")
	_, _ = fw.Write([]byte("hello"))
	_ = zw.Close()

	proc := HandlersMap{
		"a.xml": {func(s string) (string, error) {
			return strings.ToUpper(s), nil
		}},
	}

	err := ProcessedOutput([]HandlersMap{proc}, &buf, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := readZipEntry(t, buf.Bytes(), "a.xml")
	if got != "HELLO" {
		t.Errorf("got %q, want %q", got, "HELLO")
	}
}

func TestProcessedOutput_ErrorOnProcessorFailure(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("err.xml")
	_, _ = fw.Write([]byte("data"))
	_ = zw.Close()

	proc := HandlersMap{
		"err.xml": {func(s string) (string, error) {
			return "", assertAnError{}
		}},
	}

	err := ProcessedOutput([]HandlersMap{proc}, &buf, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProcessedOutput_InvalidZip(t *testing.T) {
	buf := bytes.NewBuffer([]byte("not-a-zip"))
	err := ProcessedOutput([]HandlersMap{{}}, buf, "pre")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProcessedOutput_WildcardKey(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("a.xml")
	_, _ = fw.Write([]byte("before"))
	// b.txt не xml/rels — wildcard не применяется, должно остаться нетронутым.
	fw2, _ := zw.Create("b.txt")
	_, _ = fw2.Write([]byte("untouched"))
	fw3, _ := zw.Create("c.rels")
	_, _ = fw3.Write([]byte("links"))
	_ = zw.Close()

	proc := HandlersMap{
		"*": {func(s string) (string, error) {
			return strings.ToUpper(s), nil
		}},
	}

	err := ProcessedOutput([]HandlersMap{proc}, &buf, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotA := readZipEntry(t, buf.Bytes(), "a.xml")
	if gotA != "BEFORE" {
		t.Errorf("a.xml got %q, want %q", gotA, "BEFORE")
	}
	gotB := readZipEntry(t, buf.Bytes(), "b.txt")
	if gotB != "untouched" {
		t.Errorf("b.txt (non-text) got %q, want %q", gotB, "untouched")
	}
	gotC := readZipEntry(t, buf.Bytes(), "c.rels")
	if gotC != "LINKS" {
		t.Errorf("c.rels got %q, want %q", gotC, "LINKS")
	}
}

func TestProcessedOutput_WildcardDoesNotCorruptBinary(t *testing.T) {
	// Бинарные данные: PNG-заголовок + произвольные байты, включая невалидный UTF-8.
	binaryData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG magic
		0xFF, 0xFE, 0x00, 0x01,                             // невалидный UTF-8
		0x7B, 0x7B, 0x58, 0x2E, 0x59, 0x7D, 0x7D,          // случайный {{X.Y}}
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("word/media/image1.png")
	_, _ = fw.Write(binaryData)
	fw2, _ := zw.Create("word/document.xml")
	_, _ = fw2.Write([]byte("<p>data</p>"))
	_ = zw.Close()

	original := make([]byte, len(binaryData))
	copy(original, binaryData)
	// создаём заглушку-хендлер: если он применится к бинарному файлу —
	// будет видно по содержимому
	double := func(s string) (string, error) { return s + s, nil }

	proc := HandlersMap{
		"*": {double},
	}

	err := ProcessedOutput([]HandlersMap{proc}, &buf, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotPNG := readZipEntryBytes(t, buf.Bytes(), "word/media/image1.png")
	if !bytes.Equal(gotPNG, original) {
		t.Errorf("binary file corrupted: len before=%d, after=%d", len(original), len(gotPNG))
	}

	gotXML := readZipEntry(t, buf.Bytes(), "word/document.xml")
	if gotXML != "<p>data</p><p>data</p>" {
		t.Errorf("document.xml not processed by wildcard: got %q", gotXML)
	}
}

func TestProcessedOutput_WildcardDoesNotOverrideExactKey(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("a.xml")
	_, _ = fw.Write([]byte("data"))
	_ = zw.Close()

	proc := HandlersMap{
		"*": {func(s string) (string, error) {
			return "wildcard", nil
		}},
		"a.xml": {func(s string) (string, error) {
			return "exact", nil
		}},
	}

	err := ProcessedOutput([]HandlersMap{proc}, &buf, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := readZipEntry(t, buf.Bytes(), "a.xml")
	if got != "exact" {
		t.Errorf("got %q, want %q", got, "exact")
	}
}

func TestProcessedOutput_MultipleMaps(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("f.txt")
	_, _ = fw.Write([]byte("abc"))
	_ = zw.Close()

	double := func(s string) (string, error) {
		return s + s, nil
	}

	first := HandlersMap{"f.txt": {double}}
	second := HandlersMap{"f.txt": {double}}

	err := ProcessedOutput([]HandlersMap{first, second}, &buf, "multi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := readZipEntry(t, buf.Bytes(), "f.txt")
	if got != "abcabcabcabc" {
		t.Errorf("got %q, want %q", got, "abcabcabcabc")
	}
}

type assertAnError struct{}

func (assertAnError) Error() string { return "mock error" }

type errFileSource struct{}

func (e *errFileSource) ReadFile(name string) ([]byte, bool, error) {
	return nil, false, assertAnError{}
}
func (e *errFileSource) Each(fn func(name string) error) error {
	return fn("any.txt")
}

func newTestSrc(files map[string]string) *testZipSrc {
	return &testZipSrc{files: files}
}

type testZipSrc struct {
	files map[string]string
}

func (s *testZipSrc) ReadFile(name string) ([]byte, bool, error) {
	data, ok := s.files[name]
	if !ok {
		return nil, false, nil
	}
	return []byte(data), true, nil
}

func (s *testZipSrc) Each(fn func(name string) error) error {
	for name := range s.files {
		if err := fn(name); err != nil {
			return err
		}
	}
	return nil
}

func readZipEntryBytes(t *testing.T, data []byte, name string) []byte {
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
			return buf.Bytes()
		}
	}
	t.Fatalf("entry %q not found in zip", name)
	return nil
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
