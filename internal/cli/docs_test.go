package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocsHelpListsRender(t *testing.T) {
	code, out, _ := runCLI("docs", "--help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "render") {
		t.Fatalf("docs help missing render: %q", out)
	}
}

func TestDocsAloneShowsHelp(t *testing.T) {
	code, out, _ := runCLI("docs")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "render") {
		t.Fatalf("docs output missing render: %q", out)
	}
}

func TestDocsRenderHelpListsFlags(t *testing.T) {
	code, out, _ := runCLI("docs", "render", "--help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	for _, want := range []string{"--output", "-o,", "--chromium"} {
		if !strings.Contains(out, want) {
			t.Fatalf("render help missing %q: %q", want, out)
		}
	}
}

func TestRenderArgs(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"zero args", []string{"docs", "render"}, "accepts 1 arg(s), received 0"},
		{"two args", []string{"docs", "render", dir, dir}, "accepts 1 arg(s), received 2"},
		{"missing dir", []string{"docs", "render", filepath.Join(dir, "nope")}, "does not exist"},
		{"file path", []string{"docs", "render", file}, "is a file, not a directory"},
		{"valid dir", []string{"docs", "render", dir}, "render is not implemented yet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, errOut := runCLI(tt.args...)
			if code == 0 {
				t.Fatal("exit code = 0, want non-zero")
			}
			if !strings.Contains(errOut, tt.want) {
				t.Fatalf("stderr = %q, want it to contain %q", errOut, tt.want)
			}
		})
	}
}
