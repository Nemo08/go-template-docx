package zio

import (
	goziputils "github.com/JJJJJJack/go-zip-utils"
)

// zipMapSrc provides a FileSource backed by a goziputils.ZipMap.
type zipMapSrc struct {
	zm goziputils.ZipMap
}

func (a *zipMapSrc) ReadFile(name string) ([]byte, bool, error) {
	f, ok := a.zm[name]
	if !ok {
		return nil, false, nil
	}
	data, err := goziputils.ReadZipFileContent(f)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (a *zipMapSrc) Each(fn func(name string) error) error {
	for name := range a.zm {
		if err := fn(name); err != nil {
			return err
		}
	}
	return nil
}

// newZipMapFromBytes creates a ZipMap from zip bytes.
func newZipMapFromBytes(data []byte) (goziputils.ZipMap, error) {
	return goziputils.NewZipMapFromBytes(data)
}

// FromGozip wraps a goziputils.ZipMap as a FileSource.
func FromGozip(zm goziputils.ZipMap) FileSource {
	return &zipMapSrc{zm: zm}
}
