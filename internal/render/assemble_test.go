package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

func makeBundle(t *testing.T, files map[string]string, order ...string) *bundle.Bundle {
	t.Helper()
	dir := t.TempDir()
	yml := "title: T\ndocuments:\n"
	for _, name := range order {
		yml += "  - " + name + "\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(files[name]), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "spun.yaml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := bundle.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAssembleOrder(t *testing.T) {
	b := makeBundle(t, map[string]string{"b.md": "# Bee\n", "a.md": "# Ay\n"}, "b.md", "a.md")
	docs, err := Assemble(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[0].Path != "b.md" || docs[1].Path != "a.md" {
		t.Fatalf("docs = %+v", docs)
	}
	if !strings.Contains(docs[0].HTML, `<h1 id="b--bee"><span class="heading-number">1</span> Bee</h1>`) {
		t.Errorf("html = %q", docs[0].HTML)
	}
	if string(docs[1].Source) != "# Ay\n" {
		t.Errorf("source = %q", docs[1].Source)
	}
	body := Body(docs)
	if strings.Index(body, `data-source="b.md"`) > strings.Index(body, `data-source="a.md"`) {
		t.Errorf("body order wrong: %s", body)
	}
	if strings.Count(body, "break-before: page") != 1 {
		t.Errorf("want exactly one page break: %s", body)
	}
	if strings.Contains(body, `data-source="b.md" style`) {
		t.Errorf("first section must not break: %s", body)
	}
}

func TestAssembleGFMTable(t *testing.T) {
	b := makeBundle(t, map[string]string{"a.md": "| a | b |\n|---|---|\n| 1 | 2 |\n\n~~x~~\n"}, "a.md")
	docs, err := Assemble(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(docs[0].HTML, "<table>") || !strings.Contains(docs[0].HTML, "<del>x</del>") {
		t.Errorf("html = %q", docs[0].HTML)
	}
}

func TestAssembleFootnote(t *testing.T) {
	b := makeBundle(t, map[string]string{"a.md": "Text[^1].\n\n[^1]: Note.\n"}, "a.md")
	docs, err := Assemble(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(docs[0].HTML, "footnote") || !strings.Contains(docs[0].HTML, "Note.") {
		t.Errorf("html = %q", docs[0].HTML)
	}
}

func TestAssembleEmptyDocument(t *testing.T) {
	b := makeBundle(t, map[string]string{"a.md": ""}, "a.md")
	docs, err := Assemble(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if docs[0].HTML != "" {
		t.Errorf("html = %q", docs[0].HTML)
	}
	want := "<section class=\"document\" data-source=\"a.md\">\n</section>\n"
	if got := Body(docs); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestAssembleFrontMatterRejected(t *testing.T) {
	b := makeBundle(t, map[string]string{"a.md": "ok\n", "intro.md": "---\ntitle: x\n---\n# Hi\n"}, "a.md", "intro.md")
	_, err := Assemble(b, nil)
	if err == nil || !strings.Contains(err.Error(), "intro.md") {
		t.Fatalf("err = %v", err)
	}
}

func TestAssembleThematicBreakAllowed(t *testing.T) {
	b := makeBundle(t, map[string]string{"a.md": "text\n\n---\n\nmore\n"}, "a.md")
	if _, err := Assemble(b, nil); err != nil {
		t.Fatal(err)
	}
}

type shout struct{}

func (shout) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(util.Prioritized(shout{}, 100)))
}

func (shout) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := n.(*ast.Text); ok && entering {
			seg := t.Segment
			t.Segment = text.NewSegment(seg.Start, seg.Start)
			s := ast.NewString([]byte(strings.ToUpper(string(seg.Value(reader.Source())))))
			s.SetRaw(true)
			t.Parent().ReplaceChild(t.Parent(), t, s)
		}
		return ast.WalkContinue, nil
	})
}

func TestAssembleExtraExtension(t *testing.T) {
	b := makeBundle(t, map[string]string{"a.md": "hello\n"}, "a.md")
	docs, err := Assemble(b, []goldmark.Extender{shout{}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(docs[0].HTML, "HELLO") {
		t.Errorf("html = %q", docs[0].HTML)
	}
}

func TestAssembleRawHTMLNotPassedThrough(t *testing.T) {
	b := makeBundle(t, map[string]string{"a.md": "<script>alert(1)</script>\n\ninline <b>x</b>\n"}, "a.md")
	docs, err := Assemble(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(docs[0].HTML, "<script>") || strings.Contains(docs[0].HTML, "<b>") {
		t.Errorf("html = %q", docs[0].HTML)
	}
}
