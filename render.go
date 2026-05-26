package gotemplatedocx

import (
	"text/template"

	"github.com/JJJJJJack/go-template-docx/xml"
)

// RenderOption configures the DOCX template rendering.
type RenderOption func(*docxTemplate)

// WithFuncs adds custom template functions for v2 Render API.
func WithFuncs(fns template.FuncMap) RenderOption {
	return func(t *docxTemplate) {
		t.AddTemplateFuncs(fns)
	}
}

// WithImage adds a media file to embed in the output DOCX.
func WithImage(filename string, data []byte) RenderOption {
	return func(t *docxTemplate) {
		t.Media(filename, data)
	}
}

// WithPreProcessors adds XML pre-processing steps.
func WithPreProcessors(m ...xml.HandlersMap) RenderOption {
	return func(t *docxTemplate) {
		t.AddPreProcessors(m...)
	}
}

// WithPostProcessors adds XML post-processing steps.
func WithPostProcessors(m ...xml.HandlersMap) RenderOption {
	return func(t *docxTemplate) {
		t.AddPostProcessors(m...)
	}
}

// WithRemoveEmptyTableRows enables or disables empty table row removal (default: true).
func WithRemoveEmptyTableRows(v bool) RenderOption {
	return func(t *docxTemplate) {
		t.removeEmptyTableRows = v
	}
}

// WithRemoveRangeRows enables or disables range directive row removal (default: false).
func WithRemoveRangeRows(v bool) RenderOption {
	return func(t *docxTemplate) {
		t.removeRangeRows = v
	}
}

// WithIgnoreMissingKey enables or disables missing key ignoring (default: false).
func WithIgnoreMissingKey(v bool) RenderOption {
	return func(t *docxTemplate) {
		t.ignoreMissingKey = v
	}
}

// Render applies the DOCX template with the provided data and returns the result bytes.
// This is the main entry point of the v2 functional-options API.
//
// When the number of options is dynamic (e.g. images loaded in a loop),
// build a []RenderOption slice and spread it:
//
//	var opts []gotemplatedocx.RenderOption
//	for name, data := range images {
//	    opts = append(opts, gotemplatedocx.WithImage(name, data))
//	}
//	result, err := gotemplatedocx.Render(docxBytes, data, opts...)
//
// Or use NewRenderBuilder for a fluent builder pattern.
func Render(docxBytes []byte, data any, opts ...RenderOption) ([]byte, error) {
	tpl, err := NewDocxTemplateFromBytes(docxBytes)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(tpl)
	}
	if err := tpl.Apply(data); err != nil {
		return nil, err
	}
	return tpl.Bytes(), nil
}

// RenderBuilder builds a DOCX render step by step, useful when options
// are added dynamically (e.g. images in a loop).
type RenderBuilder struct {
	tpl *docxTemplate
}

// NewRenderBuilder creates a RenderBuilder from DOCX bytes.
func NewRenderBuilder(docxBytes []byte) (*RenderBuilder, error) {
	tpl, err := NewDocxTemplateFromBytes(docxBytes)
	if err != nil {
		return nil, err
	}
	return &RenderBuilder{tpl: tpl}, nil
}

// WithFuncs adds custom template functions.
func (b *RenderBuilder) WithFuncs(fns template.FuncMap) *RenderBuilder {
	b.tpl.AddTemplateFuncs(fns)
	return b
}

// WithImage adds a media file to embed.
func (b *RenderBuilder) WithImage(filename string, data []byte) *RenderBuilder {
	b.tpl.Media(filename, data)
	return b
}

// WithPreProcessors adds XML pre-processing steps.
func (b *RenderBuilder) WithPreProcessors(m ...xml.HandlersMap) *RenderBuilder {
	b.tpl.AddPreProcessors(m...)
	return b
}

// WithPostProcessors adds XML post-processing steps.
func (b *RenderBuilder) WithPostProcessors(m ...xml.HandlersMap) *RenderBuilder {
	b.tpl.AddPostProcessors(m...)
	return b
}

// WithRemoveEmptyTableRows enables or disables empty table row removal.
func (b *RenderBuilder) WithRemoveEmptyTableRows(v bool) *RenderBuilder {
	b.tpl.removeEmptyTableRows = v
	return b
}

// WithRemoveRangeRows enables or disables range directive row removal.
func (b *RenderBuilder) WithRemoveRangeRows(v bool) *RenderBuilder {
	b.tpl.removeRangeRows = v
	return b
}

// WithIgnoreMissingKey enables or disables missing key ignoring.
func (b *RenderBuilder) WithIgnoreMissingKey(v bool) *RenderBuilder {
	b.tpl.ignoreMissingKey = v
	return b
}

// Apply runs the template and returns the result bytes.
func (b *RenderBuilder) Apply(data any) ([]byte, error) {
	if err := b.tpl.Apply(data); err != nil {
		return nil, err
	}
	return b.tpl.Bytes(), nil
}
