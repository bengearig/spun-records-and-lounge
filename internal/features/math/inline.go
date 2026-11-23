package math

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type inlineParser struct{}

func (inlineParser) Trigger() []byte { return []byte{'$'} }

func (inlineParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()
	if escaped(block.Source(), segment.Start) {
		return nil
	}
	if len(line) > 1 && line[1] == '$' {
		if n := parseDisplay(block); n != nil {
			return n
		}
		block.Advance(2)
		return ast.NewTextSegment(segment.WithStop(segment.Start + 2))
	}
	return parseInline(block, line)
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

func parseInline(block text.Reader, line []byte) ast.Node {
	if len(line) < 2 || util.IsSpace(line[1]) {
		return nil
	}
	for i := 1; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '$':
			if i == 1 || util.IsSpace(line[i-1]) {
				continue
			}
			if i+1 < len(line) && util.IsNumeric(line[i+1]) {
				continue
			}
			block.Advance(i + 1)
			return &InlineNode{TeX: string(line[1:i])}
		}
	}
	return nil
}

// parseDisplay parses $$...$$ across lines, restoring the reader on failure.
func parseDisplay(block text.Reader) ast.Node {
	startLine, startSeg := block.Position()
	var tex []byte
	for first := true; ; first = false {
		line, _ := block.PeekLine()
		if line == nil {
			break
		}
		from := 0
		if first {
			from = 2
		}
		if j := indexClose(line, from); j >= 0 {
			tex = append(tex, line[from:j]...)
			block.Advance(j + 2)
			trimmed := string(bytes.TrimSpace(tex))
			if trimmed == "" {
				break
			}
			return &InlineNode{TeX: trimmed, Display: true}
		}
		tex = append(tex, line[from:]...)
		block.AdvanceLine()
	}
	block.SetPosition(startLine, startSeg)
	return nil
}

// indexClose returns the index of the first unescaped $$ at or after from.
func indexClose(line []byte, from int) int {
	for i := from; i+1 < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '$':
			if line[i+1] == '$' {
				return i
			}
		}
	}
	return -1
}
