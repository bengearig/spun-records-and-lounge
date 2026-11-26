package costs

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// KindInline is the node kind of InlineNode.
var KindInline = ast.NewNodeKind("CostsInline")

// InlineNode is a {{costs:<path>}} directive; Offset is its byte offset in the source.
type InlineNode struct {
	ast.BaseInline
	Path   string
	Offset int
}

func (n *InlineNode) Kind() ast.NodeKind { return KindInline }

func (n *InlineNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Path": n.Path}, nil)
}

const directivePrefix = "{{costs:"

type inlineParser struct{}

func (inlineParser) Trigger() []byte { return []byte{'{'} }

func (inlineParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()
	if escaped(block.Source(), segment.Start) || !bytes.HasPrefix(line, []byte(directivePrefix)) {
		return nil
	}
	rest := line[len(directivePrefix):]
	end := bytes.Index(rest, []byte("}}"))
	if end < 0 {
		return nil
	}
	path := rest[:end]
	if bytes.ContainsAny(path, "{} \t\r\n") {
		return nil
	}
	block.Advance(len(directivePrefix) + end + 2)
	return &InlineNode{Path: string(path), Offset: segment.Start}
}

// escaped reports whether the byte at pos is preceded by an odd number of
// backslashes.
func escaped(src []byte, pos int) bool {
	n := 0
	for i := pos - 1; i >= 0 && src[i] == '\\'; i-- {
		n++
	}
	return n%2 == 1
}

type nodeRenderer struct {
	data *Data
}

func (r nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindInline, r.renderInline)
	reg.Register(KindBlock, r.renderBlock)
}

func (r nodeRenderer) renderInline(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*InlineNode)
	out, err := resolve(r.data, n.Path)
	if err != nil {
		line := 1 + bytes.Count(source[:min(n.Offset, len(source))], []byte{'\n'})
		return ast.WalkStop, fmt.Errorf("line %d: costs: %w in %q", line, err, directivePrefix+n.Path+"}}")
	}
	_, _ = w.Write(util.EscapeHTML([]byte(out)))
	return ast.WalkSkipChildren, nil
}

// resolve formats the figure a directive path names.
func resolve(d *Data, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	t, err := d.Totals()
	if err != nil {
		return "", err
	}
	parts := strings.Split(path, ".")
	var rng Range
	var rest []string
	switch parts[0] {
	case "total", "startup", "reserve", "monthly":
		rest = parts[1:]
		switch parts[0] {
		case "total":
			rng = t.Total
		case "startup":
			rng = t.Startup
		case "reserve":
			rng = t.Reserve
		default:
			rng = t.Monthly
		}
	case "category", "item", "level", "goal":
		if len(parts) < 2 {
			return "", fmt.Errorf("unknown path %q", path)
		}
		id := parts[1]
		rest = parts[2:]
		switch parts[0] {
		case "category":
			if _, ok := d.Category(id); !ok {
				return "", fmt.Errorf("unknown category %q", id)
			}
			for _, c := range t.Categories {
				if c.ID == id {
					rng = c.Range
				}
			}
		case "item":
			it, ok := d.Item(id)
			if !ok {
				return "", fmt.Errorf("unknown item %q", id)
			}
			rng = itemRange(it)
		default:
			_, i, ok := d.Level(id)
			if !ok {
				return "", fmt.Errorf("unknown level %q", id)
			}
			if parts[0] == "level" {
				rng = t.Levels[i].Adds
			} else {
				rng = t.Levels[i].Goal
			}
		}
	default:
		return "", fmt.Errorf("unknown path %q", path)
	}
	switch len(rest) {
	case 0:
		if rng.Low == rng.High {
			return rng.Low.String(), nil
		}
		return rng.Low.String() + " to " + rng.High.String(), nil
	case 1:
		switch rest[0] {
		case "low":
			return rng.Low.String(), nil
		case "expected":
			return rng.Expected.String(), nil
		case "high":
			return rng.High.String(), nil
		}
		return "", fmt.Errorf("unknown suffix %q", rest[0])
	}
	return "", fmt.Errorf("unknown path %q", path)
}
