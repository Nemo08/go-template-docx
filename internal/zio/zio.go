package zio

import (
	"archive/zip"
	"fmt"
)

// FileSource provides content-based access to files within an archive.
type FileSource interface {
	ReadFile(name string) (data []byte, found bool, err error)
	Each(fn func(name string) error) error
}

// NewFromBytes creates a FileSource from zip bytes.
func NewFromBytes(data []byte) (FileSource, error) {
	zm, err := newZipMapFromBytes(data)
	if err != nil {
		return nil, err
	}
	return &zipMapSrc{zm: zm}, nil
}

// CopyToZip copies a file from the source archive into the zip writer.
func CopyToZip(w *zip.Writer, src FileSource, name string) error {
	data, found, err := src.ReadFile(name)
	if err != nil {
		return fmt.Errorf("unable to read file '%s': %w", name, err)
	}
	if !found {
		return fmt.Errorf("file '%s' not found in source archive", name)
	}
	fw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("unable to create '%s' in zip: %w", name, err)
	}
	if _, err := fw.Write(data); err != nil {
		return fmt.Errorf("unable to write '%s' to zip: %w", name, err)
	}
	return nil
}

// RewriteToZip replaces a file in the zip writer with new content.
func RewriteToZip(w *zip.Writer, src FileSource, name string, content []byte) error {
	fw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("unable to create '%s' in zip: %w", name, err)
	}
	if _, err := fw.Write(content); err != nil {
		return fmt.Errorf("unable to write '%s' to zip: %w", name, err)
	}
	return nil
}
