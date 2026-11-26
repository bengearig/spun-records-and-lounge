package costs

import (
	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/bengearig/spun-records-and-lounge/internal/feature"
	"github.com/bengearig/spun-records-and-lounge/internal/render"
	"os"
	"path/filepath"
	"testing"
)

const exampleYAML = `levels:
  - id: base
    name: Base
    reserve_months: {low: 1, expected: 1, high: 2}
  - id: plus
    name: Plus
    reserve_months: {low: 0, expected: 1, high: 1}
categories:
  - {id: a, name: Alpha}
  - {id: b, name: Beta}
items:
  - {id: a1, category: a, level: base, name: A one, low: 100, expected: 200, high: 300, status: estimate}
  - {id: a2, category: a, level: plus, name: A two, low: 50, expected: 60, high: 70, status: estimate, owner_input: true}
  - {id: b1, category: b, level: base, name: B one, low: 1000.50, expected: 1500, high: 2000, status: estimate}
monthly:
  - {id: m1, level: base, name: M one, low: 10, expected: 20, high: 30, status: estimate}
  - {id: m2, level: plus, name: M two, low: 5, expected: 5, high: 5, status: estimate}
`

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// assemble renders markdown as the only document of a bundle with costs enabled.
func assemble(t *testing.T, data, markdown string) ([]render.Doc, error) {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "spun.yaml", "title: T\ndocuments:\n  - a.md\nfeatures:\n  costs:\n    file: costs.yaml\n")
	writeFile(t, dir, "costs.yaml", data)
	writeFile(t, dir, "a.md", markdown)
	b, err := bundle.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	f, err := feature.Default.Lookup(Name)
	if err != nil {
		t.Fatal(err)
	}
	cf, err := f.(feature.Configurable).Configure(feature.BundleInfo{Dir: b.Dir, Options: b.Config.Features[Name].Options})
	if err != nil {
		t.Fatal(err)
	}
	return render.Assemble(b, cf.(feature.MarkdownExtender).Extenders())
}
