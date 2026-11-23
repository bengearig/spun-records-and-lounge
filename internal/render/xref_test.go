package render

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/bengearig/spun-records-and-lounge/internal/pdf"
)

const lorem = "The quick brown fox jumps over the lazy dog while the label tallies another royalty statement for the quarter. "

func filler(paragraphs int) string {
	var sb strings.Builder
	for i := 0; i < paragraphs; i++ {
		sb.WriteString(strings.Repeat(lorem, 6))
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// xrefBundle writes a multi-page bundle and returns its directory. The
// heading "Margins" is document 2's first subsection, so it is numbered 2.1.
func xrefBundle(t *testing.T, config string, intro string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"spun.yaml":  "title: Xref Fixture\ndocuments:\n  - intro.md\n  - details.md\n" + config,
		"intro.md":   intro,
		"details.md": "# Details\n\n" + filler(40) + "## Margins\n\nMargin text.\n\n" + filler(5),
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func countPrints(t *testing.T) *atomic.Int32 {
	t.Helper()
	var n atomic.Int32
	orig := printPDF
	printPDF = func(ctx context.Context, exe string, html []byte, opts pdf.Options, ids []string) ([]byte, map[string]int, error) {
		n.Add(1)
		return orig(ctx, exe, html, opts, ids)
	}
	t.Cleanup(func() { printPDF = orig })
	return &n
}

func runXref(t *testing.T, dir string) (text []string, prints int32) {
	t.Helper()
	exe, err := pdf.Find("")
	if err != nil {
		t.Fatalf("chromium required: %v", err)
	}
	b, err := bundle.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := countPrints(t)
	out, err := Run(context.Background(), b, RunOptions{Output: filepath.Join(t.TempDir(), "out.pdf"), Chromium: exe})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return pdfPageText(t, data), n.Load()
}

const xrefIntro = "# Intro\n\nSee [](details.md#margins) and [Pricing margins](details.md#margins) or [titled](details.md#margins \"xref\").\n"

func squash(s string) string { return strings.Join(strings.Fields(s), "") }

// headingPage finds the page whose text holds the numbered heading.
func headingPage(t *testing.T, pages []string, heading string) int {
	t.Helper()
	for i, p := range pages {
		if strings.Contains(p, squash(heading)) && !strings.Contains(p, "Section") {
			return i + 1
		}
	}
	t.Fatalf("heading %q not found on any page", heading)
	return 0
}

func TestXref(t *testing.T) {
	t.Run("section and page", func(t *testing.T) {
		pages, prints := runXref(t, xrefBundle(t, "", xrefIntro))
		want := headingPage(t, pages, "2.1 Margins")
		if want < 3 {
			t.Fatalf("fixture too small: heading on page %d", want)
		}
		wantText := squash(fmt.Sprintf("Section 2.1, p. %d", want))
		if !strings.Contains(pages[1], wantText) {
			t.Errorf("page 2 lacks %q:\n%s", wantText, pages[1])
		}
		if got := strings.Count(strings.Join(pages, ""), wantText); got != 2 {
			t.Errorf("xref text appears %d times, want 2 (empty text and title xref)", got)
		}
		if !strings.Contains(pages[1], squash("Pricing margins")) {
			t.Error("plain link lost its text")
		}
		if prints < 2 || prints > maxPrints {
			t.Errorf("prints = %d, want 2..%d", prints, maxPrints)
		}
	})

	t.Run("numbering disabled", func(t *testing.T) {
		pages, _ := runXref(t, xrefBundle(t, "numbering:\n  enabled: false\n", xrefIntro))
		want := headingPage(t, pages, "Margins")
		all := strings.Join(pages, "")
		if strings.Contains(all, "Section") {
			t.Error("section text present with numbering disabled")
		}
		if !strings.Contains(pages[1], squash(fmt.Sprintf("See p. %d and", want))) {
			t.Errorf("page 2 lacks page-only xref for page %d:\n%s", want, pages[1])
		}
	})

	t.Run("page numbers disabled", func(t *testing.T) {
		pages, prints := runXref(t, xrefBundle(t, "page:\n  numbers: false\n", xrefIntro))
		if !strings.Contains(pages[1], squash("See Section 2.1 and")) {
			t.Errorf("page 2 lacks section-only xref:\n%s", pages[1])
		}
		if strings.Contains(strings.Join(pages, ""), "p.") {
			t.Error("page text present with page.numbers false")
		}
		if prints != 1 {
			t.Errorf("prints = %d, want 1", prints)
		}
	})

	t.Run("no xrefs prints once", func(t *testing.T) {
		intro := "# Intro\n\nSee [Pricing margins](details.md#margins).\n"
		pages, prints := runXref(t, xrefBundle(t, "", intro))
		if prints != 1 {
			t.Errorf("prints = %d, want 1", prints)
		}
		if !strings.Contains(pages[1], squash("Pricing margins")) {
			t.Error("plain link lost its text")
		}
	})
}

func TestXrefFallsBackToHeadingText(t *testing.T) {
	dir := xrefBundle(t, "numbering:\n  enabled: false\npage:\n  numbers: false\n", xrefIntro)
	b, err := bundle.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	docs, ids, err := assembleWithPages(b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Errorf("ids = %v, want none", ids)
	}
	if got := strings.Count(docs[0].HTML, `<a href="#details--margins">Margins</a>`); got != 2 {
		t.Errorf("heading-text xrefs = %d, want 2:\n%s", got, docs[0].HTML)
	}
	if strings.Contains(docs[0].HTML, `title="xref"`) {
		t.Error("xref title attribute not removed")
	}
}
