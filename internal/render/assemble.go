package render

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"strings"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// Doc is one rendered bundle document.
type Doc struct {
	// Path is the document path as listed in spun.yaml.
	Path   string
	HTML   string
	Source []byte
}

func Assemble(b *bundle.Bundle, extensions []goldmark.Extender) ([]Doc, error) {
	exts := []goldmark.Extender{extension.GFM, extension.Footnote}
	for _, e := range extensions {
		if e != nil {
			exts = append(exts, e)
		}
	}
	md := goldmark.New(goldmark.WithExtensions(exts...))

	docs := make([]Doc, 0, len(b.DocumentPaths))
	for i, path := range b.DocumentPaths {
		name := b.Config.Documents[i]
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if hasFrontMatter(src) {
			return nil, fmt.Errorf("%s: YAML front matter is not supported; remove the leading --- block", name)
		}
		var buf bytes.Buffer
		if err := md.Convert(src, &buf); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		docs = append(docs, Doc{Path: name, HTML: buf.String(), Source: src})
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
