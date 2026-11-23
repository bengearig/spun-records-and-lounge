package templates

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
)

//go:embed default
var embedded embed.FS

const (
	defaultName  = "default"
	templateFile = "template.html"
	styleFile    = "style.css"
)

// HeadInjections carries feature-provided assets for the document head.
type HeadInjections struct {
	HTML   string
	Style  string
	Script string
}

type pageData struct {
	Size    string
	Margins string
	Numbers bool
}

type data struct {
	Title    string
	Subtitle string
	Author   string
	Date     string
	Body     template.HTML
	Page     pageData
	HeadHTML template.HTML
	Style    template.CSS
	Script   template.JS
}

func Render(b *bundle.Bundle, body string, head HeadInjections) ([]byte, error) {
	cfg := b.Config
	src, css, err := load(b)
	if err != nil {
		return nil, err
	}
	t, err := template.New(cfg.Template).Parse(src)
	if err != nil {
		return nil, fmt.Errorf("template %q: %w", cfg.Template, err)
	}
	style := css
	if head.Style != "" {
		style += "\n" + head.Style
	}
	d := data{
		Title:    cfg.Title,
		Subtitle: cfg.Subtitle,
		Author:   cfg.Author,
		Date:     cfg.Date,
		Body:     template.HTML(body),
		Page:     pageData{Size: cfg.Page.Size, Margins: cfg.Page.Margins, Numbers: cfg.Page.Numbers},
		HeadHTML: template.HTML(head.HTML),
		Style:    template.CSS(style),
		Script:   template.JS(head.Script),
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, d); err != nil {
		return nil, fmt.Errorf("template %q: %w", cfg.Template, err)
	}
	return buf.Bytes(), nil
}

func load(b *bundle.Bundle) (src, css string, err error) {
	name := b.Config.Template
	if name == "" || name == defaultName {
		return readFS(embedded, defaultName)
	}
	notFound := fmt.Errorf("template %q not found", name)
	if filepath.IsAbs(name) {
		return "", "", notFound
	}
	path := filepath.Join(b.Dir, name)
	if rel, relErr := filepath.Rel(b.Dir, path); relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", notFound
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return "", "", notFound
		}
		return "", "", fmt.Errorf("template %q: %w", name, statErr)
	}
	if info.IsDir() {
		return readFS(os.DirFS(path), ".")
	}
	if !strings.EqualFold(filepath.Ext(path), ".html") {
		return "", "", notFound
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		return "", "", fmt.Errorf("template %q: %w", name, readErr)
	}
	return string(raw), "", nil
}

func readFS(fsys fs.FS, dir string) (src, css string, err error) {
	html, err := fs.ReadFile(fsys, filepath.ToSlash(filepath.Join(dir, templateFile)))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", "", fmt.Errorf("template directory has no %s", templateFile)
		}
		return "", "", err
	}
	style, err := fs.ReadFile(fsys, filepath.ToSlash(filepath.Join(dir, styleFile)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", "", err
	}
	return string(html), string(style), nil
}

// FooterTemplate returns Chromium's footer HTML; an empty element hides the
// header or footer.
func FooterTemplate(numbers bool) string {
	if !numbers {
		return `<div></div>`
	}
	return `<div style="width:100%;font-size:9px;font-family:Helvetica,Arial,sans-serif;color:#666;text-align:center;">` +
		`<span class="pageNumber"></span> / <span class="totalPages"></span></div>`
}
