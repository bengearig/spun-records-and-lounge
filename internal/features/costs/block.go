package costs

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// KindBlock is the node kind of BlockNode.
var KindBlock = ast.NewNodeKind("CostsBlock")

// BlockNode is a fenced costs block; Line is the 1-based line of its opening fence.
type BlockNode struct {
	ast.BaseBlock
	Body string
	Line int
}

func (n *BlockNode) Kind() ast.NodeKind { return KindBlock }

func (n *BlockNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Body": n.Body}, nil)
}

type blockTransformer struct{}

func (blockTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	var found []*ast.FencedCodeBlock
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if fcb, ok := n.(*ast.FencedCodeBlock); ok && entering && isCostsFence(fcb, source) {
			found = append(found, fcb)
		}
		return ast.WalkContinue, nil
	})
	for _, fcb := range found {
		var body bytes.Buffer
		lines := fcb.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			body.Write(seg.Value(source))
		}
		n := &BlockNode{
			Body: body.String(),
			Line: 1 + bytes.Count(source[:min(fcb.Info.Segment.Start, len(source))], []byte{'\n'}),
		}
		fcb.Parent().ReplaceChild(fcb.Parent(), fcb, n)
	}
}

func isCostsFence(fcb *ast.FencedCodeBlock, source []byte) bool {
	return fcb.Info != nil && string(bytes.TrimSpace(fcb.Info.Segment.Value(source))) == Name
}

// cell is one table cell; num cells are right-aligned figures.
type cell struct {
	text string
	num  bool
}

func txt(s string) cell     { return cell{text: s} }
func money(m Money) cell    { return cell{text: m.String(), num: true} }
func count(n int) cell      { return cell{text: strconv.Itoa(n), num: true} }
func head(s string) cell    { return txt(s) }
func numHead(s string) cell { return cell{text: s, num: true} }

func rangeCells(r Range) []cell {
	return []cell{money(r.Low), money(r.Expected), money(r.High)}
}

func row(name string, cells ...cell) []cell {
	return append([]cell{txt(name)}, cells...)
}

type table struct {
	head []cell
	rows [][]cell
}

// tableKind builds one kind of table; options lists the keys it accepts.
type tableKind struct {
	options []string
	build   func(d *Data, opts map[string]string) (table, error)
}

// tableKinds is the one place table kinds are registered.
var tableKinds = map[string]tableKind{
	"categories": {build: buildCategories},
	"items":      {options: []string{"category", "level", "source"}, build: buildItems},
	"reserve":    {build: buildReserve},
}

func buildCategories(d *Data, _ map[string]string) (table, error) {
	t, err := d.Totals()
	if err != nil {
		return table{}, err
	}
	tb := table{head: []cell{head("Category"), numHead("Low"), numHead("Expected"), numHead("High")}}
	for _, c := range t.Categories {
		tb.rows = append(tb.rows, row(c.Name, rangeCells(c.Range)...))
	}
	tb.rows = append(tb.rows,
		row("Startup subtotal", rangeCells(t.Startup)...),
		row("Cash reserve", rangeCells(t.Reserve)...),
		row("Total to raise", rangeCells(t.Total)...),
	)
	return tb, nil
}

func buildItems(d *Data, opts map[string]string) (table, error) {
	catID, hasCat := opts["category"]
	if hasCat {
		if _, ok := d.Category(catID); !ok {
			return table{}, fmt.Errorf("unknown category %q", catID)
		}
	}
	levelID, hasLevel := opts["level"]
	if hasLevel {
		if _, _, ok := d.Level(levelID); !ok {
			return table{}, fmt.Errorf("unknown level %q", levelID)
		}
	}
	showSource := false
	if v, ok := opts["source"]; ok {
		switch v {
		case "true":
			showSource = true
		case "false":
		default:
			return table{}, fmt.Errorf("source must be true or false")
		}
	}
	tb := table{head: []cell{head("Item"), numHead("Low"), numHead("Expected"), numHead("High"), head("Status")}}
	if showSource {
		tb.head = append(tb.head, head("Source"))
	}
	for _, c := range d.Categories {
		if hasCat && c.ID != catID {
			continue
		}
		for _, it := range d.Items {
			if it.Category != c.ID || (hasLevel && it.Level != levelID) {
				continue
			}
			status := it.Status
			if it.OwnerInput {
				status += " (owner input needed)"
			}
			r := append(row(it.Name, rangeCells(itemRange(it))...), txt(status))
			if showSource {
				r = append(r, txt(it.Source))
			}
			tb.rows = append(tb.rows, r)
		}
	}
	return tb, nil
}

func buildReserve(d *Data, _ map[string]string) (table, error) {
	t, err := d.Totals()
	if err != nil {
		return table{}, err
	}
	tb := table{head: []cell{head("Item"), head("Level"), numHead("Low"), numHead("Expected"), numHead("High")}}
	for _, it := range d.Monthly {
		levelName := it.Level
		if l, _, ok := d.Level(it.Level); ok {
			levelName = l.Name
		}
		tb.rows = append(tb.rows, row(it.Name, append([]cell{txt(levelName)}, rangeCells(itemRange(it))...)...))
	}
	var months ReserveMonths
	if n := len(t.Levels); n > 0 {
		months = t.Levels[n-1].ReserveMonths
	}
	tb.rows = append(tb.rows,
		row("Monthly total", append([]cell{txt("")}, rangeCells(t.Monthly)...)...),
		row("Reserve months", txt(""), count(months.Low), count(months.Expected), count(months.High)),
		row("Reserve", append([]cell{txt("")}, rangeCells(t.Reserve)...)...),
	)
	return tb, nil
}

// parseBlock splits a block body into its table kind and options.
func parseBlock(body string) (string, tableKind, map[string]string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", tableKind{}, nil, fmt.Errorf("empty block")
	}
	if strings.ContainsAny(body, "\r\n") {
		return "", tableKind{}, nil, fmt.Errorf("expected one line, got more")
	}
	fields := strings.Fields(body)
	kind, ok := tableKinds[fields[0]]
	if !ok {
		return "", tableKind{}, nil, fmt.Errorf("unknown table %q", fields[0])
	}
	opts := map[string]string{}
	for _, f := range fields[1:] {
		k, v, ok := strings.Cut(f, "=")
		if !ok || k == "" {
			return "", tableKind{}, nil, fmt.Errorf("bad option %q", f)
		}
		if !slices.Contains(kind.options, k) {
			return "", tableKind{}, nil, fmt.Errorf("unknown option %q for %s", k, fields[0])
		}
		opts[k] = v
	}
	return fields[0], kind, opts, nil
}

func (r nodeRenderer) renderBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*BlockNode)
	_, kind, opts, err := parseBlock(n.Body)
	var tb table
	if err == nil {
		tb, err = kind.build(r.data, opts)
	}
	if err != nil {
		return ast.WalkStop, fmt.Errorf("line %d: costs block: %w", n.Line, err)
	}
	writeTable(w, tb)
	return ast.WalkSkipChildren, nil
}

func writeTable(w util.BufWriter, tb table) {
	_, _ = w.WriteString("<table class=\"costs-table\">\n<thead>\n<tr>")
	for _, c := range tb.head {
		writeCell(w, "th", c)
	}
	_, _ = w.WriteString("</tr>\n</thead>\n<tbody>\n")
	for _, r := range tb.rows {
		_, _ = w.WriteString("<tr>")
		for _, c := range r {
			writeCell(w, "td", c)
		}
		_, _ = w.WriteString("</tr>\n")
	}
	_, _ = w.WriteString("</tbody>\n</table>\n")
}

func writeCell(w util.BufWriter, tag string, c cell) {
	_, _ = w.WriteString("<" + tag)
	if c.num {
		_, _ = w.WriteString(` class="num"`)
	}
	_, _ = w.WriteString(">")
	_, _ = w.Write(util.EscapeHTML([]byte(c.text)))
	_, _ = w.WriteString("</" + tag + ">")
}

// Styles right-aligns figures in costs tables.
func (configured) Styles() string {
	return ".costs-table .num { text-align: right; font-variant-numeric: tabular-nums; }\n"
}

func (configured) HeadHTML() string { return "" }

func (configured) Scripts() string { return "" }
