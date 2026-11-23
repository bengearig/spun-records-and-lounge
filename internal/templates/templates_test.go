package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
)

func newBundle(t *testing.T, mutate func(*bundle.Config)) *bundle.Bundle {
	t.Helper()
	cfg := bundle.Config{
		Title:    "Annual Plan",
		Subtitle: "Fiscal Year",
		Author:   "Ada Lovelace",
		Date:     "2026-01-01",
		Template: "default",
		Page:     bundle.PageConfig{Size: "letter", Margins: "1in", Numbers: true},
	}
	if mutate != nil {
		mutate(&cfg)
	}
	return &bundle.Bundle{Dir: t.TempDir(), Config: cfg}
}

func render(t *testing.T, b *bundle.Bundle, head HeadInjections) string {
	t.Helper()
	out, err := Render(b, "<p>BODY-MARK</p>", head)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return string(out)
}

func TestDefaultRender(t *testing.T) {
	out := render(t, newBundle(t, nil), HeadInjections{})
	for _, want := range []string{"Annual Plan", "Ada Lovelace", "Fiscal Year", "2026-01-01", "<p>BODY-MARK</p>", "size: letter", "margin: 1in", "font-family"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestEmptyFieldsOmitted(t *testing.T) {
	b := newBundle(t, func(c *bundle.Config) { c.Subtitle, c.Author, c.Date = "", "", "" })
	out := render(t, b, HeadInjections{})
	for _, bad := range []string{`class="cover-subtitle"`, `class="cover-author"`, `class="cover-date"`, `name="author"`} {
		if strings.Contains(out, bad) {
			t.Errorf("output should omit %q", bad)
		}
	}
	if !strings.Contains(out, `class="cover-title"`) {
		t.Error("title missing")
	}
}

func TestPageCSS(t *testing.T) {
	b := newBundle(t, func(c *bundle.Config) { c.Page.Size, c.Page.Margins = "a4", "20mm 15mm" })
	out := render(t, b, HeadInjections{})
	if !strings.Contains(out, "size: a4") || !strings.Contains(out, "margin: 20mm 15mm") {
		t.Errorf("@page not driven by config:\n%s", out)
	}
}

func TestLocalDirectoryOverride(t *testing.T) {
	b := newBundle(t, func(c *bundle.Config) { c.Template = "tpl" })
	dir := filepath.Join(b.Dir, "tpl")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "template.html"), []byte("<html><style>{{.Style}}</style>DIR {{.Title}} {{.Body}}</html>"), 0o644)
	os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{color:red}"), 0o644)
	out := render(t, b, HeadInjections{})
	if !strings.Contains(out, "DIR Annual Plan") || !strings.Contains(out, "BODY-MARK") || !strings.Contains(out, "color:red") {
		t.Errorf("unexpected output: %s", out)
	}
	if strings.Contains(out, "cover-title") {
		t.Error("default template leaked into override")
	}
}

func TestLocalDirectoryWithoutStyle(t *testing.T) {
	b := newBundle(t, func(c *bundle.Config) { c.Template = "tpl" })
	dir := filepath.Join(b.Dir, "tpl")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "template.html"), []byte("X{{.Title}}"), 0o644)
	if out := render(t, b, HeadInjections{}); out != "XAnnual Plan" {
		t.Errorf("got %q", out)
	}
}

func TestLocalSingleFile(t *testing.T) {
	b := newBundle(t, func(c *bundle.Config) { c.Template = "my.html" })
	os.WriteFile(filepath.Join(b.Dir, "my.html"), []byte("FILE {{.Author}} {{.Body}}"), 0o644)
	out := render(t, b, HeadInjections{})
	if out != "FILE Ada Lovelace <p>BODY-MARK</p>" {
		t.Errorf("got %q", out)
	}
}

func TestMissingTemplate(t *testing.T) {
	for _, name := range []string{"nope", "../outside", "/abs/path"} {
		b := newBundle(t, func(c *bundle.Config) { c.Template = name })
		_, err := Render(b, "", HeadInjections{})
		if err == nil || !strings.Contains(err.Error(), `template "`+name+`" not found`) {
			t.Errorf("%q: got err %v", name, err)
		}
	}
}

func TestDirectoryWithoutTemplateHTML(t *testing.T) {
	b := newBundle(t, func(c *bundle.Config) { c.Template = "empty" })
	os.MkdirAll(filepath.Join(b.Dir, "empty"), 0o755)
	if _, err := Render(b, "", HeadInjections{}); err == nil || !strings.Contains(err.Error(), "template.html") {
		t.Errorf("got %v", err)
	}
}

func TestMetadataEscaped(t *testing.T) {
	b := newBundle(t, func(c *bundle.Config) {
		c.Title = "<script>alert(1)</script>"
		c.Author = "<script>evil()</script>"
	})
	out := render(t, b, HeadInjections{})
	if strings.Contains(out, "<script>alert(1)") || strings.Contains(out, "<script>evil()") {
		t.Errorf("metadata not escaped:\n%s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Error("expected escaped script text")
	}
}

func TestInjections(t *testing.T) {
	out := render(t, newBundle(t, nil), HeadInjections{
		HTML:   `<link rel="stylesheet" href="x.css">`,
		Style:  ".injected{color:blue}",
		Script: "window.injected=1;",
	})
	head := out[:strings.Index(out, "</head>")]
	for _, want := range []string{`<link rel="stylesheet" href="x.css">`, ".injected{color:blue}", "window.injected=1;"} {
		if !strings.Contains(head, want) {
			t.Errorf("head missing %q", want)
		}
	}
}

func TestFooterTemplate(t *testing.T) {
	if off := FooterTemplate(false); strings.Contains(off, "pageNumber") || strings.Contains(off, "totalPages") {
		t.Errorf("numbers off should have no page spans: %q", off)
	}
	on := FooterTemplate(true)
	if !strings.Contains(on, `<span class="pageNumber">`) || !strings.Contains(on, `<span class="totalPages">`) {
		t.Errorf("numbers on missing spans: %q", on)
	}
}
