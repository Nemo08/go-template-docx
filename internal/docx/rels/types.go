package rels

const (
	ImageMediaType = iota + 1
)

type MediaRel struct {
	Type   uint
	RefID  string
	Source string
}
