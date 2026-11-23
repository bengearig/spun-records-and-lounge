package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const timeout = 60 * time.Second

// Margins are in inches.
type Margins struct {
	Top, Right, Bottom, Left float64
}

// Options configures Print. Paper sizes and margins are in inches.
type Options struct {
	PaperWidth, PaperHeight float64
	Margins                 Margins
	FooterTemplate          string
	// WaitForFlag names a window property to wait for before printing.
	WaitForFlag string
	// ErrorFlag names a window property holding a page-reported failure,
	// checked after WaitForFlag is set.
	ErrorFlag string
}

// PaperSize returns the width and height in inches for a page size name.
func PaperSize(size string) (width, height float64, err error) {
	switch strings.ToLower(size) {
	case "letter":
		return 8.5, 11, nil
	case "a4":
		return 8.27, 11.69, nil
	}
	return 0, 0, fmt.Errorf("unknown page size %q (want letter or a4)", size)
}

// Print renders html to PDF with headless Chromium at exe.
func Print(ctx context.Context, exe string, html []byte, opts Options) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dir, err := os.MkdirTemp("", "spun-pdf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "index.html")
	if err := os.WriteFile(file, html, 0o600); err != nil {
		return nil, err
	}

	allocOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocOpts = append(allocOpts, chromedp.ExecPath(exe))
	if os.Getenv(envNoSandbox) == "1" {
		allocOpts = append(allocOpts, chromedp.NoSandbox)
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, allocOpts...)
	defer cancelAlloc()
	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	steps := []chromedp.Action[chromedp.Void]{chromedp.Navigate("file://" + filepath.ToSlash(file))}
	if opts.WaitForFlag != "" {
		expr := fmt.Sprintf("window[%q] === true", opts.WaitForFlag)
		steps = append(steps, chromedp.Poll[chromedp.Void](expr, chromedp.WithPollingInterval(50*time.Millisecond)))
	}
	if err := chromedp.Do(taskCtx, steps...); err != nil {
		return nil, fmt.Errorf("load html: %w", err)
	}

	if opts.ErrorFlag != "" {
		expr := fmt.Sprintf("String(window[%q] || '')", opts.ErrorFlag)
		msg, err := chromedp.Run(taskCtx, chromedp.Evaluate[string](expr))
		if err != nil {
			return nil, fmt.Errorf("read page error: %w", err)
		}
		if msg != "" {
			return nil, fmt.Errorf("page error: %s", msg)
		}
	}

	params := page.PrintToPDFParams{
		PreferCSSPageSize: ptr(true),
		PrintBackground:   ptr(true),
		PaperWidth:        opts.PaperWidth,
		PaperHeight:       opts.PaperHeight,
		MarginTop:         ptr(opts.Margins.Top),
		MarginRight:       ptr(opts.Margins.Right),
		MarginBottom:      ptr(opts.Margins.Bottom),
		MarginLeft:        ptr(opts.Margins.Left),
	}
	if opts.FooterTemplate != "" {
		params.DisplayHeaderFooter = ptr(true)
		params.FooterTemplate = opts.FooterTemplate
	}
	buf, err := chromedp.Run(taskCtx, chromedp.PrintToPDF(
		func(p *page.PrintToPDFParams) { *p = params },
	))
	if err != nil {
		return nil, fmt.Errorf("print pdf: %w", err)
	}
	return buf, nil
}

func ptr[T any](v T) *T { return &v }
