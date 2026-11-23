package render

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"strings"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Doc is one rendered bundle document.
type Doc struct {
	// Path is the document path as listed in spun.yaml.
	Path     string
	HTML     string
	Source   []byte
	Headings []Heading
}

func Assemble(b *bundle.Bundle, extensions []goldmark.Extender) ([]Doc, error) {
	exts := []goldmark.Extender{extension.GFM, extension.Footnote}
	for _, e := range extensions {
		if e != nil {
			exts = append(exts, e)
		}
	}
	md := goldmark.New(
		goldmark.WithExtensions(exts...),
		goldmark.WithParserOptions(parser.WithHeadingAttribute()),
	)

	srcs := make([][]byte, len(b.DocumentPaths))
	roots := make([]ast.Node, len(b.DocumentPaths))
	for i, path := range b.DocumentPaths {
		name := b.Config.Documents[i]
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if hasFrontMatter(src) {
			return nil, fmt.Errorf("%s: YAML front matter is not supported; remove the leading --- block", name)
		}
		srcs[i] = src
		roots[i] = md.Parser().Parse(text.NewReader(src))
	}

	headings, err := assignAnchors(b.Config, b.Config.Documents, srcs, roots)
	if err != nil {
		return nil, err
	}
	if err := resolveLinks(b, srcs, roots, headings); err != nil {
		return nil, err
	}
	docs := make([]Doc, 0, len(roots))
	for i, root := range roots {
		name := b.Config.Documents[i]
		var buf bytes.Buffer
		if err := md.Renderer().Render(&buf, srcs[i], root); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		docs = append(docs, Doc{Path: name, HTML: buf.String(), Source: srcs[i], Headings: headings[i]})
	}
	return docs, nil
}

func hasFrontMatter(src []byte) bool {
	lines := strings.Split(strings.TrimPrefix(string(src), "\ufeff"), "\n")
	if strings.TrimRight(lines[0], " \t\r") != "---" {
		return false
	}
	for _, l := range lines[1:] {
		switch strings.TrimRight(l, " \t\r") {
		case "---", "...":
			return true
		}
	}
	return false
}

func Body(docs []Doc) string {
	var sb strings.Builder
	for i, d := range docs {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(`<section class="document" data-source="`)
		sb.WriteString(html.EscapeString(d.Path))
		sb.WriteByte('"')
		if i > 0 {
			sb.WriteString(` style="break-before: page; page-break-before: always;"`)
		}
		sb.WriteString(">\n")
		sb.WriteString(d.HTML)
		sb.WriteString("</section>\n")
	}
	return sb.String()
}
