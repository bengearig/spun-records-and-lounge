package render

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/bengearig/spun-records-and-lounge/internal/feature"
	"github.com/bengearig/spun-records-and-lounge/internal/pdf"
	"github.com/bengearig/spun-records-and-lounge/internal/templates"
	"github.com/yuin/goldmark"
)

type RunOptions struct {
	// Output overrides the bundle's output path; relative paths resolve
	// against the working directory.
	Output string
	// Chromium is the executable path; empty falls back to pdf.Find's search.
	Chromium string
}

// Run renders b to a PDF and returns the path written.
func Run(ctx context.Context, b *bundle.Bundle, opts RunOptions) (string, error) {
	enabled := make(map[string]bool, len(b.Config.Features))
	for name, fc := range b.Config.Features {
		enabled[name] = fc.Enabled
	}
	features, err := feature.Default.Enabled(enabled)
	if err != nil {
		return "", err
	}

	var extensions []goldmark.Extender
	var head templates.HeadInjections
	for _, f := range features {
		if m, ok := f.(feature.MarkdownExtender); ok {
			extensions = append(extensions, m.Extenders()...)
		}
		if a, ok := f.(feature.Assets); ok {
			head.HTML = joinNonEmpty(head.HTML, a.HeadHTML())
			head.Style = joinNonEmpty(head.Style, a.Styles())
			head.Script = joinNonEmpty(head.Script, a.Scripts())
		}
	}

	out, err := outputPath(b, opts)
	if err != nil {
		return "", err
	}
	pdfOpts, err := pdfOptions(b.Config)
	if err != nil {
		return "", err
	}

	docs, err := Assemble(b, extensions)
	if err != nil {
		return "", err
	}
	html, err := templates.Render(b, Body(docs), head)
	if err != nil {
		return "", err
	}
	exe, err := pdf.Find(opts.Chromium)
	if err != nil {
		return "", err
	}
	data, err := pdf.Print(ctx, exe, html, pdfOpts)
	if err != nil {
		return "", err
	}
	if err := writeAtomic(out, data); err != nil {
		return "", err
	}
	return out, nil
}

func joinNonEmpty(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + "\n" + b
}

func outputPath(b *bundle.Bundle, opts RunOptions) (string, error) {
	path := opts.Output
	if path == "" {
		path = b.Config.Output
		if path == "" {
			path = filepath.Base(b.Dir) + ".pdf"
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(b.Dir, path)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("output path %q: %w", path, err)
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		return "", fmt.Errorf("output path %q is a directory", path)
	}
	return abs, nil
}

func pdfOptions(cfg bundle.Config) (pdf.Options, error) {
	w, h, err := pdf.PaperSize(cfg.Page.Size)
	if err != nil {
		return pdf.Options{}, fmt.Errorf("page.size: %w", err)
	}
	m, err := cssInches(cfg.Page.Margins)
	if err != nil {
		return pdf.Options{}, fmt.Errorf("page.margins: %w", err)
	}
	return pdf.Options{
		PaperWidth:     w,
		PaperHeight:    h,
		Margins:        pdf.Margins{Top: m, Right: m, Bottom: m, Left: m},
		FooterTemplate: templates.FooterTemplate(cfg.Page.Numbers),
	}, nil
}

var cssUnits = []struct {
	suffix string
	inches float64
}{
	{"in", 1},
	{"mm", 1 / 25.4},
	{"cm", 1 / 2.54},
	{"pt", 1.0 / 72},
	{"px", 1.0 / 96},
}

// cssInches converts a single CSS length (in, mm, cm, pt, px) to inches.
func cssInches(s string) (float64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	for _, u := range cssUnits {
		if num, ok := strings.CutSuffix(s, u.suffix); ok {
			v, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
			if err != nil || v < 0 {
				break
			}
			return v * u.inches, nil
		}
	}
	return 0, fmt.Errorf("unsupported length %q (use in, mm, cm, pt, or px)", s)
}

func writeAtomic(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".spun-*.pdf.tmp")
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	defer func() {
		if err != nil {
			os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write output: %w", err)
	}
	if err = tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fmt.Errorf("write output: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
