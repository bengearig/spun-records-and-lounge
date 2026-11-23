package math

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type nodeRenderer struct{}

func (nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindInline, renderInline)
	reg.Register(KindBlock, renderBlock)
}

func renderInline(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*InlineNode)
	if n.Display {
		writeMath(w, "div", "math display", n.TeX)
	} else {
		writeMath(w, "span", "math inline", n.TeX)
	}
	return ast.WalkSkipChildren, nil
}

func renderBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	var sb strings.Builder
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		sb.Write(seg.Value(source))
	}
	writeMath(w, "div", "math display", strings.TrimSpace(sb.String()))
	_ = w.WriteByte('\n')
	return ast.WalkSkipChildren, nil
}

func writeMath(w util.BufWriter, tag, class, tex string) {
	esc := util.EscapeHTML([]byte(tex))
	_, _ = w.WriteString("<" + tag + ` class="` + class + `" data-tex="`)
	_, _ = w.Write(esc)
	_, _ = w.WriteString(`">`)
	_, _ = w.Write(esc)
	_, _ = w.WriteString("</" + tag + ">")
}
