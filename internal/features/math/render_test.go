package math_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/bengearig/spun-records-and-lounge/internal/pdf"
	"github.com/bengearig/spun-records-and-lounge/internal/render"
)

func renderDoc(t *testing.T, md string) (string, error) {
	t.Helper()
	if _, err := pdf.Find(""); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	cfg := "title: T\ndocuments:\n  - a.md\nfeatures:\n  math:\n    enabled: true\n"
	for name, body := range map[string]string{"spun.yaml": cfg, "a.md": md} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := bundle.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return render.Run(context.Background(), b, render.RunOptions{Output: filepath.Join(dir, "out.pdf")})
}

func TestRender(t *testing.T) {
	out, err := renderDoc(t, "Inline $a+b$ here.\n\n$$x^2$$\n")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %q", data[:min(8, len(data))])
	}
	if !bytes.Contains(data, []byte("KaTeX_Main")) && !bytes.Contains(data, []byte("KaTeX_Math")) {
		t.Error("PDF does not embed a KaTeX font")
	}
}

func TestRenderInvalidTeX(t *testing.T) {
	_, err := renderDoc(t, `Bad $\frac{$ math.`)
	if err == nil {
		t.Fatal("want error for invalid TeX")
	}
	if !strings.Contains(err.Error(), `\frac{`) {
		t.Errorf("error lacks TeX snippet: %v", err)
	}
}

func TestVendoredMetadata(t *testing.T) {
	for _, f := range []string{"VERSION", "LICENSE", "katex.min.js", "katex.min.css"} {
		if _, err := os.Stat(filepath.Join("assets", f)); err != nil {
			t.Error(err)
		}
	}
}
