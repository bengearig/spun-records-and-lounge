package render

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/yuin/goldmark/ast"
)

var schemeRE = regexp.MustCompile(`^(?:[a-zA-Z][a-zA-Z0-9+.-]*:|//)`)

// resolveLinks rewrites relative document and fragment links to internal
// "#<id>" links, using the heading table built by assignAnchors. All problems
// are collected and returned together. Links with empty text or the title
// "xref" get generated cross-reference text; pages supplies their page
// numbers (nil on the first pass, which prints "?"), and the ids of every
// xref target that needs a page number are returned.
func resolveLinks(b *bundle.Bundle, srcs [][]byte, roots []ast.Node, headings [][]Heading, pages map[string]int) ([]string, error) {
	byID := map[string]Heading{}
	for _, hs := range headings {
		for _, h := range hs {
			byID[h.ID] = h
		}
	}
	var xrefIDs []string
	seen := map[string]bool{}
	names := b.Config.Documents
	index := make(map[string]int, len(names))
	for i, name := range names {
		index[path.Clean(filepath.ToSlash(name))] = i
	}

	var errs []error
	for i, root := range roots {
		err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			link, ok := n.(*ast.Link)
			if !ok || !entering {
				return ast.WalkContinue, nil
			}
			dest := string(link.Destination)
			if dest == "" || schemeRE.MatchString(dest) {
				return ast.WalkContinue, nil
			}
			id, reason := resolveLink(b, index, headings, i, dest)
			if reason != "" {
				errs = append(errs, fmt.Errorf("%s:%d: link %q: %s", names[i], linkLine(link, srcs[i]), dest, reason))
				return ast.WalkContinue, nil
			}
			link.Destination = []byte("#" + id)
			if link.FirstChild() == nil || string(link.Title) == "xref" {
				link.Title = nil
				setXrefText(link, b.Config, byID[id], pages)
				if b.Config.Page.Numbers && !seen[id] {
					seen[id] = true
					xrefIDs = append(xrefIDs, id)
				}
			}
			return ast.WalkContinue, nil
		})
		if err != nil {
			return nil, err
		}
	}
	return xrefIDs, errors.Join(errs...)
}

// setXrefText replaces the link's children with "Section N, p. P", leaving
// out whichever parts are disabled and falling back to the heading text.
func setXrefText(link *ast.Link, cfg bundle.Config, h Heading, pages map[string]int) {
	var parts []string
	if h.Number != "" {
		parts = append(parts, "Section "+h.Number)
	}
	if cfg.Page.Numbers {
		page := "?"
		if p, ok := pages[h.ID]; ok {
			page = strconv.Itoa(p)
		}
		parts = append(parts, "p. "+page)
	}
	text := h.Text
	if len(parts) > 0 {
		text = strings.Join(parts, ", ")
	}
	for c := link.FirstChild(); c != nil; c = link.FirstChild() {
		link.RemoveChild(link, c)
	}
	s := ast.NewString([]byte(html.EscapeString(text)))
	s.SetCode(true)
	link.AppendChild(link, s)
}

// resolveLink returns the full heading id for dest, or a failure reason.
func resolveLink(b *bundle.Bundle, index map[string]int, headings [][]Heading, from int, dest string) (string, string) {
	rawPath, rawFrag, _ := strings.Cut(dest, "#")
	p, err := url.PathUnescape(rawPath)
	if err != nil {
		return "", fmt.Sprintf("invalid path encoding: %v", err)
	}
	frag, err := url.PathUnescape(rawFrag)
	if err != nil {
		return "", fmt.Sprintf("invalid fragment encoding: %v", err)
	}

	target := from
	if p == "" {
		if frag == "" {
			return "", "empty fragment"
		}
	} else {
		ext := strings.ToLower(path.Ext(p))
		if ext != ".md" && ext != ".markdown" {
			return "", "not a markdown document"
		}
		rel := path.Join(path.Dir(filepath.ToSlash(b.Config.Documents[from])), p)
		i, ok := index[rel]
		if !ok {
			if rel == ".." || strings.HasPrefix(rel, "../") {
				return "", "file is outside the bundle"
			}
			if _, err := os.Stat(filepath.Join(b.Dir, filepath.FromSlash(rel))); err != nil {
				return "", "file not found"
			}
			return "", "file is not listed in documents"
		}
		target = i
	}

	hs := headings[target]
	doc := b.Config.Documents[target]
	if frag == "" {
		if len(hs) == 0 {
			return "", fmt.Sprintf("%s has no headings", doc)
		}
		return hs[0].ID, ""
	}
	return matchHeading(hs, doc, frag)
}

// matchHeading finds the heading for frag, trying the full id first, then a
// heading slug.
func matchHeading(hs []Heading, doc, frag string) (string, string) {
	for _, h := range hs {
		if h.ID == frag {
			return h.ID, ""
		}
	}
	prefix := docSlug(doc) + "--"
	var found []Heading
	for _, h := range hs {
		if headingSlug(h.Text) == frag || h.ID == prefix+frag {
			found = append(found, h)
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Sprintf("heading %q not found in %s", frag, doc)
	case 1:
		return found[0].ID, ""
	}
	lines := make([]string, len(found))
	for i, h := range found {
		lines[i] = fmt.Sprintf("line %d", h.Line)
	}
	return "", fmt.Sprintf("heading %q is ambiguous in %s (%s); use a full or custom id", frag, doc, strings.Join(lines, ", "))
}

// linkLine reports the 1-based source line of a link or image, falling back to the
// enclosing block when the link has no text of its own.
func linkLine(link ast.Node, src []byte) int {
	offset := -1
	_ = ast.Walk(link, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := n.(*ast.Text); ok && entering {
			offset = t.Segment.Start
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	if offset < 0 {
		for p := link.Parent(); p != nil; p = p.Parent() {
			if lines := p.Lines(); p.Type() == ast.TypeBlock && lines != nil && lines.Len() > 0 {
				offset = lines.At(0).Start
				break
			}
		}
	}
	if offset < 0 {
		return 1
	}
	return bytes.Count(src[:offset], []byte("\n")) + 1
}
