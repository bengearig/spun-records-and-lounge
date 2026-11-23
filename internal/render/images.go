package render

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

var remoteRE = regexp.MustCompile(`(?i)^(?:https?:)?//`)

var svgRE = regexp.MustCompile(`(?i)<svg[\s>]`)

// embedImages replaces every image destination with a data URI. All problems
// are collected and returned together.
func embedImages(b *bundle.Bundle, srcs [][]byte, roots []ast.Node) error {
	root, err := filepath.EvalSymlinks(b.Dir)
	if err != nil {
		return fmt.Errorf("bundle directory: %w", err)
	}
	var errs []error
	for i, node := range roots {
		name := b.Config.Documents[i]
		docDir := filepath.Dir(b.DocumentPaths[i])
		err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			img, ok := n.(*ast.Image)
			if !ok || !entering {
				return ast.WalkContinue, nil
			}
			dest := string(img.Destination)
			uri, reason := loadImage(b.Dir, root, docDir, dest)
			if reason != "" {
				errs = append(errs, fmt.Errorf("%s:%d: image %q: %s", name, linkLine(img, srcs[i]), dest, reason))
				return ast.WalkContinue, nil
			}
			img.Destination = []byte(uri)
			return ast.WalkContinue, nil
		})
		if err != nil {
			return err
		}
	}
	return errors.Join(errs...)
}

// loadImage returns the data URI for dest, or a failure reason. dir is the
// bundle directory and root its symlink-resolved form.
func loadImage(dir, root, docDir, dest string) (string, string) {
	if dest == "" {
		return "", "empty source"
	}
	if remoteRE.MatchString(dest) {
		return "", "remote images are not supported; add the file to the bundle"
	}
	if schemeRE.MatchString(dest) {
		return "", "unsupported source scheme"
	}
	p, err := url.PathUnescape(dest)
	if err != nil {
		return "", fmt.Sprintf("invalid path encoding: %v", err)
	}
	full := filepath.Join(docDir, filepath.FromSlash(p))
	if !within(dir, full) {
		return "", "path is outside the bundle"
	}
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "file not found"
		}
		return "", err.Error()
	}
	if !within(root, resolved) {
		return "", "path is outside the bundle"
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err.Error()
	}
	if !info.Mode().IsRegular() {
		return "", "not a regular file"
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err.Error()
	}
	mime, ok := imageMIME(data, resolved)
	if !ok {
		return "", "unsupported image type (want PNG, JPEG, GIF, SVG, or WebP)"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), ""
}

func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func imageMIME(data []byte, name string) (string, bool) {
	switch m := http.DetectContentType(data); m {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return m, true
	}
	if strings.EqualFold(filepath.Ext(name), ".svg") && svgRE.Match(data) {
		return "image/svg+xml", true
	}
	return "", false
}

// imageExtender renders images with the stock renderer in unsafe mode, which
// goldmark needs to emit the data:image/svg+xml URIs produced by embedImages.
// Every image destination is a data URI generated here, so nothing
// author-controlled reaches the output unchecked.
type imageExtender struct{}

type funcCapture map[ast.NodeKind]renderer.NodeRendererFunc

func (c funcCapture) Register(k ast.NodeKind, f renderer.NodeRendererFunc) { c[k] = f }

type imageRenderer struct{ fn renderer.NodeRendererFunc }

func (r imageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.fn)
}

func (imageExtender) Extend(md goldmark.Markdown) {
	c := funcCapture{}
	html.NewRenderer(html.WithUnsafe()).RegisterFuncs(c)
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(imageRenderer{c[ast.KindImage]}, 100)))
}
