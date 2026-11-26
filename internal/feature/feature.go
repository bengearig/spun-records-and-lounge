package feature

import (
	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
)

type Feature interface {
	Name() string
}

// BundleInfo describes the bundle a feature is enabled in.
type BundleInfo struct {
	Dir     string         // absolute bundle directory
	Options map[string]any // the feature's spun.yaml options, without "enabled"
}

// Configurable features return the feature to use for one bundle.
type Configurable interface {
	Feature
	Configure(BundleInfo) (Feature, error)
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
