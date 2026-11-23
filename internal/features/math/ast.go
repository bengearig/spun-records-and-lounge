package math

import "github.com/yuin/goldmark/ast"

// KindInline is the node kind of InlineNode.
var KindInline = ast.NewNodeKind("MathInline")

// KindBlock is the node kind of BlockNode.
var KindBlock = ast.NewNodeKind("MathBlock")

// InlineNode is math inside running text. Display is set for $$...$$ that
// appears within a paragraph.
type InlineNode struct {
	ast.BaseInline
	TeX     string
	Display bool
}

func (n *InlineNode) Kind() ast.NodeKind { return KindInline }

func (n *InlineNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"TeX": n.TeX}, nil)
}

// BlockNode is standalone display math whose TeX lives in the node's lines.
type BlockNode struct {
	ast.BaseBlock
	closed bool
}

func (n *BlockNode) Kind() ast.NodeKind { return KindBlock }

func (n *BlockNode) IsRaw() bool { return true }

func (n *BlockNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}
