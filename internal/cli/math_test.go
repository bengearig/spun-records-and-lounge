package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	_ "github.com/bengearig/spun-records-and-lounge/internal/features/math"
	"github.com/bengearig/spun-records-and-lounge/internal/pdf"
)

func TestFeatureMathRender(t *testing.T) {
	if _, err := pdf.Find(""); err != nil {
		t.Skip(err)
	}
	renderFixture := func(t *testing.T, fixture string) []byte {
		t.Helper()
		out := filepath.Join(t.TempDir(), "out.pdf")
		if code, _, stderr := runCLI("docs", "render", fixture, "-o", out); code != 0 {
			t.Fatalf("exit code = %d, stderr = %q", code, stderr)
		}
		return readPDF(t, out)
	}

	t.Run("enabled", func(t *testing.T) {
		if data := renderFixture(t, "testdata/math-bundle"); !bytes.Contains(data, []byte("KaTeX_")) {
			t.Error("PDF does not embed KaTeX fonts")
		}
	})
	t.Run("disabled", func(t *testing.T) {
		data := renderFixture(t, "testdata/math-disabled-bundle")
		if bytes.Contains(data, []byte("KaTeX_")) {
			t.Error("PDF embeds KaTeX fonts")
		}
	})
}
