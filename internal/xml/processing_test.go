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

func readZipEntry(t *testing.T, data []byte, name string) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	for _, f := range r.File {
		if f.Name == name {
			rc, _ := f.Open()
			defer rc.Close()
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(rc)
			return buf.String()
		}
	}
	t.Fatalf("entry %q not found in zip", name)
	return ""
}
