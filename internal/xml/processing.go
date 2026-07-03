// Package xml provides the pre/post-processing framework for XML content
// inside DOCX archives.
package xml

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JJJJJJack/go-template-docx/internal/zio"
)

// isTextPart возвращает true для частей docx-архива, содержимое которых
// представляет собой текст/XML, а не бинарные данные (изображения, шрифты,
// встроенные объекты, OLE и т.п.). Wildcard-ключ "*" в HandlersMap
// применяется ТОЛЬКО к таким частям, чтобы избежать повреждения бинарных
// данных при конвертации []byte → string → []byte.
func isTextPart(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".xml" || ext == ".rels"
}

// Handler takes the content of a file and returns the modified
// content that will replace it.
type Handler func(content string) (string, error)

// HandlersMap maps filenames to a [Handler] functions chain. Each file content
// will be modified sequentially by each function in the []Handler slice.
// The final output will overwrite the original.
type HandlersMap map[string][]Handler

// processFileWithHandlers reads a file from src, applies the handler chain,
// and writes the result to outputZipWriter. If no handlers are provided, the
// file is copied as-is.
func processFileWithHandlers(outputZipWriter zio.ZipWriter, src zio.FileSource, filename string, processors []Handler, stage string) error {
	if len(processors) == 0 {
		return zio.CopyToZip(outputZipWriter, src, filename)
	}

	fileContent, found, err := src.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("unable to read file '%s' during %s-processing: %w", filename, stage, err)
	}
	if !found {
		return fmt.Errorf("file '%s' not found during %s-processing", filename, stage)
	}

	xmlOutput := string(fileContent)
	for _, processor := range processors {
		xmlOutput, err = processor(xmlOutput)
		if err != nil {
			return fmt.Errorf("error %s processing file '%s': %w", stage, filename, err)
		}
	}

	return zio.RewriteToZip(outputZipWriter, src, filename, []byte(xmlOutput))
}

// ProcessedOutput applies pre/post-processor handlers to all files in a DOCX zip buffer.
// Каждая HandlersMap в слайсе обрабатывается независимо и последовательно:
// для каждого файла сначала ищется точный ключ (filename), и только если
// его нет — применяется wildcard-ключ "*" (для .xml/.rels частей).
// Точный ключ и wildcard в рамках одной HandlersMap — взаимоисключающий
// выбор (exact побеждает). Но если точный ключ лежит в одной карте,
// а wildcard — в другой (следующей в слайсе), они оба применяются
// к одному и тому же файлу. Это ожидаемое поведение, а не баг.
func ProcessedOutput(filesProcessorsMaps []HandlersMap, outputBuffer *bytes.Buffer, preOrPost string) error {
	for _, filesPostProcessorsMap := range filesProcessorsMaps {
		zipBytes := append([]byte(nil), outputBuffer.Bytes()...)

		src, err := zio.NewFromBytes(zipBytes)
		if err != nil {
			return fmt.Errorf("unable to create zip source during %s-processing: %w", preOrPost, err)
		}

		outputBuffer.Reset()
		outputZipWriter := zio.NewZipWriter(outputBuffer)

		err = src.Each(func(filename string) error {
			handlers := filesPostProcessorsMap[filename]
			if handlers == nil && isTextPart(filename) {
				handlers = filesPostProcessorsMap["*"]
			}
			return processFileWithHandlers(outputZipWriter, src, filename, handlers, preOrPost)
		})
		if err != nil {
			_ = outputZipWriter.Close()
			return err
		}

		if err := outputZipWriter.Close(); err != nil {
			return fmt.Errorf("unable to close zip writer after %s-processing: %w", preOrPost, err)
		}
	}

	return nil
}
