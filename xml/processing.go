package xml

import (
	"archive/zip"
	"bytes"
	"fmt"

	"github.com/JJJJJJack/go-template-docx/internal/zio"
)

// Handler takes the content of a file and returns the modified
// content that will replace it.
type Handler func(content string) (string, error)

// HandlersMap maps filenames to a [Handler] functions chain. Each file content
// will be modified sequentially by each function in the []Handler slice.
// The final output will overwrite the original.
type HandlersMap map[string][]Handler

func ProcessedOutput(filesProcessorsMaps []HandlersMap, outputBuffer *bytes.Buffer, preOrPost string) error {
	for _, filesPostProcessorsMap := range filesProcessorsMaps {
		zipBytes := append([]byte(nil), outputBuffer.Bytes()...)

		src, err := zio.NewFromBytes(zipBytes)
		if err != nil {
			return fmt.Errorf("unable to create zip source during %s-processing: %w", preOrPost, err)
		}

		outputBuffer.Reset()
		outputZipWriter := zip.NewWriter(outputBuffer)

		err = src.Each(func(filename string) error {
			processors := filesPostProcessorsMap[filename]
			if len(processors) == 0 {
				if err := zio.CopyToZip(outputZipWriter, src, filename); err != nil {
					return fmt.Errorf("unable to copy original file '%s' during %s-processing: %w", filename, preOrPost, err)
				}
				return nil
			}

			fileContent, found, err := src.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("unable to read file '%s' during %s-processing: %w", filename, preOrPost, err)
			}
			if !found {
				return fmt.Errorf("file '%s' not found during %s-processing", filename, preOrPost)
			}

			xmlOutput := string(fileContent)
			for _, processor := range processors {
				xmlOutput, err = processor(xmlOutput)
				if err != nil {
					return fmt.Errorf("error %s processing file '%s': %w", preOrPost, filename, err)
				}
			}

			if err := zio.RewriteToZip(outputZipWriter, src, filename, []byte(xmlOutput)); err != nil {
				return fmt.Errorf("unable to rewrite %s-processed file '%s': %w", preOrPost, filename, err)
			}
			return nil
		})
		if err != nil {
			return err
		}

		if err := outputZipWriter.Close(); err != nil {
			return fmt.Errorf("unable to close zip writer after %s-processing: %w", preOrPost, err)
		}
	}

	return nil
}
