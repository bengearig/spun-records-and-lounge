package render

import (
	"strings"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
)

func assembleFixture(t *testing.T, name string) ([]Doc, error) {
	t.Helper()
	b, err := bundle.Load("testdata/links/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return Assemble(b, nil)
}

func TestLinksResolved(t *testing.T) {
	docs, err := assembleFixture(t, "ok")
	if err != nil {
		t.Fatal(err)
	}
	a, b := docs[0].HTML, docs[1].HTML
	for _, tc := range []struct {
		name, html, want string
	}{
		{"same-document fragment", a, `<a href="#a--details">d</a>`},
		{"cross-document fragment", a, `<a href="#b--deep">b</a>`},
		{"document without fragment", a, `<a href="#b--bee">b</a>`},
		{"self link with fragment", a, `<a href="#a--details">self</a>`},
		{"full id", a, `<a href="#b--deep">full</a>`},
		{"custom id", a, `<a href="#top-id">c</a>`},
		{"empty link text becomes an xref", a, `<a href="#b--deep">Section 2.1, p. ?</a>`},
		{"external http", a, `<a href="https://example.com/x.md#y">h</a>`},
		{"mailto", a, `<a href="mailto:a@example.com">m</a>`},
		{"parent directory", b, `<a href="#a--details">a</a>`},
		{"parent directory without fragment", b, `<a href="#a--alpha">a</a>`},
		{"dot-relative self", b, `<a href="#b--deep">a</a>`},
	} {
		if !strings.Contains(tc.html, tc.want) {
			t.Errorf("%s: missing %s in:\n%s", tc.name, tc.want, tc.html)
		}
	}
}

func TestLinksErrors(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		want    []string
	}{
		{"not-listed", []string{`a.md:3: link "extra.md": file is not listed in documents`}},
		{"missing-file", []string{`a.md:3: link "nope.md#y": file not found`}},
		{"missing-heading", []string{`a.md:3: link "#nothing": heading "nothing" not found in a.md`}},
		{"non-markdown", []string{`a.md:3: link "logo.png": not a markdown document`}},
		{"ambiguous", []string{`a.md:7: link "#same": heading "same" is ambiguous in a.md (line 3, line 5)`}},
		{"multi", []string{
			`a.md:3: link "#nothing": heading "nothing" not found`,
			`a.md:5: link "gone.md": file not found`,
			`b.md:3: link "a.md#absent": heading "absent" not found in a.md`,
		}},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			_, err := assembleFixture(t, tc.fixture)
			if err == nil {
				t.Fatal("expected error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q missing %q", err, w)
				}
			}
		})
	}
}

func TestLinksAmbiguousSuffixIDResolves(t *testing.T) {
	docs, err := assembleFixture(t, "ambiguous")
	if err == nil {
		t.Fatalf("expected ambiguity error, got %v", docs)
	}
	if strings.Contains(err.Error(), `"same-1"`) {
		t.Errorf("suffixed slug should be unique: %v", err)
	}
}
