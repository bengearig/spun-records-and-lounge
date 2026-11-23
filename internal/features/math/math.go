// Package math adds LaTeX math to markdown: $...$ for inline math and $$...$$
// for display math. It emits marked-up HTML for client-side or headless KaTeX
// rendering.
package math

import (
	"github.com/bengearig/spun-records-and-lounge/internal/feature"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Name is the feature name used in spun.yaml.
const Name = "math"

func init() {
	feature.Default.Register(Feature{})
}

// Feature is the optional math feature.
type Feature struct{}

func (Feature) Name() string { return Name }

func (Feature) Extenders() []goldmark.Extender {
	return []goldmark.Extender{Extension}
}

type extension struct{}

// Extension is the goldmark extension that parses and renders math.
var Extension goldmark.Extender = extension{}

func (extension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(util.Prioritized(inlineParser{}, 200)),
		parser.WithBlockParsers(util.Prioritized(blockParser{}, 500)),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(util.Prioritized(nodeRenderer{}, 500)),
	)
}
