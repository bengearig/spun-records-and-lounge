package render

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
)

func ids(hs []Heading) []string {
	var out []string
	for _, h := range hs {
		out = append(out, h.ID)
	}
	return out
}

func numbers(hs []Heading) []string {
	var out []string
	for _, h := range hs {
		out = append(out, h.Number)
	}
	return out
}

func assemble(t *testing.T, files map[string]string, order ...string) []Doc {
	t.Helper()
	docs, err := Assemble(makeBundle(t, files, order...), nil)
	if err != nil {
		t.Fatal(err)
	}
	return docs
}

func TestAnchorsIDs(t *testing.T) {
	docs := assemble(t, map[string]string{
		"Plan Doc.md": "# Overview\n\n## Overview\n\n## Overview\n\n# Café Ünï 日本\n\n# Hello, World! (v2.0) -- *ok*_x\n\n# !!!\n",
	}, "Plan Doc.md")
	want := []string{
		"plan-doc--overview",
		"plan-doc--overview-1",
		"plan-doc--overview-2",
		"plan-doc--café-ünï-日本",
		"plan-doc--hello-world-v20----ok_x",
		"plan-doc--heading",
	}
	if got := ids(docs[0].Headings); !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %q, want %q", got, want)
	}
	if !strings.Contains(docs[0].HTML, `id="plan-doc--overview-1"`) {
		t.Errorf("html missing id: %s", docs[0].HTML)
	}
	if h := docs[0].Headings[0]; h.Text != "Overview" || h.Level != 1 || h.DocPath != "Plan Doc.md" || h.Line != 1 {
		t.Errorf("heading = %+v", h)
	}
	if h := docs[0].Headings[1]; h.Line != 3 || h.Level != 2 {
		t.Errorf("heading = %+v", h)
	}
}

func TestAnchorsDuplicateAcrossDocuments(t *testing.T) {
	docs := assemble(t, map[string]string{
		"a b.md": "# Intro\n",
		"a-b.md": "# Intro\n",
	}, "a b.md", "a-b.md")
	if got := ids(docs[0].Headings); got[0] != "a-b--intro" {
		t.Errorf("first = %q", got)
	}
	if got := ids(docs[1].Headings); got[0] != "a-b--intro-1" {
		t.Errorf("second = %q", got)
	}
}

func TestAnchorsStable(t *testing.T) {
	b := makeBundle(t, map[string]string{"a.md": "# X\n\n## X\n", "b.md": "# X\n"}, "a.md", "b.md")
	first, err := Assemble(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Assemble(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("renders differ:\n%+v\n%+v", first, second)
	}
}

func TestAnchorsCustomID(t *testing.T) {
	docs := assemble(t, map[string]string{
		"a.md": "# Pricing {#pricing}\n\n## Pricing\n",
		"b.md": "# Other {#other}\n",
	}, "a.md", "b.md")
	if got, want := ids(docs[0].Headings), []string{"pricing", "a--pricing"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %q, want %q", got, want)
	}
	if docs[0].Headings[0].Text != "Pricing" {
		t.Errorf("text = %q", docs[0].Headings[0].Text)
	}
	if !strings.Contains(docs[1].HTML, `id="other"`) {
		t.Errorf("html = %s", docs[1].HTML)
	}
}

func TestAnchorsCustomIDWinsOverGenerated(t *testing.T) {
	docs := assemble(t, map[string]string{"a.md": "# Foo\n\n# Bar {#a--foo}\n"}, "a.md")
	if got, want := ids(docs[0].Headings), []string{"a--foo-1", "a--foo"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %q, want %q", got, want)
	}
}

func TestAnchorsDuplicateCustomIDErrors(t *testing.T) {
	b := makeBundle(t, map[string]string{
		"a.md": "# A {#same}\n",
		"b.md": "# Intro\n\n# B {#same}\n",
	}, "a.md", "b.md")
	_, err := Assemble(b, nil)
	if err == nil || !strings.Contains(err.Error(), "b.md:3") || !strings.Contains(err.Error(), `"same"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestAnchorsNumbering(t *testing.T) {
	docs := assemble(t, map[string]string{
		"a.md": "# A\n\n## A1\n\n## A2\n\n### A2a\n\n# B\n\n### B-deep\n\n## B1\n",
		"b.md": "# C\n\n## C1\n",
	}, "a.md", "b.md")
	want := []string{"1", "1.1", "1.2", "1.2.1", "2", "2.1", "2.2"}
	if got := numbers(docs[0].Headings); !reflect.DeepEqual(got, want) {
		t.Errorf("numbers = %q, want %q", got, want)
	}
	if got, want := numbers(docs[1].Headings), []string{"3", "3.1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("second doc numbers = %q, want %q", got, want)
	}
	if !strings.Contains(docs[0].HTML, `<h2 id="a--a1"><span class="heading-number">1.1</span> A1</h2>`) {
		t.Errorf("html = %s", docs[0].HTML)
	}
}

func TestAnchorsNumberingResetAfterShallower(t *testing.T) {
	docs := assemble(t, map[string]string{"a.md": "# A\n\n## x\n\n## y\n\n# B\n\n## z\n"}, "a.md")
	if got, want := numbers(docs[0].Headings), []string{"1", "1.1", "1.2", "2", "2.1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("numbers = %q, want %q", got, want)
	}
}

func writeConfig(t *testing.T, extra string, files map[string]string, order ...string) []Doc {
	t.Helper()
	b := makeBundle(t, files, order...)
	dir := b.Dir
	yml := "title: T\ndocuments:\n"
	for _, n := range order {
		yml += "  - " + n + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "spun.yaml"), []byte(yml+extra), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := bundle.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	docs, err := Assemble(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	return docs
}

func TestAnchorsDepthLimit(t *testing.T) {
	docs := writeConfig(t, "numbering:\n  depth: 2\n", map[string]string{"a.md": "# A\n\n## B\n\n### C\n"}, "a.md")
	if got, want := numbers(docs[0].Headings), []string{"1", "1.1", ""}; !reflect.DeepEqual(got, want) {
		t.Errorf("numbers = %q, want %q", got, want)
	}
	if !strings.Contains(docs[0].HTML, `<h3 id="a--c">C</h3>`) {
		t.Errorf("html = %s", docs[0].HTML)
	}
}

func TestAnchorsNumberingDisabled(t *testing.T) {
	docs := writeConfig(t, "numbering:\n  enabled: false\n", map[string]string{"a.md": "# A\n\n## B\n"}, "a.md")
	if strings.Contains(docs[0].HTML, "heading-number") {
		t.Errorf("html = %s", docs[0].HTML)
	}
	if got := numbers(docs[0].Headings); got[0] != "" || got[1] != "" {
		t.Errorf("numbers = %q", got)
	}
	if !strings.Contains(docs[0].HTML, `<h1 id="a--a">A</h1>`) {
		t.Errorf("html = %s", docs[0].HTML)
	}
}

func TestAnchorsDocumentStartsDeeper(t *testing.T) {
	docs := assemble(t, map[string]string{
		"a.md": "## First\n\n### Sub\n\n## Second\n",
		"b.md": "# Top\n\n## Under\n",
	}, "a.md", "b.md")
	if got, want := numbers(docs[0].Headings), []string{"1", "1.1", "2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("numbers = %q, want %q", got, want)
	}
	if got, want := numbers(docs[1].Headings), []string{"3", "3.1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("second doc numbers = %q, want %q", got, want)
	}
}
