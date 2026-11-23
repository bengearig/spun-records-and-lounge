package render

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/yuin/goldmark/ast"
)

// Heading is one entry in a bundle's heading table.
type Heading struct {
	ID      string
	Text    string
	Number  string // empty when the heading is unnumbered
	Level   int
	DocPath string
	Line    int
}

type headingNode struct {
	node   *ast.Heading
	doc    int
	custom string
	h      Heading
}

// assignAnchors sets ids and number prefixes on the headings of every parsed
// document and returns each document's headings, in bundle order.
func assignAnchors(cfg bundle.Config, names []string, srcs [][]byte, roots []ast.Node) ([][]Heading, error) {
	var nodes []headingNode
	for i, root := range roots {
		err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			h, ok := n.(*ast.Heading)
			if !ok || !entering {
				return ast.WalkContinue, nil
			}
			hn := headingNode{node: h, doc: i}
			hn.h = Heading{
				Text:    headingText(h, srcs[i]),
				Level:   h.Level,
				DocPath: names[i],
				Line:    headingLine(h, srcs[i]),
			}
			if v, ok := h.AttributeString("id"); ok {
				if b, ok := v.([]byte); ok && len(b) > 0 {
					hn.custom = string(b)
				}
			}
			nodes = append(nodes, hn)
			return ast.WalkSkipChildren, nil
		})
		if err != nil {
			return nil, err
		}
	}

	used := make(map[string]bool, len(nodes))
	for i := range nodes {
		hn := &nodes[i]
		if hn.custom == "" {
			continue
		}
		if used[hn.custom] {
			return nil, fmt.Errorf("%s:%d: duplicate custom heading id %q", hn.h.DocPath, hn.h.Line, hn.custom)
		}
		used[hn.custom] = true
		hn.h.ID = hn.custom
	}
	for i := range nodes {
		hn := &nodes[i]
		if hn.custom != "" {
			continue
		}
		base := docSlug(hn.h.DocPath) + "--" + headingSlug(hn.h.Text)
		id := base
		for n := 1; used[id]; n++ {
			id = base + "-" + strconv.Itoa(n)
		}
		used[id] = true
		hn.h.ID = id
	}

	var counter numberer
	out := make([][]Heading, len(roots))
	for i := range nodes {
		hn := &nodes[i]
		num := counter.next(hn.h.Level)
		if cfg.Numbering.Enabled && num.depth <= cfg.Numbering.Depth {
			hn.h.Number = num.label
		}
		hn.node.SetAttributeString("id", []byte(hn.h.ID))
		if hn.h.Number != "" {
			span := ast.NewString([]byte(`<span class="heading-number">` + hn.h.Number + `</span> `))
			span.SetCode(true)
			hn.node.InsertBefore(hn.node, hn.node.FirstChild(), span)
		}
		out[hn.doc] = append(out[hn.doc], hn.h)
	}
	return out, nil
}

type numbering struct {
	label string
	depth int
}

// numberer numbers headings by nesting rather than absolute level, so a
// document that opens at ## still starts at 1. last holds the most recent
// count at each nesting depth, so a sibling that follows a skipped level
// continues the sequence instead of repeating a number.
type numberer struct {
	levels []int
	last   [8]int
}

func (n *numberer) next(level int) numbering {
	for len(n.levels) > 0 && n.levels[len(n.levels)-1] > level {
		n.levels = n.levels[:len(n.levels)-1]
	}
	if len(n.levels) == 0 || n.levels[len(n.levels)-1] < level {
		n.levels = append(n.levels, level)
	}
	d := len(n.levels) - 1
	n.last[d]++
	for i := d + 1; i < len(n.last); i++ {
		n.last[i] = 0
	}
	parts := make([]string, d+1)
	for i := range parts {
		parts[i] = strconv.Itoa(n.last[i])
	}
	return numbering{label: strings.Join(parts, "."), depth: d + 1}
}

func docSlug(name string) string {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return '-'
	}, base)
}

func headingSlug(text string) string {
	var sb strings.Builder
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(unicode.ToLower(r))
		case unicode.IsSpace(r), r == '-':
			sb.WriteByte('-')
		case r == '_':
			sb.WriteByte('_')
		}
	}
	if s := strings.Trim(sb.String(), "-"); s != "" {
		return s
	}
	return "heading"
}

func headingText(h *ast.Heading, src []byte) string {
	var sb strings.Builder
	var collect func(n ast.Node)
	collect = func(n ast.Node) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch t := c.(type) {
			case *ast.Text:
				sb.Write(t.Segment.Value(src))
				if t.SoftLineBreak() || t.HardLineBreak() {
					sb.WriteByte(' ')
				}
			case *ast.String:
				sb.Write(t.Value)
			default:
				collect(c)
			}
		}
	}
	collect(h)
	return strings.TrimSpace(sb.String())
}

func headingLine(h *ast.Heading, src []byte) int {
	lines := h.Lines()
	if lines == nil || lines.Len() == 0 {
		return 1
	}
	return bytes.Count(src[:lines.At(0).Start], []byte("\n")) + 1
}
