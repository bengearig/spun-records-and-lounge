package render

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/bengearig/spun-records-and-lounge/internal/pdf"
)

// imageBundle copies the images fixture into a temp dir and adds files.
func imageBundle(t *testing.T, files map[string]string) *bundle.Bundle {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS("testdata/images/ok")); err != nil {
		t.Fatal(err)
	}
	if _, ok := files["sub/b.md"]; !ok {
		files["sub/b.md"] = "# B\n"
	}
	files["spun.yaml"] = "title: T\ndocuments:\n  - a.md\n  - sub/b.md\n"
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := bundle.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestImagesEmbedded(t *testing.T) {
	b := imageBundle(t, map[string]string{
		"a.md":     "# A\n\n![A pixel](pixel.png \"Pixel title\")\n\n![Logo](logo.svg)\n",
		"sub/b.md": "# B\n\n![Sub](pixel.png)\n",
	})
	docs, err := Assemble(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := docs[0].HTML
	for _, want := range []string{
		`<img src="data:image/png;base64,iVBORw0KGgo`,
		`alt="A pixel" title="Pixel title"`,
		`<img src="data:image/svg+xml;base64,`,
		`alt="Logo"`,
	} {
		if !strings.Contains(a, want) {
			t.Errorf("missing %s in:\n%s", want, a)
		}
	}
	if !strings.Contains(docs[1].HTML, `<img src="data:image/png;base64,`) {
		t.Errorf("subdirectory image not inlined relative to its document:\n%s", docs[1].HTML)
	}
}

func TestImagesErrors(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.png"), mustRead(t, "testdata/images/ok/pixel.png"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		files map[string]string
		link  bool
		want  []string
	}{
		{"missing", map[string]string{"a.md": "# A\n\n![x](nope.png)\n"}, false,
			[]string{`a.md:3: image "nope.png": file not found`}},
		{"escapes", map[string]string{"a.md": "# A\n\n![x](../outside.png)\n"}, false,
			[]string{`a.md:3: image "../outside.png": path is outside the bundle`}},
		{"symlink escape", map[string]string{"a.md": "# A\n\n![x](link.png)\n"}, true,
			[]string{`a.md:3: image "link.png": path is outside the bundle`}},
		{"unsupported", map[string]string{"a.md": "# A\n\n![x](notes.txt)\n", "notes.txt": "hello"}, false,
			[]string{`a.md:3: image "notes.txt": unsupported image type`}},
		{"svg extension without svg content", map[string]string{"a.md": "# A\n\n![x](bad.svg)\n", "bad.svg": "hello"}, false,
			[]string{`a.md:3: image "bad.svg": unsupported image type`}},
		{"remote", map[string]string{"a.md": "# A\n\n![x](https://127.0.0.1:1/x.png)\n"}, false,
			[]string{`a.md:3: image "https://127.0.0.1:1/x.png": remote images are not supported`}},
		{"multiple", map[string]string{
			"a.md":     "# A\n\n![x](nope.png)\n\n![y](http://example.com/y.png)\n",
			"sub/b.md": "# B\n\n![z](../../out.png)\n",
		}, false, []string{
			`a.md:3: image "nope.png": file not found`,
			`a.md:5: image "http://example.com/y.png": remote images are not supported`,
			`b.md:3: image "../../out.png": path is outside the bundle`,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := imageBundle(t, tc.files)
			if tc.link {
				if err := os.Symlink(filepath.Join(outside, "secret.png"), filepath.Join(b.Dir, "link.png")); err != nil {
					t.Skip(err)
				}
			}
			_, err := Assemble(b, nil)
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("missing %q in:\n%v", w, err)
				}
			}
		})
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestImagesRenderPDF(t *testing.T) {
	exe, err := pdf.Find("")
	if err != nil {
		t.Fatalf("chromium required: %v", err)
	}
	b := imageBundle(t, map[string]string{
		"a.md":     "# A\n\n![A pixel](pixel.png)\n",
		"sub/b.md": "# B\n",
	})
	out, err := Run(context.Background(), b, RunOptions{Output: filepath.Join(t.TempDir(), "out.pdf"), Chromium: exe})
	if err != nil {
		t.Fatal(err)
	}
	data := mustRead(t, out)
	if !bytes.Contains(data, []byte("/Subtype /Image")) && !bytes.Contains(data, []byte("/Subtype/Image")) {
		t.Error("PDF has no image XObject")
	}
}
