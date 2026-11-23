package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copyBundle(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "plan")
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	return dst
}

func readPDF(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("%s does not start with %%PDF-", path)
	}
	return data
}

func TestRenderEndToEnd(t *testing.T) {
	const fixture = "testdata/minimal-bundle"

	t.Run("output flag", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "flag.pdf")
		code, stdout, stderr := runCLI("docs", "render", fixture, "-o", out)
		if code != 0 {
			t.Fatalf("exit code = %d, stderr = %q", code, stderr)
		}
		if strings.TrimSpace(stdout) != out {
			t.Fatalf("stdout = %q, want %q", stdout, out)
		}
		data := readPDF(t, out)
		if !bytes.Contains(data, []byte("/Title (Minimal Bundle Title)")) {
			t.Error("pdf does not contain the title")
		}
	})

	t.Run("flag overrides output field", func(t *testing.T) {
		dir := copyBundle(t, fixture)
		cfg := filepath.Join(dir, "spun.yaml")
		raw, _ := os.ReadFile(cfg)
		if err := os.WriteFile(cfg, append(raw, "output: from-config.pdf\n"...), 0o644); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(t.TempDir(), "flag.pdf")
		if code, _, stderr := runCLI("docs", "render", dir, "-o", out); code != 0 {
			t.Fatalf("exit code = %d, stderr = %q", code, stderr)
		}
		readPDF(t, out)
		if _, err := os.Stat(filepath.Join(dir, "from-config.pdf")); !os.IsNotExist(err) {
			t.Errorf("config output written despite --output: %v", err)
		}
	})

	t.Run("output field relative to bundle with nested dirs", func(t *testing.T) {
		dir := copyBundle(t, fixture)
		cfg := filepath.Join(dir, "spun.yaml")
		raw, _ := os.ReadFile(cfg)
		if err := os.WriteFile(cfg, append(raw, "output: build/nested/out.pdf\n"...), 0o644); err != nil {
			t.Fatal(err)
		}
		code, stdout, stderr := runCLI("docs", "render", dir)
		if code != 0 {
			t.Fatalf("exit code = %d, stderr = %q", code, stderr)
		}
		want := filepath.Join(dir, "build", "nested", "out.pdf")
		if strings.TrimSpace(stdout) != want {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
		readPDF(t, want)
	})

	t.Run("default name", func(t *testing.T) {
		dir := copyBundle(t, fixture)
		code, stdout, stderr := runCLI("docs", "render", dir)
		if code != 0 {
			t.Fatalf("exit code = %d, stderr = %q", code, stderr)
		}
		want := filepath.Join(dir, "plan.pdf")
		if strings.TrimSpace(stdout) != want {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
		readPDF(t, want)
	})

	t.Run("failure leaves no output", func(t *testing.T) {
		dir := copyBundle(t, fixture)
		out := filepath.Join(t.TempDir(), "sub", "fail.pdf")
		if err := os.Remove(filepath.Join(dir, "details.md")); err != nil {
			t.Fatal(err)
		}
		code, _, stderr := runCLI("docs", "render", dir, "-o", out)
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		if stderr == "" {
			t.Error("expected an error on stderr")
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Errorf("output exists after failure: %v", err)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 2 {
			t.Errorf("bundle dir has stray files: %v", entries)
		}
	})
}
