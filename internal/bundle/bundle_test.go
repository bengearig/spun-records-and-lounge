package bundle

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/feature"
	_ "github.com/bengearig/spun-records-and-lounge/internal/features/math"
)

type fake string

func (f fake) Name() string { return string(f) }

func init() {
	feature.Default.Register(fake("demo"))
	feature.Default.Register(fake("other"))
}

func fixture(name string) string { return filepath.Join("testdata", name) }

func TestLoadMinimalDefaults(t *testing.T) {
	b, err := Load(fixture("minimal"))
	if err != nil {
		t.Fatal(err)
	}
	c := b.Config
	if c.Title != "T" {
		t.Errorf("title = %q", c.Title)
	}
	if c.Template != "default" {
		t.Errorf("template = %q", c.Template)
	}
	if c.Output != "minimal.pdf" {
		t.Errorf("output = %q", c.Output)
	}
	if c.Page.Size != "letter" || c.Page.Margins != "1in" || !c.Page.Numbers {
		t.Errorf("page = %+v", c.Page)
	}
	if !c.Numbering.Enabled || c.Numbering.Depth != 3 {
		t.Errorf("numbering = %+v", c.Numbering)
	}
	if len(c.Features) != 0 {
		t.Errorf("features = %v", c.Features)
	}
	abs, _ := filepath.Abs(fixture("minimal"))
	if b.Dir != abs {
		t.Errorf("dir = %q, want %q", b.Dir, abs)
	}
	if want := []string{filepath.Join(abs, "a.md")}; !reflect.DeepEqual(b.DocumentPaths, want) {
		t.Errorf("paths = %v, want %v", b.DocumentPaths, want)
	}
}

func TestLoadFull(t *testing.T) {
	b, err := Load(fixture("full"))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		Title: "Full", Subtitle: "Sub", Author: "Me", Date: "2026-01-01",
		Documents: []string{"a.md", "./b.md"},
		Template:  "custom", Output: "out/x.pdf",
		Page:      PageConfig{Size: "a4", Margins: "20mm", Numbers: false},
		Numbering: NumberingConfig{Enabled: false, Depth: 6},
		Features: map[string]FeatureConfig{
			"demo": {Enabled: false, Options: map[string]any{"mode": "fancy"}},
		},
	}
	if !reflect.DeepEqual(b.Config, want) {
		t.Errorf("config = %+v\nwant     %+v", b.Config, want)
	}
	if len(b.DocumentPaths) != 2 || filepath.Base(b.DocumentPaths[1]) != "b.md" {
		t.Errorf("paths = %v", b.DocumentPaths)
	}
}

func TestLoadFeatureForms(t *testing.T) {
	b, err := Load(fixture("feature-bool"))
	if err != nil {
		t.Fatal(err)
	}
	if f := b.Config.Features["demo"]; !f.Enabled || f.Options != nil {
		t.Errorf("demo = %+v", f)
	}
	if f := b.Config.Features["other"]; f.Enabled {
		t.Errorf("other = %+v", f)
	}

	b, err = Load(fixture("feature-mapping"))
	if err != nil {
		t.Fatal(err)
	}
	f := b.Config.Features["demo"]
	if !f.Enabled || f.Options["level"] != 2 {
		t.Errorf("demo = %+v", f)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		dir  string
		want []string
	}{
		{"empty-dir", []string{"spun.yaml", "not found"}},
		{"invalid-yaml", []string{"spun.yaml"}},
		{"unknown-field", []string{"spun.yaml", "bogus"}},
		{"no-title", []string{"spun.yaml: title"}},
		{"empty-documents", []string{"spun.yaml: documents"}},
		{"no-documents", []string{"spun.yaml: documents"}},
		{"missing-document", []string{"spun.yaml: documents[2]", "nope.md"}},
		{"escape", []string{"spun.yaml: documents[1]", "escapes"}},
		{"duplicate", []string{"spun.yaml: documents[2]", "more than once"}},
		{"dir-document", []string{"spun.yaml: documents[0]", "regular file"}},
		{"bad-size", []string{"spun.yaml: page.size", "legal"}},
		{"depth-0", []string{"spun.yaml: numbering.depth"}},
		{"depth-7", []string{"spun.yaml: numbering.depth"}},
		{"unknown-feature", []string{"spun.yaml: features.nonesuch", "unknown feature"}},
		{"feature-wrong-type", []string{"spun.yaml: features.demo", "must be a bool or a mapping"}},
	}
	for _, tc := range tests {
		t.Run(tc.dir, func(t *testing.T) {
			_, err := Load(fixture(tc.dir))
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

func TestLoadSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "bundle")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.md")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link.md")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	cfg := "title: T\ndocuments: [link.md]\n"
	if err := os.WriteFile(filepath.Join(dir, "spun.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "spun.yaml: documents[0]") || !strings.Contains(err.Error(), "escapes") {
		t.Errorf("err = %v", err)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "spun.yaml"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "spun.yaml: title") {
		t.Errorf("err = %v", err)
	}
}

func TestFeatureMath(t *testing.T) {
	b, err := Load(fixture("math-mapping"))
	if err != nil {
		t.Fatal(err)
	}
	if f := b.Config.Features["math"]; !f.Enabled {
		t.Errorf("math = %+v, want enabled", f)
	}

	_, err = Load(fixture("math-typo"))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, w := range []string{"features.matth", "unknown feature", `"matth"`, "math"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q missing %q", err, w)
		}
	}
}
