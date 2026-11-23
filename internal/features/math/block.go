package math

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var delim = []byte("$$")

type blockParser struct{}

func (blockParser) Trigger() []byte { return []byte{'$'} }

func (blockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	trimmed := bytes.TrimSpace(line)
	if !bytes.HasPrefix(trimmed, delim) {
		return nil, parser.NoChildren
	}
	node := &BlockNode{}
	if len(trimmed) == len(delim) {
		reader.Advance(lineLen(line))
		return node, parser.NoChildren
	}
	if len(trimmed) > 2*len(delim) && bytes.HasSuffix(trimmed, delim) {
		inner := trimmed[len(delim) : len(trimmed)-len(delim)]
		if indexClose(inner, 0) < 0 && len(bytes.TrimSpace(inner)) > 0 {
			start := segment.Start + bytes.Index(line, inner)
			node.Lines().Append(text.NewSegment(start, start+len(inner)))
			node.closed = true
			reader.Advance(lineLen(line))
			return node, parser.NoChildren
		}
	}
	return nil, parser.NoChildren
}

func (blockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	n := node.(*BlockNode)
	if n.closed {
		return parser.Close
	}
	line, segment := reader.PeekLine()
	if bytes.Equal(bytes.TrimSpace(line), delim) {
		n.closed = true
		reader.Advance(lineLen(line))
		return parser.Close
	}
	n.Lines().Append(segment)
	reader.Advance(lineLen(line))
	return parser.Continue | parser.NoChildren
}

func (blockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (blockParser) CanInterruptParagraph() bool { return false }

func (blockParser) CanAcceptIndentedLine() bool { return false }

// lineLen is the line length without its trailing newline.
func lineLen(line []byte) int {
	return len(bytes.TrimRight(line, "\r\n"))
}
