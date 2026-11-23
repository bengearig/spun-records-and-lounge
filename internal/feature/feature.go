package feature

import (
	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
)

type Feature interface {
	Name() string
}

type MarkdownExtender interface {
	Extenders() []goldmark.Extender
}

// Phase orders pipeline stages; lower phases run first.
type Phase int

const (
	// PreRender stages transform markdown source before it is rendered.
	PreRender Phase = iota
	// PostRender stages transform the HTML produced by rendering.
	PostRender
)

func (p Phase) String() string {
	switch p {
	case PreRender:
		return "PreRender"
	case PostRender:
		return "PostRender"
	}
	return "Phase(unknown)"
}

type Stage interface {
	StageName() string
	Phase() Phase
	Process(doc string) (string, error)
}

type CommandProvider interface {
	Commands() []*cobra.Command
}

type Assets interface {
	HeadHTML() string
	Styles() string
	Scripts() string
}
