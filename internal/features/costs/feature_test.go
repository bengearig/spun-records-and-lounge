package costs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/feature"
)

func TestRegistered(t *testing.T) {
	f, err := feature.Default.Lookup(Name)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.(feature.Configurable); !ok {
		t.Fatalf("%T does not implement feature.Configurable", f)
	}
	if _, ok := f.(feature.MarkdownExtender); ok {
		t.Fatalf("%T must not implement feature.MarkdownExtender", f)
	}
}

func TestConfigure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "costs.yaml", exampleYAML)
	writeFile(t, dir, "bad.yaml", "levels: []\n")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(dir, "costs.yaml")

	tests := []struct {
		name    string
		options map[string]any
		wantErr string
	}{
		{"valid", map[string]any{"file": "costs.yaml"}, ""},
		{"missing file", map[string]any{}, "file is required"},
		{"nil options", nil, "file is required"},
		{"empty file", map[string]any{"file": ""}, "file is required"},
		{"non-string file", map[string]any{"file": 3}, "file must be a string"},
		{"absolute", map[string]any{"file": abs}, "file must be relative to the bundle"},
		{"escape", map[string]any{"file": "../escape.yaml"}, "file escapes the bundle"},
		{"nonexistent", map[string]any{"file": "nope.yaml"}, `file "nope.yaml" does not exist`},
		{"directory", map[string]any{"file": "sub"}, `file "sub" is not a regular file`},
		{"invalid data", map[string]any{"file": "bad.yaml"}, "bad.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Feature{}.Configure(feature.BundleInfo{Dir: dir, Options: tt.options})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Name() != Name {
				t.Fatalf("Name = %q", got.Name())
			}
			m, ok := got.(feature.MarkdownExtender)
			if !ok {
				t.Fatalf("%T does not implement feature.MarkdownExtender", got)
			}
			if n := len(m.Extenders()); n != 1 {
				t.Fatalf("Extenders = %d, want 1", n)
			}
		})
	}
}
